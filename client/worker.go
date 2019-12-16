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
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	jaegerpb "github.com/owais/jaegerpb"
	ocTrancelator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
	"go.opencensus.io/trace"

	sapmpb "github.com/signalfx/sapm-proto/gen"
)

const (
	defaultRateLimitingBackoffSeconds = 8
	headerAccessToken                 = "X-SF-Token"
	headerRetryAfter                  = "Retry-After"
	headerContentEncoding             = "Content-Encoding"
	headerContentType                 = "Content-Type"
	headerValueGZIP                   = "gzip"
	headerValueXProtobuf              = "application/x-protobuf"
	maxHTTPBodyReadBytes              = 256 << 10
)

// worker is not safe to be called from multiple goroutines. Each caller must use locks to avoid races
// and data corruption. In case a caller needs to export multiple requests at the same time, it should
// use one worker per request.
type worker struct {
	closeCh     chan struct{}
	client      *http.Client
	accessToken string
	endpoint    string
	gzipWriter  *gzip.Writer
	maxRetries  uint
}

func newWorker(closeCh chan struct{}, client *http.Client, endpoint string, accessToken string, maxRetries uint) *worker {
	w := &worker{
		closeCh:     closeCh,
		client:      client,
		accessToken: accessToken,
		endpoint:    endpoint,
		gzipWriter:  gzip.NewWriter(nil),
		maxRetries:  maxRetries,
	}
	return w
}

func (w *worker) export(ctx context.Context, batch *jaegerpb.Batch) error {
	_, span := trace.StartSpan(ctx, "export")
	defer span.End()

	span.AddAttributes(trace.Int64Attribute("spans", int64(len(batch.Spans))))

	if len(batch.Spans) == 0 {
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

	retries := uint(0)
	for {
		err := w.send(ctx, sr)
		if err == nil {
			recordSuccess(ctx, sr)
			break
		}
		recordSendFailure(ctx, sr)

		if err.Permanent {
			span.SetStatus(trace.Status{
				Code: ocTrancelator.OCStatusCodeFromHTTP(int32(err.StatusCode)),
			})
			recordDrops(ctx, sr)
			return err
		}

		if err.RetryDelaySeconds > 0 {
			if retries >= w.maxRetries {
				span.SetStatus(trace.Status{
					Code: trace.StatusCodeDeadlineExceeded,
				})
				recordDrops(ctx, sr)
				err.Permanent = true
				return err
			}

			select {
			case <-time.After(time.Duration(err.RetryDelaySeconds) * time.Second):
				retries++
				continue
			case <-w.closeCh:
				err.Permanent = true
				return err
			}
		}
		return err
	}
	return nil
}

func (w *worker) send(ctx context.Context, r *sendRequest) *ErrHTTPSend {
	_, span := trace.StartSpan(ctx, "export")
	defer span.End()

	req, err := http.NewRequest("POST", w.endpoint, bytes.NewBuffer(r.message))
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInvalidArgument,
			Message: err.Error(),
		})
		return &ErrHTTPSend{Err: err, Permanent: true}
	}
	req.Header.Add(headerContentType, headerValueXProtobuf)
	req.Header.Add(headerContentEncoding, headerValueGZIP)
	if w.accessToken != "" {
		req.Header.Add(headerAccessToken, w.accessToken)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeInternal,
			Message: err.Error(),
		})
		return &ErrHTTPSend{Err: err}
	}
	io.CopyN(ioutil.Discard, resp.Body, maxHTTPBodyReadBytes)
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}

	// Drop the batch if server thinks it is malformed in some way or client is not authorized
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("server responded with: %d", resp.StatusCode)
		span.SetStatus(trace.Status{
			Code:    ocTrancelator.OCStatusCodeFromHTTP(int32(resp.StatusCode)),
			Message: msg,
		})
		return &ErrHTTPSend{
			Err:        fmt.Errorf("dropping request: %s", msg),
			StatusCode: http.StatusBadRequest,
			Permanent:  true,
		}
	}

	// Check if server is overwhelmed and requested to pause sending for a while.
	// Pause this worker from sending more data till the specified number of seconds in the Retry-After header
	// Fallback to defaultRateLimitingBackoffSeconds if the header is not present
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := defaultRateLimitingBackoffSeconds
		if val := resp.Header.Get(headerRetryAfter); val != "" {
			if seconds, err := strconv.Atoi(val); err == nil {
				retryAfter = seconds
			}
		}
		span.SetStatus(trace.Status{Code: trace.StatusCodeResourceExhausted})
		return &ErrHTTPSend{
			Err:               errors.New("server responded with 429"),
			StatusCode:        resp.StatusCode,
			RetryDelaySeconds: retryAfter,
		}
	}

	// TODO: handle 301, 307, 308
	// redirects are not handled right now but should be to confirm with the spec.

	span.SetStatus(trace.Status{Code: ocTrancelator.OCStatusCodeFromHTTP(int32(resp.StatusCode))})
	return &ErrHTTPSend{
		Err:        fmt.Errorf("error exporting spans. server responded with status %d", resp.StatusCode),
		StatusCode: resp.StatusCode,
	}
}

// prepare takes a jaeger batch, converts it to a SAPM PostSpansRequest, compresses it and returns a request ready
// to be sent.
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
