// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/cmd/filtering-report/api"
	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/filtering-report/signer/signertest"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func TestForwarder_ForwardsMessages(t *testing.T) {
	pemPath, endpoint := NewMockExternalEndpoint(t)

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
	forwarder := NewTestForwarder(t, queueClient, endpoint.URL(), pemPath)
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
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	filteringReportClient := stack.Attach()
	t.Cleanup(func() { filteringReportClient.Close() })

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
	if err := filteringReportClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	forwarder := NewTestForwarder(t, queueClient, externalEndpointServer.URL, pemPath)
	forwarder.pollAndForward(ctx)

	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on endpoint failure, got %d", len(deleted))
	}
}

func TestForwarder_EmptyQueue(t *testing.T) {
	pemPath, endpoint := NewMockExternalEndpoint(t)
	queueClient := &sqsclient.MockQueueClient{}

	forwarder := NewTestForwarder(t, queueClient, endpoint.URL(), pemPath)
	interval := forwarder.pollAndForward(t.Context())

	if got := endpoint.ReceivedCount(); got != 0 {
		t.Fatalf("expected no HTTP calls on empty queue, got %d", got)
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
	pemPath, endpoint := NewMockExternalEndpoint(t)
	queueClient := &sqsclient.MockQueueClient{
		ReceiveErr: fmt.Errorf("simulated SQS error"),
	}

	forwarder := NewTestForwarder(t, queueClient, endpoint.URL(), pemPath)
	interval := forwarder.pollAndForward(t.Context())

	if interval != forwarder.config.PollInterval {
		t.Fatalf("expected poll interval %v on receive error, got %v", forwarder.config.PollInterval, interval)
	}
}

func TestForwarder_FailsConstructionOnExpiredLeaf(t *testing.T) {
	opts := signertest.DefaultLeafOptions(signertest.DefaultTestSAN)
	opts.NotAfter = time.Now().Add(-time.Minute)
	pemPath, _ := signertest.SigningFixture(t, opts)

	signerCfg := signer.DefaultConfig
	signerCfg.PEMFile = pemPath
	config := &Config{
		Workers:            1,
		PollInterval:       10 * time.Millisecond,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     "http://127.0.0.1:0",
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
		Signer: signerCfg,
	}
	_, err := New(config, &sqsclient.MockQueueClient{})
	if err == nil {
		t.Fatal("expected New to fail on expired leaf")
	}
	if !strings.Contains(err.Error(), "leaf certificate") {
		t.Fatalf("expected signer leaf-certificate error, got: %v", err)
	}
}

func TestForwarder_DeleteError(t *testing.T) {
	pemPath, endpoint := NewMockExternalEndpoint(t)

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

	forwarder := NewTestForwarder(t, queueClient, endpoint.URL(), pemPath)
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
