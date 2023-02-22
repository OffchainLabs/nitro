package validator

// import (
// 	"context"
// 	"errors"
// 	"io"
// 	"testing"
// 	"time"

// 	"github.com/OffchainLabs/challenge-protocol-v2/protocol/go-implementation"
// 	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
// 	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
// 	"github.com/OffchainLabs/challenge-protocol-v2/util"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/sirupsen/logrus"
// 	"github.com/sirupsen/logrus/hooks/test"
// 	"github.com/stretchr/testify/require"
// )

// func init() {
// 	logrus.SetLevel(logrus.DebugLevel)
// 	logrus.SetOutput(io.Discard)
// }

// func Test_track(t *testing.T) {
// 	tx := &goimpl.ActiveTx{}
// 	hook := test.NewGlobal()
// 	tkr := newVertexTracker(util.NewArtificialTimeReference(), time.Millisecond, &goimpl.Challenge{}, &goimpl.ChallengeVertex{
// 		Commitment: util.HistoryCommitment{},
// 		Validator:  common.Address{},
// 	}, nil, nil, "", common.Address{})
// 	tkr.awaitingOneStepFork = true
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5)
// 	defer cancel()
// 	tkr.track(ctx, tx)
// 	AssertLogsContain(t, hook, "Tracking challenge vertex")
// 	AssertLogsContain(t, hook, "Challenge goroutine exiting")
// }

