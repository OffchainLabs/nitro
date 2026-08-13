// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// TestPGAHappyPath runs the sequencer end-to-end with the PGA orderer active
func TestPGAHappyPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_60)
	builder.execConfig.Sequencer.MaxBlockSpeed = 100 * time.Millisecond
	builder.execConfig.Sequencer.ExperimentalPGA.DangerousForceFIFO = false
	builder.execConfig.Sequencer.ExperimentalPGA.RoundsPerBlock = 2
	cleanup := builder.Build(t)
	defer cleanup()

	// PGA only activates on a collect-tips chain.
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbOwner.SetCollectTips(&ownerAuth, true)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// One funded account per competing tx, so the burst has no nonce interdependencies.
	tips := []int64{5, 11, 23, 47, 97}
	names := make([]string, len(tips))
	for i := range tips {
		names[i] = fmt.Sprintf("PgaBidder%d", i)
		builder.L2Info.GenerateAccount(names[i])
		builder.L2.TransferBalance(t, "Faucet", names[i], big.NewInt(1e18), builder.L2Info)
	}

	txs := signPGABurst(t, builder, names, tips)

	// A paused sequencer parks the whole burst in its queue, so it drains into a single auction:
	// a tx is only reordered against the txs it shares a drain with.
	sequencer := builder.L2.ExecNode.Sequencer
	sequencer.Pause()

	// Submissions block until sequenced, so the burst needs concurrent senders.
	var wg sync.WaitGroup
	sendErrs := make([]error, len(txs))
	for i, tx := range txs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendErrs[i] = builder.L2.Client.SendTransaction(ctx, tx)
		}()
		// Wait for each tx to reach the queue before sending the next, so the queue holds the
		// burst in ascending tip order: that is what FIFO would sequence, so the auction's
		// descending order can't pass by accident.
		waitForQueuedTxs(t, i+1, 5*time.Second)
	}
	sequencer.Activate()
	wg.Wait()

	receipts := make([]*types.Receipt, len(txs))
	for i, tx := range txs {
		Require(t, sendErrs[i])
		receipt, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		receipts[i] = receipt
	}

	for i, receipt := range receipts {
		blockBaseFee := builder.L2.GetBaseFeeAt(t, receipt.BlockNumber)
		want := new(big.Int).Add(blockBaseFee, big.NewInt(tips[i]))
		if receipt.EffectiveGasPrice.Cmp(want) != 0 {
			Fatal(t, "tx paid wrong price", "tip", tips[i], "want", want, "got", receipt.EffectiveGasPrice)
		}
		if receipt.BlockNumber.Cmp(receipts[0].BlockNumber) != 0 {
			Fatal(t, "burst split across blocks", "inclusions", formatPGAInclusions(receipts, tips))
		}
	}
	assertTipsDescendByTxIndex(t, receipts, tips)
}

// TestPGASameSenderNonceChain submits a sender's consecutive-nonce pair with inverted tips: the
// higher-tipped follow-up must not outbid its own predecessor, in either queue order.
func TestPGASameSenderNonceChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_60)
	// The queue-length gauge below is process-global; run alone so no other node writes to it.
	builder.parallelise = false
	builder.execConfig.Sequencer.MaxBlockSpeed = 100 * time.Millisecond
	builder.execConfig.Sequencer.ExperimentalPGA.DangerousForceFIFO = false
	builder.execConfig.Sequencer.ExperimentalPGA.RoundsPerBlock = 2
	// Keep the parked follow-up alive while the paused sequencer holds the burst.
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = time.Minute
	cleanup := builder.Build(t)
	defer cleanup()

	// PGA only activates on a collect-tips chain.
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbOwner.SetCollectTips(&ownerAuth, true)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	name := "PgaNonceChainer"
	builder.L2Info.GenerateAccount(name)
	builder.L2.TransferBalance(t, "Faucet", name, big.NewInt(1e18), builder.L2Info)

	for burst, reversed := range []bool{false, true} {
		tips := []int64{5, 50}
		txs := signPGABurst(t, builder, []string{name, name}, tips)
		submitted := txs
		if reversed {
			submitted = []*types.Transaction{txs[1], txs[0]}
		}

		// A paused sequencer parks the pair in its queue, so both drain into the same auction.
		sequencer := builder.L2.ExecNode.Sequencer
		sequencer.Pause()
		waitForExactQueuedTxs(t, 0, 5*time.Second)

		// Submissions block until sequenced, so the burst needs concurrent senders.
		var wg sync.WaitGroup
		sendErrs := make([]error, len(submitted))
		for i, tx := range submitted {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendErrs[i] = builder.L2.Client.SendTransaction(ctx, tx)
			}()
			// Wait for each tx to reach the queue before sending the next, pinning the order.
			waitForExactQueuedTxs(t, i+1, 5*time.Second)
		}
		sequencer.Activate()
		wg.Wait()

		receipts := make([]*types.Receipt, len(txs))
		for i, tx := range txs {
			Require(t, sendErrs[i])
			receipt, err := builder.L2.EnsureTxSucceeded(tx)
			Require(t, err)
			receipts[i] = receipt
		}

		predecessor, followUp := receipts[0], receipts[1]
		if predecessor.BlockNumber.Cmp(followUp.BlockNumber) != 0 {
			Fatal(t, "nonce chain split across blocks", "burst", burst, "inclusions", formatPGAInclusions(receipts, tips))
		}
		if predecessor.TransactionIndex >= followUp.TransactionIndex {
			Fatal(t, "follow-up sequenced before its predecessor", "burst", burst, "inclusions", formatPGAInclusions(receipts, tips))
		}
		// Each tx pays its own tip.
		blockBaseFee := builder.L2.GetBaseFeeAt(t, predecessor.BlockNumber)
		for i, receipt := range receipts {
			want := new(big.Int).Add(blockBaseFee, big.NewInt(tips[i]))
			if receipt.EffectiveGasPrice.Cmp(want) != 0 {
				Fatal(t, "tx paid wrong price", "burst", burst, "tip", tips[i], "want", want, "got", receipt.EffectiveGasPrice)
			}
		}
	}
}

