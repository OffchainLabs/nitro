package validator

import (
	"testing"

	"errors"
	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/testing/mocks"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
