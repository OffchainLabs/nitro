// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/mocks"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSkipsProcessingAssertionFromEvilFork(t *testing.T) {
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockBridge(),
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

	rollupAdminBindings, err := rollupgen.NewRollupAdminLogic(setup.Addrs.Rollup, setup.Backend)
	require.NoError(t, err)
	_, err = rollupAdminBindings.SetMinimumAssertionPeriod(setup.Accounts[0].TxOpts, big.NewInt(1))
	require.NoError(t, err)
	setup.Backend.Commit()

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
	bobPostState, err := bobStateManager.ExecutionStateAfterBatchCount(ctx, 1)
	require.NoError(t, err)
	bobAssertion, err := bobChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		bobPostState,
	)
	require.NoError(t, err)

	// We have Alice process the assertion and post a rival, honest assertion to the one
	// at batch 1.
	aliceChalManager, err := challengemanager.New(
		ctx,
		aliceChain,
		setup.Backend,
		aliceStateManager,
		setup.Addrs.Rollup,
		challengemanager.WithMode(types.DefensiveMode),
		challengemanager.WithEdgeTrackerWakeInterval(time.Hour),
	)
	require.NoError(t, err)

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
		time.Hour, // poll interval
		time.Hour, // confirmation attempt interval
		aliceStateManager,
		time.Hour, // poll interval
		time.Second*1,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, aliceAssertionManager.ProcessAssertionCreationEvent(ctx, bobAssertion.Id()))

	// Get the parent assertion of what bob posted.
	creationInfo, err := bobChain.ReadAssertionCreationInfo(ctx, bobAssertion.Id())
	require.NoError(t, err)

	// Check that it has an honest children now as a result of Alice processing Bob's assertion
	// and posting her own rival, and we check she did indeed submit this rival.
	require.True(t, aliceAssertionManager.AssertionHasHonestChild(protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash}))
	require.Equal(t, uint64(1), aliceAssertionManager.SubmittedRivals())

	// We have bob post an assertion at batch 2.
	dataHash := [32]byte{1}
	_, err = bridgeBindings.EnqueueSequencerMessage(setup.Accounts[0].TxOpts, dataHash, big.NewInt(1), big.NewInt(1), big.NewInt(2))
	require.NoError(t, err)
	setup.Backend.Commit()

	bobPostState, err = bobStateManager.ExecutionStateAfterBatchCount(ctx, uint64(2))
	require.NoError(t, err)
	bobAssertion, err = bobChain.StakeOnNewAssertion(
		ctx,
		creationInfo,
		bobPostState,
	)
	require.NoError(t, err)

	// Once Alice sees this, she should do nothing
	// as it is from a bad fork and she already posted the correct child to their earliest
	// valid ancestor here.
	for i := 0; i < 10; i++ {
		require.NoError(t, aliceAssertionManager.ProcessAssertionCreationEvent(ctx, bobAssertion.Id()))
	}
	// We should only have attempted to submit 1 honest rival.
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
		setup.WithMockBridge(),
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

	rollupAdminBindings, err := rollupgen.NewRollupAdminLogic(setup.Addrs.Rollup, setup.Backend)
	require.NoError(t, err)
	_, err = rollupAdminBindings.SetMinimumAssertionPeriod(setup.Accounts[0].TxOpts, big.NewInt(1))
	require.NoError(t, err)
	setup.Backend.Commit()

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

	alicePostState, err := aliceStateManager.ExecutionStateAfterBatchCount(ctx, 1)
	require.NoError(t, err)

	t.Logf("New stake from alice at post state %+v\n", alicePostState)
	aliceAssertion, err := aliceChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		alicePostState,
	)
	require.NoError(t, err)

	bobPostState, err := bobStateManager.ExecutionStateAfterBatchCount(ctx, 1)
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
		_, err = bridgeBindings.EnqueueSequencerMessage(setup.Accounts[0].TxOpts, dataHash, big.NewInt(1), big.NewInt(count), big.NewInt(count+1))
		require.NoError(t, err)
		setup.Backend.Commit()
		count += 1

		prevInfo, err2 := aliceChain.ReadAssertionCreationInfo(ctx, aliceAssertion.Id())
		require.NoError(t, err2)
		alicePostState, err = aliceStateManager.ExecutionStateAfterBatchCount(ctx, uint64(batch))
		require.NoError(t, err)
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
		setup.Backend,
		charlieStateManager,
		setup.Addrs.Rollup,
		challengemanager.WithMode(types.DefensiveMode),
		challengemanager.WithEdgeTrackerWakeInterval(time.Hour),
	)
	require.NoError(t, err)

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
		time.Hour, // poll interval
		time.Hour, // confirmation attempt interval
		charlieStateManager,
		time.Hour, // poll interval
		time.Second*1,
		nil,
	)
	require.NoError(t, err)

	err = charlieAssertionManager.ProcessAssertionCreationEvent(ctx, aliceAssertion.Id())
	require.NoError(t, err)

	// Assert that Charlie posted the rival assertion at batch 4,
	// and also initiated a challenge.
	charlieSubmitted := charlieAssertionManager.AssertionsSubmittedInProcess()
	require.Equal(t, 1, len(charlieSubmitted))
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