// signPGABurst signs one self-transfer per account, one per tip.
func signPGABurst(t *testing.T, builder *NodeBuilder, names []string, tips []int64) []*types.Transaction {
	t.Helper()
	baseFee := builder.L2.GetBaseFee(t)

	txs := make([]*types.Transaction, len(tips))
	for i, tip := range tips {
		info := builder.L2Info.GetInfoWithPrivKey(names[i])
		// A generous fee cap keeps the effective tip equal to the tip cap despite base fee drift.
		gasFeeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(4)), big.NewInt(tip))
		txs[i] = builder.L2Info.SignTxAs(names[i], &types.DynamicFeeTx{
			To:        &info.Address,
			Gas:       builder.L2Info.TransferGas,
			GasTipCap: big.NewInt(tip),
			GasFeeCap: gasFeeCap,
			Value:     big.NewInt(1),
			Nonce:     info.Nonce.Add(1) - 1,
		})
	}
	return txs
}

// sequencerQueueLengthMetric is refreshed every sequencer poll interval, even while paused.
const sequencerQueueLengthMetric = "arb/sequencer/queue/length"

// sequencerQueueLengthGauge looks the gauge up instead of registering it, so a renamed metric
// fails here rather than handing out a fresh gauge that never leaves zero.
func sequencerQueueLengthGauge(t *testing.T) *metrics.Gauge {
	t.Helper()
	gauge, ok := metrics.DefaultRegistry.Get(sequencerQueueLengthMetric).(*metrics.Gauge)
	if !ok {
		Fatal(t, "sequencer queue length metric is not a registered gauge", "metric", sequencerQueueLengthMetric)
	}
	return gauge
}

// waitForQueuedTxs waits until the sequencer's queue holds want txs.
func waitForQueuedTxs(t *testing.T, want int, timeout time.Duration) {
	t.Helper()
	gauge := sequencerQueueLengthGauge(t)
	deadline := time.Now().Add(timeout)
	for {
		if got := gauge.Snapshot().Value(); got >= int64(want) {
			return
		}
		if time.Now().After(deadline) {
			Fatal(t, "timed out waiting for queued txs", "metric", sequencerQueueLengthMetric,
				"want", want, "got", gauge.Snapshot().Value())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// waitForExactQueuedTxs waits until the sequencer's queue holds exactly want txs, so a stale
// gauge value from earlier activity can't end the wait early.
func waitForExactQueuedTxs(t *testing.T, want int, timeout time.Duration) {
	t.Helper()
	gauge := sequencerQueueLengthGauge(t)
	deadline := time.Now().Add(timeout)
	for {
		if got := gauge.Snapshot().Value(); got == int64(want) {
			return
		}
		if time.Now().After(deadline) {
			Fatal(t, "timed out waiting for queued txs", "metric", sequencerQueueLengthMetric,
				"want", want, "got", gauge.Snapshot().Value())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// assertTipsDescendByTxIndex checks that the receipts' tips strictly descend in block order.
func assertTipsDescendByTxIndex(t *testing.T, receipts []*types.Receipt, tips []int64) {
	t.Helper()
	inclusions := pgaInclusions(receipts, tips)
	for i := 1; i < len(inclusions); i++ {
		if inclusions[i-1].tip <= inclusions[i].tip {
			Fatal(t, "block is not ordered by descending tip", "inclusions", formatPGAInclusions(receipts, tips))
		}
	}
}

type pgaInclusion struct {
	blockNumber *big.Int
	txIndex     uint
	tip         int64
}

// pgaInclusions pairs each tip with where its tx landed, sorted by block order.
func pgaInclusions(receipts []*types.Receipt, tips []int64) []pgaInclusion {
	inclusions := make([]pgaInclusion, len(receipts))
	for i, receipt := range receipts {
		inclusions[i] = pgaInclusion{receipt.BlockNumber, receipt.TransactionIndex, tips[i]}
	}
	sort.Slice(inclusions, func(a, b int) bool {
		if cmp := inclusions[a].blockNumber.Cmp(inclusions[b].blockNumber); cmp != 0 {
			return cmp < 0
		}
		return inclusions[a].txIndex < inclusions[b].txIndex
	})
	return inclusions
}

func formatPGAInclusions(receipts []*types.Receipt, tips []int64) string {
	out := ""
	for _, inclusion := range pgaInclusions(receipts, tips) {
		out += fmt.Sprintf("[block %v tx %d tip %d]", inclusion.blockNumber, inclusion.txIndex, inclusion.tip)
	}
	return out
}
