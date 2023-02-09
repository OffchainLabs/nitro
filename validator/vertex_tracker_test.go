package validator

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
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
	tx := &protocol.ActiveTx{}
	hook := test.NewGlobal()
	tkr := newVertexTracker(util.NewArtificialTimeReference(), time.Millisecond, &protocol.Challenge{}, &protocol.ChallengeVertex{
		Commitment: util.HistoryCommitment{},
		Validator:  common.Address{},
	}, nil, nil, "", common.Address{})
	tkr.awaitingOneStepFork = true
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5)
	defer cancel()
	tkr.track(ctx, tx)
	AssertLogsContain(t, hook, "Tracking challenge vertex")
	AssertLogsContain(t, hook, "Challenge goroutine exiting")
}

func Test_actOnBlockChallenge(t *testing.T) {
	tx := &protocol.ActiveTx{}
	challengeCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.ChallengeCommitHash(challengeCommit.Hash())
	ctx := context.Background()
	t.Run("does nothing if awaiting one step fork", func(t *testing.T) {
		tkr := &vertexTracker{
			awaitingOneStepFork: true,
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)
	})
	t.Run("fails to fetch vertex by history commit", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			errors.New("something went wrong"),
		)
		vertex = &protocol.ChallengeVertex{
			Commitment: history,
		}
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.ErrorContains(t, err, "could not refresh vertex")
	})
	t.Run("fails to check if at one-step-fork", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
			Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			})),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		p.On("Completed", &protocol.ActiveTx{}).Return(
			false,
		)
		p.On("HasConfirmedSibling", &protocol.ActiveTx{}, vertex.SequenceNum).Return(
			false, nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus},
			challengeCommitHash,
			history,
			parentHistory,
		).Return(
			false, errors.New("something went wrong"),
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.ErrorContains(t, err, "something went wrong")
	})
	t.Run("logs one-step-fork and returns", func(t *testing.T) {
		hook := test.NewGlobal()
		history := util.HistoryCommitment{
			Height: 1,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
			Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			})),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus},
			challengeCommitHash,
			history,
			parentHistory,
		).Return(
			true, nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Reached one-step-fork at 0")
	})
	t.Run("vertex's prev is nil and returns", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
			Prev:       util.None[protocol.ChallengeVertexInterface](),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.ErrorIs(t, err, ErrPrevNone)
	})
	t.Run("vertex confirmed and returns", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
			Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			})),
			Status: protocol.ConfirmedAssertionState,
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.ErrorIs(t, err, ErrConfirmed)
	})
	t.Run("challenge completed and returns", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
			Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			})),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		tkr := &vertexTracker{
			chain:  p,
			vertex: vertex,
			challenge: &protocol.Challenge{
				WinnerAssertion: util.Some(&protocol.Assertion{}),
			},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.ErrorIs(t, err, ErrChallengeCompleted)
	})
	t.Run("takes no action is presumptive", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 2,
		}
		parentHistory := util.HistoryCommitment{
			Height: 0,
		}
		p := &mocks.MockProtocol{}
		vertex := &protocol.ChallengeVertex{
			Commitment: history,
		}
		prev := &protocol.ChallengeVertex{
			Commitment:           parentHistory,
			PresumptiveSuccessor: util.Some(protocol.ChallengeVertexInterface(vertex)),
		}
		vertex.Prev = util.Some(protocol.ChallengeVertexInterface(prev))
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus},
			challengeCommitHash,
			history,
			parentHistory,
		).Return(
			false, nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)
	})
	t.Run("bisects", func(t *testing.T) {
		hook := test.NewGlobal()
		trk := setupNonPSTracker(t, ctx, tx)
		err := trk.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
	})
	t.Run("merges", func(t *testing.T) {
		hook := test.NewGlobal()
		trk := setupNonPSTracker(t, ctx, tx)
		err := trk.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)

		// Get the challenge vertex from the other validator. It should share a history
		// with the vertex we just bisected to, so it should try to merge instead.
		var vertex *protocol.ChallengeVertex
		v, err := trk.stateManager.HistoryCommitmentUpTo(ctx, 5)
		require.NoError(t, err)
		err = trk.chain.Call(func(tx *protocol.ActiveTx) error {
			parentStateCommitment, err := trk.challenge.ParentStateCommitment(ctx, tx)
			if err != nil {
				return err
			}
			vertex, err = trk.chain.ChallengeVertexByCommitHash(tx, protocol.ChallengeCommitHash(parentStateCommitment.Hash()), protocol.VertexCommitHash(v.Hash()))
			if err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, vertex)
		trk.vertex = vertex

		err = trk.actOnBlockChallenge(ctx, tx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
		AssertLogsContain(t, hook, "Successfully merged to vertex with height 4")
	})
}

func Test_isAtOneStepFork(t *testing.T) {
	tx := &protocol.ActiveTx{}
	ctx := context.Background()
	challengeCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.ChallengeCommitHash(challengeCommit.Hash())
	commitA := util.HistoryCommitment{
		Height: 1,
	}
	commitB := util.HistoryCommitment{
		Height: 2,
	}
	vertex := &protocol.ChallengeVertex{
		Commitment: commitA,
		Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
			Commitment: commitB,
		})),
	}
	t.Run("fails", func(t *testing.T) {
		p := &mocks.MockProtocol{}
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus},
			challengeCommitHash,
			commitA,
			commitB,
		).Return(
			false, errors.New("something went wrong"),
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.isAtOneStepFork(ctx, tx)
		require.ErrorContains(t, err, "something went wrong")
	})
	t.Run("OK", func(t *testing.T) {
		p := &mocks.MockProtocol{}
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus},
			challengeCommitHash,
			commitA,
			commitB,
		).Return(
			true, nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		ok, err := tkr.isAtOneStepFork(ctx, tx)
		require.NoError(t, err)
		require.True(t, ok)
	})
}

