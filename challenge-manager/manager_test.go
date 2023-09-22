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
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/OffchainLabs/bold/testing/setup"
	customTime "github.com/OffchainLabs/bold/time"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

var _ = types.ChallengeManager(&Manager{})

func TestEdgeTracker_act(t *testing.T) {
	ctx := context.Background()
	t.Run("bisects", func(t *testing.T) {
		tkr, _ := setupNonPSTracker(ctx, t)
		err := tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, 4, int(tkr.CurrentState()))
		err = tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, int(tkr.CurrentState()))
		err = tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, int(tkr.CurrentState()))
	})
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

func setupNonPSTracker(ctx context.Context, t *testing.T) (*edgetracker.Tracker, *edgetracker.Tracker) {
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
	require.NoError(t, err)

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

	evilEdge, _, err := evilValidator.addBlockChallengeLevelZeroEdge(ctx, createdData.Leaf2)
	require.NoError(t, err)

	// Check presumptive statuses.
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
