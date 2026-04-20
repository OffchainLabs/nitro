// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
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

func newTestForwarder(queueClient *sqsclient.MockQueueClient, endpointURL string) *Forwarder {
	config := &Config{
		Workers:              1,
		PollInterval:         time.Second,
		SQSWaitTimeSeconds:   DefaultConfig.SQSWaitTimeSeconds,
		SQSVisibilityTimeout: DefaultConfig.SQSVisibilityTimeout,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
		MaxRetries:        DefaultConfig.MaxRetries,
		InitialBackoff:    time.Millisecond,
		MaxBackoff:        5 * time.Millisecond,
		BackoffMultiplier: DefaultConfig.BackoffMultiplier,
	}
	return New(config, queueClient)
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
	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
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
	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
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

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
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

// sendOneReport pushes a single filtered-tx report through the API so the
// queue has exactly one message for the forwarder to process.
func sendOneReport(t *testing.T, client *rpc.Client) {
	t.Helper()
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
	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}
}

func TestForwarder_RetriesOn5xxThenSucceeds(t *testing.T) {
	const failuresBeforeSuccess = 2
	var hits atomic.Int32
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) <= failuresBeforeSuccess {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)
	sendOneReport(t, filteringReportClient)

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(t.Context())

	if got := hits.Load(); got != failuresBeforeSuccess+1 {
		t.Fatalf("expected %d endpoint hits (retries then success), got %d", failuresBeforeSuccess+1, got)
	}
	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 1 {
		t.Fatalf("expected 1 delete after successful retry, got %d", len(deleted))
	}
}

func TestForwarder_ExhaustsRetriesOn5xxDoesNotDelete(t *testing.T) {
	var hits atomic.Int32
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)
	sendOneReport(t, filteringReportClient)

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(t.Context())

	// #nosec G115 -- small test-config values
	expected := int32(forwarder.config.MaxRetries) + 1
	if got := hits.Load(); got != expected {
		t.Fatalf("expected %d endpoint hits (all retries exhausted), got %d", expected, got)
	}
	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 0 {
		t.Fatalf("expected 0 deletes after exhausted retries, got %d", len(deleted))
	}
}

func TestForwarder_4xxNotRetried(t *testing.T) {
	var hits atomic.Int32
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)
	sendOneReport(t, filteringReportClient)

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(t.Context())

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 endpoint hit (4xx not retried), got %d", got)
	}
	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on 4xx, got %d", len(deleted))
	}
}

func TestForwarder_RetriesOnTransportError(t *testing.T) {
	// Start a server then close it so the URL points at a non-listening port;
	// every attempt will fail with a transport error, exercising the transport-
	// error retry path.
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	endpointURL := externalEndpointServer.URL
	externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	filteringReportClient := newTestStack(t, queueClient)
	sendOneReport(t, filteringReportClient)

	forwarder := newTestForwarder(queueClient, endpointURL)
	forwarder.pollAndForward(t.Context())

	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on transport-error exhaustion, got %d", len(deleted))
	}
}

func TestForwarder_ContextCancellationAbortsRetries(t *testing.T) {
	var hits atomic.Int32
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	// Stretch the backoff so we can cancel during the first sleep.
	forwarder.config.InitialBackoff = 500 * time.Millisecond
	forwarder.config.MaxBackoff = time.Second

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := forwarder.forwardToEndpoint(ctx, "{}")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// One attempt happens before the first backoff sleep; the second attempt
	// must not run once the context has been canceled.
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 endpoint hit before cancellation, got %d", got)
	}
}

func TestConfig_Validate_RejectsRetryBudgetExceedingVisibilityTimeout(t *testing.T) {
	cfg := DefaultConfig
	cfg.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: 10 * time.Second}
	cfg.MaxRetries = 5
	cfg.InitialBackoff = 200 * time.Millisecond
	cfg.MaxBackoff = 5 * time.Second
	cfg.BackoffMultiplier = 2.0
	// Worst case: 6 * 10s request timeout + ~10s of capped backoff = ~70s.
	cfg.SQSVisibilityTimeout = 15 * time.Second

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "sqs-visibility-timeout") {
		t.Fatalf("expected sqs-visibility-timeout validation error, got %v", err)
	}
}

func TestConfig_Validate_AcceptsBudgetWithinVisibilityTimeout(t *testing.T) {
	cfg := DefaultConfig
	cfg.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: 5 * time.Second}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected default config to validate, got %v", err)
	}
}

func TestConfig_Validate_RejectsZeroExternalEndpointTimeout(t *testing.T) {
	cfg := DefaultConfig
	cfg.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: 0}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "external-endpoint.timeout") {
		t.Fatalf("expected external-endpoint.timeout validation error, got %v", err)
	}
}

func TestConfig_Validate_RejectsZeroInitialBackoff(t *testing.T) {
	cfg := DefaultConfig
	cfg.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: 5 * time.Second}
	cfg.InitialBackoff = 0

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "initial-backoff") {
		t.Fatalf("expected initial-backoff validation error, got %v", err)
	}
}

func TestConfig_Validate_RejectsOverflowRetryBudget(t *testing.T) {
	cfg := DefaultConfig
	cfg.ExternalEndpoint = genericconf.HTTPClientConfig{URL: "http://example.com", Timeout: time.Hour}
	// Picking values large enough that (MaxRetries+1) * Timeout overflows
	// int64 nanoseconds forces worstCaseRetryBudget to saturate to MaxInt64,
	// which must then exceed any sane SQSVisibilityTimeout.
	cfg.MaxRetries = 1_000_000
	cfg.SQSVisibilityTimeout = 24 * time.Hour

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "sqs-visibility-timeout") {
		t.Fatalf("expected sqs-visibility-timeout validation error on overflow, got %v", err)
	}
}
