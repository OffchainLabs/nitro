// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestRegularBlocksAreMaxBlockSpeedApart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// MaxBlockSpeed is deliberately large so the measured spacing dwarfs
	// scheduling jitter, keeping the assertions non-flaky.
	const maxBlockSpeed = time.Second

	// This test measures wall-clock spacing between blocks, so it must not run in
	// parallel with other tests, whose CPU contention would skew the timing.
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise()
	builder.isSequencer = true
	builder.execConfig.Sequencer.MaxBlockSpeed = maxBlockSpeed
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Recipient")
	transferEach := big.NewInt(1e12)

	// SendTransaction blocks until the tx is sequenced into a block, so submitting
	// sequentially puts each tx in its own block and the return time marks when
	// that block was produced. The throttle, not the submission rate, then paces
	// these to one per MaxBlockSpeed.
	const numTxs = 12
	blockTimes := make([]time.Time, 0, numTxs)
	var lastHash common.Hash
	for i := 0; i < numTxs; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "Recipient", builder.L2Info.TransferGas, transferEach, nil)
		require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
		blockTimes = append(blockTimes, time.Now())
		lastHash = tx.Hash()
	}

	_, err := WaitForTx(ctx, builder.L2.Client, lastHash, 30*time.Second)
	require.NoError(t, err, "last tx should be sequenced")

	// The recipient received every transfer.
	bal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Recipient"), nil)
	require.NoError(t, err)
	expected := new(big.Int).Mul(big.NewInt(numTxs), transferEach)
	require.Equal(t, expected, bal, "recipient should receive every transfer")

	gaps := make([]time.Duration, 0, len(blockTimes)-1)
	for i := 1; i < len(blockTimes); i++ {
		gaps = append(gaps, blockTimes[i].Sub(blockTimes[i-1]))
	}

	// Every gap should be close to MaxBlockSpeed: not faster (the throttle floor)
	// and not much slower (blocks aren't starved). The tolerance absorbs scheduling
	// jitter, which is tiny relative to MaxBlockSpeed.
	const tolerance = 200 * time.Millisecond
	for i, gap := range gaps {
		require.GreaterOrEqual(t, gap, maxBlockSpeed-tolerance,
			"block gap %d (%v) is shorter than MaxBlockSpeed %v; all gaps=%v", i, gap, maxBlockSpeed, gaps)
		require.LessOrEqual(t, gap, maxBlockSpeed+tolerance,
			"block gap %d (%v) is longer than MaxBlockSpeed %v; all gaps=%v", i, gap, maxBlockSpeed, gaps)
	}
}
