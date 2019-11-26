// Copyright 2019 Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/golang/protobuf/proto"
	jaegerpb "github.com/jaegertracing/jaeger/model"
	ocTrancelator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
	"go.opencensus.io/trace"

	sapmpb "github.com/signalfx/sapm-proto/gen"
)

const (
	default429BackoffSeconds = 3
	headerAuthToken          = "X-SF-Token"
	headerRetryAfter         = "Retry-After"
	headerContentEncoding    = "Content-Encoding"
	headerContentType        = "Content-Type"
	headerValueGZIP          = "gzip"
	headerValueXProtobuf     = "application/x-protobuf"
)

type worker struct {
	authToken  string
	client     *http.Client
	endpoint   string
	gzipWriter *gzip.Writer
	maxRetries uint64
	doneCh     chan struct{}
}

func newWorker(client *http.Client, endpoint string, authToken string, maxRetries uint64) *worker {
	w := &worker{
		authToken:  authToken,
		client:     client,
		endpoint:   endpoint,
		gzipWriter: gzip.NewWriter(nil),
		maxRetries: maxRetries,
		doneCh:     make(chan struct{}),
	}
	return w
}

func (w *worker) export(ctx context.Context, batch *jaegerpb.Batch) error {
	_, span := trace.StartSpan(ctx, "export")
	defer span.End()

	if len(batch.Spans) == 0 {
		span.AddAttributes(trace.BoolAttribute("empty_batch", true))
		return nil
	}

	sr, err := w.prepare(ctx, batch)
	if err != nil {
		recordEncodingFailure(ctx, sr)
		span.SetStatus(trace.Status{
			Code: trace.StatusCodeInvalidArgument,
		})
		return err
	}

	ticker := backoff.NewTicker(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), w.maxRetries))

	retries := uint64(0)
	for range ticker.C {
		err := w.send(ctx, sr)
		if err == nil {
			recordSuccess(ctx, sr)
			ticker.Stop()
			break
		}

		if err.permanent {
			span.SetStatus(trace.Status{
				Code: ocTrancelator.OCStatusCodeFromHTTP(int32(err.httpCode)),
			})
			recordDrops(ctx, sr)
			ticker.Stop()
			return err
		}

		if err.retryDelaySeconds > 0 {
			time.Sleep(time.Duration(err.retryDelaySeconds) * time.Second)
		}

		if retries >= w.maxRetries {
			span.SetStatus(trace.Status{
				Code: trace.StatusCodeDeadlineExceeded,
			})
			recordDrops(ctx, sr)
			ticker.Stop()
			return err
		}
		recordSendFailure(ctx, sr)
		retries++
	}
	return nil
}

func (w *worker) send(ctx context.Context, r *sendRequest) *errSendFailure {
	_, span := trace.StartSpan(ctx, "export")
	defer span.End()

	req, _ := http.NewRequest("POST", w.endpoint, bytes.NewBuffer(r.message))
	req.Header.Add(headerContentType, headerValueXProtobuf)
	req.Header.Add(headerContentEncoding, headerValueGZIP)
	if w.authToken != "" {
		req.Header.Add(headerAuthToken, w.authToken)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInternal,
			Message: err.Error(),
		})
		return &errSendFailure{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299 {
		return nil
	}

	// Drop the batch if server thinks it is malformed in some way or client is not authorized
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("server responded with: %d", resp.StatusCode)
		span.SetStatus(trace.Status{
			Code:    ocTrancelator.OCStatusCodeFromHTTP(int32(resp.StatusCode)),
			Message: msg,
		})
		return &errSendFailure{
			err: fmt.Errorf("dropping request: %s", msg),
			httpCode:  http.StatusBadRequest,
			permanent: true,
		}
	}

	// Check if server is overwhelmed and requested to pause sending for a while.
	// Pause this worker from sending more data till the specified number of senconds in the Retry-After haeder
	// Fallback to default429BackoffSeconds if the header is not present
	// Enqueue the request for retrying later
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := default429BackoffSeconds
		if val := resp.Header.Get(headerRetryAfter); val != "" {
			if seconds, err := strconv.Atoi(val); err == nil {
				retryAfter = seconds
			}
		}
		span.SetStatus(trace.Status{Code: trace.StatusCodeResourceExhausted})
		return &errSendFailure{
			err:               errors.New("server responded with 429"),
			httpCode:          resp.StatusCode,
			retryDelaySeconds: retryAfter,
		}
	}

	// TODO: handle 301, 307, 308
	// redirects are not handled right now but should be to confirm with the spec.

	span.SetStatus(trace.Status{Code: ocTrancelator.OCStatusCodeFromHTTP(int32(resp.StatusCode))})
	return &errSendFailure{
		err:      fmt.Errorf("error exporting spans. server responded with status %d", resp.StatusCode),
		httpCode: resp.StatusCode,
	}
}

// prepare takes a jaeger batch, converts it to a SAPM PostSpansRequest, compresses it and returns a request ready
// to be sent. The method is not safe to be called from multiple goroutines. Each caller must use locks to avoid races
// and data corruption.
func (w *worker) prepare(ctx context.Context, batch *jaegerpb.Batch) (*sendRequest, error) {
	_, span := trace.StartSpan(ctx, "export")
	defer span.End()

	buf := bytes.NewBuffer([]byte{})
	w.gzipWriter.Reset(buf)

	batches := make([]*jaegerpb.Batch, 1)
	batches[0] = batch
	psr := &sapmpb.PostSpansRequest{
		Batches: batches,
	}

	encoded, err := proto.Marshal(psr)
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInvalidArgument,
			Message: "failed to marshal request",
		})
		return nil, err
	}

	_, err = w.gzipWriter.Write(encoded)
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInvalidArgument,
			Message: "failed to gzip request",
		})
		return nil, err
	}

	if err := w.gzipWriter.Close(); err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInvalidArgument,
			Message: "failed to gzip request",
		})
		return nil, err
	}
	sr := &sendRequest{
		message: buf.Bytes(),
		batches: 1,
		spans:   int64(len(batch.Spans)),
	}
	return sr, nil
}
