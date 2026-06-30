// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// TestTraceFilteredTxBalancedCallstack reproduces the "incorrect number of top-level
// calls" trace failure caused by onchain-filtered transactions.
//
// When a tx hash is in the onchain FilteredTransactions list, TxProcessor.RevertedTxHook
// bumps the nonce, consumes all remaining gas, and returns ErrFilteredTx. That makes
// state_transition skip evm.Call entirely, so the EVM never fires the depth-0 OnEnter.
// Without a faked top-level frame the tracer's callstack stays empty and GetResult fails
// with "incorrect number of top-level calls". emitSkippedCallFrame now fakes the frame, so
// the trace must succeed.
//
// The regression target is a LEGACY (type 0) transaction, matching the on-chain failure
// observed on robinhood-testnet block 0x48F051C.
func TestTraceFilteredTxBalancedCallstack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := setupFilteredTxTestBuilder(t, ctx)

	builder.L2Info.GenerateAccount("FilteredUser")
	builder.L2Info.GenerateAccount("Sender")
	builder.L2Info.GenerateAccount("Filterer")

	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "Filterer", big.NewInt(1e18), builder.L2Info)

	// Grant Filterer the transaction-filterer role so it can add tx hashes to the
	// onchain filter (filtering is enabled at genesis by setupFilteredTxTestBuilder).
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, builder.L2Info.GetAddress("Filterer"))
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Block FilteredUser via the address filter so the delayed message halts the
	// delayed sequencer, mirroring the production flow that lands a tx in the onchain
	// filter before it is (re-)sequenced.
	filteredAddr := builder.L2Info.GetAddress("FilteredUser")
	addrFilter := newHashedChecker([]common.Address{filteredAddr})
	builder.L2.ExecNode.ExecEngine.SetAddressChecker(t, addrFilter)

	// A legacy (type 0) value transfer to the filtered address, sent via the delayed inbox.
	senderInfo := builder.L2Info.GetInfoWithPrivKey("Sender")
	nonce := senderInfo.Nonce.Add(1) - 1
	delayedTx := builder.L2Info.SignTxAs("Sender", &types.LegacyTx{
		Nonce:    nonce,
		To:       &filteredAddr,
		Value:    big.NewInt(1e12),
		Gas:      builder.L2Info.TransferGas,
		GasPrice: new(big.Int).Set(builder.L2Info.GasPrice),
	})
	require.Equal(t, uint8(types.LegacyTxType), delayedTx.Type(), "regression target must be a legacy tx")
	txHash := sendDelayedTx(t, ctx, builder, delayedTx)

	advanceL1ForDelayed(t, ctx, builder)
	waitForDelayedSequencerHaltOnHashes(t, ctx, builder, []common.Hash{txHash}, 10*time.Second)

	// Operator adds the tx hash to the onchain filter; the sequencer resumes and the tx is
	// sequenced but skipped by RevertedTxHook (all gas consumed, failed status).
	addTxHashToOnChainFilter(t, ctx, builder, txHash, "Filterer")
	waitForDelayedSequencerResume(t, ctx, builder, 10*time.Second)
	advanceL1ForDelayed(t, ctx, builder)

	receipt, err := WaitForTx(ctx, builder.L2.Client, txHash, 10*time.Second)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "filtered tx should be mined with failed status")
	require.Equal(t, delayedTx.Gas(), receipt.GasUsed, "filtered tx should consume all gas (punishment)")

	// Before the fix, both calls return "incorrect number of top-level calls" because the
	// filtered tx skipped the EVM without faking a top-level frame.
	l2rpc := builder.L2.Stack.Attach()
	defer l2rpc.Close()
	for _, tracer := range []string{"callTracer", "flatCallTracer", "erc7562Tracer"} {
		var txTrace json.RawMessage
		err := l2rpc.CallContext(ctx, &txTrace, "debug_traceTransaction", txHash,
			map[string]interface{}{"tracer": tracer})
		require.NoErrorf(t, err, "debug_traceTransaction with %s must not fail on a filtered tx", tracer)
		require.NotEmpty(t, txTrace, "tracer %s returned an empty trace", tracer)

		var blockTrace json.RawMessage
		err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber",
			rpc.BlockNumber(receipt.BlockNumber.Int64()),
			map[string]interface{}{"tracer": tracer})
		require.NoErrorf(t, err, "debug_traceBlockByNumber with %s must not fail on a block with a filtered tx", tracer)
	}
}