func TestScanner_ProcessAssertionCreation(t *testing.T) {
	ctx := context.Background()
	t.Run("no fork detected", func(t *testing.T) {
		manager, _, mockStateProvider, cfg := setupChallengeManager(t)

		prev := &mocks.MockAssertion{
			MockPrevId:         mockId(1),
			MockId:             mockId(1),
			MockStateHash:      common.Hash{},
			MockHasSecondChild: false,
		}
		ev := &mocks.MockAssertion{
			MockPrevId:         mockId(1),
			MockId:             mockId(2),
			MockStateHash:      common.BytesToHash([]byte("bar")),
			MockHasSecondChild: false,
		}

		p := &mocks.MockProtocol{}
		cm := &mocks.MockSpecChallengeManager{}
		p.On("SpecChallengeManager", ctx).Return(cm, nil)
		p.On("ReadAssertionCreationInfo", ctx, mockId(2)).Return(&protocol.AssertionCreatedInfo{
			ParentAssertionHash: mockId(1).Hash,
			AfterState:          rollupgen.ExecutionState{},
		}, nil)
		p.On("GetAssertion", ctx, mockId(2)).Return(ev, nil)
		p.On("GetAssertion", ctx, mockId(1)).Return(prev, nil)
		mockStateProvider.On("AgreesWithExecutionState", ctx, &protocol.ExecutionState{}).Return(nil)
		scanner, err := assertions.NewManager(p, mockStateProvider, cfg.Backend, manager, cfg.Addrs.Rollup, manager.ChallengeManagerAddress(), "", time.Second, time.Second, &mocks.MockStateManager{}, time.Second, time.Second, nil)
		require.NoError(t, err)

		err = scanner.ProcessAssertionCreationEvent(ctx, ev.Id())
		require.NoError(t, err)
		require.Equal(t, uint64(1), scanner.AssertionsProcessed())
		require.Equal(t, uint64(0), scanner.ForksDetected())
		require.Equal(t, uint64(0), scanner.ChallengesSubmitted())
	})
	t.Run("fork leads validator to challenge leaf", func(t *testing.T) {
		ctx := context.Background()
		createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
			DivergeBlockHeight: 5,
		}, setup.WithMockOneStepProver())
		require.NoError(t, err)

		manager, err := challengemanager.New(
			ctx,
			createdData.Chains[1],
			createdData.Backend,
			createdData.HonestStateManager,
			createdData.Addrs.Rollup,
			challengemanager.WithMode(types.MakeMode),
			challengemanager.WithEdgeTrackerWakeInterval(100*time.Millisecond),
		)
		require.NoError(t, err)

		scanner, err := assertions.NewManager(createdData.Chains[1], createdData.HonestStateManager, createdData.Backend, manager, createdData.Addrs.Rollup, manager.ChallengeManagerAddress(), "", time.Second, time.Second, createdData.HonestStateManager, time.Second, time.Second, nil)
		require.NoError(t, err)

		err = scanner.ProcessAssertionCreationEvent(ctx, createdData.Leaf2.Id())
		require.NoError(t, err)

		otherManager, err := challengemanager.New(
			ctx,
			createdData.Chains[0],
			createdData.Backend,
			createdData.EvilStateManager,
			createdData.Addrs.Rollup,
			challengemanager.WithMode(types.MakeMode),
			challengemanager.WithEdgeTrackerWakeInterval(100*time.Millisecond),
		)
		require.NoError(t, err)

		otherScanner, err := assertions.NewManager(createdData.Chains[0], createdData.EvilStateManager, createdData.Backend, otherManager, createdData.Addrs.Rollup, otherManager.ChallengeManagerAddress(), "", time.Second, time.Second, createdData.EvilStateManager, time.Second, time.Second, nil)
		require.NoError(t, err)

		err = otherScanner.ProcessAssertionCreationEvent(ctx, createdData.Leaf1.Id())
		require.NoError(t, err)

		require.Equal(t, uint64(1), otherScanner.AssertionsProcessed())
		require.Equal(t, uint64(1), otherScanner.ChallengesSubmitted())
		require.Equal(t, uint64(1), scanner.AssertionsProcessed())
		require.Equal(t, uint64(1), scanner.ChallengesSubmitted())
	})
	t.Run("defensive validator can still challenge leaf", func(t *testing.T) {
		ctx := context.Background()
		createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
			DivergeBlockHeight: 5,
		}, setup.WithMockOneStepProver())
		require.NoError(t, err)

		manager, err := challengemanager.New(
			ctx,
			createdData.Chains[1],
			createdData.Backend,
			createdData.HonestStateManager,
			createdData.Addrs.Rollup,
			challengemanager.WithMode(types.DefensiveMode),
			challengemanager.WithEdgeTrackerWakeInterval(100*time.Millisecond),
		)
		require.NoError(t, err)
		scanner, err := assertions.NewManager(createdData.Chains[1], createdData.HonestStateManager, createdData.Backend, manager, createdData.Addrs.Rollup, manager.ChallengeManagerAddress(), "", time.Second, time.Second, createdData.HonestStateManager, time.Second, time.Second, nil)
		require.NoError(t, err)

		err = scanner.ProcessAssertionCreationEvent(ctx, createdData.Leaf2.Id())
		require.NoError(t, err)

		otherManager, err := challengemanager.New(
			ctx,
			createdData.Chains[0],
			createdData.Backend,
			createdData.EvilStateManager,
			createdData.Addrs.Rollup,
			challengemanager.WithMode(types.DefensiveMode),
			challengemanager.WithEdgeTrackerWakeInterval(100*time.Millisecond),
		)
		require.NoError(t, err)

		otherScanner, err := assertions.NewManager(createdData.Chains[0], createdData.EvilStateManager, createdData.Backend, otherManager, createdData.Addrs.Rollup, otherManager.ChallengeManagerAddress(), "", time.Second, time.Second, createdData.EvilStateManager, time.Second, time.Second, nil)
		require.NoError(t, err)

		err = otherScanner.ProcessAssertionCreationEvent(ctx, createdData.Leaf1.Id())
		require.NoError(t, err)

		require.Equal(t, uint64(1), otherScanner.AssertionsProcessed())
		require.Equal(t, uint64(1), otherScanner.ChallengesSubmitted())
		require.Equal(t, uint64(1), scanner.AssertionsProcessed())
		require.Equal(t, uint64(1), scanner.ChallengesSubmitted())
	})
}

func setupChallengeManager(t *testing.T) (*challengemanager.Manager, *mocks.MockProtocol, *mocks.MockStateManager, *setup.ChainSetup) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	cm := &mocks.MockSpecChallengeManager{}
	cm.On("NumBigSteps", ctx).Return(uint8(1), nil)
	p.On("CurrentChallengeManager", ctx).Return(cm, nil)
	p.On("SpecChallengeManager", ctx).Return(cm, nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager(setup.WithMockOneStepProver())
	require.NoError(t, err)
	v, err := challengemanager.New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup, challengemanager.WithMode(types.MakeMode), challengemanager.WithEdgeTrackerWakeInterval(100*time.Millisecond))
	require.NoError(t, err)
	return v, p, s, cfg
}

func mockId(x uint64) protocol.AssertionHash {
	return protocol.AssertionHash{Hash: common.BytesToHash([]byte(fmt.Sprintf("%d", x)))}
}
