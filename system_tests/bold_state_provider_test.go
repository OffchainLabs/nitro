// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

//asdgo:build challengetest && !race

package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/valnode"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	prefixproofs "github.com/OffchainLabs/bold/state-commitments/prefix-proofs"
	mockmanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
)

func TestChallengeProtocolBOLD_Bisections(t *testing.T) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l2node, l1info, l2info, l1stack, l1client, stateManager, blockValidator := setupBoldStateProvider(t, ctx)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	l2info.GenerateAccount("Destination")
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	seqInbox := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInbox, l1client)
	Require(t, err)

	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)

	honestUpgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("UpgradeExecutor"), l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	honestRollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = honestUpgradeExec.ExecuteCall(&honestRollupOwnerOpts, seqInbox, data)
	Require(t, err)

	// We will make two batches, with 5 messages in each batch.
	numMessagesPerBatch := int64(5)
	divergeAt := int64(-1) // No divergence.
	makeBoldBatch(t, l2node, l2info, l1client, &sequencerTxOpts, seqInboxBinding, seqInbox, numMessagesPerBatch, divergeAt)
	numMessagesPerBatch = int64(10)
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
		lastInfo, err := blockValidator.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		batchMsgCount, err := l2node.InboxTracker.GetBatchMessageCount(lastInfo.GlobalState.Batch)
		if err != nil {
			continue
		}
		Require(t, err)
		t.Log("lastValidatedMessageCount", batchMsgCount, "totalMessageCount", totalMessageCount)
		if batchMsgCount >= totalMessageCount {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	historyCommitter := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager, []l2stateprovider.Height{
			1 << 5,
			1 << 5,
			1 << 5,
		},
		stateManager,
		nil, // api db
	)
	bisectionHeight := l2stateprovider.Height(16)
	request := &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              common.Hash{},
		FromBatch:                   1,
		ToBatch:                     3,
		UpperChallengeOriginHeights: []l2stateprovider.Height{},
		FromHeight:                  0,
		UpToHeight:                  option.Some(bisectionHeight),
	}
	bisectionCommitment, err := historyCommitter.HistoryCommitment(ctx, request)
	Require(t, err)

	request.UpToHeight = option.None[l2stateprovider.Height]()
	packedProof, err := historyCommitter.PrefixProof(ctx, request, bisectionHeight)
	Require(t, err)

	dataItem, err := mockmanager.ProofArgs.Unpack(packedProof)
	Require(t, err)
	preExpansion, ok := dataItem[0].([][32]byte)
	if !ok {
		Fatal(t, "wrong type")
	}

	hashes := make([]common.Hash, len(preExpansion))
	for i, h := range preExpansion {
		hash := h
		hashes[i] = hash
	}

	computed, err := prefixproofs.Root(hashes)
	Require(t, err)
	if computed != bisectionCommitment.Merkle {
		Fatal(t, "wrong commitment")
	}
}

