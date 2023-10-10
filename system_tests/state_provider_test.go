// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/valnode"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
)

func TestStateProvider_BOLD(t *testing.T) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l2node, l1info, l2info, l1stack, l1client, stateManager := setupBoldStateProvider(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	l2info.GenerateAccount("Destination")
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	seqInbox := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInbox, l1client)
	Require(t, err)

	// We will make two batches, with 5 messages in each batch.
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1) // No divergence.
	makeBoldBatch(t, l2node, l2info, l1client, &sequencerTxOpts, seqInboxBinding, seqInbox, numMessagesPerBatch, divergeAt)
	makeBoldBatch(t, l2node, l2info, l1client, &sequencerTxOpts, seqInboxBinding, seqInbox, numMessagesPerBatch, divergeAt)

	bridgeBinding, err := bridgegen.NewBridge(l1info.GetAddress("Bridge"), l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()
	totalMessageCount, err := l2node.InboxTracker.GetBatchMessageCount(totalBatches - 1)
	Require(t, err)

	// Wait until the validator has validated the batches.
	for {
		if _, err := l2node.TxStreamer.ResultAtCount(arbutil.MessageIndex(totalMessageCount)); err == nil {
			break
		}
	}

	t.Run("StatesInBatchRange", func(t *testing.T) {
		fromBatch := l2stateprovider.Batch(1)
		toBatch := l2stateprovider.Batch(3)
		fromHeight := l2stateprovider.Height(0)
		toHeight := l2stateprovider.Height(16)
		stateRoots, states, err := stateManager.StatesInBatchRange(fromHeight, toHeight, fromBatch, toBatch)
		Require(t, err)

		if len(stateRoots) != 17 {
			Fatal(t, "wrong number of state roots")
		}
		if len(states) == 0 {
			Fatal(t, "no states returned")
		}
		firstState := states[0]
		if firstState.Batch != 1 && firstState.PosInBatch != 0 {
			Fatal(t, "wrong first state")
		}
		lastState := states[len(states)-1]
		if lastState.Batch != 1 && lastState.PosInBatch != 0 {
			Fatal(t, "wrong last state")
		}
	})
	t.Run("AgreesWithExecutionState", func(t *testing.T) {
		// Non-zero position in batch shoould fail.
		err = stateManager.AgreesWithExecutionState(ctx, &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				Batch:      0,
				PosInBatch: 1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		})
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !strings.Contains(err.Error(), "position in batch must be zero") {
			Fatal(t, "wrong error message")
		}

		// Always agrees with genesis.
		err = stateManager.AgreesWithExecutionState(ctx, &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				Batch:      0,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		})
		Require(t, err)

		// Always agrees with the init message.
		err = stateManager.AgreesWithExecutionState(ctx, &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		})
		Require(t, err)

		// Chain catching up if it has not seen batch 10.
		err = stateManager.AgreesWithExecutionState(ctx, &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				Batch:      10,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		})
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !errors.Is(err, staker.ErrChainCatchingUp) {
			Fatal(t, "wrong error")
		}

		// Check if we agree with the last posted batch to the inbox.
		result, err := l2node.TxStreamer.ResultAtCount(arbutil.MessageIndex(totalMessageCount))
		Require(t, err)

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  result.BlockHash,
				SendRoot:   result.SendRoot,
				Batch:      3,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		err = stateManager.AgreesWithExecutionState(ctx, state)
		Require(t, err)

		// See if we agree with one batch immediately after that and see that we fail with
		// "ErrChainCatchingUp".
		state.GlobalState.Batch += 1

		err = stateManager.AgreesWithExecutionState(ctx, state)
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !errors.Is(err, staker.ErrChainCatchingUp) {
			Fatal(t, "wrong error")
		}
	})
	t.Run("ExecutionStateAfterBatchCount", func(t *testing.T) {
		_, err = stateManager.ExecutionStateAfterBatchCount(ctx, 0)
		if err == nil {
			Fatal(t, "should have failed")
		}
		if !strings.Contains(err.Error(), "batch count cannot be zero") {
			Fatal(t, "wrong error message")
		}

		execState, err := stateManager.ExecutionStateAfterBatchCount(ctx, totalBatches)
		Require(t, err)

		// We should agree with the last posted batch to the inbox based on our
		// retrieved execution state.
		err = stateManager.AgreesWithExecutionState(ctx, execState)
		Require(t, err)
	})
}

func setupBoldStateProvider(t *testing.T, ctx context.Context) (*arbnode.Node, *BlockchainTestInfo, *BlockchainTestInfo, *node.Node, *ethclient.Client, *staker.StateManager) {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)

	_, l2node, _, _, l1info, _, l1client, l1stack, _, _ := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, l2chainConfig, nil, l2info)

	valConfig := staker.L1ValidatorConfig{}
	valConfig.Strategy = "MakeNodes"
	valnode.TestValidationConfig.UseJit = false
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

	stateManager, err := staker.NewStateManager(
		stateless,
		"",
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		"good",
		staker.DisableCache(),
	)
	Require(t, err)
	return l2node, l1info, l2info, l1stack, l1client, stateManager
}
