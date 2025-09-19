// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package assertions_test

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/assertions"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/casttest"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

func TestSkipsProcessingAssertionFromEvilFork(t *testing.T) {
	t.Skip("Flakey test, needs investigation")
	testData, err := setup.ChainsWithEdgeChallengeManager(
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

	bridgeBindings, err := mocksgen.NewBridgeStub(testData.Addrs.Bridge, testData.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(testData.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := testData.Chains[0]
	bobChain := testData.Chains[1]
	charlieChain := testData.Chains[2]

	ctx := context.Background()
	genesisHash, err := testData.Chains[1].GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisCreationInfo, err := testData.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
	)
	aliceStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	// Bob diverges from Alice at batch 1.
	stateManagerOpts = testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
		stateprovider.WithBlockDivergenceHeight(1),
		stateprovider.WithMachineDivergenceStep(1),
	)
	bobStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	aliceChalManager, err := challengemanager.NewChallengeStack(
		aliceChain,
		aliceStateManager,
		challengemanager.StackWithMode(types.DefensiveMode),
		challengemanager.StackWithName("alice"),
	)
	require.NoError(t, err)
	aliceChalManager.Start(ctx)

	// We have bob post an assertion at batch 1.
	//
	// It is important that this assertion is posted after alice's challenge
	// manager is started, and before Charlie's assertion manager is started. If
	// this happens before alice's challenge manager then, alice and charlie will
	// race to post the rival assertion to the one from bob. Even though, alice's
	// polling interval is a minute, the very first time she poll's, she'll
	// already see bob's rival assertion and attempt to post the rival. Only one
	// rival assertion can be posted with identical content, so, if alice wins,
	// charlie's attempt below will fail because the rival assertion he is trying
	// to post already exists.
	genesisGlobalState := protocol.GoGlobalStateFromSolidity(genesisCreationInfo.AfterState.GlobalState)
	bobPostState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesisGlobalState)
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

	// Setup an assertion manager for Charlie, and have it process Alice's
	// assertion creation event at batch 1.
	charlieAssertionManager, err := assertions.NewManager(
		charlieChain,
		aliceStateManager,
		"charlie",
		types.DefensiveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithPostingDisabled(),
	)
	require.NoError(t, err)
	charlieAssertionManager.SetRivalHandler(aliceChalManager)

	charlieAssertionManager.Start(ctx)

	// Check that charlie submitted a rival to Bob's assertion after some time.
	// Charlie does this because he agrees with Alice's assertion at batch 1.
	time.Sleep(time.Second)
	require.Equal(t, uint64(1), charlieAssertionManager.SubmittedRivals())

	// We have bob post an assertion at batch 2.
	dataHash := [32]byte{1}
	enqueueSequencerMessageAsExecutor(
		t,
		testData.Accounts[0].TxOpts,
		testData.Addrs.UpgradeExecutor,
		testData.Backend,
		testData.Addrs.Bridge,
		seqMessage{
			dataHash:                 dataHash,
			afterDelayedMessagesRead: big.NewInt(1),
			prevMessageCount:         big.NewInt(1),
			newMessageCount:          big.NewInt(2),
		},
	)

	genesisState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	preState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesisState.GlobalState)
	require.NoError(t, err)
	bobPostState, err = bobStateManager.ExecutionStateAfterPreviousState(ctx, 2, preState.GlobalState)
	require.NoError(t, err)
	_, err = bobChain.StakeOnNewAssertion(
		ctx,
		bobAssertionInfo,
		bobPostState,
	)
	require.NoError(t, err)

	// Once Charlie sees this, he should do nothing as it is from a bad fork and
	// he already posted the correct child to their earliest valid ancestor here.
	// Charlie should only have attempted to submit 1 honest rival.
	time.Sleep(time.Second)
	require.Equal(t, uint64(1), charlieAssertionManager.SubmittedRivals())
}

