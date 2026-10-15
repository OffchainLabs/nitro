// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sequencer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

const (
	sizeSamplingMaxTxDataSize = 5000
	sizeSamplingMinTxSize     = 1024
	sizeSamplingBigTxCount    = 6
)

var withSizeSampling = systest.WithExecConfigOverride(func(c *gethexec.Config) {
	c.Sequencer.MaxTxDataSize = sizeSamplingMaxTxDataSize
	c.Sequencer.FIFO.SizeSamplingMaxExponent = 10
	c.Sequencer.FIFO.SizeSamplingGraceBlocks = 0
	c.Sequencer.FIFO.SizeSamplingMinTxSize = sizeSamplingMinTxSize
})

var sizeSamplingTests = []systest.Scenario{
	systest.Test(testRunFIFOSizeSamplingLetsSmallTxOvertakeBigOnes, withSizeSampling),
}

func testRunFIFOSizeSamplingLetsSmallTxOvertakeBigOnes(env *systest.Env) {
	var bigTxs []*pgaBid
	for i := range sizeSamplingBigTxCount {
		sender := fmt.Sprintf("Big%d", i)
		bigTxs = append(bigTxs, newBidOfSize(env, sender, sizeSamplingMaxTxDataSize))
	}
	env.L2.Info.GenerateAccount("Small")
	env.L2.TransferBalance("Faucet", "Small", big.NewInt(params.Ether))
	small := &pgaBid{label: "small", tx: env.L2.Info.PrepareTx("Small", "Small", env.L2.Info.TransferGas, big.NewInt(1), nil)}
	env.True(txSize(small.tx) <= sizeSamplingMinTxSize, "small tx of %d bytes is above the sampling floor", txSize(small.tx))

	bids := append(append([]*pgaBid{}, bigTxs...), small)
	enqueueAndSequence(env, bids)
	requireIncluded(env, bids)

	firstBigBlock := bigTxs[0].receipt.BlockNumber
	lastBigBlock := bigTxs[len(bigTxs)-1].receipt.BlockNumber
	smallBlock := small.receipt.BlockNumber
	env.True(smallBlock.Cmp(firstBigBlock) > 0, "small tx did not wait for the first block: %s", describeInclusions(bids))
	env.True(smallBlock.Cmp(lastBigBlock) < 0, "small tx waited for every big tx: %s", describeInclusions(bids))
}

func newBidOfSize(env *systest.Env, sender string, size int) *pgaBid {
	env.L2.Info.GenerateAccount(sender)
	env.L2.TransferBalance("Faucet", sender, big.NewInt(params.Ether))
	info := env.L2.Info.GetInfoWithPrivKey(sender)
	nonce := info.Nonce.Load()
	dataLen := size / 2
	for range 16 {
		tx := env.L2.Info.SignTxAs(sender, &types.DynamicFeeTx{
			To:        &info.Address,
			Gas:       env.L2.Info.TransferGas,
			GasTipCap: env.L2.Info.GasPrice,
			GasFeeCap: env.L2.Info.GasPrice,
			Nonce:     nonce,
			Data:      make([]byte, dataLen),
		})
		if txSize(tx) == size {
			info.Nonce.Add(1)
			return &pgaBid{label: fmt.Sprintf("%s size %d", sender, size), tx: tx}
		}
		dataLen += size - txSize(tx)
	}
	env.True(false, "could not build a %d-byte tx for %s", size, sender)
	return nil
}
