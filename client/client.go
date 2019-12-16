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
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	jaegerpb "github.com/owais/jaegerpb"
	"go.opencensus.io/stats/view"
)

const (
	idleConnTimeout     = 30 * time.Second
	tlsHandshakeTimeout = 10 * time.Second
	dialerTimeout       = 30 * time.Second
	dialerKeepAlive     = 30 * time.Second

	// default values
	defaultNumWorkers  uint = 8
	defaultMaxRetries  uint = 8
	defaultMaxIdleCons      = 100
	defaultHTTPTimeout      = 10 * time.Second
)

type sendRequest struct {
	message []byte
	spans   int64
	batches int64
}

// Client implements an HTTP sender for the SAPM protocol
type Client struct {
	numWorkers  uint
	maxIdleCons uint
	maxRetries  uint
	endpoint    string
	accessToken string
	httpClient  *http.Client
	closeCh     chan struct{}

	workers chan *worker
}

// New creates a new SAPM Client
func New(opts ...Option) (*Client, error) {
	views := metricViews()
	if err := view.Register(views...); err != nil {
		return nil, err
	}

	c := &Client{
		numWorkers:  defaultNumWorkers,
		maxRetries:  defaultMaxRetries,
		maxIdleCons: defaultMaxIdleCons,
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	if c.endpoint == "" {
		return nil, fmt.Errorf(
			"endpoint cannot be empty. WithEndpoint option must be called with a valid endpoint value",
		)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   dialerTimeout,
					KeepAlive: dialerKeepAlive,
				}).DialContext,
				MaxIdleConns:        int(c.maxIdleCons),
				MaxIdleConnsPerHost: int(c.maxIdleCons),
				IdleConnTimeout:     idleConnTimeout,
				TLSHandshakeTimeout: tlsHandshakeTimeout,
			},
		}
	}

	c.closeCh = make(chan struct{})
	c.workers = make(chan *worker, c.numWorkers)
	for i := uint(0); i < c.numWorkers; i++ {
		w := newWorker(c.closeCh, c.httpClient, c.endpoint, c.accessToken, c.maxRetries)
		c.workers <- w
	}

	return c, nil
}

// Export takes a Jaeger batch and uses one of the available workers to export it synchronously.
// It internally retries with a back-off in case of the server responding with HTTP 429.
func (sa *Client) Export(ctx context.Context, batch *jaegerpb.Batch) error {
	w := <-sa.workers
	err := w.export(ctx, batch)
	sa.workers <- w
	return err
}

// Stop waits for all inflight requests to finish and then drains the worker pool so no more work can be done.
// It returns once all workers are drained from the pool. Note that the client can accept new requests while
// Stop() waits for other requests to finish.
func (sa *Client) Stop() {
	wg := sync.WaitGroup{}
	wg.Add(int(sa.numWorkers))
	close(sa.closeCh)
	for i := uint(0); i < sa.numWorkers; i++ {
		go func() {
			<-sa.workers
			wg.Done()
		}()
	}
	wg.Wait()
}
