// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/filtering-report/api"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func newTestStack(t *testing.T, queueClient *sqsclient.MockQueueClient) *rpc.Client {
	t.Helper()
	stackConfig := api.DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := api.NewStack(&stackConfig, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })
	client := stack.Attach()
	t.Cleanup(func() { client.Close() })
	return client
}

func newTestForwarder(t *testing.T, queueClient *sqsclient.MockQueueClient, endpointURL string) *Forwarder {
	t.Helper()
	config := &Config{
		Workers:            1,
		PollInterval:       time.Second,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
		CircuitBreaker: CircuitBreakerConfig{Enabled: false},
	}
	fwd, err := New(config, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}

func TestForwarder_ForwardsMessages(t *testing.T) {
	var mu sync.Mutex
	var receivedBodiesByExternalEndpoint []string
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mu.Lock()
		receivedBodiesByExternalEndpoint = append(receivedBodiesByExternalEndpoint, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)

	reports := []addressfilter.FilteredTxReport{
		{
			ID:                "",
			TxHash:            common.HexToHash("0x01"),
			TxRLP:             nil,
			FilteredAddresses: nil,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		},
		{
			ID:                "",
			TxHash:            common.HexToHash("0x02"),
			TxRLP:             nil,
			FilteredAddresses: nil,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		},
	}
	if err := filteringReportClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	forwarder := newTestForwarder(t, queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(ctx)
	forwarder.pollAndForward(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedBodiesByExternalEndpoint) != 2 {
		t.Fatalf("expected 2 forwarded messages, got %d", len(receivedBodiesByExternalEndpoint))
	}

	var expectedBodies []string
	for _, report := range reports {
		b, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		expectedBodies = append(expectedBodies, string(b))
	}
	sort.Strings(expectedBodies)
	sort.Strings(receivedBodiesByExternalEndpoint)
	for i := range expectedBodies {
		if receivedBodiesByExternalEndpoint[i] != expectedBodies[i] {
			t.Fatalf("body mismatch at index %d: expected %q, got %q", i, expectedBodies[i], receivedBodiesByExternalEndpoint[i])
		}
	}

	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 2 {
		t.Fatalf("expected 2 deletes, got %d", len(deleted))
	}
}

func TestForwarder_EndpointFailure_DoesNotDelete(t *testing.T) {
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)

	reports := []addressfilter.FilteredTxReport{
		{
			ID:                "",
			TxHash:            common.HexToHash("0x01"),
			TxRLP:             nil,
			FilteredAddresses: nil,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		},
	}
	if err := filteringReportClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	forwarder := newTestForwarder(t, queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(ctx)

	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on endpoint failure, got %d", len(deleted))
	}
}

func TestForwarder_EmptyQueue(t *testing.T) {
	externalEndpointServerCalled := false
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalEndpointServerCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}

	forwarder := newTestForwarder(t, queueClient, externalEndpointServer.URL)
	interval := forwarder.pollAndForward(t.Context())

	if externalEndpointServerCalled {
		t.Fatal("expected no HTTP calls on empty queue")
	}
	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on empty queue, got %d", len(deleted))
	}
	if interval != forwarder.config.PollInterval {
		t.Fatalf("expected poll interval %v on empty queue, got %v", forwarder.config.PollInterval, interval)
	}
}

func TestForwarder_ReceiveError(t *testing.T) {
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected no HTTP calls when Receive fails")
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{
		ReceiveErr: fmt.Errorf("simulated SQS error"),
	}

	forwarder := newTestForwarder(t, queueClient, externalEndpointServer.URL)
	interval := forwarder.pollAndForward(t.Context())

	if interval != forwarder.config.PollInterval {
		t.Fatalf("expected poll interval %v on receive error, got %v", forwarder.config.PollInterval, interval)
	}
}

func TestForwarder_DeleteError(t *testing.T) {
	var endpointCalled bool
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpointCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{
		DeleteErr: fmt.Errorf("simulated SQS delete error"),
	}
	rpcClient := newTestStack(t, queueClient)

	reports := []addressfilter.FilteredTxReport{{
		ID:                "",
		TxHash:            common.HexToHash("0x01"),
		TxRLP:             nil,
		FilteredAddresses: nil,
		BlockNumber:       0,
		ParentBlockHash:   common.Hash{},
		PositionInBlock:   0,
		FilteredAt:        time.Time{},
		IsDelayed:         false,
		DelayedReportData: nil,
	}}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	forwarder := newTestForwarder(t, queueClient, externalEndpointServer.URL)
	interval := forwarder.pollAndForward(t.Context())

	if !endpointCalled {
		t.Fatal("expected forward to succeed before delete failure")
	}
	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on delete error, got %d", len(deleted))
	}
	if interval != 0 {
		t.Fatalf("expected immediate re-poll (0) on delete error, got %v", interval)
	}
}

func testBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:         true,
		WindowDuration:  time.Minute,
		MinSamples:      2,
		OpenThreshold:   0.5,
		OpenCooldown:    time.Hour,
		HalfOpenTimeout: time.Hour,
	}
}

func newBreakerForwarder(t *testing.T, queueClient *sqsclient.MockQueueClient, endpointURL string, cb CircuitBreakerConfig) *Forwarder {
	t.Helper()
	config := &Config{
		Workers:            1,
		PollInterval:       time.Second,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
		CircuitBreaker: cb,
	}
	fwd, err := New(config, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}

func sendReports(t *testing.T, client *rpc.Client, n int) {
	t.Helper()
	reports := make([]addressfilter.FilteredTxReport, n)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{TxHash: common.BigToHash(common.Big1)}
	}
	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}
}

func TestForwarder_5xxTripsBreakerAndStopsReceives(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	queueClient := &sqsclient.MockQueueClient{}
	rpcClient := newTestStack(t, queueClient)
	sendReports(t, rpcClient, 3)

	cb := testBreakerConfig()
	fwd := newBreakerForwarder(t, queueClient, srv.URL, cb)

	// First two polls: each hits the 500 endpoint, breaker records failures.
	fwd.pollAndForward(t.Context())
	fwd.pollAndForward(t.Context())

	if got := hits.Load(); got != 2 {
		t.Fatalf("expected 2 endpoint hits before breaker trips, got %d", got)
	}
	receivesAfterTrip := queueClient.ReceiveCalls()

	// Third poll: breaker is Open (>=2 failures, rate 1.0 >= 0.5). No Receive,
	// no HTTP call.
	interval := fwd.pollAndForward(t.Context())
	if interval != fwd.config.PollInterval {
		t.Fatalf("expected PollInterval while Open, got %v", interval)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("expected no more endpoint hits after trip, got %d", got)
	}
	if queueClient.ReceiveCalls() != receivesAfterTrip {
		t.Fatalf("expected no Receive calls while Open, got %d extra", queueClient.ReceiveCalls()-receivesAfterTrip)
	}
	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 0 {
		t.Fatalf("expected 0 deletes while Open, got %d", len(deleted))
	}
}

func TestForwarder_429TripsBreaker(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	queueClient := &sqsclient.MockQueueClient{}
	rpcClient := newTestStack(t, queueClient)
	sendReports(t, rpcClient, 3)

	fwd := newBreakerForwarder(t, queueClient, srv.URL, testBreakerConfig())
	fwd.pollAndForward(t.Context())
	fwd.pollAndForward(t.Context())
	interval := fwd.pollAndForward(t.Context())

	if got := hits.Load(); got != 2 {
		t.Fatalf("expected 2 hits before 429 trips breaker, got %d", got)
	}
	if interval != fwd.config.PollInterval {
		t.Fatalf("expected PollInterval while Open, got %v", interval)
	}
}

func TestForwarder_4xxDoesNotTripBreaker(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	queueClient := &sqsclient.MockQueueClient{}
	rpcClient := newTestStack(t, queueClient)
	sendReports(t, rpcClient, 4)

	fwd := newBreakerForwarder(t, queueClient, srv.URL, testBreakerConfig())
	for i := 0; i < 4; i++ {
		fwd.pollAndForward(t.Context())
	}

	if got := hits.Load(); got != 4 {
		t.Fatalf("expected all 4 messages to hit endpoint (400 doesn't trip breaker), got %d", got)
	}
}

func TestConfig_Validate_CircuitBreaker(t *testing.T) {
	base := DefaultConfig
	base.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: time.Second}

	t.Run("default is valid", func(t *testing.T) {
		cfg := base
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected default to validate, got %v", err)
		}
	})

	t.Run("disabled skips breaker checks", func(t *testing.T) {
		cfg := base
		cfg.CircuitBreaker = CircuitBreakerConfig{Enabled: false}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("disabled breaker should skip its own validation, got %v", err)
		}
	})

	t.Run("out-of-range open threshold rejected", func(t *testing.T) {
		cfg := base
		cfg.CircuitBreaker.OpenThreshold = 1.5
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for out-of-range OpenThreshold")
		}
	})

	t.Run("zero window rejected", func(t *testing.T) {
		cfg := base
		cfg.CircuitBreaker.WindowDuration = 0
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for zero WindowDuration")
		}
	})
}
