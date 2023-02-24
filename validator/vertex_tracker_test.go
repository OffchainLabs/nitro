package validator

import (
	"context"
	"io"
	"testing"
	"time"

	"errors"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
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

func Test_track(t *testing.T) {
	t.Skip("Needs mocks")
	hook := test.NewGlobal()
	tkr := newVertexTracker(
		util.NewArtificialTimeReference(),
		time.Millisecond,
		&solimpl.Challenge{},
		&solimpl.ChallengeVertex{},
		&solimpl.AssertionChain{},
		&mocks.MockStateManager{},
		"mock-validator",
		common.Address{},
	)

	tkr.awaitingOneStepFork = true
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5)
	defer cancel()
	tkr.track(ctx)
	AssertLogsContain(t, hook, "Tracking challenge vertex")
	AssertLogsContain(t, hook, "Challenge goroutine exiting")
}

func Test_actOnBlockChallenge(t *testing.T) {
	ctx := context.Background()
	t.Run("does nothing if awaiting one step fork", func(t *testing.T) {
		tkr := &vertexTracker{
			awaitingOneStepFork: true,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.NoError(t, err)
	})
	t.Run("fails to fetch vertex by history commit", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		p.On("CurrentChallengeManager", ctx, &mocks.MockActiveTx{}).Return(
			&solimpl.ChallengeManager{},
			errors.New("something went wrong"),
		)
		vertex := &mocks.MockChallengeVertex{
			MockHistory: history,
		}
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: nil, // TODO: Populate
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.ErrorContains(t, err, "could not refresh vertex")
	})
	t.Run("pre-checks before checking is at one-step-fork", func(t *testing.T) {
		tx := &mocks.MockActiveTx{ReadWriteTx: false}
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
			tx,
		).Return(
			false, errors.New("something went wrong"),
		)
		vertex := &mocks.MockChallengeVertex{
			MockId:      common.Hash{},
			MockHistory: history,
			MockPrev:    util.Some(protocol.ChallengeVertex(prevV)),
			MockStatus:  protocol.AssertionConfirmed,
		}
		challenge := &mocks.MockChallenge{}
		p.On("CurrentChallengeManager", ctx, tx).Return(
			manager,
			nil,
		)
		manager.On("GetVertex", ctx, tx, protocol.VertexHash(vertex.Id())).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)

		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: challenge,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.ErrorIs(t, err, ErrConfirmed)

		vertex.MockStatus = protocol.AssertionPending
		challenge.On("Completed", ctx, tx).Return(
			true, nil,
		)
		vertex.On("HasConfirmedSibling", ctx, tx).Return(
			false, nil,
		)

		err = tkr.actOnBlockChallenge(ctx)
		require.ErrorIs(t, err, ErrChallengeCompleted)

		tkr = &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &solimpl.Challenge{},
		}
		err = tkr.actOnBlockChallenge(ctx)
		require.ErrorContains(t, err, "something went wrong")
	})
	t.Run("logs one-step-fork and returns", func(t *testing.T) {
		hook := test.NewGlobal()
		tx := &mocks.MockActiveTx{ReadWriteTx: false}
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
			tx,
		).Return(
			true, nil,
		)
		vertex := &mocks.MockChallengeVertex{
			MockId:      common.Hash{},
			MockHistory: history,
			MockPrev:    util.Some(protocol.ChallengeVertex(prevV)),
			MockStatus:  protocol.AssertionPending,
		}
		challenge := &mocks.MockChallenge{}
		p.On("CurrentChallengeManager", ctx, tx).Return(
			manager,
			nil,
		)
		manager.On("GetVertex", ctx, tx, protocol.VertexHash(vertex.Id())).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		challenge.On("Completed", ctx, tx).Return(
			false, nil,
		)
		vertex.On("HasConfirmedSibling", ctx, tx).Return(
			false, nil,
		)

		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: challenge,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Reached one-step-fork at 0")
	})
	t.Run("vertex prev is nil and returns", func(t *testing.T) {
		tx := &mocks.MockActiveTx{ReadWriteTx: false}
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		manager := &mocks.MockChallengeManager{}
		p.On("CurrentChallengeManager", ctx, tx).Return(
			manager,
			nil,
		)
		vertex := &mocks.MockChallengeVertex{
			MockHistory: history,
			MockPrev:    util.None[protocol.ChallengeVertex](),
		}
		manager.On("GetVertex", ctx, tx, protocol.VertexHash{}).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &mocks.MockChallenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.ErrorIs(t, err, ErrPrevNone)
	})
	t.Run("takes no action is presumptive", func(t *testing.T) {
		tx := &mocks.MockActiveTx{ReadWriteTx: false}
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
			tx,
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
		p.On("CurrentChallengeManager", ctx, tx).Return(
			manager,
			nil,
		)
		manager.On("GetVertex", ctx, tx, protocol.VertexHash(vertex.Id())).Return(
			util.Some(protocol.ChallengeVertex(vertex)),
			nil,
		)
		challenge.On("Completed", ctx, tx).Return(
			false, nil,
		)
		vertex.On("HasConfirmedSibling", ctx, tx).Return(
			false, nil,
		)
		vertex.On("IsPresumptiveSuccessor", ctx, tx).Return(
			true, nil,
		)

		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: challenge,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.NoError(t, err)
	})
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		trk := setupNonPSTracker(t, ctx)
		err := trk.actOnBlockChallenge(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
	})
	t.Run("merges", func(t *testing.T) {
		hook := test.NewGlobal()
		trk := setupNonPSTracker(t, ctx)
		err := trk.actOnBlockChallenge(ctx)
		require.NoError(t, err)

		// Get the challenge vertex from the other validator. It should share a history
		// with the vertex we just bisected to, so it should try to merge instead.
		var vertex protocol.ChallengeVertex
		honestCommit, err := trk.stateManager.HistoryCommitmentUpTo(ctx, 64)
		require.NoError(t, err)

		err = trk.chain.Call(func(tx protocol.ActiveTx) error {
			genesisId, err := trk.chain.GetAssertionId(ctx, tx, protocol.AssertionSequenceNumber(0))
			require.NoError(t, err)
			manager, err := trk.chain.CurrentChallengeManager(ctx, tx)
			require.NoError(t, err)
			chalIdComputed, err := manager.CalculateChallengeHash(ctx, tx, common.Hash(genesisId), protocol.BlockChallenge)
			require.NoError(t, err)
			vertexId, err := manager.CalculateChallengeVertexId(ctx, tx, chalIdComputed, honestCommit)
			require.NoError(t, err)
			vertexV, err := manager.GetVertex(ctx, tx, vertexId)
			require.NoError(t, err)
			vertex = vertexV.Unwrap()
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, vertex)

		trk.vertex = vertex

		err = trk.actOnBlockChallenge(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
		AssertLogsContain(t, hook, "Successfully merged to vertex with height 64")
	})
}

func setupNonPSTracker(t *testing.T, ctx context.Context) *vertexTracker {
	logsHook := test.NewGlobal()
	createdData := createTwoValidatorFork(t, ctx, 65 /* divergence point */)

	honestManager := statemanager.New(createdData.honestValidatorStateRoots)
	honestValidator, err := New(ctx, createdData.assertionChains[1], honestManager)
	require.NoError(t, err)

	evilManager := statemanager.New(createdData.evilValidatorStateRoots)
	evilValidator, err := New(ctx, createdData.assertionChains[2], evilManager)
	require.NoError(t, err)

	err = honestValidator.onLeafCreated(ctx, createdData.leaf1)
	require.NoError(t, err)
	err = honestValidator.onLeafCreated(ctx, createdData.leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

	var vertexToBisect protocol.ChallengeVertex
	var challenge protocol.Challenge

	err = evilValidator.chain.Tx(func(tx protocol.ActiveTx) error {
		genesisId, err := evilValidator.chain.GetAssertionId(ctx, tx, protocol.AssertionSequenceNumber(0))
		require.NoError(t, err)
		manager, err := evilValidator.chain.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		chalIdComputed, err := manager.CalculateChallengeHash(ctx, tx, common.Hash(genesisId), protocol.BlockChallenge)
		require.NoError(t, err)

		chal, err := manager.GetChallenge(ctx, tx, chalIdComputed)
		require.NoError(t, err)
		require.Equal(t, false, chal.IsNone())
		assertion, err := evilValidator.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(2))
		require.NoError(t, err)

		honestCommit, err := evilValidator.stateManager.HistoryCommitmentUpTo(ctx, assertion.Height())
		require.NoError(t, err)
		vToBisect, err := chal.Unwrap().AddBlockChallengeLeaf(ctx, tx, assertion, honestCommit)
		require.NoError(t, err)
		vertexToBisect = vToBisect
		challenge = chal.Unwrap()
		return nil
	})
	require.NoError(t, err)

	// Check presumptive statuses.
	err = evilValidator.chain.Tx(func(tx protocol.ActiveTx) error {
		isPs, err := vertexToBisect.IsPresumptiveSuccessor(ctx, tx)
		require.NoError(t, err)
		require.Equal(t, false, isPs)
		return nil
	})
	require.NoError(t, err)
	return newVertexTracker(util.NewArtificialTimeReference(), time.Second, challenge, vertexToBisect, evilValidator.chain, evilValidator.stateManager, evilValidator.name, evilValidator.address)
}

// func Test_vertexTracker_canConfirm(t *testing.T) {
// 	ctx := context.Background()
// 	tx := &goimpl.ActiveTx{}
// 	tracker := setupNonPSTracker(t, ctx, tx)

// 	// Can't confirm is vertex is confirmed or rejected
// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.ConfirmedAssertionState
// 	confirmed, err := tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.False(t, confirmed)
// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.RejectedAssertionState
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.False(t, confirmed)

// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.PendingAssertionState
// 	// Can't confirm is parent isn't confirmed
// 	tracker.vertex.(*goimpl.ChallengeVertex).Prev = util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 		Status: goimpl.PendingAssertionState,
// 	}))
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.False(t, confirmed)

// 	// Can confirm if vertex has won subchallenge
// 	tracker.vertex.(*goimpl.ChallengeVertex).Prev = util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 		Status: goimpl.ConfirmedAssertionState,
// 		SubChallenge: util.Some(goimpl.ChallengeInterface(&goimpl.Challenge{
// 			WinnerVertex: util.Some(tracker.vertex),
// 		})),
// 	}))
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.True(t, confirmed)

// 	// Can't confirm if vertex is in the middle of subchallenge
// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.PendingAssertionState
// 	tracker.vertex.(*goimpl.ChallengeVertex).Prev = util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 		Status: goimpl.ConfirmedAssertionState,
// 		SubChallenge: util.Some(goimpl.ChallengeInterface(&goimpl.Challenge{
// 			WinnerVertex: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{})),
// 		})),
// 	}))
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.False(t, confirmed)

// 	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.PendingAssertionState
// 	tracker.vertex.(*goimpl.ChallengeVertex).Prev = util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 		Status:       goimpl.ConfirmedAssertionState,
// 		SubChallenge: util.None[goimpl.ChallengeInterface](),
// 	}))
// 	psTimer, err := tracker.vertex.GetPsTimer(ctx, tx)
// 	require.NoError(t, err)
// 	psTimer.Add(1000000001)
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.True(t, confirmed)

// 	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
// 	tracker.vertex.(*goimpl.ChallengeVertex).Status = goimpl.PendingAssertionState
// 	tracker.vertex.(*goimpl.ChallengeVertex).Prev = util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 		Status:               goimpl.ConfirmedAssertionState,
// 		SubChallenge:         util.None[goimpl.ChallengeInterface](),
// 		PresumptiveSuccessor: util.Some(tracker.vertex),
// 	}))
// 	confirmed, err = tracker.confirmed(ctx, tx)
// 	require.NoError(t, err)
// 	require.True(t, confirmed)
// }
