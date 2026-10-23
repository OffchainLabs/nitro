// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// The tests in this file cover the sequencer's rejection of a second transaction for a sender
// and nonce that is already queued. Each test parks the first transaction in a paused
// sequencer's queue and submits the second one while it is parked.

const sameNonceSender = "SameNonceSender"

// sameNonceBuilder starts a collect-tips chain. The sequencer orders by PGA unless forceFIFO
// is set.
func sameNonceBuilder(t *testing.T, ctx context.Context, forceFIFO bool) (*NodeBuilder, func()) {
	t.Helper()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_60)
	builder.execConfig.Sequencer.MaxBlockSpeed = 100 * time.Millisecond
	builder.execConfig.Sequencer.QueueTimeout = time.Minute
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = time.Minute
	builder.execConfig.Sequencer.PGA.DangerousForceFIFO = forceFIFO
	cleanup := builder.Build(t)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbOwner.SetCollectTips(&ownerAuth, true)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	builder.L2Info.GenerateAccount(sameNonceSender)
	builder.L2.TransferBalance(t, "Faucet", sameNonceSender, big.NewInt(1e18), builder.L2Info)
	return builder, cleanup
}

// requirePGAAdmission produces one more block after SetCollectTips, since tx admission only
// learns the strategy from a block creation attempt, and the attempt that included
// SetCollectTips still ran under FIFO.
func requirePGAAdmission(t *testing.T, builder *NodeBuilder) {
	t.Helper()
	builder.L2.TransferBalance(t, "Faucet", "Faucet", common.Big1, builder.L2Info)
	if got := builder.L2.ExecNode.Sequencer.OrderingStrategyForTesting(); got != "PGA" {
		Fatal(t, "tx admission is not running under PGA", "strategy", got)
	}
}

// signSameNonceTx signs a self-transfer with the given tip. A nil feeCap picks a cap generous
// enough to keep the effective tip equal to the tip despite basefee drift.
func signSameNonceTx(t *testing.T, builder *NodeBuilder, nonce uint64, tip int64, feeCap *big.Int) *types.Transaction {
	t.Helper()
	info := builder.L2Info.GetInfoWithPrivKey(sameNonceSender)
	if feeCap == nil {
		baseFee := builder.L2.GetBaseFee(t)
		feeCap = new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(4)), big.NewInt(tip))
	}
	return builder.L2Info.SignTxAs(sameNonceSender, &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       builder.L2Info.TransferGas,
		GasTipCap: big.NewInt(tip),
		GasFeeCap: feeCap,
		Value:     common.Big1,
		Nonce:     nonce,
	})
}

func waitForSequencerQueue(t *testing.T, builder *NodeBuilder, want int) {
	t.Helper()
	sequencer := builder.L2.ExecNode.Sequencer
	deadline := time.Now().Add(5 * time.Second)
	for sequencer.QueuedTxCount() != want {
		if time.Now().After(deadline) {
			Fatal(t, "timed out waiting for queued txs", "want", want, "got", sequencer.QueuedTxCount())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// parkTx submits tx to the paused sequencer and returns the channel its result lands on once
// the sequencer resumes.
func parkTx(t *testing.T, ctx context.Context, builder *NodeBuilder, tx *types.Transaction) <-chan error {
	t.Helper()
	queued := builder.L2.ExecNode.Sequencer.QueuedTxCount()
	result := make(chan error, 1)
	go func() { result <- builder.L2.Client.SendTransaction(ctx, tx) }()
	waitForSequencerQueue(t, builder, queued+1)
	return result
}

func awaitSendResult(t *testing.T, result <-chan error, label string) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Minute):
		Fatal(t, "submission did not return", "tx", label)
		return nil
	}
}

func requirePaidTip(t *testing.T, builder *NodeBuilder, receipt *types.Receipt, tip int64) {
	t.Helper()
	baseFee := builder.L2.GetBaseFeeAt(t, receipt.BlockNumber)
	want := new(big.Int).Add(baseFee, big.NewInt(tip))
	if receipt.EffectiveGasPrice.Cmp(want) != 0 {
		Fatal(t, "tx paid wrong price", "tip", tip, "want", want, "got", receipt.EffectiveGasPrice)
	}
}

