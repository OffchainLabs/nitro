package validator

import (
	"context"
	"testing"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	statemanager "github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_merge(t *testing.T) {
	ctx := context.Background()
	genesisCommit := protocol.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.CommitHash(genesisCommit.Hash())

	t.Run("fails to verify prefix proof", func(t *testing.T) {
		logsHook := test.NewGlobal()
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

		err := validator.onLeafCreated(ctx, leaf1)
		require.NoError(t, err)
		err = validator.onLeafCreated(ctx, leaf2)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

		vertex := &protocol.ChallengeVertex{
			Prev: &protocol.ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 0,
					Merkle: common.BytesToHash([]byte{0}),
				},
			},
			Commitment: util.HistoryCommitment{
				Height: 7,
				Merkle: common.BytesToHash([]byte("SOME JUNK DATA")),
			},
		}
		mergingTo := protocol.VertexSequenceNumber(1)
		err = validator.merge(
			ctx, challengeCommitHash, vertex, mergingTo,
		)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("good prefix proof", func(t *testing.T) {
		t.Skip()
		logsHook := test.NewGlobal()
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

		err := validator.onLeafCreated(ctx, leaf1)
		require.NoError(t, err)
		err = validator.onLeafCreated(ctx, leaf2)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

		historyCommit, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
		require.NoError(t, err)

		genesisCommit := protocol.StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		}

		id := protocol.CommitHash(genesisCommit.Hash())
		err = validator.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			assertion, fetchErr := p.AssertionBySequenceNum(tx, protocol.AssertionSequenceNumber(1))
			if fetchErr != nil {
				return fetchErr
			}
			challenge, challErr := p.ChallengeByCommitHash(tx, id)
			if challErr != nil {
				return challErr
			}
			// We add a leaf at height 5.
			if _, err = challenge.AddLeaf(tx, assertion, historyCommit, validator.address); err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)

		// Get the challenge from the chain itself.
		var vertexToMerge *protocol.ChallengeVertex
		err = validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			vertexToMerge, err = p.ChallengeVertexBySequenceNum(tx, id, protocol.VertexSequenceNumber(1))
			if err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, vertexToMerge)

		err = validator.merge(ctx, challengeCommitHash, vertexToMerge, protocol.VertexSequenceNumber(1))
		require.NoError(t, err)

		AssertLogsContain(t, logsHook, "Successfully merged vertex")
	})
}

func generateStateRoots(numBlocks uint64) []common.Hash {
	ret := []common.Hash{}
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(i))
	}
	return ret
}
