package arbtest

import (
	"context"
	"errors"
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

func TestSequencerFilter(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.isSequencer = true
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User")
	var latestL2 uint64
	var err error
	for i := 0; latestL2 < 3; i++ {
		_, _ = builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e18), builder.L2Info)
		latestL2, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
	}

	preTxFilter := func(withBlock bool) func(_ *params.ChainConfig, _ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions, _ common.Address, _ *arbos.L1Info) error {
		return func(_ *params.ChainConfig, _ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions, _ common.Address, _ *arbos.L1Info) error {
			if _, ok := tx.GetInner().(*types.DynamicFeeTx); ok {
				statedb.FilterTx(withBlock)
			}
			return nil
		}
	}
	postTxFilter := func(_ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, _ *types.Transaction, _ common.Address, _ uint64, _ *core.ExecutionResult) error {
		if statedb.IsTxInvalid() {
			return errors.New("internal error")
		}
		return nil
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
	txes = append(txes, builder.L2Info.PrepareTx("Owner", "User", builder.L2Info.TransferGas, big.NewInt(1e12), nil))
	txes = append(txes, builder.L2Info.PrepareTx("User", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), nil))

	hooks := &arbos.SequencingHooks{TxErrors: []error{}, DiscardInvalidTxsEarly: false, PreTxFilter: preTxFilter(false), PostTxFilter: postTxFilter, ConditionalOptionsForTx: nil}
	block, err := builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, txes, hooks)
	if block != nil {
		t.Fatal("block shouldn't be generated when all txes have failed")
	}
	Require(t, err) // There shouldn't be any error in block generation
	if len(hooks.TxErrors) != 2 {
		t.Fatalf("expected 2 tx errors, found: %d", len(hooks.TxErrors))
	}
	for _, err := range hooks.TxErrors {
		if err.Error() != state.ErrArbTxFilter.Error() {
			t.Fatalf("expected ErrArbTxFilter, found: %s", err.Error())
		}
	}

	hooks.TxErrors = []error{}
	hooks.PreTxFilter = preTxFilter(true)
	block, err = builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, txes, hooks)
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
