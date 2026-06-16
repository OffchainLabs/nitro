// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/filtering-report/signer/signertest"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

type MockExternalEndpoint struct {
	server       *httptest.Server
	reports      chan *addressfilter.FilteredTxReport
	requestCount atomic.Int64
}

// NewMockExternalEndpoint returns a signer whose identity the endpoint's
// verifier accepts, together with the endpoint itself.
func NewMockExternalEndpoint(t *testing.T) (sgn *signer.Signer, endpoint *MockExternalEndpoint) {
	t.Helper()
	leaf := signertest.DefaultLeafOptions(signertest.DefaultTestSAN)
	pemPath, caPath := signertest.SigningFixture(t, leaf)
	verifier, err := signertest.NewVerifier(&signertest.VerifierConfig{
		CARootPEMFile: caPath,
		ExpectedSAN:   leaf.URI,
		TimestampSkew: signertest.DefaultTimestampSkew,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	sgn, err = signer.NewSigner(&signer.Config{PEMFile: pemPath, ReloadInterval: time.Minute})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	m := &MockExternalEndpoint{
		reports: make(chan *addressfilter.FilteredTxReport, 100),
	}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestCount.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := verifier.VerifyHTTPRequest(r, body); err != nil {
			t.Errorf("verifier rejected signed request: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
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
	return sgn, m
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

func (m *MockExternalEndpoint) ReceivedCount() int {
	return int(m.requestCount.Load())
}

func (m *MockExternalEndpoint) AssertNoReport(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case r := <-m.reports:
		t.Fatalf("unexpected report: %s", r.TxHash.Hex())
	case <-time.After(within):
	}
}

func NewTestForwarder(t *testing.T, queueClient sqsclient.QueueClient, poisonQueueClient sqsclient.QueueClient, endpointURL string, sgn *signer.Signer) *Forwarder {
	t.Helper()
	config := &Config{
		Workers:            1,
		PollInterval:       10 * time.Millisecond,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
		ExternalEndpointRetryableErrorSlowdown: DefaultExternalEndpointRetryableErrorSlowdownConfig,
		Signer:                                 signer.DefaultConfig,
	}
	fwd, err := New(config, queueClient, poisonQueueClient, sgn)
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}
