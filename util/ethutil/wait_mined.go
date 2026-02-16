// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package ethutil

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// WaitForTx waits for a transaction to be mined and returns its receipt.
// It tries to subscribe to new heads for near-instant notification (requires
// a WebSocket connection). If subscriptions aren't supported (HTTP), it falls
// back to polling at the given interval.
func WaitForTx(ctx context.Context, client *ethclient.Client, tx *types.Transaction, pollInterval time.Duration) (*types.Receipt, error) {
	heads := make(chan *types.Header, 1)
	sub, subErr := client.SubscribeNewHead(ctx, heads)
	if subErr != nil {
		return pollForReceipt(ctx, client, tx, pollInterval)
	}
	defer sub.Unsubscribe()

	for {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-sub.Err():
			if err != nil {
				return nil, fmt.Errorf("head subscription error while waiting for tx: %w", err)
			}
			return nil, errors.New("head subscription closed unexpectedly")
		case <-heads:
		}
	}
}

func pollForReceipt(ctx context.Context, client *ethclient.Client, tx *types.Transaction, pollInterval time.Duration) (*types.Receipt, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
