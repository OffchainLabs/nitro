// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"net/http"
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
	stack, err := NewStack(&stackConfig, sqsclient.NewQueueClient(&sqsclient.MockClient{}, "https://sqs.test/queue"))
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
