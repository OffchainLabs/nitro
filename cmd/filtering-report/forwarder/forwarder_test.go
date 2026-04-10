// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"context"
	"encoding/json"
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
		Workers:         1,
		PollInterval:    time.Second,
		WaitTimeSeconds: DefaultConfig.WaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
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

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(context.Background())
	forwarder.pollAndForward(context.Background())

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

	forwarder := newTestForwarder(queueClient, externalEndpointServer.URL)
	forwarder.pollAndForward(context.Background())

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
	interval := forwarder.pollAndForward(context.Background())

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
