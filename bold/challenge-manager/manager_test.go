// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package challengemanager

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/chain-watcher"
	"github.com/offchainlabs/nitro/bold/challenge-manager/edge-tracker"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/testing/mocks"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	customTime "github.com/offchainlabs/nitro/bold/time"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

var _ = types.RivalHandler(&Manager{})

func TestEdgeTracker_Act(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, t, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	tkr, _ := setupEdgeTrackersForBisection(t, ctx, createdData, option.None[uint64]())
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeBisecting, tkr.CurrentState())

	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeAwaitingChallengeCompletion, tkr.CurrentState())

	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeAwaitingChallengeCompletion, tkr.CurrentState())
}

func TestEdgeTracker_Act_ConfirmedByTime(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, t, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	chalManager := createdData.Chains[0].SpecChallengeManager()
	chalPeriodBlocks := chalManager.ChallengePeriodBlocks()

	// Delay the evil root edge creation by a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks)
	honestTracker, evilTracker := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)

	honestEdgeOpt, err := chalManager.GetEdge(ctx, honestTracker.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, honestEdgeOpt.IsNone())

	evilEdgeOpt, err := chalManager.GetEdge(ctx, evilTracker.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, evilEdgeOpt.IsNone())

	// Expect our edge to be confirmed right away.
	err = honestTracker.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeAwaitingChallengeCompletion, honestTracker.CurrentState())
	require.Equal(t, true, honestTracker.ShouldDespawn(ctx))
}

type verifiedHonestMock struct {
	*mocks.MockSpecEdge
}

func (verifiedHonestMock) Honest() {}

func Test_getEdgeTrackers(t *testing.T) {
	ctx := context.Background()

	v, m, s := setupValidator(ctx, t)
	edge := &mocks.MockSpecEdge{}
	honest := &mocks.MockHonestEdge{MockSpecEdge: edge}
	edge.On("Id").Return(protocol.EdgeId{Hash: common.BytesToHash([]byte("foo"))})
	edge.On("GetReversedChallengeLevel").Return(protocol.ChallengeLevel(2))
	edge.On("MutualId").Return(protocol.MutualId{})
	edge.On("OriginId").Return(protocol.OriginId{})
	edge.On("CreatedAtBlock").Return(uint64(1), nil)
	parentAssertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("par"))}
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("bar"))}
	edge.On("ClaimId").Return(option.Some(protocol.ClaimId(assertionHash.Hash)))
	edge.On("AssertionHash", ctx).Return(assertionHash, nil)
	edge.On("StartCommitment").Return(protocol.Height(0), common.Hash{})
	edge.On("EndCommitment").Return(protocol.Height(0), common.Hash{})
	edge.On("GetChallengeLevel").Return(protocol.ChallengeLevel(0))
	edge.On("MarkAsHonest").Return()
	edge.On("AsVerifiedHonest").Return(honest, true)
	m.On("ReadAssertionCreationInfo", ctx, assertionHash).Return(&protocol.AssertionCreatedInfo{
		BeforeState: rollupgen.AssertionState{
			GlobalState: rollupgen.GlobalState{
				U64Vals: [2]uint64{1, 0},
			},
		},
		AfterState: rollupgen.AssertionState{
			GlobalState: rollupgen.GlobalState{
				U64Vals: [2]uint64{100, 0},
			},
		},
		ParentAssertionHash: parentAssertionHash,
	}, nil)
	m.On("ReadAssertionCreationInfo", ctx, parentAssertionHash).Return(&protocol.AssertionCreatedInfo{
		InboxMaxCount: big.NewInt(100),
	}, nil)
	s.On("ExecutionStateMsgCount", ctx, &protocol.ExecutionState{}).Return(uint64(1), nil)

	require.NoError(t, v.watcher.AddVerifiedHonestEdge(ctx, verifiedHonestMock{edge}))
	edge.MarkAsHonest()
	verifiedRoyal, _ := edge.AsVerifiedHonest()
	trk, err := v.getTrackerForEdge(ctx, verifiedRoyal)
	require.NoError(t, err)

	require.Equal(t, l2stateprovider.Batch(1), l2stateprovider.Batch(trk.AssertionInfo().FromState.Batch))
	require.Equal(t, l2stateprovider.Batch(100), trk.AssertionInfo().BatchLimit)
}

