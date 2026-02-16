// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package challenge_testing includes all non-production code used in BoLD.
package challenge_testing

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TxSucceeded(
	ctx context.Context,
	tx *types.Transaction,
	addr common.Address,
	backend bind.DeployBackend,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	if waitErr := WaitForTx(ctx, backend, tx); waitErr != nil {
		return errors.Wrap(waitErr, "error waiting for tx to be mined")
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("tx receipt not successful")
	}
	code, err := backend.CodeAt(ctx, addr, nil)
	if err != nil {
		return err
	}
	if len(code) == 0 {
		return errors.New("contract not deployed")
	}
	return nil
}

type committer interface {
	Commit() common.Hash
}

type headSubscriber interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

// WaitForTx waits for a transaction to be mined. It triggers .Commit() on
// simulated backends. If the backend supports head subscriptions, those are
// used for near-instant notification; otherwise it falls back to
// bind.WaitMined (1-second polling).
func WaitForTx(ctx context.Context, be bind.DeployBackend, tx *types.Transaction) error {
	if simulated, ok := be.(committer); ok {
		simulated.Commit()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	if subscriber, ok := be.(headSubscriber); ok {
		return waitMinedWithSubscription(ctx, be, subscriber, tx)
	}
	_, err := bind.WaitMined(ctx, be, tx)
	return err
}

func waitMinedWithSubscription(ctx context.Context, be bind.DeployBackend, subscriber headSubscriber, tx *types.Transaction) error {
	heads := make(chan *types.Header, 1)
	sub, err := subscriber.SubscribeNewHead(ctx, heads)
	if err != nil {
		_, err = bind.WaitMined(ctx, be, tx)
		return err
	}
	defer sub.Unsubscribe()

	for {
		receipt, err := be.TransactionReceipt(ctx, tx.Hash())
		if err == nil && receipt != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			if err != nil {
				return fmt.Errorf("head subscription error: %w", err)
			}
			return errors.New("head subscription closed unexpectedly")
		case <-heads:
		case <-time.After(time.Second):
		}
	}
}
