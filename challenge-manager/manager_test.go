// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challengemanager

import (
	"context"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/OffchainLabs/bold/testing/setup"
	customTime "github.com/OffchainLabs/bold/time"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var _ = types.ChallengeManager(&Manager{})

func TestEdgeTracker_Act(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	tkr, _ := setupEdgeTrackersForBisection(t, ctx, createdData, option.None[uint64]())
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeBisecting, tkr.CurrentState())

	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())

	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())
}

func TestEdgeTracker_Act_ChallengedEdgeCannotConfirmByTime(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)

	// Delay the evil root edge creation by half a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks / 2)

	tkr, _ := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeBisecting, tkr.CurrentState())

	// After bisecting, our edge should be in the confirming state.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())

	// However, it should not be confirmable yet as we are halfway through the challenge period.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())

	someEdge, err := chalManager.GetEdge(ctx, tkr.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, someEdge.IsNone())
	edge := someEdge.Unwrap()
	assertionHash, err := edge.AssertionHash(ctx)
	require.NoError(t, err)

	pathTimerBefore, _, _, err := tkr.Watcher().ComputeHonestPathTimer(ctx, assertionHash, edge.Id())
	require.NoError(t, err)

	// Advance our backend way beyond the challenge period.
	for i := uint64(0); i < chalPeriodBlocks; i++ {
		createdData.Backend.Commit()
	}

	pathTimerAfter, _, _, err := tkr.Watcher().ComputeHonestPathTimer(ctx, assertionHash, edge.Id())
	require.NoError(t, err)
	require.Equal(t, pathTimerBefore, pathTimerAfter)

	// Despite a lot of time having passed since the edge was created, its timer stopped halfway
	// through the challenge period as it gained a rival. That is, no matter how much time passes,
	// our edge will still not be confirmed by time.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())
}

func TestEdgeTracker_Act_ConfirmedByTime(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)

	// Delay the evil root edge creation by a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks)
	honestTracker, evilTracker := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)

	honestEdgeOpt, err := chalManager.GetEdge(ctx, honestTracker.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, honestEdgeOpt.IsNone())
	honestEdge := honestEdgeOpt.Unwrap()

	evilEdgeOpt, err := chalManager.GetEdge(ctx, evilTracker.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, evilEdgeOpt.IsNone())
	evilEdge := evilEdgeOpt.Unwrap()

	// Expect that neither of the edges have a confirmed rival.
	hasConfirmedRival, err := honestEdge.HasConfirmedRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, hasConfirmedRival)
	hasConfirmedRival, err = evilEdge.HasConfirmedRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, hasConfirmedRival)

	// Expect our edge to be confirmed right away.
	err = honestTracker.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirmed, honestTracker.CurrentState())
	require.Equal(t, true, honestTracker.ShouldDespawn(ctx))

	// Expect that the evil edge now has a confirmed rival.
	hasConfirmedRival, err = evilEdge.HasConfirmedRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, hasConfirmedRival)

	// Expect the evil tracker should despawn because it has a confirmed rival.
	require.Equal(t, true, evilTracker.ShouldDespawn(ctx))
}

