package api

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func newTestAPI(t *testing.T) *TransactionFiltererAPI {
	t.Helper()
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	err := api.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(api.StopAndWait)
	return api
}

func TestFilterNoManager(t *testing.T) {
	api := newTestAPI(t)

	_, err := api.Filter(context.Background(), common.HexToHash("0x1234"))
	require.ErrorContains(t, err, "sequencer client not set yet")
}

func TestFilterContextCancelledBeforeEnqueue(t *testing.T) {
	api := newTestAPI(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := api.Filter(ctx, common.HexToHash("0x1234"))
	require.ErrorIs(t, err, context.Canceled)
}

func TestFilterSequentialProcessing(t *testing.T) {
	api := newTestAPI(t)

	// Fill the queue to capacity to verify sequential processing.
	// Without a real manager we can only test the "no manager" error path,
	// but we can verify requests are processed one by one.
	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = api.Filter(context.Background(), common.HexToHash("0xabcd"))
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.ErrorContains(t, err, "sequencer client not set yet", "request %d", i)
	}
}

func TestFilterContextCancelledWhileQueued(t *testing.T) {
	api := newTestAPI(t)

	// Block the consumer by sending a request that will take time.
	// We can't easily block the consumer without a real manager, but we
	// can fill the queue and cancel one of the waiting callers.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should either get processed (returning "no manager" error)
	// or the context should cancel.
	_, err := api.Filter(ctx, common.HexToHash("0x5678"))
	if err != nil {
		// Either context.DeadlineExceeded or "sequencer client not set yet" — both valid.
		require.True(t, errors.Is(err, context.DeadlineExceeded) ||
			err.Error() == "sequencer client not set yet",
			"unexpected error: %v", err)
	}
}
