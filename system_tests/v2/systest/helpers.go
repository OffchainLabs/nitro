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
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
)

// Internal tx-wait plumbing shared by the chain handles and node builders.

// DefaultTxWaitTimeout is the default wait-for-receipt timeout.
const DefaultTxWaitTimeout = 30 * time.Second

// DefaultSetupTxTimeout is the wait-for-receipt timeout used during node setup.
const DefaultSetupTxTimeout = 10 * time.Second

// ensureTxSucceededWithin polls until tx is mined, then runs v1's
// EnsureTxSucceeded success checks.
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

func isTxIndexing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "indexing is in progress")
}

// waitForSafeBlock blocks until the chain's safe block reaches target, the
// timeout elapses, or ctx is cancelled.
func waitForSafeBlock(ctx context.Context, client *ethclient.Client, target *big.Int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		safe, err := client.HeaderByNumber(ctx, big.NewInt(int64(rpc.SafeBlockNumber)))
		if err != nil {
			return err
		}
		if safe.Number.Cmp(target) >= 0 {
			return nil
		}
		select {
		case <-time.After(headerreader.TestConfig.Dangerous.WaitForTxApprovalSafePoll):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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