func TestEdgeTracker_Act_ShouldDespawn_HasConfirmableAncestor(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)

	// Delay the evil root edge creation by a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks)
	honestParent, _ := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)

	// We manually bisect the honest, root level edge and initialize
	// edge trackers for its children.
	history, proof, err := honestParent.DetermineBisectionHistoryWithProof(ctx)
	require.NoError(t, err)
	edge, err := chalManager.GetEdge(ctx, honestParent.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, edge.IsNone())

	child1, child2, err := edge.Unwrap().Bisect(ctx, history.Merkle, proof)
	require.NoError(t, err)
	require.NoError(t, honestParent.Watcher().AddVerifiedHonestEdge(ctx, child1))
	require.NoError(t, honestParent.Watcher().AddVerifiedHonestEdge(ctx, child2))

	assertionInfo := &edgetracker.AssociatedAssertionMetadata{
		FromBatch:      0,
		ToBatch:        1,
		WasmModuleRoot: common.Hash{},
	}
	childTracker1, err := edgetracker.New(
		ctx,
		child1,
		createdData.Chains[1],
		createdData.HonestStateManager,
		honestParent.Watcher(),
		honestParent.ChallengeManager(),
		assertionInfo,
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
	)
	require.NoError(t, err)
	childTracker2, err := edgetracker.New(
		ctx,
		child2,
		createdData.Chains[1],
		createdData.HonestStateManager,
		honestParent.Watcher(),
		honestParent.ChallengeManager(),
		assertionInfo,
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
	)
	require.NoError(t, err)

	// However, the ancestor of both child edges is confirmable by time, so both of them
	// should not be allowed to act and should despawn.
	require.Equal(t, true, childTracker1.ShouldDespawn(ctx))
	require.Equal(t, true, childTracker2.ShouldDespawn(ctx))

	// We check we can also confirm the ancestor edge.
	err = honestParent.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirmed, honestParent.CurrentState())
	require.Equal(t, true, honestParent.ShouldDespawn(ctx))
}

type verifiedHonestMock struct {
	*mocks.MockSpecEdge
}

func (verifiedHonestMock) Honest() {}

func Test_getEdgeTrackers(t *testing.T) {
	ctx := context.Background()

	v, m, s := setupValidator(t)
	edge := &mocks.MockSpecEdge{}
	edge.On("Id").Return(protocol.EdgeId{Hash: common.BytesToHash([]byte("foo"))})
	edge.On("GetReversedChallengeLevel").Return(protocol.ChallengeLevel(2))
	edge.On("MutualId").Return(protocol.MutualId{})
	edge.On("CreatedAtBlock").Return(uint64(1), nil)
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash([]byte("bar"))}
	edge.On("ClaimId").Return(option.Some(protocol.ClaimId(assertionHash.Hash)))
	edge.On("AssertionHash", ctx).Return(assertionHash, nil)
	m.On("ReadAssertionCreationInfo", ctx, assertionHash).Return(&protocol.AssertionCreatedInfo{
		BeforeState: rollupgen.ExecutionState{
			GlobalState: rollupgen.GlobalState{
				U64Vals: [2]uint64{1, 0},
			},
		},
		AfterState: rollupgen.ExecutionState{
			GlobalState: rollupgen.GlobalState{
				U64Vals: [2]uint64{100, 0},
			},
		},
	}, nil)
	m.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{}).Return(&protocol.AssertionCreatedInfo{}, nil)
	s.On("ExecutionStateMsgCount", ctx, &protocol.ExecutionState{}).Return(uint64(1), nil)

	require.NoError(t, v.watcher.AddVerifiedHonestEdge(ctx, verifiedHonestMock{edge}))

	trk, err := v.getTrackerForEdge(ctx, protocol.SpecEdge(edge))
	require.NoError(t, err)

	require.Equal(t, l2stateprovider.Batch(1), trk.AssertionInfo().FromBatch)
	require.Equal(t, l2stateprovider.Batch(100), trk.AssertionInfo().ToBatch)
}

