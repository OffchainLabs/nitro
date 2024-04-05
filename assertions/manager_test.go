// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions_test

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSkipsProcessingAssertionFromEvilFork(t *testing.T) {
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithMockBridge(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)

	bridgeBindings, err := mocksgen.NewBridgeStub(setup.Addrs.Bridge, setup.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(util.GetSafeCallOpts(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := setup.Chains[0]
	bobChain := setup.Chains[1]

	ctx := context.Background()
	genesisHash, err := setup.Chains[1].GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisCreationInfo, err := setup.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
	)
	aliceStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	// Bob diverges from Alice at batch 1.
	stateManagerOpts = setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
		statemanager.WithBlockDivergenceHeight(1),
		statemanager.WithMachineDivergenceStep(1),
	)
	bobStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	// We have bob post an assertion at batch 1.
	genesisGlobalState := protocol.GoGlobalStateFromSolidity(genesisCreationInfo.AfterState.GlobalState)
	bobPostState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, &genesisGlobalState, 1<<26)
	require.NoError(t, err)
	t.Logf("%+v", bobPostState)
	bobAssertion, err := bobChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		bobPostState,
	)
	require.NoError(t, err)
	bobAssertionInfo, err := bobChain.ReadAssertionCreationInfo(ctx, bobAssertion.Id())
	require.NoError(t, err)

	// We have Alice process the assertion and post a rival, honest assertion to the one
	// at batch 1.
	aliceChalManager, err := challengemanager.New(
		ctx,
		aliceChain,
		aliceStateManager,
		setup.Addrs.Rollup,
		challengemanager.WithMode(types.DefensiveMode),
		challengemanager.WithEdgeTrackerWakeInterval(time.Hour),
	)
	require.NoError(t, err)
	aliceChalManager.Start(ctx)

	// Setup an assertion manager for Charlie, and have it process Alice's
	// assertion creation event at batch 4.
	aliceAssertionManager, err := assertions.NewManager(
		aliceChain,
		aliceStateManager,
		setup.Backend,
		aliceChalManager,
		aliceChain.RollupAddress(),
		aliceChalManager.ChallengeManagerAddress(),
		"alice",
		time.Millisecond*200, // poll interval for assertions
		time.Hour,            // confirmation attempt interval
		aliceStateManager,
		time.Hour, // poll interval
		time.Second*1,
		nil,
		assertions.WithPostingDisabled(),
	)
	require.NoError(t, err)

	aliceAssertionManager.Start(ctx)

	// Check that Alice submitted a rival to Bob's assertion after some time.
	time.Sleep(time.Second)
	require.Equal(t, uint64(1), aliceAssertionManager.SubmittedRivals())

	// We have bob post an assertion at batch 2.
	dataHash := [32]byte{1}
	enqueueSequencerMessageAsExecutor(
		t,
		setup.Accounts[0].TxOpts,
		setup.Addrs.UpgradeExecutor,
		setup.Backend,
		setup.Addrs.Bridge,
		seqMessage{
			dataHash:                 dataHash,
			afterDelayedMessagesRead: big.NewInt(1),
			prevMessageCount:         big.NewInt(1),
			newMessageCount:          big.NewInt(2),
		},
	)

	genesisState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 0, nil, 1<<26)
	require.NoError(t, err)
	preState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, &genesisState.GlobalState, 1<<26)
	require.NoError(t, err)
	bobPostState, err = bobStateManager.ExecutionStateAfterPreviousState(ctx, 2, &preState.GlobalState, 1<<26)
	require.NoError(t, err)
	_, err = bobChain.StakeOnNewAssertion(
		ctx,
		bobAssertionInfo,
		bobPostState,
	)
	require.NoError(t, err)

	// Once Alice sees this, she should do nothing as it is from a bad fork and
	// she already posted the correct child to their earliest valid ancestor here.
	// We should only have attempted to submit 1 honest rival.
	time.Sleep(time.Second)
	require.Equal(t, uint64(1), aliceAssertionManager.SubmittedRivals())
}

