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
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	jaegerpb "github.com/jaegertracing/jaeger/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gen "github.com/signalfx/sapm-proto/gen"
)

var testBatch = &jaegerpb.Batch{
	Process: &jaegerpb.Process{
		ServiceName: "serviceA",
		Tags:        []jaegerpb.KeyValue{{Key: "k", VStr: "v", VType: jaegerpb.ValueType_STRING}},
	},
	Spans: []*jaegerpb.Span{{
		TraceID:       jaegerpb.NewTraceID(1, 1),
		SpanID:        jaegerpb.NewSpanID(1),
		OperationName: "op1",
	}, {
		TraceID:       jaegerpb.NewTraceID(2, 2),
		SpanID:        jaegerpb.NewSpanID(2),
		OperationName: "op2",
	}},
}

func newTestWorker(c *http.Client) *worker {
	return newWorker(c, "http://local", "")
}

func TestPrepare(t *testing.T) {
	w := newTestWorker(newMockHTTPClient(&mockTransport{}))
	sr, err := w.prepare(context.Background(), testBatch)
	assert.NoError(t, err)

	assert.Equal(t, 1, int(sr.batches))
	assert.Equal(t, 2, len(testBatch.Spans))

	gz, err := gzip.NewReader(bytes.NewReader(sr.message))
	require.NoError(t, err)
	defer gz.Close()

	contents, err := ioutil.ReadAll(gz)
	require.NoError(t, err)

	psr := &gen.PostSpansRequest{}
	err = proto.Unmarshal(contents, psr)
	require.NoError(t, err)

	require.Len(t, psr.Batches, 1)

	want := psr.Batches[0]

	if !reflect.DeepEqual(testBatch, want) {
		t.Errorf("got:\n%+v\nwant:\n%+v\n", testBatch, want)
	}

}

func TestWorkerSend(t *testing.T) {
	transport := &mockTransport{}
	w := newTestWorker(newMockHTTPClient(transport))

	ctx := context.Background()
	sr, err := w.prepare(ctx, testBatch)
	require.NoError(t, err)

	err = w.send(ctx, sr)
	require.Nil(t, err)

	received := transport.requests()
	require.Len(t, received, 1)

	r := received[0].r
	assert.Equal(t, r.Method, "POST")
	assert.Equal(t, r.Header.Get(headerContentEncoding), headerValueGZIP)
	assert.Equal(t, r.Header.Get(headerContentType), headerValueXProtobuf)
}

func TestWorkerSendErrors(t *testing.T) {
	transport := &mockTransport{statusCode: 400}
	w := newTestWorker(newMockHTTPClient(transport))

	ctx := context.Background()
	sr, err := w.prepare(ctx, testBatch)
	require.NoError(t, err)

	sendErr := w.send(ctx, sr)
	require.NotNil(t, sendErr)
	assert.Equal(t, 400, sendErr.StatusCode)
	assert.True(t, sendErr.Permanent)
	assert.Equal(t, sendErr.RetryDelaySeconds, 0)

	transport.reset(500)
	sendErr = w.send(ctx, sr)
	require.NotNil(t, sendErr)
	assert.Equal(t, 500, sendErr.StatusCode)
	assert.False(t, sendErr.Permanent)
	assert.Equal(t, sendErr.RetryDelaySeconds, 0)

	transport.reset(429)
	sendErr = w.send(ctx, sr)
	require.NotNil(t, sendErr)
	assert.Equal(t, 429, sendErr.StatusCode)
	assert.False(t, sendErr.Permanent)
	assert.Equal(t, sendErr.RetryDelaySeconds, defaultRateLimitingBackoffSeconds)

	transport.reset(429)
	transport.headers = map[string]string{headerRetryAfter: "100"}
	sendErr = w.send(ctx, sr)
	require.NotNil(t, sendErr)
	assert.Equal(t, 429, sendErr.StatusCode)
	assert.False(t, sendErr.Permanent)
	assert.Equal(t, sendErr.RetryDelaySeconds, 100)

	transport.reset(200)
	transport.err = errors.New("test error")
	sendErr = w.send(ctx, sr)
	require.NotNil(t, sendErr)
	assert.Equal(t, sendErr.Error(), "Post http://local: test error")
	assert.Equal(t, 0, sendErr.StatusCode)
	assert.False(t, sendErr.Permanent)
	assert.Equal(t, sendErr.RetryDelaySeconds, 0)
}
