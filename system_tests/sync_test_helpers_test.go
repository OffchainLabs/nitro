// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// waitForBlocksToCatchup has a time "limit" factor to limit running this function forever in weird cases such as running with race detection in nightly CI
func waitForBlocksToCatchup(ctx context.Context, t *testing.T, clientA *ethclient.Client, clientB *ethclient.Client, limit time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
			headerA, err := clientA.HeaderByNumber(ctx, nil)
			Require(t, err)
			headerB, err := clientB.HeaderByNumber(ctx, nil)
			Require(t, err)
			if headerA.Number.Cmp(headerB.Number) == 0 {
				return
			}
		case <-time.After(limit):
			t.Fatal("waitForBlocksToCatchup didnt finish")
		}
	}
}

func createTransactionTillBatchCount(ctx context.Context, t *testing.T, builder *NodeBuilder, finalCount uint64) {
	// We run the loop for 6000 iterations ~ maximum of 10 minutes of run time before failing. This is to avoid
	// running this function forever in weird cases such as running with race detection in nightly CI
	for i := uint64(0); i < 6000; i++ {
		Require(t, ctx.Err())
		tx := builder.L2Info.PrepareTx("Faucet", "BackgroundUser", builder.L2Info.TransferGas, big.NewInt(1), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		count, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		if count > finalCount {
			return
		}
		time.Sleep(100 * time.Millisecond) // give some time for other components (reader/tracker) to read the batches from L1
	}
	t.Fatal("createTransactionTillBatchCount didnt finish")
}