// func Test_actOnBlockChallenge(t *testing.T) {
// 	tx := &goimpl.ActiveTx{}
// 	challengeCommit := util.StateCommitment{
// 		Height:    0,
// 		StateRoot: common.Hash{},
// 	}
// 	challengeCommitHash := goimpl.ChallengeCommitHash(challengeCommit.Hash())
// 	ctx := context.Background()
// 	t.Run("does nothing if awaiting one step fork", func(t *testing.T) {
// 		tkr := &vertexTracker{
// 			awaitingOneStepFork: true,
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)
// 	})
// 	t.Run("fails to fetch vertex by history commit", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		p := &mocks.MockProtocol{}
// 		var vertex *goimpl.ChallengeVertex
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			errors.New("something went wrong"),
// 		)
// 		vertex = &goimpl.ChallengeVertex{
// 			Commitment: history,
// 		}
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.ErrorContains(t, err, "could not refresh vertex")
// 	})
// 	t.Run("fails to check if at one-step-fork", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		parentHistory := util.HistoryCommitment{
// 			Height: 0,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 			Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 				Commitment: parentHistory,
// 			})),
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		p.On("Completed", &goimpl.ActiveTx{}).Return(
// 			false,
// 		)
// 		p.On("HasConfirmedSibling", &goimpl.ActiveTx{}, vertex.SequenceNum).Return(
// 			false, nil,
// 		)
// 		p.On(
// 			"IsAtOneStepFork",
// 			&goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus},
// 			challengeCommitHash,
// 			history,
// 			parentHistory,
// 		).Return(
// 			false, errors.New("something went wrong"),
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.ErrorContains(t, err, "something went wrong")
// 	})
// 	t.Run("logs one-step-fork and returns", func(t *testing.T) {
// 		hook := test.NewGlobal()
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		parentHistory := util.HistoryCommitment{
// 			Height: 0,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 			Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 				Commitment: parentHistory,
// 			})),
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		p.On(
// 			"IsAtOneStepFork",
// 			&goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus},
// 			challengeCommitHash,
// 			history,
// 			parentHistory,
// 		).Return(
// 			true, nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)
// 		AssertLogsContain(t, hook, "Reached one-step-fork at 0")
// 	})
// 	t.Run("vertex's prev is nil and returns", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 			Prev:       util.None[goimpl.ChallengeVertexInterface](),
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.ErrorIs(t, err, ErrPrevNone)
// 	})
// 	t.Run("vertex confirmed and returns", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		parentHistory := util.HistoryCommitment{
// 			Height: 0,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 			Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 				Commitment: parentHistory,
// 			})),
// 			Status: goimpl.ConfirmedAssertionState,
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.ErrorIs(t, err, ErrConfirmed)
// 	})
// 	t.Run("challenge completed and returns", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		parentHistory := util.HistoryCommitment{
// 			Height: 0,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 			Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 				Commitment: parentHistory,
// 			})),
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:  p,
// 			vertex: vertex,
// 			challenge: &goimpl.Challenge{
// 				WinnerAssertion: util.Some(&goimpl.Assertion{}),
// 			},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.ErrorIs(t, err, ErrChallengeCompleted)
// 	})
// 	t.Run("takes no action is presumptive", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 2,
// 		}
// 		parentHistory := util.HistoryCommitment{
// 			Height: 0,
// 		}
// 		p := &mocks.MockProtocol{}
// 		vertex := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 		}
// 		prev := &goimpl.ChallengeVertex{
// 			Commitment:           parentHistory,
// 			PresumptiveSuccessor: util.Some(goimpl.ChallengeVertexInterface(vertex)),
// 		}
// 		vertex.Prev = util.Some(goimpl.ChallengeVertexInterface(prev))
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			nil,
// 		)
// 		p.On(
// 			"IsAtOneStepFork",
// 			&goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus},
// 			challengeCommitHash,
// 			history,
// 			parentHistory,
// 		).Return(
// 			false, nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		err := tkr.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)
// 	})
// 	t.Run("bisects", func(t *testing.T) {
// 		hook := test.NewGlobal()
// 		trk := setupNonPSTracker(t, ctx, tx)
// 		err := trk.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)
// 		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
// 		AssertLogsContain(t, hook, "Successfully bisected to vertex")
// 	})
// 	t.Run("merges", func(t *testing.T) {
// 		hook := test.NewGlobal()
// 		trk := setupNonPSTracker(t, ctx, tx)
// 		err := trk.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)

// 		// Get the challenge vertex from the other validator. It should share a history
// 		// with the vertex we just bisected to, so it should try to merge instead.
// 		var vertex *goimpl.ChallengeVertex
// 		v, err := trk.stateManager.HistoryCommitmentUpTo(ctx, 5)
// 		require.NoError(t, err)
// 		err = trk.chain.Call(func(tx *goimpl.ActiveTx) error {
// 			var parentStateCommitment util.StateCommitment
// 			parentStateCommitment, err = trk.challenge.ParentStateCommitment(ctx, tx)
// 			if err != nil {
// 				return err
// 			}
// 			vertex, err = trk.chain.ChallengeVertexByCommitHash(tx, goimpl.ChallengeCommitHash(parentStateCommitment.Hash()), goimpl.VertexCommitHash(v.Hash()))
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		})
// 		require.NoError(t, err)
// 		require.NotNil(t, vertex)
// 		trk.vertex = vertex

// 		err = trk.actOnBlockChallenge(ctx, tx)
// 		require.NoError(t, err)
// 		AssertLogsContain(t, hook, "Challenge vertex goroutine acting")
// 		AssertLogsContain(t, hook, "Successfully bisected to vertex")
// 		AssertLogsContain(t, hook, "Successfully merged to vertex with height 4")
// 	})
// }

// func Test_isAtOneStepFork(t *testing.T) {
// 	tx := &goimpl.ActiveTx{}
// 	ctx := context.Background()
// 	challengeCommit := util.StateCommitment{
// 		Height:    0,
// 		StateRoot: common.Hash{},
// 	}
// 	challengeCommitHash := goimpl.ChallengeCommitHash(challengeCommit.Hash())
// 	commitA := util.HistoryCommitment{
// 		Height: 1,
// 	}
// 	commitB := util.HistoryCommitment{
// 		Height: 2,
// 	}
// 	vertex := &goimpl.ChallengeVertex{
// 		Commitment: commitA,
// 		Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 			Commitment: commitB,
// 		})),
// 	}
// 	t.Run("fails", func(t *testing.T) {
// 		p := &mocks.MockProtocol{}
// 		p.On(
// 			"IsAtOneStepFork",
// 			&goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus},
// 			challengeCommitHash,
// 			commitA,
// 			commitB,
// 		).Return(
// 			false, errors.New("something went wrong"),
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		_, err := tkr.isAtOneStepFork(ctx, tx)
// 		require.ErrorContains(t, err, "something went wrong")
// 	})
// 	t.Run("OK", func(t *testing.T) {
// 		p := &mocks.MockProtocol{}
// 		p.On(
// 			"IsAtOneStepFork",
// 			&goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus},
// 			challengeCommitHash,
// 			commitA,
// 			commitB,
// 		).Return(
// 			true, nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			vertex:    vertex,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		ok, err := tkr.isAtOneStepFork(ctx, tx)
// 		require.NoError(t, err)
// 		require.True(t, ok)
// 	})
// }

// func Test_fetchVertexByHistoryCommit(t *testing.T) {
// 	ctx := context.Background()
// 	challengeCommit := util.StateCommitment{
// 		Height:    0,
// 		StateRoot: common.Hash{},
// 	}
// 	challengeCommitHash := goimpl.ChallengeCommitHash(challengeCommit.Hash())

// 	t.Run("nil vertex", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		p := &mocks.MockProtocol{}
// 		var vertex *goimpl.ChallengeVertex
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex, nil,
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		_, err := tkr.fetchVertexByHistoryCommit(ctx, goimpl.VertexCommitHash(history.Hash()))
// 		require.ErrorContains(t, err, "fetched nil challenge")
// 	})
// 	t.Run("fetching error", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		p := &mocks.MockProtocol{}
// 		var vertex *goimpl.ChallengeVertex
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(
// 			vertex,
// 			errors.New("something went wrong"),
// 		)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		_, err := tkr.fetchVertexByHistoryCommit(ctx, goimpl.VertexCommitHash(history.Hash()))
// 		require.ErrorContains(t, err, "something went wrong")
// 	})
// 	t.Run("OK", func(t *testing.T) {
// 		history := util.HistoryCommitment{
// 			Height: 1,
// 		}
// 		p := &mocks.MockProtocol{}
// 		want := &goimpl.ChallengeVertex{
// 			Commitment: history,
// 		}
// 		p.On("ChallengeVertexByCommitHash", &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}, challengeCommitHash, goimpl.VertexCommitHash(history.Hash())).Return(want, nil)
// 		tkr := &vertexTracker{
// 			chain:     p,
// 			challenge: &goimpl.Challenge{},
// 		}
// 		got, err := tkr.fetchVertexByHistoryCommit(ctx, goimpl.VertexCommitHash(history.Hash()))
// 		require.NoError(t, err)
// 		require.Equal(t, want, got)
// 	})
// }

// func setupNonPSTracker(t *testing.T, ctx context.Context, tx *goimpl.ActiveTx) *vertexTracker {
// 	stateRoots := generateStateRoots(10)
// 	manager := statemanager.New(stateRoots)
// 	leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)
// 	err := validator.onLeafCreated(ctx, tx, leaf1)
// 	require.NoError(t, err)
// 	err = validator.onLeafCreated(ctx, tx, leaf2)
// 	require.NoError(t, err)

// 	historyCommit, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
// 	require.NoError(t, err)

// 	genesisCommit := util.StateCommitment{
// 		Height:    0,
// 		StateRoot: common.Hash{},
// 	}

// 	id := goimpl.ChallengeCommitHash(genesisCommit.Hash())
// 	var challenge goimpl.ChallengeInterface
// 	err = validator.chain.Tx(func(tx *goimpl.ActiveTx) error {
// 		assertion, fetchErr := validator.chain.AssertionBySequenceNum(tx, goimpl.AssertionSequenceNumber(1))
// 		if fetchErr != nil {
// 			return fetchErr
// 		}
// 		challenge, err = validator.chain.ChallengeByCommitHash(tx, id)
// 		if err != nil {
// 			return err
// 		}
// 		if _, err = challenge.AddLeaf(ctx, tx, assertion, historyCommit, validator.address); err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// 	require.NoError(t, err)

// 	// Get the challenge vertex.
// 	c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, 6)
// 	require.NoError(t, err)

// 	var vertex goimpl.ChallengeVertexInterface
// 	err = validator.chain.Call(func(tx *goimpl.ActiveTx) error {
// 		vertex, err = validator.chain.ChallengeVertexByCommitHash(tx, id, goimpl.VertexCommitHash(c.Hash()))
// 		if err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// 	require.NoError(t, err)
// 	require.NotNil(t, vertex)

// 	return newVertexTracker(util.NewArtificialTimeReference(), time.Second, challenge, vertex, validator.chain, validator.stateManager, validator.name, validator.address)
// }

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