func TestComplexAssertionForkScenario(t *testing.T) {
	// Chain state looks like this:
	// 1 ->2->3->4
	//  \->2'
	//
	// and then we have another validator that disagrees with 4, so Charlie
	// should open a 4' that branches off 3.
	testData, err := setup.ChainsWithEdgeChallengeManager(
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

	bridgeBindings, err := mocksgen.NewBridgeStub(testData.Addrs.Bridge, testData.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(testData.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := testData.Chains[0]
	bobChain := testData.Chains[1]

	ctx := context.Background()
	genesisHash, err := testData.Chains[1].GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisCreationInfo, err := testData.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
	)
	aliceStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	// Bob diverges from Alice at batch 1.
	stateManagerOpts = testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
		stateprovider.WithBlockDivergenceHeight(1),
		stateprovider.WithMachineDivergenceStep(1),
	)
	bobStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	genesisState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	alicePostState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesisState.GlobalState)
	require.NoError(t, err)

	t.Logf("New stake from alice at post state %+v\n", alicePostState)
	aliceAssertion, err := aliceChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		alicePostState,
	)
	require.NoError(t, err)

	bobPostState, err := bobStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesisState.GlobalState)
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
			testData.Accounts[0].TxOpts,
			testData.Addrs.UpgradeExecutor,
			testData.Backend,
			testData.Addrs.Bridge,
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
		preState, err2 := aliceStateManager.ExecutionStateAfterPreviousState(ctx, casttest.ToUint64(t, batch-1), prevGlobalState)
		require.NoError(t, err2)
		alicePostState, err2 = aliceStateManager.ExecutionStateAfterPreviousState(ctx, casttest.ToUint64(t, batch), preState.GlobalState)
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
	charlieChain := testData.Chains[2]

	stateManagerOpts = []stateprovider.Opt{
		stateprovider.WithNumBatchesRead(5),
		stateprovider.WithBlockDivergenceHeight(36), // TODO: Make this more intuitive. This translates to batch 4 due to how our mock works.
		stateprovider.WithMachineDivergenceStep(1),
	}
	charlieStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	// Setup an assertion manager for Charlie, and have it process Alice's
	// assertion creation event at batch 4.
	charlieAssertionManager, err := assertions.NewManager(
		charlieChain,
		charlieStateManager,
		"charlie",
		types.DefensiveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithPostingDisabled(),
	)
	require.NoError(t, err)

	chalManager, err := challengemanager.NewChallengeStack(
		charlieChain,
		charlieStateManager,
		challengemanager.OverrideAssertionManager(charlieAssertionManager),
		challengemanager.StackWithMode(types.DefensiveMode),
		challengemanager.StackWithName("charlie"),
	)
	require.NoError(t, err)
	chalManager.Start(ctx)

	time.Sleep(time.Second * 2)

	// Assert that Charlie posted the rival assertion at batch 4.
	charlieSubmitted := charlieAssertionManager.AssertionsSubmittedInProcess()
	require.Equal(t, true, len(charlieSubmitted) > 0)
	charlieAssertion := charlieSubmitted[0]
	charlieAssertionInfo, err := charlieChain.ReadAssertionCreationInfo(ctx, charlieAssertion)
	require.NoError(t, err)
	charliePostState := protocol.GoExecutionStateFromSolidity(charlieAssertionInfo.AfterState)

	// Alice and Charlie batch should match.
	require.Equal(t, charliePostState.GlobalState.Batch, alicePostState.GlobalState.Batch)
	require.Equal(t, charliePostState.GlobalState.PosInBatch, alicePostState.GlobalState.PosInBatch)

	// But blockhash should not match.
	require.NotEqual(t, charliePostState.GlobalState.BlockHash, alicePostState.GlobalState.BlockHash)
}

func TestFastConfirmation(t *testing.T) {
	ctx := context.Background()
	testData, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithAutoDeposit(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
		setup.WithFastConfirmation(),
	)
	require.NoError(t, err)

	bridgeBindings, err := mocksgen.NewBridgeStub(testData.Addrs.Bridge, testData.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(testData.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := testData.Chains[0]

	stateManagerOpts := testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
	)
	stateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	assertionManager, err := assertions.NewManager(
		aliceChain,
		stateManager,
		"alice",
		types.ResolveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithFastConfirmation(),
	)
	require.NoError(t, err)

	chalManager, err := challengemanager.NewChallengeStack(
		aliceChain,
		stateManager,
		challengemanager.OverrideAssertionManager(assertionManager),
		challengemanager.StackWithMode(types.ResolveMode),
		challengemanager.StackWithName("alice"),
	)
	require.NoError(t, err)
	chalManager.Start(ctx)

	preState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	postState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 1, preState.GlobalState)
	require.NoError(t, err)

	time.Sleep(time.Second)

	posted, err := assertionManager.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())
	creationInfo, err := aliceChain.ReadAssertionCreationInfo(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	require.Equal(t, postState, protocol.GoExecutionStateFromSolidity(creationInfo.AfterState))

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	expectAssertionConfirmed(t, ctx, aliceChain.Backend(), aliceChain.RollupAddress())
}

