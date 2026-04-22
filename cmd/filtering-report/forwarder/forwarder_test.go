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
	_ = queueClient.Send(t.Context(), `{"test":"message"}`)

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
