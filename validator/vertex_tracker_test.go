package validator

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
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
		manager := &mocks.MockChallengeManager{}
		prevV := &mocks.MockChallengeVertex{
			MockHistory: parentHistory,
		}
		prevV.On(
			"ChildrenAreAtOneStepFork",
			ctx,
		).Return(
			true, nil,
		)
		vertex := &mocks.MockChallengeVertex{
			MockId:      common.Hash{},
			MockHistory: history,
			MockPrev:    util.Some(protocol.ChallengeVertex(prevV)),
			MockStatus:  protocol.AssertionPending,
		}
		vertex.On(
			"IsPresumptiveSuccessor",
			ctx,
		).Return(
			false, nil,
		)
		challenge := &mocks.MockChallenge{}
		p.On("CurrentChallengeManager", ctx).Return(
			manager,
			nil,
		)
		manager.On("GetVertex", ctx, protocol.VertexHash(vertex.Id())).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		challenge.On("Completed", ctx).Return(
			false, nil,
		)
		vertex.On("HasConfirmedSibling", ctx).Return(
			false, nil,
		)

		tkr, err := newVertexTracker(
			&vertexTrackerConfig{
				chain: p,
			},
			challenge,
			vertex,
		)
		require.NoError(t, err)
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(trackerAtOneStepFork), int(tkr.fsm.Current().State))
		err = tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(trackerOpeningSubchallenge), int(tkr.fsm.Current().State))
		AssertLogsContain(t, hook, "Reached one-step-fork at 0")
	})
	t.Run("vertex prev is nil and returns", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		manager := &mocks.MockChallengeManager{}
		p.On("CurrentChallengeManager", ctx).Return(
			manager,
			nil,
		)
		vertex := &mocks.MockChallengeVertex{
			MockHistory: history,
			MockPrev:    util.None[protocol.ChallengeVertex](),
		}
		manager.On("GetVertex", ctx, protocol.VertexHash{}).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		tkr, err := newVertexTracker(
			&vertexTrackerConfig{
				chain: p,
			},
			&mocks.MockChallenge{},
			vertex,
		)
		require.NoError(t, err)
		err = tkr.act(ctx)
		require.ErrorIs(t, err, ErrPrevNone)
	})
	t.Run("takes no action is presumptive", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 2,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		manager := &mocks.MockChallengeManager{}
		prevV := &mocks.MockChallengeVertex{
			MockHistory: parentHistory,
		}
		prevV.On(
			"ChildrenAreAtOneStepFork",
			ctx,
		).Return(
			false, nil,
		)
		vertex := &mocks.MockChallengeVertex{
			MockId:      common.Hash{},
			MockHistory: history,
			MockPrev:    util.Some(protocol.ChallengeVertex(prevV)),
			MockStatus:  protocol.AssertionPending,
		}
		challenge := &mocks.MockChallenge{}
		p.On("CurrentChallengeManager", ctx).Return(
			manager,
			nil,
		)
		manager.On("GetVertex", ctx, protocol.VertexHash(vertex.Id())).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		challenge.On("Completed", ctx).Return(
			false, nil,
		)
		vertex.On("HasConfirmedSibling", ctx).Return(
			false, nil,
		)
		vertex.On("IsPresumptiveSuccessor", ctx).Return(
			true, nil,
		)

		tkr, err := newVertexTracker(
			&vertexTrackerConfig{
				chain: p,
			},
			challenge,
			vertex,
		)
		require.NoError(t, err)
		err = tkr.act(ctx)
		require.NoError(t, err)
	})
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		tkr, _ := setupNonPSTracker(t, ctx)
		err := tkr.act(ctx)
		require.NoError(t, err)
		require.Equal(t, int(trackerBisecting), int(tkr.fsm.Current().State))
		err = tkr.act(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
	})
}