func setupEdgeTrackersForBisection(
	t *testing.T,
	ctx context.Context,
	createdData *setup.CreatedValidatorFork,
	delayEvilRootEdgeCreationByBlocks option.Option[uint64],
) (*edgetracker.Tracker, *edgetracker.Tracker) {
	honestValidator, err := New(
		ctx,
		createdData.Chains[0],
		createdData.Backend,
		createdData.HonestStateManager,
		createdData.Addrs.Rollup,
		WithName("alice"),
		WithMode(types.MakeMode),
		WithEdgeTrackerWakeInterval(100*time.Millisecond),
	)
	require.NoError(t, err)

	evilValidator, err := New(
		ctx,
		createdData.Chains[1],
		createdData.Backend,
		createdData.EvilStateManager,
		createdData.Addrs.Rollup,
		WithName("bob"),
		WithMode(types.MakeMode),
		WithEdgeTrackerWakeInterval(100*time.Millisecond),
	)
	require.NoError(t, err)

	honestEdge, _, _, err := honestValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1)
	require.NoError(t, err)

	// If we specify an optional amount of blocks to delay the evil root edge creation by, do so
	// by committing blocks to the simulated backend.
	if !delayEvilRootEdgeCreationByBlocks.IsNone() {
		delay := delayEvilRootEdgeCreationByBlocks.Unwrap()
		for i := uint64(0); i < delay; i++ {
			createdData.Backend.Commit()
		}
	}

	evilEdge, _, _, err := evilValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf2)
	require.NoError(t, err)

	// Check unrivaled statuses.
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	managerBindings, err := challengeV2gen.NewEdgeChallengeManagerCaller(chalManager.Address(), createdData.Backend)
	require.NoError(t, err)
	numBigStepLevelsRaw, err := managerBindings.NUMBIGSTEPLEVEL(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)
	numBigStepLevels := numBigStepLevelsRaw

	honestWatcher, err := watcher.New(honestValidator.chain, honestValidator, honestValidator.stateManager, createdData.Backend, time.Second, numBigStepLevels, "alice")
	require.NoError(t, err)
	honestValidator.watcher = honestWatcher
	assertionInfo := &edgetracker.AssociatedAssertionMetadata{
		FromBatch:      0,
		ToBatch:        1,
		WasmModuleRoot: common.Hash{},
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

	evilWatcher, err := watcher.New(evilValidator.chain, evilValidator, evilValidator.stateManager, createdData.Backend, time.Second, numBigStepLevels, "alice")
	require.NoError(t, err)
	evilValidator.watcher = evilWatcher
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
	require.NoError(t, honestWatcher.AddEdge(ctx, evilEdge))
	require.NoError(t, evilWatcher.AddVerifiedHonestEdge(ctx, evilEdge))
	require.NoError(t, evilWatcher.AddEdge(ctx, honestEdge))

	return tracker1, tracker2
}

func setupValidator(t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	cm := &mocks.MockSpecChallengeManager{}
	p.On("CurrentChallengeManager", ctx).Return(cm, nil)
	p.On("SpecChallengeManager", ctx).Return(cm, nil)
	cm.On("NumBigSteps", ctx).Return(uint8(1), nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager(setup.WithMockOneStepProver())
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup, WithMode(types.MakeMode), WithEdgeTrackerWakeInterval(100*time.Millisecond))
	require.NoError(t, err)
	return v, p, s
}

func TestNewRandomWakeupInterval(t *testing.T) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	cm := &mocks.MockSpecChallengeManager{}
	p.On("CurrentChallengeManager", ctx).Return(cm, nil)
	p.On("SpecChallengeManager", ctx).Return(cm, nil)
	cm.On("NumBigSteps", ctx).Return(uint8(1), nil)
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, &mocks.MockStateManager{}, cfg.Addrs.Rollup, WithMode(types.MakeMode))
	require.NoError(t, err)
	require.NotEqual(t, 0, v.edgeTrackerWakeInterval.Milliseconds())
}

func TestCanSetAPIEndpoint(t *testing.T) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	cm := &mocks.MockSpecChallengeManager{}
	p.On("CurrentChallengeManager", ctx).Return(cm, nil)
	p.On("SpecChallengeManager", ctx).Return(cm, nil)
	cm.On("NumBigSteps", ctx).Return(uint8(1), nil)
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)

	// Test we need the RPC client to enable the API service.
	_, err = New(context.Background(), p, cfg.Backend, &mocks.MockStateManager{}, cfg.Addrs.Rollup,
		WithMode(types.MakeMode), WithAPIEnabled("localhost:1234"))
	require.ErrorContains(t, err, "go-ethereum RPC client required to enable API service")

	// Test we can set the API endpoint.
	v, err := New(context.Background(), p, cfg.Backend, &mocks.MockStateManager{}, cfg.Addrs.Rollup,
		WithMode(types.MakeMode), WithAPIEnabled("localhost:1234"), WithRPCClient(&rpc.Client{}))
	require.NoError(t, err)
	require.Equal(t, "localhost:1234", v.apiAddr)
}
