package challenge_testing

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TxSucceeded(
	ctx context.Context,
	tx *types.Transaction,
	addr common.Address,
	backend *backends.SimulatedBackend,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("tx failed")
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