func setupNonPSTracker(t *testing.T, ctx context.Context) (*vertexTracker, *vertexTracker) {
	logsHook := test.NewGlobal()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		DivergeHeight: 32,
		NumBlocks:     63,
	})

	honestManager, err := statemanager.New(createdData.HonestValidatorStateRoots)
	require.NoError(t, err)

	honestValidator, err := New(
		ctx,
		createdData.Chains[0],
		createdData.Backend,
		honestManager,
		createdData.Addrs.Rollup,
	)
	require.NoError(t, err)

	evilManager, err := statemanager.New(createdData.EvilValidatorStateRoots)
	require.NoError(t, err)

	evilValidator, err := New(
		ctx,
		createdData.Chains[1],
		createdData.Backend,
		evilManager,
		createdData.Addrs.Rollup,
	)
	require.NoError(t, err)

	err = honestValidator.onLeafCreated(ctx, createdData.Leaf1)
	require.NoError(t, err)
	err = honestValidator.onLeafCreated(ctx, createdData.Leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New assertion appended")
	AssertLogsContain(t, logsHook, "New assertion appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

	var honestLeafVertex protocol.ChallengeVertex
	var leafVertexToBisect protocol.ChallengeVertex
	var challenge protocol.Challenge

	genesisId, err := evilValidator.chain.GetAssertionId(ctx, protocol.AssertionSequenceNumber(0))
	require.NoError(t, err)
	manager, err := evilValidator.chain.CurrentChallengeManager(ctx)
	require.NoError(t, err)
	chalIdComputed, err := manager.CalculateChallengeHash(ctx, common.Hash(genesisId), protocol.BlockChallenge)
	require.NoError(t, err)

	chal, err := manager.GetChallenge(ctx, chalIdComputed)
	require.NoError(t, err)
	require.Equal(t, false, chal.IsNone())
	assertion, err := evilValidator.chain.AssertionBySequenceNum(ctx, protocol.AssertionSequenceNumber(2))
	require.NoError(t, err)

	assertionHeight, err := assertion.Height()
	require.NoError(t, err)
	evilCommit, err := evilValidator.stateManager.HistoryCommitmentUpTo(ctx, assertionHeight)
	require.NoError(t, err)
	honestCommit, err := honestValidator.stateManager.HistoryCommitmentUpTo(ctx, assertionHeight)
	require.NoError(t, err)
	vToBisect, err := chal.Unwrap().AddBlockChallengeLeaf(ctx, assertion, evilCommit)
	require.NoError(t, err)

	honestLeafId, err := manager.CalculateChallengeVertexId(ctx, chalIdComputed, honestCommit)
	require.NoError(t, err)
	honestLeaf, err := manager.GetVertex(ctx, honestLeafId)
	require.NoError(t, err)

	honestLeafVertex = honestLeaf.Unwrap()
	leafVertexToBisect = vToBisect
	challenge = chal.Unwrap()

	// Check presumptive statuses.
	isPs, err := leafVertexToBisect.IsPresumptiveSuccessor(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	tracker1, err := newVertexTracker(
		&vertexTrackerConfig{
			timeRef:               util.NewArtificialTimeReference(),
			challengePeriodLength: time.Second,
			chain:                 evilValidator.chain,
			stateManager:          evilValidator.stateManager,
			validatorName:         evilValidator.name,
			validatorAddress:      evilValidator.address,
		},
		challenge,
		leafVertexToBisect,
		util.WithTrackedTransitions[vertexTrackerAction, vertexTrackerState](),
	)
	require.NoError(t, err)
	tracker2, err := newVertexTracker(
		&vertexTrackerConfig{
			timeRef:               util.NewArtificialTimeReference(),
			challengePeriodLength: time.Second,
			chain:                 honestValidator.chain,
			stateManager:          honestValidator.stateManager,
			validatorName:         honestValidator.name,
			validatorAddress:      honestValidator.address,
		},
		challenge,
		honestLeafVertex,
		util.WithTrackedTransitions[vertexTrackerAction, vertexTrackerState](),
	)
	require.NoError(t, err)
	return tracker1, tracker2
}

func Test_vertexTracker_canConfirm(t *testing.T) {
	ctx := context.Background()

	t.Run("already confirmed", func(t *testing.T) {
		vertex := &mocks.MockChallengeVertex{
			MockStatus: protocol.AssertionConfirmed,
		}
		tracker, err := newVertexTracker(
			&vertexTrackerConfig{},
			nil,
			vertex,
		)
		require.NoError(t, err)
		confirmed, err := tracker.confirmed(ctx)
		require.NoError(t, err)
		require.False(t, confirmed)
	})
	t.Run("no prev", func(t *testing.T) {
		vertex := &mocks.MockChallengeVertex{
			MockStatus: protocol.AssertionPending,
		}
		p := &mocks.MockProtocol{}
		tracker, err := newVertexTracker(
			&vertexTrackerConfig{
				chain: p,
			},
			nil,
			vertex,
		)
		require.NoError(t, err)
		confirmed, err := tracker.confirmed(ctx)
		require.ErrorContains(t, err, "no prev vertex")
		require.False(t, confirmed)
	})
	t.Run("prev is not confirmed", func(t *testing.T) {
		vertex := &mocks.MockChallengeVertex{
			MockStatus: protocol.AssertionPending,
			MockPrev: util.Some(protocol.ChallengeVertex(&mocks.MockChallengeVertex{
				MockStatus: protocol.AssertionPending,
			})),
		}
		p := &mocks.MockProtocol{}
		tracker, err := newVertexTracker(
			&vertexTrackerConfig{
				chain: p,
			},
			nil,
			vertex,
		)
		require.NoError(t, err)
		confirmed, err := tracker.confirmed(ctx)
		require.NoError(t, err)
		require.False(t, confirmed)
	})
}
