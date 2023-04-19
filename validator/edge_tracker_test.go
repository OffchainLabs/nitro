package validator

import (
	"context"
	"io"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
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
	t.Run("logs one-step-fork and returns", func(t *testing.T) {
		hook := test.NewGlobal()
		history := util.HistoryCommitment{
			Height: 1,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		manager := &mocks.MockSpecChallengeManager{}
		edge := &mocks.MockSpecEdge{}
		edge.On("StartCommitment").Return(protocol.Height(0), parentHistory.Merkle)
		edge.On("EndCommitment").Return(protocol.Height(1), history.Merkle)
		edge.On("Id").Return(protocol.EdgeId([32]byte{}))
		edge.On("GetType").Return(protocol.BlockChallengeEdge)
		edge.On(
			"HasLengthOneRival",
			ctx,
		).Return(
			true, nil,
		)
		edge.On(
			"HasRival",
			ctx,
		).Return(
			true, nil,
		)
		p.On("SpecChallengeManager", ctx).Return(
			manager,
			nil,
		)
		manager.On("GetEdge", ctx, edge.Id()).Return(
			util.Some(protocol.SpecEdge(edge)),
			nil,
		)
		tkr, err := newEdgeTracker(
			ctx,
			&edgeTrackerConfig{
				chain: p,
			},
			edge,
			0,
			1,
		)
		require.NoError(t, err)
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeAtOneStepFork), int(tkr.fsm.Current().State))
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeAddingSubchallengeLeaf), int(tkr.fsm.Current().State))
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 0")
	})
	t.Run("takes no action is presumptive", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 2,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		manager := &mocks.MockSpecChallengeManager{}
		edge := &mocks.MockSpecEdge{}
		edge.On("StartCommitment").Return(protocol.Height(0), parentHistory.Merkle)
		edge.On("EndCommitment").Return(protocol.Height(1), history.Merkle)
		edge.On("Id").Return(protocol.EdgeId([32]byte{}))
		edge.On("GetType").Return(protocol.BlockChallengeEdge)
		edge.On(
			"HasLengthOneRival",
			ctx,
		).Return(
			false, nil,
		)
		edge.On(
			"HasRival",
			ctx,
		).Return(
			false, nil,
		)
		p.On("SpecChallengeManager", ctx).Return(
			manager,
			nil,
		)
		manager.On("GetEdge", ctx, edge.Id()).Return(
			util.Some(protocol.SpecEdge(edge)),
			nil,
		)

		tkr, err := newEdgeTracker(
			ctx,
			&edgeTrackerConfig{
				chain: p,
			},
			edge,
			0,
			1,
		)
		require.NoError(t, err)
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgePresumptive), int(tkr.fsm.Current().State))
	})
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		tkr, _ := setupNonPSTracker(ctx, t)
		err := tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(edgeBisecting), int(tkr.fsm.Current().State))
		err = tkr.act(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Successfully bisected")
	})
}

func setupNonPSTracker(ctx context.Context, t *testing.T) (*edgeTracker, *edgeTracker) {
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		DivergeHeight: 0,
		NumBlocks:     7,
	})
	require.NoError(t, err)

	honestManager, err := statemanager.NewWithAssertionStates(createdData.HonestValidatorStates, createdData.HonestValidatorInboxCounts)
	require.NoError(t, err)

	honestValidator, err := New(
		ctx,
		createdData.Chains[0],
		createdData.Backend,
		honestManager,
		createdData.Addrs.Rollup,
		WithName("alice"),
	)
	require.NoError(t, err)

	evilManager, err := statemanager.NewWithAssertionStates(createdData.EvilValidatorStates, createdData.EvilValidatorInboxCounts)
	require.NoError(t, err)

	evilValidator, err := New(
		ctx,
		createdData.Chains[1],
		createdData.Backend,
		evilManager,
		createdData.Addrs.Rollup,
		WithName("bob"),
	)
	require.NoError(t, err)

	honestValidator.assertions[createdData.Leaf1.SeqNum()] = createdData.Leaf1
	honestValidator.assertions[createdData.Leaf2.SeqNum()] = createdData.Leaf2
	honestEdge, err := honestValidator.addBlockChallengeLevelZeroEdge(ctx, 1)
	require.NoError(t, err)

	evilValidator.assertions[createdData.Leaf1.SeqNum()] = createdData.Leaf1
	evilValidator.assertions[createdData.Leaf2.SeqNum()] = createdData.Leaf2
	evilEdge, err := evilValidator.addBlockChallengeLevelZeroEdge(ctx, 1)
	require.NoError(t, err)

	// Check presumptive statuses.
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	tracker1, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          util.NewArtificialTimeReference(),
			chain:            honestValidator.chain,
			stateManager:     honestValidator.stateManager,
			validatorName:    honestValidator.name,
			validatorAddress: honestValidator.address,
		},
		honestEdge,
		0,
		1,
	)
	require.NoError(t, err)

	tracker2, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          util.NewArtificialTimeReference(),
			chain:            evilValidator.chain,
			stateManager:     evilValidator.stateManager,
			validatorName:    evilValidator.name,
			validatorAddress: evilValidator.address,
		},
		evilEdge,
		0,
		1,
	)
	require.NoError(t, err)
	require.NoError(t, err)
	return tracker1, tracker2
}