func setupEdgeTrackersForBisection(
	t *testing.T,
	ctx context.Context,
	createdData *setup.CreatedValidatorFork,
	delayEvilRootEdgeCreationByBlocks option.Option[uint64],
) (*edgetracker.Tracker, *edgetracker.Tracker) {
	t.Helper()
	confInterval := time.Second * 10
	avgBlockTime := time.Second * 12
	honestOpts := []StackOpt{
		StackWithMode(types.MakeMode),
		StackWithName("alice"),
		StackWithConfirmationInterval(confInterval),
		StackWithAverageBlockCreationTime(avgBlockTime),
	}
	honestValidator, err := NewChallengeStack(
		createdData.Chains[0],
		createdData.HonestStateManager,
		honestOpts...,
	)
	require.NoError(t, err)

	evilOpts := honestOpts
	evilOpts = append(evilOpts, StackWithName("bob"))
	evilValidator, err := NewChallengeStack(
		createdData.Chains[1],
		createdData.EvilStateManager,
		evilOpts...,
	)
	require.NoError(t, err)

	honestEdge, _, _, _, err := honestValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1)
	require.NoError(t, err)

	// If we specify an optional amount of blocks to delay the evil root edge creation by, do so
	// by committing blocks to the simulated backend.
	if !delayEvilRootEdgeCreationByBlocks.IsNone() {
		delay := delayEvilRootEdgeCreationByBlocks.Unwrap()
		for i := uint64(0); i < delay; i++ {
			createdData.Backend.Commit()
		}
	}

	evilEdge, _, _, _, err := evilValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf2)
	require.NoError(t, err)

	// Check unrivaled statuses.
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	honestWatcher, err := watcher.New(
		honestValidator.chain,
		honestValidator.stateManager,
		"alice",
		nil,
		confInterval,
		avgBlockTime,
		nil,
		10,
	)
	require.NoError(t, err)
	honestWatcher.SetEdgeManager(honestValidator)
	honestValidator.watcher = honestWatcher
	assertionInfo := &l2stateprovider.AssociatedAssertionMetadata{
		FromState:            protocol.GoGlobalState{Batch: 0, PosInBatch: 0},
		BatchLimit:           1,
		WasmModuleRoot:       common.Hash{},
		ClaimedAssertionHash: createdData.Leaf1.Id(),
	}
	tracker1, err := edgetracker.New(
		ctx,
		honestEdge,
		honestValidator.chain,
		createdData.HonestStateManager,
		honestWatcher,
		honestValidator,
		assertionInfo,
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
		edgetracker.WithValidatorName(honestValidator.name),
	)
	require.NoError(t, err)

	evilWatcher, err := watcher.New(
		evilValidator.chain,
		evilValidator.stateManager,
		"bob",
		nil,
		confInterval,
		avgBlockTime,
		nil,
		10,
	)
	require.NoError(t, err)
	evilWatcher.SetEdgeManager(evilValidator)
	evilValidator.watcher = evilWatcher
	assertionInfo = &l2stateprovider.AssociatedAssertionMetadata{
		FromState:            protocol.GoGlobalState{Batch: 0, PosInBatch: 0},
		BatchLimit:           1,
		WasmModuleRoot:       common.Hash{},
		ClaimedAssertionHash: createdData.Leaf2.Id(),
	}
	tracker2, err := edgetracker.New(
		ctx,
		evilEdge,
		evilValidator.chain,
		createdData.EvilStateManager,
		evilWatcher,
		evilValidator,
		assertionInfo,
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
		edgetracker.WithValidatorName(evilValidator.name),
	)
	require.NoError(t, err)

	require.NoError(t, honestWatcher.AddVerifiedHonestEdge(ctx, honestEdge))
	_, err = honestWatcher.AddEdge(ctx, evilEdge)
	require.NoError(t, err)
	require.NoError(t, evilWatcher.AddVerifiedHonestEdge(ctx, evilEdge))
	_, err = evilWatcher.AddEdge(ctx, honestEdge)
	require.NoError(t, err)

	return tracker1, tracker2
}

func setupValidator(ctx context.Context, t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	cm := &mocks.MockSpecChallengeManager{}
	p.On("CurrentChallengeManager", ctx).Return(cm, nil)
	p.On("SpecChallengeManager").Return(cm)
	p.On("MaxAssertionsPerChallengePeriod").Return(uint64(100))
	cm.On("NumBigSteps").Return(uint8(1))
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager(setup.WithMockOneStepProver())
	require.NoError(t, err)
	p.On("Backend").Return(cfg.Backend, nil)
	p.On("RollupAddress").Return(cfg.Addrs.Rollup)
	p.On("StakerAddress").Return(cfg.Chains[0].StakerAddress())
	v, err := NewChallengeStack(
		p,
		s,
		StackWithMode(types.MakeMode),
		StackWithName("alice"),
	)
	require.NoError(t, err)
	return v, p, s
}
