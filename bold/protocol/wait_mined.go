// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package protocol

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// WaitMined waits for a transaction to be mined by subscribing to new head
// notifications from the backend. This is faster than bind.WaitMined's
// hardcoded 1s polling because ChainBackend always supports head
// subscriptions. Falls back to bind.WaitMined if the subscription fails.
func WaitMined(ctx context.Context, b ChainBackend, tx *types.Transaction) (*types.Receipt, error) {
	txHash := tx.Hash()
	heads := make(chan *types.Header, 1)
	sub, err := b.SubscribeNewHead(ctx, heads)
	if err != nil {
		log.Warn("Could not subscribe to new heads for WaitMined, falling back to polling", "err", err)
		return bind.WaitMined(ctx, b, tx)
	}
	defer sub.Unsubscribe()

	for {
		receipt, err := b.TransactionReceipt(ctx, txHash)
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
