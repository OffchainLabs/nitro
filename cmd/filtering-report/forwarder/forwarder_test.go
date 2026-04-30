// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/cmd/filtering-report/api"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func TestForwarder_ForwardsMessages(t *testing.T) {
	endpoint := NewMockExternalEndpoint(t)

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	filteringReportClient := stack.Attach()
	t.Cleanup(func() { filteringReportClient.Close() })

	reports := []addressfilter.FilteredTxReport{
		{
			ID:                "",
			TxHash:            common.HexToHash("0x01"),
			TxRLP:             hexutil.Bytes{},
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			IsDelayed:         false,
			DelayedReportData: nil,
		},
		{
			ID:                "",
			TxHash:            common.HexToHash("0x02"),
			TxRLP:             hexutil.Bytes{},
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			IsDelayed:         false,
			DelayedReportData: nil,
		},
	}
	if err := filteringReportClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	forwarder := NewTestForwarder(t, queueClient, endpoint.URL())
	forwarder.pollAndForward(ctx)
	forwarder.pollAndForward(ctx)

	received := []addressfilter.FilteredTxReport{
		*endpoint.NextReport(t),
		*endpoint.NextReport(t),
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].TxHash.Cmp(reports[j].TxHash) < 0 })
	sort.Slice(received, func(i, j int) bool { return received[i].TxHash.Cmp(received[j].TxHash) < 0 })
	for i := range reports {
		if !reflect.DeepEqual(received[i], reports[i]) {
			t.Fatalf("report mismatch at index %d: expected %+v, got %+v", i, reports[i], received[i])
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
	stack := api.NewTestStack(t, queueClient)
	filteringReportClient := stack.Attach()
	t.Cleanup(func() { filteringReportClient.Close() })

	reports := []addressfilter.FilteredTxReport{
		{
			ID:                "",
			TxHash:            common.HexToHash("0x01"),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
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
	forwarder := NewTestForwarder(t, queueClient, externalEndpointServer.URL)
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

	forwarder := NewTestForwarder(t, queueClient, externalEndpointServer.URL)
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

	forwarder := NewTestForwarder(t, queueClient, externalEndpointServer.URL)
	interval := forwarder.pollAndForward(t.Context())

	if interval != forwarder.config.PollInterval {
		t.Fatalf("expected poll interval %v on receive error, got %v", forwarder.config.PollInterval, interval)
	}
}

func TestForwarder_DeleteError(t *testing.T) {
	endpoint := NewMockExternalEndpoint(t)

	queueClient := &sqsclient.MockQueueClient{
		DeleteErr: fmt.Errorf("simulated SQS delete error"),
	}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	reports := []addressfilter.FilteredTxReport{{
		ID:                "",
		TxHash:            common.HexToHash("0x01"),
		TxRLP:             nil,
		FilteredAddresses: nil,
		ChainID:           0,
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

	forwarder := NewTestForwarder(t, queueClient, endpoint.URL())
	interval := forwarder.pollAndForward(t.Context())

	received := endpoint.NextReport(t)
	if received.TxHash != reports[0].TxHash {
		t.Fatalf("expected tx hash %v, got %v", reports[0].TxHash, received.TxHash)
	}
	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on delete error, got %d", len(deleted))
	}
	if interval != 0 {
		t.Fatalf("expected immediate re-poll (0) on delete error, got %v", interval)
	}
}
