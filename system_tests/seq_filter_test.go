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
	builder, header, txes, hooks, cleanup := setupSequencerFilterTest(t, false)
	defer cleanup()

	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
	Require(t, err) // There shouldn't be any error in block generation
	if block == nil {
		t.Fatal("block should be generated as second tx should pass")
	}
	if len(block.Transactions()) != 2 {
		t.Fatalf("expecting two txs, found: %d", len(block.Transactions()))
	}
	if block.Transactions()[1].Hash() != txes[1].Hash() {
		t.Fatal("tx hash mismatch, expecting second tx to be present in the block")
	}
	if len(hooks.GetTxErrors()) != 2 {
		t.Fatalf("expected 2 txErrors in hooks, found: %d", len(hooks.GetTxErrors()))
	}
	if hooks.GetTxErrors()[0].Error() != state.ErrArbTxFilter.Error() {
		t.Fatalf("expected ErrArbTxFilter, found: %s", hooks.GetTxErrors()[0].Error())
	}
	if hooks.GetTxErrors()[1] != nil {
		t.Fatalf("found a non-nil error for second transaction: %v", hooks.GetTxErrors()[1])
	}
}

func TestSequencerBlockFilterReject(t *testing.T) {
	builder, header, _, hooks, cleanup := setupSequencerFilterTest(t, true)
	defer cleanup()

	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
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
	builder, header, txes, hooks, cleanup := setupSequencerFilterTest(t, true)
	defer cleanup()
	_, _, err := hooks.NextTxToSequence() // remove first transaction from hooks
	Require(t, err)
	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
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

func setupSequencerFilterTest(t *testing.T, isBlockFilter bool) (*NodeBuilder, *arbostypes.L1IncomingMessageHeader, types.Transactions, arbos.SequencingHooks, func()) {
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

	hooks := NewTestSequencingHooks(txes, isBlockFilter, !isBlockFilter)

	cleanup := func() {
		builderCleanup()
		cancel()
	}

	return builder, header, txes, hooks, cleanup
}

type TestSequencingHooks struct {
	*arbos.NoopSequencingHooks
	isBlockFilter bool
	isTxFilter    bool
}

func (t *TestSequencingHooks) PreTxFilter(config *params.ChainConfig, header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, options *arbitrum_types.ConditionalOptions, address common.Address, info *arbos.L1Info) error {
	if t.isTxFilter {
		if len(transaction.Data()) > 0 {
			db.FilterTx()
		}
	}
	return nil
}

func (t *TestSequencingHooks) PostTxFilter(header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, address common.Address, u uint64, result *core.ExecutionResult) error {
	if t.isTxFilter {
		if db.IsTxFiltered() {
			return state.ErrArbTxFilter
		}
	}
	return nil
}

func (t *TestSequencingHooks) BlockFilter(header *types.Header, db *state.StateDB, transactions types.Transactions, receipts types.Receipts) error {
	if t.isBlockFilter {
		if len(transactions[1].Data()) > 0 {
			return state.ErrArbTxFilter
		}
	}
	return nil
}

func NewTestSequencingHooks(txes types.Transactions, isBlockFilter bool, isTxFilter bool) *TestSequencingHooks {
	return &TestSequencingHooks{
		NoopSequencingHooks: arbos.NewNoopSequencingHooks(txes, false),
		isBlockFilter:       isBlockFilter,
		isTxFilter:          isTxFilter,
	}
}
