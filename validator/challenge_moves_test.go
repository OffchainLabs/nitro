package validator

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_computePrefixProof(t *testing.T) {
	ctx := context.Background()
	stateRoots := generateStateRoots(10)
	manager := statemanager.New(stateRoots)
	commit, err := manager.HistoryCommitmentUpTo(ctx, 6)
	require.NoError(t, err)

	v := &vertexTracker{
		stateManager: manager,
	}

	bisectToCommit, err := v.determineBisectionPointWithHistory(ctx, 0, 6)
	require.NoError(t, err)

	bisectToHeight := bisectToCommit.Height
	proof, err := v.stateManager.PrefixProof(ctx, bisectToHeight, 6)
	require.NoError(t, err)

	err = util.VerifyPrefixProof(bisectToCommit, commit, proof)
	require.NoError(t, err)
}

func Test_bisect(t *testing.T) {
	ctx := context.Background()
	t.Run("bad bisection points", func(t *testing.T) {
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		_, _, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

		vertex := &protocol.ChallengeVertex{
			Prev: util.Some[protocol.ChallengeVertexInterface](&protocol.ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 3,
					Merkle: common.BytesToHash([]byte{0}),
				},
			}),
			Commitment: util.HistoryCommitment{
				Height: 0,
				Merkle: common.BytesToHash([]byte{1}),
			},
		}
		v := vertexTracker{
			chain:            validator.chain,
			stateManager:     validator.stateManager,
			validatorName:    validator.name,
			validatorAddress: validator.address,
		}
		_, err := v.bisect(ctx, vertex)
		require.ErrorContains(t, err, "determining bisection point failed")
	})
	t.Run("fails to verify prefix proof", func(t *testing.T) {
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		_, _, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

		vertex := &protocol.ChallengeVertex{
			Prev: util.Some[protocol.ChallengeVertexInterface](&protocol.ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 0,
					Merkle: common.BytesToHash([]byte{0}),
				},
			}),
			Commitment: util.HistoryCommitment{
				Height: 7,
				Merkle: common.BytesToHash([]byte("SOME JUNK DATA")),
			},
		}
		v := vertexTracker{
			chain:            validator.chain,
			stateManager:     validator.stateManager,
			validatorName:    validator.name,
			validatorAddress: validator.address,
		}
		_, err := v.bisect(ctx, vertex)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("OK", func(t *testing.T) {
		logsHook := test.NewGlobal()
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)
		bisectedVertex := runBisectionTest(t, logsHook, ctx, validator, stateRoots, leaf1, leaf2)

		// Expect to bisect to 4.
		require.Equal(t, uint64(4), bisectedVertex.GetCommitment().Height)
	})
}

func Test_merge(t *testing.T) {
	ctx := context.Background()
	genesisCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}
	challengeCommitHash := protocol.ChallengeCommitHash(genesisCommit.Hash())

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

		c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf2.StateCommitment.Height)
		require.NoError(t, err)

		var mergingTo protocol.ChallengeVertexInterface
		err = validator.chain.Call(func(tx *protocol.ActiveTx) error {
			mergingTo, err = validator.chain.ChallengeVertexByCommitHash(tx, challengeCommitHash, protocol.VertexCommitHash(c.Hash()))
			if err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, mergingTo)

		mergingFrom := &protocol.ChallengeVertex{
			Prev: util.Some(protocol.ChallengeVertexInterface(&protocol.ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 0,
					Merkle: common.BytesToHash([]byte{0}),
				},
			})),
			Commitment: util.HistoryCommitment{
				Height: 7,
				Merkle: common.BytesToHash([]byte("SOME JUNK DATA")),
			},
		}
		v := vertexTracker{
			chain:            validator.chain,
			stateManager:     validator.stateManager,
			validatorName:    validator.name,
			validatorAddress: validator.address,
		}
		_, err = v.merge(
			ctx, challengeCommitHash, mergingTo, mergingFrom,
		)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("OK", func(t *testing.T) {
		logsHook := test.NewGlobal()
		stateRoots := generateStateRoots(10)
		manager := statemanager.New(stateRoots)
		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

		// Bisect and obtain the result.
		bisectedVertex := runBisectionTest(t, logsHook, ctx, validator, stateRoots, leaf1, leaf2)

		// Expect to bisect to 4.
		require.Equal(t, uint64(4), bisectedVertex.GetCommitment().Height)

		c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
		require.NoError(t, err)

		// Get the vertex we want to merge from.
		var vertexToMergeFrom *protocol.ChallengeVertex
		err = validator.chain.Call(func(tx *protocol.ActiveTx) error {
			vertexToMergeFrom, err = validator.chain.ChallengeVertexByCommitHash(tx, challengeCommitHash, protocol.VertexCommitHash(c.Hash()))
			if err != nil {
				return err
			}
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, vertexToMergeFrom)

		// Perform a merge move to the bisected vertex from an origin.
		v := vertexTracker{
			chain:            validator.chain,
			stateManager:     validator.stateManager,
			validatorName:    validator.name,
			validatorAddress: validator.address,
		}
		mergingTo, err := v.merge(ctx, challengeCommitHash, bisectedVertex, vertexToMergeFrom)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "Successfully merged to vertex with height 4")
		require.Equal(t, bisectedVertex, mergingTo)
	})
}

func runBisectionTest(
	t *testing.T,
	logsHook *test.Hook,
	ctx context.Context,
	validator *Validator,
	stateRoots []common.Hash,
	leaf1,
	leaf2 *protocol.CreateLeafEvent,
) protocol.ChallengeVertexInterface {
	err := validator.onLeafCreated(ctx, leaf1)
	require.NoError(t, err)
	err = validator.onLeafCreated(ctx, leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

	historyCommit, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
	require.NoError(t, err)

	genesisCommit := util.StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}

	id := protocol.ChallengeCommitHash(genesisCommit.Hash())
	err = validator.chain.Tx(func(tx *protocol.ActiveTx) error {
		assertion, fetchErr := validator.chain.AssertionBySequenceNum(tx, protocol.AssertionSequenceNumber(1))
		if fetchErr != nil {
			return fetchErr
		}
		challenge, challErr := validator.chain.ChallengeByCommitHash(tx, id)
		if challErr != nil {
			return challErr
		}
		if _, err = challenge.AddLeaf(tx, assertion, historyCommit, validator.address); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf2.StateCommitment.Height)
	require.NoError(t, err)

	// Get the challenge from the chain itself.
	var vertexToBisect protocol.ChallengeVertexInterface
	err = validator.chain.Call(func(tx *protocol.ActiveTx) error {
		vertexToBisect, err = validator.chain.ChallengeVertexByCommitHash(tx, id, protocol.VertexCommitHash(c.Hash()))
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, vertexToBisect)

	v := vertexTracker{
		chain:            validator.chain,
		stateManager:     validator.stateManager,
		validatorName:    validator.name,
		validatorAddress: validator.address,
	}

	bisectedVertex, err := v.bisect(ctx, vertexToBisect)
	require.NoError(t, err)

	bisectionHeight := uint64(4)
	loExp := util.ExpansionFromLeaves(stateRoots[:bisectionHeight])

	bisectionCommit := util.HistoryCommitment{
		Height: bisectionHeight,
		Merkle: loExp.Root(),
	}
	require.Equal(t, bisectedVertex.GetCommitment().Hash(), bisectionCommit.Hash())

	AssertLogsContain(t, logsHook, "Successfully bisected to vertex")
	return bisectedVertex
}

func generateStateRoots(numBlocks uint64) []common.Hash {
	var ret []common.Hash
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(i))
	}
	return ret
}
