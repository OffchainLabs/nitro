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
	hook := test.NewGlobal()
	tkr := newVertexTracker(util.NewArtificialTimeReference(), time.Millisecond, &protocol.Challenge{}, &protocol.ChallengeVertex{
		Commitment: util.HistoryCommitment{},
		Validator:  common.Address{},
	}, &Validator{})
	tkr.awaitingOneStepFork = true
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5)
	defer cancel()
	tkr.track(ctx)
	AssertLogsContain(t, hook, "Tracking challenge vertex")
	AssertLogsContain(t, hook, "Challenge goroutine exiting")
}

func Test_actOnBlockChallenge(t *testing.T) {
	challengeCommit := protocol.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.ChallengeCommitHash(challengeCommit.Hash())
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
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			errors.New("something went wrong"),
		)
		vertex = &protocol.ChallengeVertex{
			Commitment: history,
		}
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			Prev: util.Some(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			}),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		p.On("Completed", &protocol.ActiveTx{}).Return(
			false,
		)
		p.On("HasConfirmedAboveSeqNumber", &protocol.ActiveTx{}, vertex.SequenceNum).Return(
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			Prev: util.Some(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			}),
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			Prev:       util.None[*protocol.ChallengeVertex](),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			Prev: util.Some(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			}),
			Status: protocol.ConfirmedAssertionState,
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			Prev: util.Some(&protocol.ChallengeVertex{
				Commitment: parentHistory,
			}),
		}
		p.On("ChallengeVertexByCommitHash", &protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus}, challengeCommitHash, protocol.VertexCommitHash(history.Hash())).Return(
			vertex,
			nil,
		)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{
				WinnerAssertion: util.Some(&protocol.Assertion{}),
			},
		}
		err := tkr.actOnBlockChallenge(ctx)
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
			PresumptiveSuccessor: util.Some(vertex),
		}
		vertex.Prev = util.Some(prev)
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
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
		var vertex *protocol.ChallengeVertex
		v, err := trk.validator.stateManager.HistoryCommitmentUpTo(ctx, 5)
		require.NoError(t, err)
		err = trk.validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			vertex, err = p.ChallengeVertexByCommitHash(tx, protocol.ChallengeCommitHash(trk.challenge.ParentStateCommitment().Hash()), protocol.VertexCommitHash(v.Hash()))
			if err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, vertex)
		trk.vertex = vertex

		err = trk.actOnBlockChallenge(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
		AssertLogsContain(t, hook, "Successfully bisected to vertex")
		AssertLogsContain(t, hook, "Successfully merged to vertex with height 4")
	})
}

func Test_isAtOneStepFork(t *testing.T) {
	challengeCommit := protocol.StateCommitment{
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
		Prev: util.Some(&protocol.ChallengeVertex{
			Commitment: commitB,
		}),
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.isAtOneStepFork()
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			vertex:    vertex,
			challenge: &protocol.Challenge{},
		}
		ok, err := tkr.isAtOneStepFork()
		require.NoError(t, err)
		require.True(t, ok)
	})
}

func Test_fetchVertexByHistoryCommit(t *testing.T) {
	challengeCommit := protocol.StateCommitment{
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.fetchVertexByHistoryCommit(protocol.VertexCommitHash(history.Hash()))
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			challenge: &protocol.Challenge{},
		}
		_, err := tkr.fetchVertexByHistoryCommit(protocol.VertexCommitHash(history.Hash()))
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
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator: v,
			challenge: &protocol.Challenge{},
		}
		got, err := tkr.fetchVertexByHistoryCommit(protocol.VertexCommitHash(history.Hash()))
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func setupNonPSTracker(t *testing.T, ctx context.Context) *vertexTracker {
	stateRoots := generateStateRoots(10)
	manager := statemanager.New(stateRoots)
	leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)
	err := validator.onLeafCreated(ctx, leaf1)
	require.NoError(t, err)
	err = validator.onLeafCreated(ctx, leaf2)
	require.NoError(t, err)

	historyCommit, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
	require.NoError(t, err)

	genesisCommit := protocol.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}

	id := protocol.ChallengeCommitHash(genesisCommit.Hash())
	var challenge *protocol.Challenge
	err = validator.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		assertion, fetchErr := p.AssertionBySequenceNum(tx, protocol.AssertionSequenceNumber(1))
		if fetchErr != nil {
			return fetchErr
		}
		challenge, err = p.ChallengeByCommitHash(tx, id)
		if err != nil {
			return err
		}
		if _, err = challenge.AddLeaf(tx, assertion, historyCommit, validator.address); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	// Get the challenge vertex.
	c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, 6)
	require.NoError(t, err)

	var vertex *protocol.ChallengeVertex
	err = validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		vertex, err = p.ChallengeVertexByCommitHash(tx, id, protocol.VertexCommitHash(c.Hash()))
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, vertex)

	return newVertexTracker(util.NewArtificialTimeReference(), time.Second, challenge, vertex, validator)
}

func Test_vertexTracker_canConfirm(t *testing.T) {
	tracker := setupNonPSTracker(t, context.Background())

	// Can't confirm is vertex is confirmed or rejected
	tracker.vertex.Status = protocol.ConfirmedAssertionState
	confirmed, err := tracker.confirmed()
	require.NoError(t, err)
	require.False(t, confirmed)
	tracker.vertex.Status = protocol.RejectedAssertionState
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.False(t, confirmed)

	tracker.vertex.Status = protocol.PendingAssertionState
	// Can't confirm is parent isn't confirmed
	tracker.vertex.Prev = util.Some(&protocol.ChallengeVertex{
		Status: protocol.PendingAssertionState,
	})
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.False(t, confirmed)

	// Can confirm if vertex has won subchallenge
	tracker.vertex.Prev = util.Some(&protocol.ChallengeVertex{
		Status: protocol.ConfirmedAssertionState,
		SubChallenge: util.Some(&protocol.SubChallenge{
			Winner: tracker.vertex,
		}),
	})
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.True(t, confirmed)

	// Can't confirm if vertex is in the middle of subchallenge
	tracker.vertex.Status = protocol.PendingAssertionState
	tracker.vertex.Prev = util.Some(&protocol.ChallengeVertex{
		Status: protocol.ConfirmedAssertionState,
		SubChallenge: util.Some(&protocol.SubChallenge{
			Winner: &protocol.ChallengeVertex{},
		}),
	})
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.False(t, confirmed)

	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
	tracker.vertex.Status = protocol.PendingAssertionState
	tracker.vertex.Prev = util.Some(&protocol.ChallengeVertex{
		Status:       protocol.ConfirmedAssertionState,
		SubChallenge: util.None[*protocol.SubChallenge](),
	})
	tracker.vertex.PsTimer.Add(1000000001)
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.True(t, confirmed)

	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
	tracker.vertex.Status = protocol.PendingAssertionState
	tracker.vertex.Prev = util.Some(&protocol.ChallengeVertex{
		Status:               protocol.ConfirmedAssertionState,
		SubChallenge:         util.None[*protocol.SubChallenge](),
		PresumptiveSuccessor: util.Some(tracker.vertex),
	})
	confirmed, err = tracker.confirmed()
	require.NoError(t, err)
	require.True(t, confirmed)
}