func TestComplexAssertionForkScenario(t *testing.T) {
	// Chain state looks like this:
	// 1 ->2->3->4
	//  \->2'
	//
	// and then we have another validator that disagrees with 4, so Charlie
	// should open a 4' that branches off 3.
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)

	bridgeBindings, err := mocksgen.NewBridgeStub(setup.Addrs.Bridge, setup.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(util.GetSafeCallOpts(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := setup.Chains[0]
	bobChain := setup.Chains[1]

	ctx := context.Background()
	genesisHash, err := setup.Chains[1].GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisCreationInfo, err := setup.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
	)
	aliceStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	// Bob diverges from Alice at batch 1.
	stateManagerOpts = setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
		statemanager.WithBlockDivergenceHeight(1),
		statemanager.WithMachineDivergenceStep(1),
	)
	bobStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	genesisState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 0, nil, 1<<26)
	require.NoError(t, err)
	alicePostState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 1, &genesisState.GlobalState, 1<<26)
	require.NoError(t, err)

	t.Logf("New stake from alice at post state %+v\n", alicePostState)
	aliceAssertion, err := aliceChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		alicePostState,
	)
	require.NoError(t, err)

	bobPostState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, &genesisState.GlobalState, 1<<26)
	require.NoError(t, err)
	_, err = bobChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		bobPostState,
	)
	require.NoError(t, err)
	t.Logf("New stake from bob at post state %+v\n", bobPostState)

	// Next, Alice posts more assertions on top of her assertion at batch 1.
	// We then create Charlie, who diverges from Alice at batch 4.
	count := int64(1)
	for batch := 2; batch <= 4; batch++ {
		dataHash := [32]byte{1}
		enqueueSequencerMessageAsExecutor(
			t,
			setup.Accounts[0].TxOpts,
			setup.Addrs.UpgradeExecutor,
			setup.Backend,
			setup.Addrs.Bridge,
			seqMessage{
				dataHash:                 dataHash,
				afterDelayedMessagesRead: big.NewInt(1),
				prevMessageCount:         big.NewInt(count),
				newMessageCount:          big.NewInt(count + 1),
			},
		)
		count += 1

		prevInfo, err2 := aliceChain.ReadAssertionCreationInfo(ctx, aliceAssertion.Id())
		require.NoError(t, err2)
		prevGlobalState := protocol.GoGlobalStateFromSolidity(prevInfo.AfterState.GlobalState)
		preState, err2 := aliceStateManager.ExecutionStateAfterPreviousState(ctx, uint64(batch-1), &prevGlobalState, 1<<26)
		require.NoError(t, err2)
		require.NoError(t, err2)
		alicePostState, err2 = aliceStateManager.ExecutionStateAfterPreviousState(ctx, uint64(batch), &preState.GlobalState, 1<<26)
		require.NoError(t, err2)
		t.Logf("Moving stake from alice at post state %+v\n", alicePostState)
		aliceAssertion, err = aliceChain.StakeOnNewAssertion(
			ctx,
			prevInfo,
			alicePostState,
		)
		require.NoError(t, err)
	}

	// Charlie should process Alice's creation event and determine it disagrees with batch 4
	// and then post the competing assertion.
	charlieChain := setup.Chains[2]

	stateManagerOpts = []statemanager.Opt{
		statemanager.WithNumBatchesRead(5),
		statemanager.WithBlockDivergenceHeight(36), // TODO: Make this more intuitive. This translates to batch 4 due to how our mock works.
		statemanager.WithMachineDivergenceStep(1),
	}
	charlieStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	chalManager, err := challengemanager.New(
		ctx,
		charlieChain,
		charlieStateManager,
		setup.Addrs.Rollup,
		challengemanager.WithMode(types.DefensiveMode),
		challengemanager.WithEdgeTrackerWakeInterval(time.Hour),
	)
	require.NoError(t, err)
	chalManager.Start(ctx)

	// Setup an assertion manager for Charlie, and have it process Alice's
	// assertion creation event at batch 4.
	charlieAssertionManager, err := assertions.NewManager(
		charlieChain,
		charlieStateManager,
		setup.Backend,
		chalManager,
		charlieChain.RollupAddress(),
		chalManager.ChallengeManagerAddress(),
		"charlie",
		time.Millisecond*100, // poll interval
		time.Hour,            // confirmation attempt interval
		charlieStateManager,
		time.Hour, // poll interval
		time.Second*1,
		nil,
		assertions.WithPostingDisabled(),
	)
	require.NoError(t, err)

	go charlieAssertionManager.Start(ctx)

	time.Sleep(time.Second)

	// Assert that Charlie posted the rival assertion at batch 4.
	charlieSubmitted := charlieAssertionManager.AssertionsSubmittedInProcess()
	charlieAssertion := charlieSubmitted[0]
	charlieAssertionInfo, err := charlieChain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: charlieAssertion})
	require.NoError(t, err)
	charliePostState := protocol.GoExecutionStateFromSolidity(charlieAssertionInfo.AfterState)

	// Alice and Charlie batch should match.
	require.Equal(t, charliePostState.GlobalState.Batch, alicePostState.GlobalState.Batch)
	require.Equal(t, charliePostState.GlobalState.PosInBatch, alicePostState.GlobalState.PosInBatch)

	// But blockhash should not match.
	require.NotEqual(t, charliePostState.GlobalState.BlockHash, alicePostState.GlobalState.BlockHash)
}

type seqMessage struct {
	dataHash                 common.Hash
	afterDelayedMessagesRead *big.Int
	prevMessageCount         *big.Int
	newMessageCount          *big.Int
}

func enqueueSequencerMessageAsExecutor(
	t *testing.T,
	opts *bind.TransactOpts,
	executor common.Address,
	backend *backends.SimulatedBackend,
	bridge common.Address,
	msg seqMessage,
) {
	execBindings, err := mocksgen.NewUpgradeExecutorMock(executor, backend)
	require.NoError(t, err)
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.AbsBridgeABI))
	require.NoError(t, err)
	data, err := seqInboxABI.Pack(
		"setSequencerInbox",
		executor,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()

	data, err = seqInboxABI.Pack(
		"enqueueSequencerMessage",
		msg.dataHash, msg.afterDelayedMessagesRead, msg.prevMessageCount, msg.newMessageCount,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()
}
