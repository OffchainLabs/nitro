package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestSequencerTxFilter(t *testing.T) {
	t.Parallel()

	builder, header, txes, hooks, cleanup := setupSequencerFilterTest(t, false)
	defer cleanup()

	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, txes, hooks, nil)
	Require(t, err) // There shouldn't be any error in block generation
	if block == nil {
		t.Fatal("block should be generated as second tx should pass")
	}
	if len(block.Transactions()) != 2 {
		t.Fatalf("expecting two txs found: %d", len(block.Transactions()))
	}
	if block.Transactions()[1].Hash() != txes[1].Hash() {
		t.Fatal("tx hash mismatch, expecting second tx to be present in the block")
	}
	if len(hooks.TxErrors) != 2 {
		t.Fatalf("expected 2 txErrors in hooks, found: %d", len(hooks.TxErrors))
	}
	if hooks.TxErrors[0].Error() != state.ErrArbTxFilter.Error() {
		t.Fatalf("expected ErrArbTxFilter, found: %s", err.Error())
	}
	if hooks.TxErrors[1] != nil {
		t.Fatalf("found a non-nil error for second transaction: %v", hooks.TxErrors[1])
	}
}

func TestSequencerBlockFilterReject(t *testing.T) {
	t.Parallel()

	builder, header, txes, hooks, cleanup := setupSequencerFilterTest(t, true)
	defer cleanup()

	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, txes, hooks, nil)
	if block != nil {
		t.Fatal("block shouldn't be generated when all txes have failed")
	}
	if err == nil {
		t.Fatal("expected ErrArbTxFilter but found nil")
	}
	if err.Error() != state.ErrArbTxFilter.Error() {
		t.Fatalf("expected ErrArbTxFilter, found: %s", err.Error())
	}
}

func TestSequencerBlockFilterAccept(t *testing.T) {
	t.Parallel()

	builder, header, txes, hooks, cleanup := setupSequencerFilterTest(t, true)
	defer cleanup()

	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, txes[1:], hooks, nil)
	Require(t, err)
	if block == nil {
		t.Fatal("block should be generated as the tx should pass")
	}
	if len(block.Transactions()) != 2 {
		t.Fatalf("expecting two txs found: %d", len(block.Transactions()))
	}
	if block.Transactions()[1].Hash() != txes[1].Hash() {
		t.Fatal("tx hash mismatch, expecting second tx to be present in the block")
	}
}

func setupSequencerFilterTest(t *testing.T, isBlockFilter bool) (*NodeBuilder, *arbostypes.L1IncomingMessageHeader, types.Transactions, *arbos.SequencingHooks, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	builderCleanup := builder.Build(t)

	builder.L2Info.GenerateAccount("User")
	var latestL2 uint64
	var err error
	for i := 0; latestL2 < 3; i++ {
		_, _ = builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e18), builder.L2Info)
		latestL2, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
	}

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: 1,
		Timestamp:   arbmath.SaturatingUCast[uint64](time.Now().Unix()),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	var txes types.Transactions
	txes = append(txes, builder.L2Info.PrepareTx("Owner", "User", builder.L2Info.TransferGas, big.NewInt(1e12), []byte{1, 2, 3}))
	txes = append(txes, builder.L2Info.PrepareTx("User", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), nil))

	hooks := arbos.NoopSequencingHooks()
	if isBlockFilter {
		hooks.BlockFilter = func(_ *types.Header, _ *state.StateDB, txes types.Transactions, _ types.Receipts) error {
			if len(txes[1].Data()) > 0 {
				return state.ErrArbTxFilter
			}
			return nil
		}
	} else {
		hooks.PreTxFilter = func(_ *params.ChainConfig, _ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions, _ common.Address, _ *arbos.L1Info) error {
			if len(tx.Data()) > 0 {
				statedb.FilterTx()
			}
			return nil
		}
		hooks.PostTxFilter = func(_ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, _ common.Address, _ uint64, _ *core.ExecutionResult) error {
			if statedb.IsTxFiltered() {
				return state.ErrArbTxFilter
			}
			return nil
		}
	}

	cleanup := func() {
		builderCleanup()
		cancel()
	}

	return builder, header, txes, hooks, cleanup
}
