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
		validator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			&mocks.MockStateManager{},
			createdData.addrs.Rollup,
		)
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
		validator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			manager,
			createdData.addrs.Rollup,
		)
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
	t.Run("bisects", func(t *testing.T) {
		logsHook := test.NewGlobal()
		createdData := createTwoValidatorFork(t, ctx, 10 /* divergence point */)

		honestManager := statemanager.New(createdData.honestValidatorStateRoots)
		honestValidator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			honestManager,
			createdData.addrs.Rollup,
		)
		require.NoError(t, err)

		evilManager := statemanager.New(createdData.evilValidatorStateRoots)
		evilValidator, err := New(
			ctx,
			createdData.assertionChains[2],
			createdData.backend,
			evilManager,
			createdData.addrs.Rollup,
		)
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

func Test_merge(t *testing.T) {
	ctx := context.Background()
	t.Run("fails to verify prefix proof", func(t *testing.T) {
		logsHook := test.NewGlobal()
		createdData := createTwoValidatorFork(t, ctx, 10 /* divergence point */)

		honestManager := statemanager.New(createdData.honestValidatorStateRoots)
		honestValidator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			honestManager,
			createdData.addrs.Rollup,
		)
		require.NoError(t, err)

		err = honestValidator.onLeafCreated(ctx, createdData.leaf1)
		require.NoError(t, err)
		err = honestValidator.onLeafCreated(ctx, createdData.leaf2)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

		commit, err := honestValidator.stateManager.HistoryCommitmentUpTo(ctx, createdData.leaf2.Height())
		require.NoError(t, err)

		var mergingTo protocol.ChallengeVertex
		var challengeId protocol.ChallengeHash
		err = honestValidator.chain.Call(func(tx protocol.ActiveTx) error {
			genesisId, err := honestValidator.chain.GetAssertionId(ctx, tx, protocol.AssertionSequenceNumber(0))
			require.NoError(t, err)
			manager, err := honestValidator.chain.CurrentChallengeManager(ctx, tx)
			require.NoError(t, err)
			chalId, err := manager.CalculateChallengeHash(ctx, tx, common.Hash(genesisId), protocol.BlockChallenge)
			require.NoError(t, err)

			challengeId = chalId

			vertexId, err := manager.CalculateChallengeVertexId(ctx, tx, chalId, commit)
			require.NoError(t, err)

			mergingToV, err := manager.GetVertex(ctx, tx, vertexId)
			require.NoError(t, err)
			mergingTo = mergingToV.Unwrap()
			return nil
		})
		require.NoError(t, err)

		mergingFrom := &mocks.MockChallengeVertex{
			MockPrev: util.Some(protocol.ChallengeVertex(&mocks.MockChallengeVertex{
				MockHistory: util.HistoryCommitment{
					Height: 0,
					Merkle: common.BytesToHash([]byte{0}),
				},
			})),
			MockHistory: util.HistoryCommitment{
				Height: 101,
				Merkle: common.BytesToHash([]byte("SOME JUNK DATA")),
			},
		}
		v := vertexTracker{
			chain:            honestValidator.chain,
			stateManager:     honestValidator.stateManager,
			validatorName:    honestValidator.name,
			validatorAddress: honestValidator.address,
		}
		_, err = v.merge(
			ctx, challengeId, mergingTo, mergingFrom,
		)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("OK", func(t *testing.T) {
		logsHook := test.NewGlobal()
		createdData := createTwoValidatorFork(t, ctx, 70 /* divergence point */)

		honestManager := statemanager.New(createdData.honestValidatorStateRoots)
		honestValidator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			honestManager,
			createdData.addrs.Rollup,
		)
		require.NoError(t, err)

		evilManager := statemanager.New(createdData.evilValidatorStateRoots)
		evilValidator, err := New(
			ctx,
			createdData.assertionChains[2],
			createdData.backend,
			evilManager,
			createdData.addrs.Rollup,
		)
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

		// Both validators should have the same history upon which one will try to merge into.
		require.Equal(t, createdData.evilValidatorStateRoots[64], createdData.honestValidatorStateRoots[64], "Different state root at 64")
		mergingFromHistory, err := honestValidator.stateManager.HistoryCommitmentUpTo(ctx, createdData.leaf1.Height())
		require.NoError(t, err)

		// Get the vertex we want to merge from.
		var vertexToMergeFrom protocol.ChallengeVertex
		var challengeId protocol.ChallengeHash
		err = honestValidator.chain.Call(func(tx protocol.ActiveTx) error {
			genesisId, err := honestValidator.chain.GetAssertionId(ctx, tx, protocol.AssertionSequenceNumber(0))
			require.NoError(t, err)
			manager, err := honestValidator.chain.CurrentChallengeManager(ctx, tx)
			require.NoError(t, err)
			chalId, err := manager.CalculateChallengeHash(ctx, tx, common.Hash(genesisId), protocol.BlockChallenge)
			require.NoError(t, err)

			challengeId = chalId

			vertexId, err := manager.CalculateChallengeVertexId(ctx, tx, chalId, mergingFromHistory)
			require.NoError(t, err)

			mergingFromV, err := manager.GetVertex(ctx, tx, vertexId)
			require.NoError(t, err)
			vertexToMergeFrom = mergingFromV.Unwrap()
			return nil
		})
		require.NoError(t, err)

		// Perform a merge move to the bisected vertex from an origin.
		v := vertexTracker{
			chain:            honestValidator.chain,
			stateManager:     honestValidator.stateManager,
			validatorName:    honestValidator.name,
			validatorAddress: honestValidator.address,
		}
		mergingTo, err := v.merge(ctx, challengeId, bisectedTo, vertexToMergeFrom)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "Successfully merged to vertex with height 64")
		require.Equal(t, bisectedTo.Id(), mergingTo.Id())
	})
}

func runBisectionTest(
	t *testing.T,
	logsHook *test.Hook,
	ctx context.Context,
	honestValidator,
	evilValidator *Validator,
	leaf1,
	leaf2 protocol.Assertion,
) protocol.ChallengeVertex {
	err := honestValidator.onLeafCreated(ctx, leaf1)
	require.NoError(t, err)
	err = honestValidator.onLeafCreated(ctx, leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New assertion appended")
	AssertLogsContain(t, logsHook, "New assertion appended")
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
