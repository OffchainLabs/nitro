// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

type MockExternalEndpoint struct {
	server  *httptest.Server
	mu      sync.Mutex
	reports []addressfilter.FilteredTxReport
}

func NewMockExternalEndpoint(t *testing.T) *MockExternalEndpoint {
	t.Helper()
	m := &MockExternalEndpoint{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var report addressfilter.FilteredTxReport
		if err := json.Unmarshal(body, &report); err != nil {
			t.Errorf("failed to unmarshal report: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		m.reports = append(m.reports, report)
		m.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() { m.server.Close() })
	return m
}

func (m *MockExternalEndpoint) Reports() []addressfilter.FilteredTxReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.reports)
}

func (m *MockExternalEndpoint) URL() string {
	return m.server.URL
}

func NewTestForwarder(t *testing.T, queueClient sqsclient.QueueClient, endpointURL string) *Forwarder {
	t.Helper()
	config := &Config{
		Workers:            1,
		PollInterval:       10 * time.Millisecond,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
	}
	fwd, err := New(config, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}
