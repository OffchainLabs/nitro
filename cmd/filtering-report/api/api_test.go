// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/execution/gethexec"
)

func newTestStack(t *testing.T, filterSetReportingEndpoint string) *node.Node {
	t.Helper()

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := NewStack(&stackConfig, filterSetReportingEndpoint)
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
	stack := newTestStack(t, "")

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
	stack := newTestStack(t, "")

	resp, err := http.Get(stack.HTTPEndpoint() + "/readiness")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestReportCurrentFilterSetId(t *testing.T) {
	received := make(chan string, 1)
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- payload["filterSetId"]
		w.WriteHeader(http.StatusOK)
	}))
	defer externalServer.Close()

	stack := newTestStack(t, externalServer.URL)

	// Call the RPC method
	client := stack.Attach()
	defer client.Close()

	expectedId := "test-filter-set-id-123"
	if err := client.Call(nil, "filteringreport_reportCurrentFilterSetId", expectedId); err != nil {
		t.Fatal(err)
	}

	got := <-received
	if got != expectedId {
		t.Fatalf("expected filterSetId %q, got %q", expectedId, got)
	}
}

func TestReportFilteredTransactions(t *testing.T) {
	stack := newTestStack(t, "")
	client := stack.Attach()
	defer client.Close()

	reports := []gethexec.FilteredTxReport{{
		Id:          "test-id",
		TxHash:      common.HexToHash("0x1234"),
		BlockNumber: 42,
		ChainId:     "412346",
		FilteredAddresses: []gethexec.FilteredAddressRecord{{
			Address:      common.HexToAddress("0xdead"),
			FilterReason: gethexec.FilterReason{Reason: gethexec.ReasonFrom},
		}},
	}}
	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", reports); err != nil {
		t.Fatal(err)
	}
}

func TestReportFilteredTransactionsEmpty(t *testing.T) {
	stack := newTestStack(t, "")
	client := stack.Attach()
	defer client.Close()

	if err := client.Call(nil, "filteringreport_reportFilteredTransactions", []gethexec.FilteredTxReport{}); err != nil {
		t.Fatal(err)
	}
}
