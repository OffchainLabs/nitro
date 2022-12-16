package validator

import (
	"testing"

	"context"
	"errors"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/testing/mocks"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_actOnBlockChallenge(t *testing.T) {
	challengeCommit := protocol.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.CommitHash(challengeCommit.Hash())
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
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
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
			Prev: &protocol.ChallengeVertex{
				Commitment: parentHistory,
			},
		}
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
			vertex,
			nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{},
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
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
			Prev: &protocol.ChallengeVertex{
				Commitment: parentHistory,
			},
		}
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
			vertex,
			nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{},
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Reached one-step-fork at 0")
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
			PresumptiveSuccessor: vertex,
		}
		vertex.Prev = prev
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
			vertex,
			nil,
		)
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{},
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
		}
		err := tkr.actOnBlockChallenge(ctx)
		require.NoError(t, err)
	})
}

func Test_isAtOneStepFork(t *testing.T) {
	challengeCommit := protocol.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.CommitHash(challengeCommit.Hash())
	commitA := util.HistoryCommitment{
		Height: 1,
	}
	commitB := util.HistoryCommitment{
		Height: 2,
	}
	vertex := &protocol.ChallengeVertex{
		Commitment: commitA,
		Prev: &protocol.ChallengeVertex{
			Commitment: commitB,
		},
	}
	t.Run("fails", func(t *testing.T) {
		p := &mocks.MockProtocol{}
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{},
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
		}
		_, err := tkr.isAtOneStepFork()
		require.ErrorContains(t, err, "something went wrong")
	})
	t.Run("OK", func(t *testing.T) {
		p := &mocks.MockProtocol{}
		p.On(
			"IsAtOneStepFork",
			&protocol.ActiveTx{},
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
			validator:           v,
			vertex:              vertex,
			challengeCommitHash: challengeCommitHash,
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
	challengeCommitHash := protocol.CommitHash(challengeCommit.Hash())

	t.Run("nil vertex", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
			vertex, nil,
		)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator:           v,
			challengeCommitHash: challengeCommitHash,
		}
		_, err := tkr.fetchVertexByHistoryCommit(history)
		require.ErrorContains(t, err, "fetched nil challenge")
	})
	t.Run("fetching error", func(t *testing.T) {
		history := util.HistoryCommitment{
			Height: 1,
		}
		p := &mocks.MockProtocol{}
		var vertex *protocol.ChallengeVertex
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(
			vertex,
			errors.New("something went wrong"),
		)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator:           v,
			challengeCommitHash: challengeCommitHash,
		}
		_, err := tkr.fetchVertexByHistoryCommit(history)
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
		p.On("ChallengeVertexByHistoryCommit", &protocol.ActiveTx{}, challengeCommitHash, history).Return(want, nil)
		v := &Validator{
			chain: p,
		}
		tkr := &vertexTracker{
			validator:           v,
			challengeCommitHash: challengeCommitHash,
		}
		got, err := tkr.fetchVertexByHistoryCommit(history)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}