func TestFastConfirmationWithSafe(t *testing.T) {
	ctx := context.Background()
	testData, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithAutoDeposit(),
		setup.WithMockOneStepProver(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
		setup.WithSafeFastConfirmation(),
	)
	require.NoError(t, err)

	bridgeBindings, err := mocksgen.NewBridgeStub(testData.Addrs.Bridge, testData.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(testData.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := testData.Chains[0]
	bobChain := testData.Chains[1]

	stateManagerOpts := testData.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
	)
	stateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	assertionManagerAlice, err := assertions.NewManager(
		aliceChain,
		stateManager,
		"alice",
		types.ResolveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithDangerousReadyToPost(),
		assertions.WithPostingDisabled(),
		assertions.WithFastConfirmation(),
	)
	require.NoError(t, err)

	chalManagerAlice, err := challengemanager.NewChallengeStack(
		aliceChain,
		stateManager,
		challengemanager.OverrideAssertionManager(assertionManagerAlice),
		challengemanager.StackWithMode(types.ResolveMode),
		challengemanager.StackWithName("alice"),
	)
	require.NoError(t, err)
	chalManagerAlice.Start(ctx)

	preState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	postState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 1, preState.GlobalState)
	require.NoError(t, err)

	time.Sleep(time.Second)

	posted, err := assertionManagerAlice.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())
	creationInfo, err := aliceChain.ReadAssertionCreationInfo(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	require.Equal(t, postState, protocol.GoExecutionStateFromSolidity(creationInfo.AfterState))

	<-time.After(time.Second)
	status, err := aliceChain.AssertionStatus(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	// Just one fast confirmation is not enough to confirm the assertion.
	require.Equal(t, protocol.AssertionPending, status)

	assertionManagerBob, err := assertions.NewManager(
		bobChain,
		stateManager,
		"bob",
		types.ResolveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(0),
		assertions.WithDangerousReadyToPost(),
		assertions.WithPostingDisabled(),
		assertions.WithFastConfirmation(),
	)
	require.NoError(t, err)

	chalManagerBob, err := challengemanager.NewChallengeStack(
		bobChain,
		stateManager,
		challengemanager.OverrideAssertionManager(assertionManagerBob),
		challengemanager.StackWithMode(types.ResolveMode),
		challengemanager.StackWithName("bob"),
	)
	require.NoError(t, err)
	chalManagerBob.Start(ctx)

	// Only after both Alice and Bob confirm the assertion, it should be confirmed.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	expectAssertionConfirmed(t, ctx, aliceChain.Backend(), aliceChain.RollupAddress())
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
	backend *setup.SimulatedBackendWrapper,
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

func expectAssertionConfirmed(
	t *testing.T,
	ctx context.Context,
	backend protocol.ChainBackend,
	rollupAddr common.Address,
) {
	rc, err := rollupgen.NewRollupCore(rollupAddr, backend)
	require.NoError(t, err)
	var confirmed bool
	for ctx.Err() == nil && !confirmed {
		i, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupCoreAssertionConfirmedIterator, error) {
			return rc.FilterAssertionConfirmed(nil, nil)
		})
		require.NoError(t, err)
		for i.Next() {
			assertionNode, err := retry.UntilSucceeds(ctx, func() (rollupgen.AssertionNode, error) {
				return rc.GetAssertion(&bind.CallOpts{Context: ctx}, i.Event.AssertionHash)
			})
			require.NoError(t, err)
			if assertionNode.Status != uint8(protocol.AssertionConfirmed) {
				t.Fatal("Confirmed assertion with unfinished state")
			}
			confirmed = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !confirmed {
		t.Fatal("assertion was not confirmed")
	}
}