func requireNoReceipt(t *testing.T, ctx context.Context, builder *NodeBuilder, tx *types.Transaction) {
	t.Helper()
	// The tx indexer reports indexing until it catches up with the latest head.
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err = builder.L2.Client.TransactionReceipt(ctx, tx.Hash())
		if err == nil || !strings.Contains(err.Error(), "indexing is in progress") || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !errors.Is(err, ethereum.NotFound) {
		Fatal(t, "rejected tx got a receipt", "hash", tx.Hash(), "err", err)
	}
}

// TestSequencerPGASameNonceFeeBumpWins parks a tx, then queues a same-nonce replacement that
// bumps both fees. The replacement lands and the original fails with nonce too low.
func TestSequencerPGASameNonceFeeBumpWins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder, cleanup := sameNonceBuilder(t, ctx, false)
	defer cleanup()
	requirePGAAdmission(t, builder)

	nonce := builder.L2Info.GetInfoWithPrivKey(sameNonceSender).Nonce.Load()
	first := signSameNonceTx(t, builder, nonce, 5, nil)
	replacement := signSameNonceTx(t, builder, nonce, 50, new(big.Int).Mul(first.GasFeeCap(), big.NewInt(2)))

	sequencer := builder.L2.ExecNode.Sequencer
	sequencer.Pause()
	firstResult := parkTx(t, ctx, builder, first)
	replacementResult := parkTx(t, ctx, builder, replacement)
	sequencer.Activate()

	if err := awaitSendResult(t, firstResult, "first"); err == nil || !strings.Contains(err.Error(), core.ErrNonceTooLow.Error()) {
		Fatal(t, "replaced tx returned the wrong result", "want", core.ErrNonceTooLow, "got", err)
	}
	Require(t, awaitSendResult(t, replacementResult, "replacement"))
	receipt, err := builder.L2.EnsureTxSucceeded(replacement)
	Require(t, err)
	requirePaidTip(t, builder, receipt, 50)
	requireNoReceipt(t, ctx, builder, first)
}

func TestSequencerSameTxResubmissionRejected(t *testing.T) {
	testSameNonceRejection(t, true, func(t *testing.T, builder *NodeBuilder, first *types.Transaction) *types.Transaction {
		return first
	}, txpool.ErrAlreadyKnown)
}

func TestSequencerFIFOSameNonceRejected(t *testing.T) {
	testSameNonceRejection(t, true, func(t *testing.T, builder *NodeBuilder, first *types.Transaction) *types.Transaction {
		return signSameNonceTx(t, builder, first.Nonce(), 50, nil)
	}, core.ErrNonceTooLow)
}

func TestSequencerPGASameNonceUnderpricedReplacementRejected(t *testing.T) {
	testSameNonceRejection(t, false, func(t *testing.T, builder *NodeBuilder, first *types.Transaction) *types.Transaction {
		requirePGAAdmission(t, builder)
		return signSameNonceTx(t, builder, first.Nonce(), 50, first.GasFeeCap())
	}, txpool.ErrReplaceUnderpriced)
}

// testSameNonceRejection parks a tx, submits the second one while it is parked, and checks that
// the sequencer rejects the second one with wantErr and still sequences the first.
func testSameNonceRejection(t *testing.T, forceFIFO bool, second func(*testing.T, *NodeBuilder, *types.Transaction) *types.Transaction, wantErr error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder, cleanup := sameNonceBuilder(t, ctx, forceFIFO)
	defer cleanup()

	nonce := builder.L2Info.GetInfoWithPrivKey(sameNonceSender).Nonce.Load()
	first := signSameNonceTx(t, builder, nonce, 5, nil)
	secondTx := second(t, builder, first)

	sequencer := builder.L2.ExecNode.Sequencer
	sequencer.Pause()
	firstResult := parkTx(t, ctx, builder, first)
	err := builder.L2.Client.SendTransaction(ctx, secondTx)
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		sequencer.Activate()
		Fatal(t, "second tx was not rejected as expected", "want", wantErr, "got", err)
	}
	sequencer.Activate()

	Require(t, awaitSendResult(t, firstResult, "first"))
	receipt, err := builder.L2.EnsureTxSucceeded(first)
	Require(t, err)
	requirePaidTip(t, builder, receipt, 5)
	if secondTx.Hash() != first.Hash() {
		requireNoReceipt(t, ctx, builder, secondTx)
	}
}
