// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
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
	forwarder := NewTestForwarder(t, queueClient, nil, endpoint.URL(), pemPath)
	var consecutiveRetryableErrors int
	forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)

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
	forwarder := NewTestForwarder(t, queueClient, nil, externalEndpointServer.URL, pemPath)
	var consecutiveRetryableErrors int
	forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)

	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes on endpoint failure, got %d", len(deleted))
	}
}

func TestForwarder_EmptyQueue(t *testing.T) {
	pemPath, endpoint := NewMockExternalEndpoint(t)
	queueClient := &sqsclient.MockQueueClient{}

	forwarder := NewTestForwarder(t, queueClient, nil, endpoint.URL(), pemPath)
	var consecutiveRetryableErrors int
	interval := forwarder.pollAndForward(t.Context(), &consecutiveRetryableErrors)

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

	forwarder := NewTestForwarder(t, queueClient, nil, endpoint.URL(), pemPath)
	var consecutiveRetryableErrors int
	interval := forwarder.pollAndForward(t.Context(), &consecutiveRetryableErrors)

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
	_, err := New(config, &sqsclient.MockQueueClient{}, nil)
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

	forwarder := NewTestForwarder(t, queueClient, nil, endpoint.URL(), pemPath)
	var consecutiveRetryableErrors int
	interval := forwarder.pollAndForward(t.Context(), &consecutiveRetryableErrors)

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

func TestForwarder_RetryableHTTPErrorSlowdown_AfterThreshold(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	forwarder := NewTestForwarder(t, queueClient, nil, externalEndpointServer.URL, pemPath)
	threshold := forwarder.config.ExternalEndpointRetryableErrorSlowdown.ConsecutiveRetryableErrors

	// Enqueue enough messages to exceed the threshold.
	reports := make([]addressfilter.FilteredTxReport, threshold)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:                "",
			TxHash:            common.HexToHash(fmt.Sprintf("0x%02x", i+1)),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	var consecutiveRetryableErrors int

	// First threshold-1 errors should return 0 (immediate re-poll).
	for i := 0; i < threshold-1; i++ {
		interval := forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
		if interval != 0 {
			t.Fatalf("call %d: expected 0 before threshold, got %v", i+1, interval)
		}
	}

	// The threshold-th error should trigger the slowdown.
	interval := forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	expected := forwarder.config.ExternalEndpointRetryableErrorSlowdown.Duration
	if interval != expected {
		t.Fatalf("expected slowdown duration %v at threshold, got %v", expected, interval)
	}
}

func TestForwarder_RetryableHTTPErrorSlowdown_ResetOnSuccess(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	var callCount atomic.Int32
	failUntil := 2 // first 2 calls fail, third succeeds
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(callCount.Add(1)) <= failUntil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	numReports := failUntil + 1 // retryable failures + 1 success
	reports := make([]addressfilter.FilteredTxReport, numReports)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:                "",
			TxHash:            common.HexToHash(fmt.Sprintf("0x%02x", i+1)),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	forwarder := NewTestForwarder(t, queueClient, nil, externalEndpointServer.URL, pemPath)
	ctx := t.Context()
	var consecutiveRetryableErrors int

	// Retryable errors.
	for i := 0; i < failUntil; i++ {
		forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	}
	if consecutiveRetryableErrors != failUntil {
		t.Fatalf("expected %d consecutive retryable errors, got %d", failUntil, consecutiveRetryableErrors)
	}

	// Success should reset counter.
	forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	if consecutiveRetryableErrors != 0 {
		t.Fatalf("expected counter reset to 0 after success, got %d", consecutiveRetryableErrors)
	}
}

func TestForwarder_RetryableHTTPErrorSlowdown_ResetOnNonRetryableError(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	var callCount atomic.Int32
	failRetryableUntil := 2 // first 2 calls return 500, third returns 400
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(callCount.Add(1)) <= failRetryableUntil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	numReports := failRetryableUntil + 1 // retryable failures + 1 non-retryable
	reports := make([]addressfilter.FilteredTxReport, numReports)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:                "",
			TxHash:            common.HexToHash(fmt.Sprintf("0x%02x", i+1)),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	forwarder := NewTestForwarder(t, queueClient, nil, externalEndpointServer.URL, pemPath)
	ctx := t.Context()
	var consecutiveRetryableErrors int

	// Retryable errors.
	for i := 0; i < failRetryableUntil; i++ {
		forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	}
	if consecutiveRetryableErrors != failRetryableUntil {
		t.Fatalf("expected %d consecutive retryable errors, got %d", failRetryableUntil, consecutiveRetryableErrors)
	}

	// Non-retryable error should reset counter.
	forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	if consecutiveRetryableErrors != 0 {
		t.Fatalf("expected counter reset to 0 after non-retryable error, got %d", consecutiveRetryableErrors)
	}
}

func TestForwarder_RetryableHTTPErrorSlowdown_NonRetryableErrorDoesNotCount(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400 - non-retryable client error
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	forwarder := NewTestForwarder(t, queueClient, nil, externalEndpointServer.URL, pemPath)
	threshold := forwarder.config.ExternalEndpointRetryableErrorSlowdown.ConsecutiveRetryableErrors

	reports := make([]addressfilter.FilteredTxReport, threshold+1)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:                "",
			TxHash:            common.HexToHash(fmt.Sprintf("0x%02x", i+1)),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	var consecutiveRetryableErrors int

	// Even after many 400 errors, should never trigger slowdown.
	for i := 0; i <= threshold; i++ {
		interval := forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
		if interval != 0 {
			t.Fatalf("call %d: expected 0 for non-retryable error, got %v", i+1, interval)
		}
	}
	if consecutiveRetryableErrors != 0 {
		t.Fatalf("expected 0 consecutive retryable errors for non-retryable errors, got %d", consecutiveRetryableErrors)
	}
}

func TestForwarder_PoisonQueue_NonRetryableErrorSentToPoisonQueue(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	poisonQueueClient := &sqsclient.MockQueueClient{}

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

	forwarder := NewTestForwarder(t, queueClient, poisonQueueClient, externalEndpointServer.URL, pemPath)
	var consecutiveRetryableErrors int
	forwarder.pollAndForward(t.Context(), &consecutiveRetryableErrors)

	// Message should have been sent to poison queue.
	sentBodies := poisonQueueClient.SentBodies()
	if len(sentBodies) != 1 {
		t.Fatalf("expected 1 message sent to poison queue, got %d", len(sentBodies))
	}
	var got addressfilter.FilteredTxReport
	if err := json.Unmarshal([]byte(sentBodies[0]), &got); err != nil {
		t.Fatalf("decode poison queue body: %v", err)
	}
	if got.TxHash != reports[0].TxHash {
		t.Fatalf("poison queue body TxHash mismatch: expected %s, got %s", reports[0].TxHash.Hex(), got.TxHash.Hex())
	}

	// Message should have been deleted from main queue.
	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 1 {
		t.Fatalf("expected 1 delete from main queue after poison queue send, got %d", len(deleted))
	}
}

func TestForwarder_TransportError_FallsThroughToSlowdown(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	// Stand the server up just to get a real URL, then immediately close it
	// so subsequent requests fail at the transport layer (connection refused).
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	endpointURL := externalEndpointServer.URL
	externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	poisonQueueClient := &sqsclient.MockQueueClient{}
	stack := api.NewTestStack(t, queueClient)
	rpcClient := stack.Attach()
	t.Cleanup(func() { rpcClient.Close() })

	forwarder := NewTestForwarder(t, queueClient, poisonQueueClient, endpointURL, pemPath)
	threshold := forwarder.config.ExternalEndpointRetryableErrorSlowdown.ConsecutiveRetryableErrors

	reports := make([]addressfilter.FilteredTxReport, threshold)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:                "",
			TxHash:            common.HexToHash(fmt.Sprintf("0x%02x", i+1)),
			TxRLP:             nil,
			FilteredAddresses: nil,
			ChainID:           0,
			BlockNumber:       0,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}
	if err := rpcClient.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	var consecutiveRetryableErrors int

	// First threshold-1 transport errors should return 0 (immediate re-poll).
	for i := 0; i < threshold-1; i++ {
		interval := forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
		if interval != 0 {
			t.Fatalf("call %d: expected 0 before threshold, got %v", i+1, interval)
		}
	}

	// The threshold-th transport error must trigger the slowdown.
	interval := forwarder.pollAndForward(ctx, &consecutiveRetryableErrors)
	expected := forwarder.config.ExternalEndpointRetryableErrorSlowdown.Duration
	if interval != expected {
		t.Fatalf("expected slowdown duration %v at threshold, got %v", expected, interval)
	}

	if sent := poisonQueueClient.SentBodies(); len(sent) != 0 {
		t.Fatalf("expected 0 sends to poison queue on transport error, got %d", len(sent))
	}
	if deleted := queueClient.DeletedReceiptHandles(); len(deleted) != 0 {
		t.Fatalf("expected 0 deletes from main queue on transport error, got %d", len(deleted))
	}
}

func TestForwarder_PoisonQueue_SendFailureLeavesMessageInQueue(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer externalEndpointServer.Close()

	queueClient := &sqsclient.MockQueueClient{}
	poisonQueueClient := &sqsclient.MockQueueClient{
		SendErr: fmt.Errorf("simulated poison queue send error"),
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

	forwarder := NewTestForwarder(t, queueClient, poisonQueueClient, externalEndpointServer.URL, pemPath)
	var consecutiveRetryableErrors int
	forwarder.pollAndForward(t.Context(), &consecutiveRetryableErrors)

	// Poison queue send failed, so message should NOT have been deleted from main queue.
	deleted := queueClient.DeletedReceiptHandles()
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deletes when poison queue send fails, got %d", len(deleted))
	}
}
