// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func newTestStack(t *testing.T) *node.Node {
	t.Helper()

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := NewStack(&stackConfig, &sqsclient.MockQueueClient{})
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
		ChainID:           42161,
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
	stack, err := NewStack(&stackConfig, mock)
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
