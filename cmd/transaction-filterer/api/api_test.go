// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func startTestAPI(t *testing.T) *TransactionFiltererAPI {
	t.Helper()
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := api.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(api.StopAndWait)
	return api
}

func TestFilterContextCancelledWhenQueueFull(t *testing.T) {
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{})

	for range filterQueueSize {
		api.queue <- common.Hash{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := api.Filter(ctx, common.Hash{2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestFilterConsumesFromQueue(t *testing.T) {
	api := startTestAPI(t)

	const n = 5
	for i := range n {
		if err := api.Filter(context.Background(), common.BytesToHash([]byte{byte(i)})); err != nil {
			t.Fatal(err)
		}
	}

	timeout := time.After(5 * time.Second)
	for len(api.queue) > 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for queue to drain")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