func Test_fetchVertexByHistoryCommit(t *testing.T) {
	ctx := context.Background()
	challengeCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.ChallengeCommitHash(challengeCommit.Hash())

	t.Run("nil vertex", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex, nil,
		)
		tkr := &vertexTracker{
			chain:     p,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.fetchVertexByHistoryCommit(ctx, protocol.VertexCommitHash(history.Hash()))
		require.ErrorContains(t, err, "fetched nil challenge")
	})
	t.Run("fetching error", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			errors.New("something went wrong"),
		)
		tkr := &vertexTracker{
			chain:     p,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.fetchVertexByHistoryCommit(ctx, protocol.VertexCommitHash(history.Hash()))
		require.ErrorContains(t, err, "something went wrong")
	})
	t.Run("OK", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		want := &protocol.ChallengeVertex{
			Commitment: history,
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(want, nil)
		tkr := &vertexTracker{
			chain:     p,
			challenge: &protocol.Challenge{},
		}
		got, err := tkr.fetchVertexByHistoryCommit(ctx, protocol.VertexCommitHash(history.Hash()))
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func setupNonPSTracker(t *testing.T, ctx context.Context, tx *protocol.ActiveTx) *vertexTracker {
	stateRoots := generateStateRoots(10)
	manager := statemanager.New(stateRoots)
	leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)
	err := validator.onLeafCreated(ctx, tx, leaf1)
	require.NoError(t, err)
	err = validator.onLeafCreated(ctx, tx, leaf2)
	require.NoError(t, err)

	historyCommit, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
	require.NoError(t, err)

	genesisCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}

	id := protocol.ChallengeCommitHash(genesisCommit.Hash())
	var challenge *protocol.Challenge
	err = validator.chain.Tx(func(tx *protocol.ActiveTx) error {
		assertion, fetchErr := validator.chain.AssertionBySequenceNum(tx, protocol.AssertionSequenceNumber(1))
		if fetchErr != nil {
			return fetchErr
		}
		challenge, err = validator.chain.ChallengeByCommitHash(tx, id)
		if err != nil {
			return err
		}
		if _, err = challenge.AddLeaf(ctx, tx, assertion, historyCommit, validator.address); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	// Get the challenge vertex.
	c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, 6)
	require.NoError(t, err)

	var vertex *protocol.ChallengeVertex
	err = validator.chain.Call(func(tx *protocol.ActiveTx) error {
		vertex, err = validator.chain.ChallengeVertexByCommitHash(tx, id, protocol.VertexCommitHash(c.Hash()))
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, vertex)

	return newVertexTracker(util.NewArtificialTimeReference(), time.Second, challenge, vertex, validator.chain, validator.stateManager, validator.name, validator.address)
}

func Test_vertexTracker_canConfirm(t *testing.T) {
	ctx := context.Background()
	tx := &protocol.ActiveTx{}
	tracker := setupNonPSTracker(t, ctx, tx)

	// Can't confirm is vertex is confirmed or rejected
	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.ConfirmedAssertionState
	confirmed, err := tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.False(t, confirmed)
	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.RejectedAssertionState
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.False(t, confirmed)

	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.PendingAssertionState
	// Can't confirm is parent isn't confirmed
	tracker.vertex.(*protocol.ChallengeVertex).Prev = util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
		Status: protocol.PendingAssertionState,
	}))
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.False(t, confirmed)

	// Can confirm if vertex has won subchallenge
	tracker.vertex.(*protocol.ChallengeVertex).Prev = util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
		Status: protocol.ConfirmedAssertionState,
		SubChallenge: util.Some(protocol.ChallengeInterface(&protocol.Challenge{
			WinnerVertex: util.Some(tracker.vertex),
		})),
	}))
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.True(t, confirmed)

	// Can't confirm if vertex is in the middle of subchallenge
	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.PendingAssertionState
	tracker.vertex.(*protocol.ChallengeVertex).Prev = util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
		Status: protocol.ConfirmedAssertionState,
		SubChallenge: util.Some(protocol.ChallengeInterface(&protocol.Challenge{
			WinnerVertex: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{})),
		})),
	}))
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.False(t, confirmed)

	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.PendingAssertionState
	tracker.vertex.(*protocol.ChallengeVertex).Prev = util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
		Status:       protocol.ConfirmedAssertionState,
		SubChallenge: util.None[protocol.ChallengeInterface](),
	}))
	psTimer, err := tracker.vertex.GetPsTimer(ctx, tx)
	require.NoError(t, err)
	psTimer.Add(1000000001)
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.True(t, confirmed)

	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
	tracker.vertex.(*protocol.ChallengeVertex).Status = protocol.PendingAssertionState
	tracker.vertex.(*protocol.ChallengeVertex).Prev = util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
		Status:               protocol.ConfirmedAssertionState,
		SubChallenge:         util.None[protocol.ChallengeInterface](),
		PresumptiveSuccessor: util.Some(tracker.vertex),
	}))
	confirmed, err = tracker.confirmed(ctx, tx)
	require.NoError(t, err)
	require.True(t, confirmed)
}
