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

func TestFilterContextCancelledWhileQueued(t *testing.T) {
	api := startTestAPI(t)

	blocker := make(chan struct{})
	api.queue <- func() { <-blocker }

	for range filterQueueSize {
		api.queue <- func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- api.Filter(ctx, common.Hash{2})
	}()

	close(blocker)

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Filter to return")
	}
}

func TestFilterSequentialProcessing(t *testing.T) {
	api := startTestAPI(t)

	var order []int
	const n = 5
	for i := range n {
		api.queue <- func() {
			order = append(order, i)
		}
	}

	done := make(chan struct{})
	api.queue <- func() { close(done) }

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for sequential processing")
	}

	for i, v := range order {
		if v != i {
			t.Fatalf("expected item %d at position %d, got %d (order: %v)", i, i, v, order)
		}
	}
}
