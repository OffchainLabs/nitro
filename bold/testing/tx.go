// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package challenge_testing includes all non-production code used in BoLD.
package challenge_testing

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

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

// WaitForTx to be mined. This method will trigger .Commit() on a simulated backend.
func WaitForTx(ctx context.Context, be bind.DeployBackend, tx *types.Transaction) error {
	if simulated, ok := be.(committer); ok {
		simulated.Commit()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	_, err := bind.WaitMined(ctx, be, tx)

	return err
}
