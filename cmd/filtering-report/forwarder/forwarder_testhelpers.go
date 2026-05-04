// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

type MockExternalEndpoint struct {
	server  *httptest.Server
	reports chan *addressfilter.FilteredTxReport
}

func NewMockExternalEndpoint(t *testing.T) *MockExternalEndpoint {
	t.Helper()
	m := &MockExternalEndpoint{
		reports: make(chan *addressfilter.FilteredTxReport, 100),
	}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var report addressfilter.FilteredTxReport
		if err := json.Unmarshal(body, &report); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.reports <- &report
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() { m.server.Close() })
	return m
}

func (m *MockExternalEndpoint) NextReport(t *testing.T) *addressfilter.FilteredTxReport {
	t.Helper()
	select {
	case r := <-m.reports:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for report")
		return nil
	}
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
