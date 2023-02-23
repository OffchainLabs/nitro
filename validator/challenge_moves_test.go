package validator

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
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
		createdData := createTwoValidatorFork(t, ctx, 10 /* divergence point */)
		validator, err := New(ctx, createdData.assertionChains[1], &mocks.MockStateManager{})
		require.NoError(t, err)

		vertex := &mocks.MockChallengeVertex{
			MockPrev: util.Some(protocol.ChallengeVertex(&mocks.MockChallengeVertex{
				MockHistory: util.HistoryCommitment{
					Height: 3,
					Merkle: common.BytesToHash([]byte{0}),
				},
			})),
			MockHistory: util.HistoryCommitment{
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
		_, err = v.bisect(ctx, vertex)
		require.ErrorContains(t, err, "determining bisection point failed")
	})
	t.Run("fails to verify prefix proof", func(t *testing.T) {
		createdData := createTwoValidatorFork(t, ctx, 10 /* divergence point */)
		manager := &mocks.MockStateManager{}
		manager.On("HistoryCommitmentUpTo", ctx, uint64(4)).Return(util.HistoryCommitment{}, nil)
		manager.On("PrefixProof", ctx, uint64(0), uint64(7)).Return(make([]common.Hash, 0), nil)
		validator, err := New(ctx, createdData.assertionChains[1], manager)
		require.NoError(t, err)

		vertex := &mocks.MockChallengeVertex{
			MockPrev: util.Some(protocol.ChallengeVertex(&mocks.MockChallengeVertex{
				MockHistory: util.HistoryCommitment{
					Height: 0,
					Merkle: common.BytesToHash([]byte{0}),
				},
			})),
			MockHistory: util.HistoryCommitment{
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
		_, err = v.bisect(ctx, vertex)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("OK", func(t *testing.T) {
		logsHook := test.NewGlobal()
		createdData := createTwoValidatorFork(t, ctx, 10 /* divergence point */)

		honestManager := statemanager.New(createdData.honestValidatorStateRoots)
		honestValidator, err := New(ctx, createdData.assertionChains[1], honestManager)
		require.NoError(t, err)

		evilManager := statemanager.New(createdData.evilValidatorStateRoots)
		evilValidator, err := New(ctx, createdData.assertionChains[2], evilManager)
		require.NoError(t, err)

		bisectedTo := runBisectionTest(
			t,
			logsHook,
			ctx,
			honestValidator,
			evilValidator,
			createdData.leaf1,
			createdData.leaf2,
		)

		// Expect to bisect to 64.
		commitment := bisectedTo.HistoryCommitment()
		require.Equal(t, uint64(64), commitment.Height)
	})
}

// func Test_merge(t *testing.T) {
// 	tx := &goimpl.ActiveTx{}
// 	ctx := context.Background()
// 	genesisCommit := util.StateCommitment{
// 		Height:    0,
// 		StateRoot: common.Hash{},
// 	}
// 	challengeCommitHash := goimpl.ChallengeCommitHash(genesisCommit.Hash())

// 	t.Run("fails to verify prefix proof", func(t *testing.T) {
// 		logsHook := test.NewGlobal()
// 		stateRoots := generateStateRoots(10)
// 		manager := statemanager.New(stateRoots)
// 		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

// 		err := validator.onLeafCreated(ctx, tx, leaf1)
// 		require.NoError(t, err)
// 		err = validator.onLeafCreated(ctx, tx, leaf2)
// 		require.NoError(t, err)
// 		AssertLogsContain(t, logsHook, "New leaf appended")
// 		AssertLogsContain(t, logsHook, "New leaf appended")
// 		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

// 		c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf2.StateCommitment.Height)
// 		require.NoError(t, err)

// 		var mergingTo goimpl.ChallengeVertexInterface
// 		err = validator.chain.Call(func(tx *goimpl.ActiveTx) error {
// 			mergingTo, err = validator.chain.ChallengeVertexByCommitHash(tx, challengeCommitHash, goimpl.VertexCommitHash(c.Hash()))
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		})
// 		require.NoError(t, err)
// 		require.NotNil(t, mergingTo)

// 		mergingFrom := &goimpl.ChallengeVertex{
// 			Prev: util.Some(goimpl.ChallengeVertexInterface(&goimpl.ChallengeVertex{
// 				Commitment: util.HistoryCommitment{
// 					Height: 0,
// 					Merkle: common.BytesToHash([]byte{0}),
// 				},
// 			})),
// 			Commitment: util.HistoryCommitment{
// 				Height: 7,
// 				Merkle: common.BytesToHash([]byte("SOME JUNK DATA")),
// 			},
// 		}
// 		v := vertexTracker{
// 			chain:            validator.chain,
// 			stateManager:     validator.stateManager,
// 			validatorName:    validator.name,
// 			validatorAddress: validator.address,
// 		}
// 		_, err = v.merge(
// 			ctx, tx, challengeCommitHash, mergingTo, mergingFrom,
// 		)
// 		require.ErrorIs(t, err, util.ErrIncorrectProof)
// 	})
// 	t.Run("OK", func(t *testing.T) {
// 		logsHook := test.NewGlobal()
// 		stateRoots := generateStateRoots(10)
// 		manager := statemanager.New(stateRoots)
// 		leaf1, leaf2, validator := createTwoValidatorFork(t, ctx, manager, stateRoots)

// 		// Bisect and obtain the result.
// 		bisectedVertex := runBisectionTest(t, logsHook, ctx, tx, validator, stateRoots, leaf1, leaf2)

// 		// Expect to bisect to 4.
// 		commitment, err := bisectedVertex.GetCommitment(ctx, tx)
// 		require.NoError(t, err)
// 		require.Equal(t, uint64(4), commitment.Height)

// 		c, err := validator.stateManager.HistoryCommitmentUpTo(ctx, leaf1.StateCommitment.Height)
// 		require.NoError(t, err)

// 		// Get the vertex we want to merge from.
// 		var vertexToMergeFrom *goimpl.ChallengeVertex
// 		err = validator.chain.Call(func(tx *goimpl.ActiveTx) error {
// 			vertexToMergeFrom, err = validator.chain.ChallengeVertexByCommitHash(tx, challengeCommitHash, goimpl.VertexCommitHash(c.Hash()))
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		})
// 		require.NoError(t, err)
// 		require.NotNil(t, vertexToMergeFrom)

