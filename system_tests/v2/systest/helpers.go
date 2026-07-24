// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbutil"
	arbtest "github.com/offchainlabs/nitro/system_tests"
)

// Test-facing chain helpers. All take an explicit *ethclient.Client and
// *BlockchainTestInfo so tests can vary the underlying client (in-process,
// HTTP, WS, secondary node) without rebinding to a default.

// DefaultTxWaitTimeout is the default wait-for-receipt timeout.
const DefaultTxWaitTimeout = 30 * time.Second

// DefaultSetupTxTimeout is the wait-for-receipt timeout used during node setup.
const DefaultSetupTxTimeout = 10 * time.Second

// EnsureTxSucceededWithin polls until tx is mined and the receipt block is
// reflected in the latest header, then runs the same success checks as v1's
// EnsureTxSucceeded: a multi-gas consistency check and a revert-reason decode.
// Fails the test on timeout or revert.
func EnsureTxSucceededWithin(t testing.TB, ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) *types.Receipt {
	t.Helper()
	receipt, err := ensureTxSucceededWithin(ctx, client, tx, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func ensureTxSucceededWithin(ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	receipt, err := waitForTxWithTimeout(ctx, client, tx.Hash(), timeout)
	if err != nil {
		return nil, fmt.Errorf("wait tx %s: %w", tx.Hash(), err)
	}
	// Single-gas projection of multi-dimensional gas must match gas used; skipped
	// when multigas is disabled and reports zero.
	if !receipt.MultiGasUsed.IsZero() && receipt.GasUsed != receipt.MultiGasUsed.SingleGas() {
		return nil, fmt.Errorf("tx %s: gas used %d != multigas single gas %d", tx.Hash(), receipt.GasUsed, receipt.MultiGasUsed.SingleGas())
	}
	// nil on success; on revert, re-runs the tx as a call to surface the reason.
	if err := arbutil.DetailTxError(ctx, client, tx, receipt); err != nil {
		return nil, fmt.Errorf("transaction %s: %w", tx.Hash(), err)
	}
	return receipt, nil
}

// EnsureTxFailed waits for tx to be mined and fails the test unless it reverted.
func EnsureTxFailed(t testing.TB, ctx context.Context, client *ethclient.Client, tx *types.Transaction) *types.Receipt {
	t.Helper()
	receipt, err := waitForTxWithTimeout(ctx, client, tx.Hash(), DefaultTxWaitTimeout)
	if err != nil {
		t.Fatalf("wait tx %s: %v", tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusFailed {
		t.Fatalf("transaction %s unexpectedly succeeded", tx.Hash())
	}
	return receipt
}

// TxReceiptWithin waits for tx's receipt with no success checks — caller
// decides what to verify.
func TxReceiptWithin(ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	return waitForTxWithTimeout(ctx, client, tx.Hash(), timeout)
}

// AdvanceBlocks emits n no-op self-transfers from the Owner account through
// client, waiting each to mine. Use to move the chain head forward.
func AdvanceBlocks(t testing.TB, ctx context.Context, client *ethclient.Client, info *arbtest.BlockchainTestInfo, n int) {
	t.Helper()
	for range n {
		tx := info.PrepareTx("Owner", "Owner", info.TransferGas, big.NewInt(0), nil)
		if err := client.SendTransaction(ctx, tx); err != nil {
			t.Fatalf("AdvanceBlocks send: %v", err)
		}
		EnsureTxSucceededWithin(t, ctx, client, tx, DefaultTxWaitTimeout)
	}
}

func isTxIndexing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "indexing is in progress")
}

func dialOptional(t testing.TB, url string) *ethclient.Client {
	t.Helper()
	if url == "" {
		return nil
	}
	c, err := ethclient.Dial(url)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	return c
}

func waitForTxWithTimeout(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var receipt *types.Receipt
	err := defaultBackoff.until(ctx, func() (bool, error) {
		r, err := client.TransactionReceipt(ctx, hash)
		if errors.Is(err, ethereum.NotFound) || isTxIndexing(err) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("TransactionReceipt(%s): %w", hash, err)
		}
		header, herr := client.HeaderByNumber(ctx, nil)
		if herr != nil {
			return false, fmt.Errorf("HeaderByNumber: %w", herr)
		}
		if header.Number.Cmp(r.BlockNumber) < 0 {
			return false, nil
		}
		receipt = r
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}
