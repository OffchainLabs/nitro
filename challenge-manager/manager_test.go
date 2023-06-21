package challengemanager

import (
	"context"
	"io"
	"testing"
	"time"

	watcher "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/logging"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	customTime "github.com/OffchainLabs/challenge-protocol-v2/time"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

var _ = ChallengeCreator(&Manager{})

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(io.Discard)
}

func TestEdgeTracker_act(t *testing.T) {
	ctx := context.Background()
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		tkr, _ := setupNonPSTracker(ctx, t)
		err := tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(4), int(tkr.CurrentState()))
		err = tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(5), int(tkr.CurrentState()))
		logging.AssertLogsContain(t, hook, "Successfully bisected")
		err = tkr.Act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(5), int(tkr.CurrentState()))
	})
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
	)
	require.NoError(t, err)

	evilValidator, err := New(
		ctx,
		createdData.Chains[1],
		createdData.Backend,
		createdData.EvilStateManager,
		createdData.Addrs.Rollup,
		WithName("bob"),
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

	honestWatcher := watcher.New(honestValidator.chain, honestValidator.stateManager, createdData.Backend, time.Second, "alice")
	honestValidator.watcher = honestWatcher
	tracker1, err := edgetracker.New(
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
		edgetracker.WithValidatorAddress(honestValidator.address),
		edgetracker.WithValidatorName(honestValidator.name),
	)
	require.NoError(t, err)

	syncCompleted := make(chan struct{})
	go honestWatcher.Watch(ctx, syncCompleted)
	<-syncCompleted

	evilWatcher := watcher.New(evilValidator.chain, evilValidator.stateManager, createdData.Backend, time.Second, "alice")
	evilValidator.watcher = evilWatcher
	tracker2, err := edgetracker.New(
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
		edgetracker.WithValidatorAddress(evilValidator.address),
		edgetracker.WithValidatorName(evilValidator.name),
	)
	require.NoError(t, err)

	syncCompleted = make(chan struct{})
	go evilWatcher.Watch(ctx, syncCompleted)
	<-syncCompleted
	return tracker1, tracker2
}

func setupValidator(t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	p.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup)
	require.NoError(t, err)
	return v, p, s
}
