// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package arbtest

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/valnode"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

func TestExecutionStateMsgCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	res, err := l2node.TxStreamer.ResultAtCount(2)
	Require(t, err)
	err = manager.AgreesWithExecutionState(ctx, &protocol.ExecutionState{GlobalState: protocol.GoGlobalState{Batch: 1, BlockHash: res.BlockHash}})
	Require(t, err)
}

func TestExecutionStateAtMessageNumber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2node, l1stack, manager := setupManger(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	res, err := l2node.TxStreamer.ResultAtCount(2)
	Require(t, err)
	expectedState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			Batch:     1,
			BlockHash: res.BlockHash,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	batchCount := expectedState.GlobalState.Batch + 1
	executionState, err := manager.ExecutionStateAfterBatchCount(ctx, batchCount)
	Require(t, err)
	if !reflect.DeepEqual(executionState, expectedState) {
		Fail(t, "Unexpected executionState", executionState, "(expected ", expectedState, ")")
	}
	Require(t, err)
}

func setupManger(t *testing.T, ctx context.Context) (*arbnode.Node, *node.Node, *staker.StateManager) {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	_, l2node, l2client, _, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, l2chainConfig, nil, nil, l2info)
	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2client, ctx)
	l2info.GenerateAccount("BackgroundUser")
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	tx := l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, balance, nil)
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	for i := uint64(0); i < 10; i++ {
		l2info.Accounts["BackgroundUser"].Nonce = i
		tx = l2info.PrepareTx("BackgroundUser", "BackgroundUser", l2info.TransferGas, common.Big0, nil)
		err = l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig
	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution.Recorder,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)
	manager, err := staker.NewStateManager(stateless, t.TempDir(), nil)
	Require(t, err)
	return l2node, l1stack, manager
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
