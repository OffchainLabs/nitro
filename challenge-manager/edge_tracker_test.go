package validator

import (
	"context"
	"io"
	"testing"
	"time"

	watcher "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/chain-watcher"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/logging"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	customTime "github.com/OffchainLabs/challenge-protocol-v2/time"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(io.Discard)
}

func Test_act(t *testing.T) {
	ctx := context.Background()
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		tkr, _ := setupNonPSTracker(ctx, t)
		err := tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeBisecting), int(tkr.fsm.Current().State))
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeConfirming), int(tkr.fsm.Current().State))
		logging.AssertLogsContain(t, hook, "Successfully bisected")
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeConfirming), int(tkr.fsm.Current().State))
	})
}

func setupNonPSTracker(ctx context.Context, t *testing.T) (*edgeTracker, *edgeTracker) {
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
	tracker1, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          customTime.NewArtificialTimeReference(),
			chain:            honestValidator.chain,
			stateManager:     honestValidator.stateManager,
			validatorName:    honestValidator.name,
			validatorAddress: honestValidator.address,
			chainWatcher:     honestWatcher,
			challengeManager: honestValidator,
		},
		honestEdge,
		0,
		1,
	)
	require.NoError(t, err)

	syncCompleted := make(chan struct{})
	go honestWatcher.Watch(ctx, syncCompleted)
	<-syncCompleted

	evilWatcher := watcher.New(evilValidator.chain, evilValidator.stateManager, createdData.Backend, time.Second, "alice")
	tracker2, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          customTime.NewArtificialTimeReference(),
			chain:            evilValidator.chain,
			stateManager:     evilValidator.stateManager,
			validatorName:    evilValidator.name,
			validatorAddress: evilValidator.address,
			chainWatcher:     evilWatcher,
			challengeManager: evilValidator,
		},
		evilEdge,
		0,
		1,
	)
	require.NoError(t, err)

	syncCompleted = make(chan struct{})
	go evilWatcher.Watch(ctx, syncCompleted)
	<-syncCompleted
	return tracker1, tracker2
}
