// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func newTestStack(t *testing.T) *node.Node {
	t.Helper()
	return newTestStackWithFilterSetReporting(t, genericconf.HTTPClientConfig{})
}

func newTestStackWithFilterSetReporting(t *testing.T, filterSetReport genericconf.HTTPClientConfig) *node.Node {
	t.Helper()

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := NewStack(&stackConfig, &sqsclient.MockQueueClient{}, filterSetReport)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })
	return stack
}

func TestLiveness(t *testing.T) {
	stack := newTestStack(t)

	resp, err := http.Get(stack.HTTPEndpoint() + "/liveness")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestReadiness(t *testing.T) {
	stack := newTestStack(t)

	resp, err := http.Get(stack.HTTPEndpoint() + "/readiness")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestReportFilteredTransactions(t *testing.T) {
	stack := newTestStack(t)
	client := stack.Attach()
	defer client.Close()

	reports := []addressfilter.FilteredTxReport{{
		ID:     "test-id",
		TxHash: common.HexToHash("0x1234"),
		TxRLP:  nil,
		FilteredAddresses: []filter.FilteredAddressRecord{{
			Address:      common.HexToAddress("0xdead"),
			FilterReason: filter.FilterReason{Reason: filter.ReasonFrom, EventRuleMatch: nil},
		}},
		BlockNumber:       42,
		ParentBlockHash:   common.Hash{},
		PositionInBlock:   0,
		FilteredAt:        time.Time{},
		IsDelayed:         false,
		DelayedReportData: nil,
	}}
	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}
}

func TestReportFilteredTransactionsEmpty(t *testing.T) {
	stack := newTestStack(t)
	client := stack.Attach()
	defer client.Close()

	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", []addressfilter.FilteredTxReport{}); err != nil {
		t.Fatal(err)
	}
}

func TestReportFilteredTransactionsPartialFailure(t *testing.T) {
	failOnCall := 1 // fail on the 2nd Send (0-indexed)
	mock := &failingQueueClient{failOnCall: failOnCall}

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := NewStack(&stackConfig, mock, genericconf.HTTPClientConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })

	client := stack.Attach()
	defer client.Close()

	reports := make([]addressfilter.FilteredTxReport, 3)
	for i := range reports {
		reports[i] = addressfilter.FilteredTxReport{
			ID:     fmt.Sprintf("id-%d", i),
			TxHash: common.BigToHash(big.NewInt(int64(i))),
			TxRLP:  nil,
			FilteredAddresses: []filter.FilteredAddressRecord{{
				Address: common.HexToAddress("0xdead"),
				FilterReason: filter.FilterReason{
					Reason:         filter.ReasonFrom,
					EventRuleMatch: nil,
				},
			}},
			BlockNumber:       42,
			ParentBlockHash:   common.Hash{},
			PositionInBlock:   0,
			FilteredAt:        time.Time{},
			IsDelayed:         false,
			DelayedReportData: nil,
		}
	}

	err = client.Call(nil, "filteringreport_reportFilteredTransactions", reports)
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "1/3") {
		t.Fatalf("expected error to mention 1/3 failures, got: %s", errMsg)
	}
	failedTxHash := reports[failOnCall].TxHash.Hex()
	if !strings.Contains(errMsg, failedTxHash) {
		t.Fatalf("expected error to contain txHash %s, got: %s", failedTxHash, errMsg)
	}
	// All 3 sends should have been attempted (no fail-fast)
	if mock.sendCount != 3 {
		t.Fatalf("expected 3 send attempts, got %d", mock.sendCount)
	}
}

func TestReportCurrentFilterSetId_NoEndpointIsNoOp(t *testing.T) {
	stack := newTestStack(t)
	client := stack.Attach()
	defer client.Close()

	report := addressfilter.FilterSetIdReport{
		FilterSetId: uuid.New(),
		ChainId:     big.NewInt(42161),
		ReportedAt:  time.Now().UTC(),
	}
	if err := client.Call(nil, "filteringreport_reportCurrentFilterSetId", report); err != nil {
		t.Fatalf("expected no-op call to succeed, got %v", err)
	}
}

func TestReportCurrentFilterSetId_Posts(t *testing.T) {
	var received atomic.Value
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var parsed addressfilter.FilterSetIdReport
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		received.Store(parsed)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	stack := newTestStackWithFilterSetReporting(t, genericconf.HTTPClientConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})
	client := stack.Attach()
	defer client.Close()

	id := uuid.New()
	chainID := big.NewInt(42161)
	reportedAt := time.Now().UTC().Truncate(time.Second)
	report := addressfilter.FilterSetIdReport{
		FilterSetId: id,
		ChainId:     chainID,
		ReportedAt:  reportedAt,
	}
	if err := client.Call(nil, "filteringreport_reportCurrentFilterSetId", report); err != nil {
		t.Fatalf("rpc call failed: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 POST, got %d", calls.Load())
	}
	got, ok := received.Load().(addressfilter.FilterSetIdReport)
	if !ok {
		t.Fatal("server did not record a report")
	}
	if got.FilterSetId != id {
		t.Errorf("filter-set id: want %s, got %s", id, got.FilterSetId)
	}
	if got.ChainId == nil || got.ChainId.Cmp(chainID) != 0 {
		t.Errorf("chain id: want %s, got %v", chainID, got.ChainId)
	}
	if !got.ReportedAt.Equal(reportedAt) {
		t.Errorf("reported-at: want %s, got %s", reportedAt, got.ReportedAt)
	}
}

func TestReportCurrentFilterSetId_Non2xxError(t *testing.T) {
	const errorBody = "upstream is down"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(errorBody))
	}))
	defer server.Close()

	stack := newTestStackWithFilterSetReporting(t, genericconf.HTTPClientConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})
	client := stack.Attach()
	defer client.Close()

	report := addressfilter.FilterSetIdReport{
		FilterSetId: uuid.New(),
		ChainId:     big.NewInt(1),
		ReportedAt:  time.Now().UTC(),
	}
	err := client.Call(nil, "filteringreport_reportCurrentFilterSetId", report)
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
	msg := err.Error()
	if !strings.Contains(msg, server.URL) {
		t.Errorf("error should contain endpoint URL %q, got: %s", server.URL, msg)
	}
	if !strings.Contains(msg, "500") {
		t.Errorf("error should mention status 500, got: %s", msg)
	}
	if !strings.Contains(msg, errorBody) {
		t.Errorf("error should contain response body snippet %q, got: %s", errorBody, msg)
	}
}

// failingQueueClient wraps MockQueueClient and fails on a specific Send call.
type failingQueueClient struct {
	sqsclient.MockQueueClient
	failOnCall int
	sendCount  int
}

func (f *failingQueueClient) Send(ctx context.Context, body string) error {
	callNum := f.sendCount
	f.sendCount++
	if callNum == f.failOnCall {
		return fmt.Errorf("simulated SQS failure")
	}
	return f.MockQueueClient.Send(ctx, body)
}