func TestChallengeProtocolBOLD_StateProvider(t *testing.T) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	l2node, l1info, l2info, l1stack, l1client, stateManager, blockValidator := setupBoldStateProvider(t, ctx, staker.WithoutFinalizedBatchChecks())
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()
	l2info.GenerateAccount("Destination")
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	seqInbox := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInbox, l1client)
	Require(t, err)

	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)

	honestUpgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("UpgradeExecutor"), l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	honestRollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = honestUpgradeExec.ExecuteCall(&honestRollupOwnerOpts, seqInbox, data)
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
		lastInfo, err := blockValidator.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		batchMsgCount, err := l2node.InboxTracker.GetBatchMessageCount(lastInfo.GlobalState.Batch)
		if err != nil {
			continue
		}
		t.Log("lastValidatedMessageCount", batchMsgCount, "totalMessageCount", totalMessageCount)
		if batchMsgCount >= totalMessageCount {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	maxBlocks := uint64(1 << 14)

	t.Run("StatesInBatchRange", func(t *testing.T) {
		fromBatch := l2stateprovider.Batch(1)
		toBatch := l2stateprovider.Batch(3)
		fromHeight := l2stateprovider.Height(0)
		toHeight := l2stateprovider.Height(14)
		stateRoots, states, err := stateManager.StatesInBatchRange(fromHeight, toHeight, fromBatch, toBatch)
		Require(t, err)

		if len(stateRoots) != 15 {
			Fatal(t, "wrong number of state roots")
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
		// Non-zero position in batch should fail.
		_, err = stateManager.ExecutionStateAfterPreviousState(
			ctx,
			0,
			&protocol.GoGlobalState{
				Batch:      0,
				PosInBatch: 1,
			},
			maxBlocks,
		)
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !strings.Contains(err.Error(), "max inbox count cannot be zero") {
			Fatal(t, "wrong error message")
		}

		// Always agrees with genesis.
		genesis, err := stateManager.ExecutionStateAfterPreviousState(
			ctx,
			1,
			&protocol.GoGlobalState{
				Batch:      0,
				PosInBatch: 0,
			},
			maxBlocks,
		)
		Require(t, err)
		if genesis == nil {
			Fatal(t, "genesis should not be nil")
		}

		// Always agrees with the init message.
		first, err := stateManager.ExecutionStateAfterPreviousState(
			ctx,
			2,
			&genesis.GlobalState,
			maxBlocks,
		)
		Require(t, err)
		if first == nil {
			Fatal(t, "genesis should not be nil")
		}

		// Chain catching up if it has not seen batch 10.
		_, err = stateManager.ExecutionStateAfterPreviousState(
			ctx,
			10,
			&first.GlobalState,
			maxBlocks,
		)
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !errors.Is(err, l2stateprovider.ErrChainCatchingUp) {
			Fatal(t, "wrong error")
		}

		// Check if we agree with the last posted batch to the inbox.
		result, err := l2node.TxStreamer.ResultAtCount(totalMessageCount)
		Require(t, err)
		_ = result

		state := protocol.GoGlobalState{
			BlockHash: result.BlockHash,
			SendRoot:  result.SendRoot,
			Batch:     3,
		}
		got, err := stateManager.ExecutionStateAfterPreviousState(ctx, 3, &first.GlobalState, maxBlocks)
		Require(t, err)
		if state.Batch != got.GlobalState.Batch {
			Fatal(t, "wrong batch")
		}
		if state.SendRoot != got.GlobalState.SendRoot {
			Fatal(t, "wrong send root")
		}
		if state.BlockHash != got.GlobalState.BlockHash {
			Fatal(t, "wrong batch")
		}

		// See if we agree with one batch immediately after that and see that we fail with
		// "ErrChainCatchingUp".
		_, err = stateManager.ExecutionStateAfterPreviousState(
			ctx,
			state.Batch+1,
			&got.GlobalState,
			maxBlocks,
		)
		if err == nil {
			Fatal(t, "should not agree with execution state")
		}
		if !errors.Is(err, l2stateprovider.ErrChainCatchingUp) {
			Fatal(t, "wrong error")
		}
	})
	t.Run("ExecutionStateAfterBatchCount", func(t *testing.T) {
		_, err = stateManager.ExecutionStateAfterPreviousState(ctx, 0, &protocol.GoGlobalState{}, maxBlocks)
		if err == nil {
			Fatal(t, "should have failed")
		}
		if !strings.Contains(err.Error(), "max inbox count cannot be zero") {
			Fatal(t, "wrong error message", err)
		}

		genesis, err := stateManager.ExecutionStateAfterPreviousState(ctx, 1, &protocol.GoGlobalState{}, maxBlocks)
		Require(t, err)
		execState, err := stateManager.ExecutionStateAfterPreviousState(ctx, totalBatches, &genesis.GlobalState, maxBlocks)
		Require(t, err)
		if execState == nil {
			Fatal(t, "should not be nil")
		}
	})
}

func setupBoldStateProvider(t *testing.T, ctx context.Context, opts ...staker.BOLDStateProviderOpt) (*arbnode.Node, *BlockchainTestInfo, *BlockchainTestInfo, *node.Node, *ethclient.Client, *staker.BOLDStateProvider, *staker.BlockValidator) {
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

	_, l2node, _, _, l1info, _, l1client, l1stack, _, _ := createTestNodeOnL1ForBoldProtocol(t, ctx, false, nil, l2chainConfig, nil, l2info)

	valnode.TestValidationConfig.UseJit = false
	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	Require(t, stateless.Start(ctx))

	blockValidator, err := staker.NewBlockValidator(
		stateless,
		l2node.InboxTracker,
		l2node.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)
	Require(t, blockValidator.Initialize(ctx))
	Require(t, blockValidator.Start(ctx))

	stateManager, err := staker.NewBOLDStateProvider(
		blockValidator,
		stateless,
		"",
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		"",
		opts...,
	)
	Require(t, err)

	Require(t, l2node.Start(ctx))
	return l2node, l1info, l2info, l1stack, l1client, stateManager, blockValidator
}
