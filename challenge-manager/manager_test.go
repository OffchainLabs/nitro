// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challengemanager

import (
	"context"
	"math/big"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/OffchainLabs/bold/testing/setup"
	customTime "github.com/OffchainLabs/bold/time"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

var _ = types.ChallengeManager(&Manager{})

func TestEdgeTracker_Act(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
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
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
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

	// Advance our backend way beyond the challenge period.
	for i := uint64(0); i < chalPeriodBlocks; i++ {
		createdData.Backend.Commit()
	}

	// Despite a lot of time having passed since the edge was created, its timer stopped halfway
	// through the challenge period as it gained a rival. That is, no matter how much time passes,
	// our edge will still not be confirmed by time.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirming, tkr.CurrentState())
}

func TestEdgeTracker_Act_ConfirmedByTime(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
	require.NoError(t, err)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)

	// Delay the evil root edge creation by a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks)
	tkr, _ := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)

	// Expect our edge to be confirmed right away.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirmed, tkr.CurrentState())
	require.Equal(t, true, tkr.ShouldDespawn(ctx))
}

func TestEdgeTracker_Act_ShouldDespawn_HasConfirmableAncestor(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
	require.NoError(t, err)

	chalManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)

	// Delay the evil root edge creation by a challenge period.
	delayEvilRootEdgeCreation := option.Some(chalPeriodBlocks)
	tkr, _ := setupEdgeTrackersForBisection(t, ctx, createdData, delayEvilRootEdgeCreation)

	// We manually bisect the honest, root level edge and initialize
	// edge trackers for its children.
	history, proof, err := tkr.DetermineBisectionHistoryWithProof(ctx)
	require.NoError(t, err)
	edge, err := chalManager.GetEdge(ctx, tkr.EdgeId())
	require.NoError(t, err)
	require.Equal(t, false, edge.IsNone())

	child1, child2, err := edge.Unwrap().Bisect(ctx, history.Merkle, proof)
	require.NoError(t, err)
	require.NoError(t, tkr.Watcher().AddVerifiedHonestEdge(ctx, child1))
	require.NoError(t, tkr.Watcher().AddVerifiedHonestEdge(ctx, child2))

	childTracker1, err := edgetracker.New(
		ctx,
		child1,
		createdData.Chains[1],
		createdData.HonestStateManager,
		tkr.Watcher(),
		tkr.ChallengeManager(),
		edgetracker.HeightConfig{
			StartBlockHeight:           0,
			TopLevelClaimEndBatchCount: 1,
		},
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
	)
	require.NoError(t, err)
	childTracker2, err := edgetracker.New(
		ctx,
		child2,
		createdData.Chains[1],
		createdData.HonestStateManager,
		tkr.Watcher(),
		tkr.ChallengeManager(),
		edgetracker.HeightConfig{
			StartBlockHeight:           0,
			TopLevelClaimEndBatchCount: 1,
		},
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
	)
	require.NoError(t, err)

	// However, the ancestor of both child edges is confirmable by time, so both of them
	// should not be allowed to act and should despawn.
	require.Equal(t, true, childTracker1.ShouldDespawn(ctx))
	require.Equal(t, true, childTracker2.ShouldDespawn(ctx))

	// We check we can also confirm the ancestor edge.
	err = tkr.Act(ctx)
	require.NoError(t, err)
	require.Equal(t, edgetracker.EdgeConfirmed, tkr.CurrentState())
	require.Equal(t, true, tkr.ShouldDespawn(ctx))
}

func Test_getEdgeTrackers(t *testing.T) {
	ctx := context.Background()

	v, m, s := setupValidator(t)
	edge := &mocks.MockSpecEdge{}
	edge.On("AssertionHash", ctx).Return(protocol.AssertionHash{}, nil)
	m.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{}).Return(&protocol.AssertionCreatedInfo{InboxMaxCount: big.NewInt(100)}, nil)
	s.On("ExecutionStateMsgCount", ctx, &protocol.ExecutionState{}).Return(uint64(1), nil)

	trk, err := v.getTrackerForEdge(ctx, protocol.SpecEdge(edge))
	require.NoError(t, err)

	require.Equal(t, uint64(1), trk.StartBlockHeight())
	require.Equal(t, uint64(0x64), trk.TopLevelClaimEndBatchCount())
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

	honestEdge, _, err := honestValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1)
	require.NoError(t, err)

	// If we specify an optional amount of blocks to delay the evil root edge creation by, do so
	// by committing blocks to the simulated backend.
	if !delayEvilRootEdgeCreationByBlocks.IsNone() {
		delay := delayEvilRootEdgeCreationByBlocks.Unwrap()
		for i := uint64(0); i < delay; i++ {
			createdData.Backend.Commit()
		}
	}

	evilEdge, _, err := evilValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf2)
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
	numBigStepLevels := uint8(numBigStepLevelsRaw.Uint64())

	honestWatcher := watcher.New(honestValidator.chain, honestValidator, honestValidator.stateManager, createdData.Backend, time.Second, numBigStepLevels, "alice")
	honestValidator.watcher = honestWatcher
	tracker1, err := edgetracker.New(
		ctx,
		honestEdge,
		honestValidator.chain,
		createdData.HonestStateManager,
		honestWatcher,
		honestValidator,
		edgetracker.HeightConfig{
			StartBlockHeight:           0,
			TopLevelClaimEndBatchCount: 1,
		},
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
		edgetracker.WithValidatorName(honestValidator.name),
	)
	require.NoError(t, err)

	go honestWatcher.Start(ctx)
	for {
		if honestWatcher.IsSynced() {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}

	evilWatcher := watcher.New(evilValidator.chain, evilValidator, evilValidator.stateManager, createdData.Backend, time.Second, numBigStepLevels, "alice")
	evilValidator.watcher = evilWatcher
	tracker2, err := edgetracker.New(
		ctx,
		evilEdge,
		evilValidator.chain,
		createdData.EvilStateManager,
		evilWatcher,
		evilValidator,
		edgetracker.HeightConfig{
			StartBlockHeight:           0,
			TopLevelClaimEndBatchCount: 1,
		},
		edgetracker.WithTimeReference(customTime.NewArtificialTimeReference()),
		edgetracker.WithValidatorName(evilValidator.name),
	)
	require.NoError(t, err)

	go evilWatcher.Start(ctx)
	for {
		if evilWatcher.IsSynced() {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
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
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup, WithMode(types.MakeMode), WithEdgeTrackerWakeInterval(100*time.Millisecond))
	require.NoError(t, err)
	return v, p, s
}
