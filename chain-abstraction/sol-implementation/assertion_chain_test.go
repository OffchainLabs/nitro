package solimpl_test

import (
	"context"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateAssertion(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	backend := cfg.Backend

	t.Run("OK", func(t *testing.T) {
		latestBlockHash := common.Hash{}
		for i := uint64(0); i < 100; i++ {
			latestBlockHash = backend.Commit()
		}

		createdInfo := &protocol.AssertionCreatedInfo{
			AfterState: (&protocol.ExecutionState{
				GlobalState:   protocol.GoGlobalState{},
				MachineStatus: protocol.MachineStatusFinished,
			}).AsSolidityStruct(),
			InboxMaxCount: big.NewInt(1),
		}
		postState := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  latestBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		_, err := chain.CreateAssertion(ctx, createdInfo, postState)
		require.NoError(t, err)

		_, err = chain.CreateAssertion(ctx, createdInfo, postState)
		require.ErrorContains(t, err, "ALREADY_STAKED")
	})
	t.Run("can create fork", func(t *testing.T) {
		t.Skip()
		assertionChain := cfg.Chains[1]

		for i := uint64(0); i < 100; i++ {
			backend.Commit()
		}

		creationInfo := &protocol.AssertionCreatedInfo{
			AfterState: (&protocol.ExecutionState{
				GlobalState:   protocol.GoGlobalState{},
				MachineStatus: protocol.MachineStatusFinished,
			}).AsSolidityStruct(),
			InboxMaxCount: big.NewInt(1),
		}
		postState := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  common.BytesToHash([]byte("evil hash")),
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		_, err := assertionChain.CreateAssertion(ctx, creationInfo, postState)
		require.NoError(t, err)
	})
}

func TestConfirmAssertionByChallengeWinner(t *testing.T) {
	ctx := context.Background()
	_, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) protocol.SpecEdge {
		startCommit, startErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, 0, 1)
		require.NoError(t, startErr)
		endCommit, endErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, proofErr)

		edge, edgeErr := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
			prefixProof,
		)
		require.NoError(t, edgeErr)
		return edge
	}
	honestEdge := leafAdder(createdData.HonestStateManager, createdData.Leaf1)
	s0, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s0)

	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, hasRival)

	// Adjust well beyond a challenge period.
	for i := 0; i < 200; i++ {
		createdData.Backend.Commit()
	}

	chain := createdData.Chains[0]

	latestConfirmed, err := chain.LatestConfirmed(ctx)
	require.NoError(t, err)

	t.Run("genesis case", func(t *testing.T) {
		err = chain.ConfirmAssertionByChallengeWinner(
			ctx, latestConfirmed.Id(), protocol.EdgeId{},
		)
		require.NoError(t, err)
	})
	t.Run("no level zero edge confirmed yet for the assertion", func(t *testing.T) {
		err = chain.ConfirmAssertionByChallengeWinner(
			ctx, createdData.Leaf1.Id(), honestEdge.Id(),
		)
		require.ErrorContains(t, err, "EDGE_NOT_CONFIRMED")
	})
	t.Run("level zero block edge confirmed allows assertion confirmation", func(t *testing.T) {
		err = honestEdge.ConfirmByTimer(ctx, make([]protocol.EdgeId, 0))
		require.NoError(t, err)

		err = chain.ConfirmAssertionByChallengeWinner(
			ctx, createdData.Leaf1.Id(), honestEdge.Id(),
		)
		require.NoError(t, err)

		latestConfirmed, err = chain.LatestConfirmed(ctx)
		require.NoError(t, err)
		require.Equal(t, createdData.Leaf1.Id(), latestConfirmed.Id())

		// Confirming again should just be a no-op.
		err = chain.ConfirmAssertionByChallengeWinner(
			ctx, createdData.Leaf1.Id(), honestEdge.Id(),
		)
		require.NoError(t, err)
	})
}

func TestAssertionBySequenceNum(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	latestConfirmed, err := chain.LatestConfirmed(ctx)
	require.NoError(t, err)
	_, err = chain.GetAssertion(ctx, latestConfirmed.Id())
	require.NoError(t, err)

	_, err = chain.GetAssertion(ctx, protocol.AssertionId(common.BytesToHash([]byte("foo"))))
	require.ErrorIs(t, err, solimpl.ErrNotFound)
}

func TestChallengePeriodBlocks(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]

	manager, err := chain.SpecChallengeManager(ctx)
	require.NoError(t, err)

	chalPeriod, err := manager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)
	require.Equal(t, cfg.RollupConfig.ConfirmPeriodBlocks, chalPeriod)
}