// 		// Perform a merge move to the bisected vertex from an origin.
// 		v := vertexTracker{
// 			chain:            validator.chain,
// 			stateManager:     validator.stateManager,
// 			validatorName:    validator.name,
// 			validatorAddress: validator.address,
// 		}
// 		mergingTo, err := v.merge(ctx, tx, challengeCommitHash, bisectedVertex, vertexToMergeFrom)
// 		require.NoError(t, err)
// 		AssertLogsContain(t, logsHook, "Successfully merged to vertex with height 4")
// 		require.Equal(t, bisectedVertex, mergingTo)
// 	})
// }

func runBisectionTest(
	t *testing.T,
	logsHook *test.Hook,
	ctx context.Context,
	honestValidator,
	evilValidator *Validator,
	leaf1,
	leaf2 *protocol.CreateLeafEvent,
) protocol.ChallengeVertex {
	err := honestValidator.onLeafCreated(ctx, leaf1)
	require.NoError(t, err)
	err = honestValidator.onLeafCreated(ctx, leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

	var vertexToBisect protocol.ChallengeVertex
	var chalId protocol.ChallengeHash

	err = evilValidator.chain.Tx(func(tx protocol.ActiveTx) error {
		genesisId, err := evilValidator.chain.GetAssertionId(ctx, tx, protocol.AssertionSequenceNumber(0))
		require.NoError(t, err)
		manager, err := evilValidator.chain.CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		chalIdComputed, err := manager.CalculateChallengeHash(ctx, tx, common.Hash(genesisId), protocol.BlockChallenge)
		require.NoError(t, err)

		chalId = chalIdComputed

		challenge, err := manager.GetChallenge(ctx, tx, chalId)
		require.NoError(t, err)
		require.Equal(t, false, challenge.IsNone())
		assertion, err := evilValidator.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(2))
		require.NoError(t, err)

		honestCommit, err := evilValidator.stateManager.HistoryCommitmentUpTo(ctx, assertion.Height())
		require.NoError(t, err)
		vToBisect, err := challenge.Unwrap().AddBlockChallengeLeaf(ctx, tx, assertion, honestCommit)
		require.NoError(t, err)
		vertexToBisect = vToBisect
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

	v := vertexTracker{
		chain:            evilValidator.chain,
		stateManager:     evilValidator.stateManager,
		validatorName:    evilValidator.name,
		validatorAddress: evilValidator.address,
	}

	bisectedVertex, err := v.bisect(ctx, vertexToBisect)
	require.NoError(t, err)

	shouldBisectToCommit, err := evilValidator.stateManager.HistoryCommitmentUpTo(ctx, bisectedVertex.HistoryCommitment().Height)
	require.NoError(t, err)

	commitment := bisectedVertex.HistoryCommitment()
	require.NoError(t, err)
	require.Equal(t, commitment.Hash(), shouldBisectToCommit.Hash())

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
