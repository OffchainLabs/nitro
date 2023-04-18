package solimpl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	_ = protocol.SpecEdge(&solimpl.SpecEdge{})
	_ = protocol.SpecChallengeManager(&solimpl.SpecChallengeManager{})
)

func TestEdgeChallengeManager_IsUnrivaled(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	opts := []statemanager.Opt{
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	}

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)
	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}

	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	t.Run("first leaf is presumptive", func(t *testing.T) {
		hasRival, err := honestEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, true, !hasRival)
	})

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	t.Run("neither is presumptive if rivals", func(t *testing.T) {
		hasRival, err := honestEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)

		hasRival, err = evilEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)
	})

	t.Run("bisected children are presumptive", func(t *testing.T) {
		honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 2)
		require.NoError(t, err)
		honestProof, err := honestStateManager.PrefixProof(ctx, 2, 3)
		require.NoError(t, err)

		lower, upper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		hasRival, err := lower.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, true, !hasRival)
		hasRival, err = upper.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, true, !hasRival)

		hasRival, err = honestEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)

		hasRival, err = evilEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)
	})
}

func TestEdgeChallengeManager_HasLengthOneRival(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	opts := []statemanager.Opt{
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	}

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)
	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}
	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	t.Run("lone level zero edge is not one step fork source", func(t *testing.T) {
		isOSF, err := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
	})

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	t.Run("level zero edge with rivals is not one step fork source", func(t *testing.T) {
		isOSF, err := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
		isOSF, err = evilEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
	})
	t.Run("post bisection, mutual edge is one step fork source", func(t *testing.T) {
		honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 2)
		require.NoError(t, err)
		honestProof, err := honestStateManager.PrefixProof(ctx, 2, 3)
		require.NoError(t, err)
		lowerHonest, upper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		isOSF, err := lowerHonest.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
		isOSF, err = upper.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)

		evilBisectCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 2)
		require.NoError(t, err)
		evilProof, err := evilStateManager.PrefixProof(ctx, 2, 3)
		require.NoError(t, err)
		lower, _, err := evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
		require.NoError(t, err)

		evilBisectCommit, err = evilStateManager.HistoryCommitmentUpTo(ctx, 1)
		require.NoError(t, err)
		evilProof, err = evilStateManager.PrefixProof(ctx, 1, 2)
		require.NoError(t, err)
		_, _, err = lower.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
		require.NoError(t, err)

		honestBisectCommit, err = honestStateManager.HistoryCommitmentUpTo(ctx, 1)
		require.NoError(t, err)
		honestProof, err = honestStateManager.PrefixProof(ctx, 1, 2)
		require.NoError(t, err)
		lower, upper, err = lowerHonest.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		isOSF, err = lower.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, true, isOSF)

		isOSF, err = upper.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
	})
}

func TestEdgeChallengeManager_BlockChallengeAddLevelZeroEdge(t *testing.T) {
	ctx := context.Background()
	height := uint64(3)
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     height,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	chain1 := createdData.Chains[0]
	challengeManager, err := chain1.SpecChallengeManager(ctx)
	require.NoError(t, err)

	t.Run("claim predecessor does not exist", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})
	t.Run("invalid height", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})
	t.Run("last state is not assertion claim block hash", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})
	t.Run("last state not in history", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})
	t.Run("first state not in history", func(t *testing.T) {
		t.Skip("Needs Solidity code")
	})

	leaves := make([]common.Hash, 4)
	for i := range leaves {
		leaves[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	opts := []statemanager.Opt{
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	}

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)

	start, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	end, err := honestStateManager.HistoryCommitmentUpTo(ctx, height)
	require.NoError(t, err)

	t.Run("OK", func(t *testing.T) {
		_, err = challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end)
		require.NoError(t, err)
	})
	t.Run("already exists", func(t *testing.T) {
		_, err = challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end)
		require.ErrorContains(t, err, "already exists")
	})
}

func TestEdgeChallengeManager_Bisect(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	opts := []statemanager.Opt{
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	}

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)
	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		opts...,
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}
	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	t.Run("cannot bisect presumptive", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("invalid prefix proof", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("edge has children", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("OK", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
}

func TestEdgeChallengeManager_SubChallenges(t *testing.T) {
	t.Run("leaf cannot be a fork candidate", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("lowest height not one step fork", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("has presumptive successor", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("empty history root", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("OK", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
}

func TestEdgeChallengeManager_ConfirmByClaim(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	)
	require.NoError(t, err)

	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
		statemanager.WithBigStepStateDivergenceHeight(1),
		statemanager.WithSmallStepStateDivergenceHeight(1),
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}
	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	// Adjust well beyond a challenge period.
	for i := 0; i < 100; i++ {
		createdData.Backend.Commit()
	}

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 2)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 2, 3)
	require.NoError(t, err)
	honestChildren1, _, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	require.NoError(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{}))

	t.Run("Origin id-mutual id mismatch", func(t *testing.T) {
		require.ErrorContains(t, honestChildren1.ConfirmByClaim(ctx, protocol.ClaimId(honestEdge.Id())), "execution reverted: Origin id-mutual id mismatch")
	})
}

func TestEdgeChallengeManager_ConfirmByChildren(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	)
	require.NoError(t, err)

	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
		statemanager.WithBigStepStateDivergenceHeight(1),
		statemanager.WithSmallStepStateDivergenceHeight(1),
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}
	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	s0, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s0)

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 2)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 2, 3)
	require.NoError(t, err)
	honestChildren1, honestChildren2, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	s1, err := honestChildren1.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s1)
	s2, err := honestChildren2.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s2)

	// Adjust well beyond a challenge period.
	for i := 0; i < 100; i++ {
		createdData.Backend.Commit()
	}

	require.NoError(t, honestChildren1.ConfirmByTimer(ctx, []protocol.EdgeId{}))
	require.NoError(t, honestChildren2.ConfirmByTimer(ctx, []protocol.EdgeId{}))
	s1, err = honestChildren1.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s1)
	s2, err = honestChildren2.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s2)

	require.NoError(t, honestEdge.ConfirmByChildren(ctx))
	s0, err = honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s0)
}

func TestEdgeChallengeManager_ConfirmByTimer(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height) + 1,
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}
	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	s0, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s0)

	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, hasRival)

	// Adjust well beyond a challenge period.
	for i := 0; i < 100; i++ {
		createdData.Backend.Commit()
	}

	t.Run("confirmed by timer", func(t *testing.T) {
		require.ErrorContains(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{protocol.EdgeId(common.Hash{1})}), "execution reverted: Edge does not exist")
	})
	t.Run("confirmed by timer", func(t *testing.T) {
		require.NoError(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{}))
		status, err := honestEdge.Status(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.EdgeConfirmed, status)
	})
	t.Run("can't confirm again", func(t *testing.T) {
		status, err := honestEdge.Status(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.EdgeConfirmed, status)
		require.ErrorContains(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{}), "execution reverted: Edge not pending")
	})
}

func TestEdgeChallengeManager(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(2)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
		NumBlocks:     uint64(height),
		DivergeHeight: 0,
	})
	require.NoError(t, err)

	honestStateManager, err := statemanager.New(
		createdData.HonestValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	)
	require.NoError(t, err)

	evilStateManager, err := statemanager.New(
		createdData.EvilValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
		statemanager.WithBigStepStateDivergenceHeight(1),
		statemanager.WithSmallStepStateDivergenceHeight(1),
	)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}

	honestStartCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(0))
	require.NoError(t, err)
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")
	honestEdge := leafAdder(honestStartCommit, honestEndCommit, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)
	t.Log("Alice is presumptive")

	evilStartCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(0))
	require.NoError(t, err)
	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Bob creates level zero block edge")
	evilEdge := leafAdder(evilStartCommit, evilEndCommit, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	// Honest and evil edge are rivals, neither is presumptive.
	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	t.Log("Neither is presumptive")

	// Attempt bisections down to one step fork.
	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 1, 2)
	require.NoError(t, err)

	t.Log("Alice bisects")
	_, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	evilBisectCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	evilProof, err := evilStateManager.PrefixProof(ctx, 1, 2)
	require.NoError(t, err)

	t.Log("Bob bisects")
	oneStepForkSourceEdge, _, err := evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
	require.NoError(t, err)

	isAtOneStepFork, err := oneStepForkSourceEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Lower child of bisection is at one step fork")

	// Now opening big step level zero leaves
	bigStepAdder := func(startCommit, endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			oneStepForkSourceEdge,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestBigStepCommit, err := honestStateManager.BigStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to big step */
	)
	require.NoError(t, err)

	t.Log("Alice creates level zero big step challenge edge")
	honestEdge = bigStepAdder(honestStartCommit, honestBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, honestEdge.GetType())
	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	t.Log("Alice is presumptive")

	evilBigStepCommit, err := evilStateManager.BigStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to big step */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero big step challenge edge")
	evilEdge = bigStepAdder(evilStartCommit, evilBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, evilEdge.GetType())

	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	t.Log("Neither is presumptive")

	isAtOneStepFork, err = honestEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step fork at big step challenge level")

	claimHeight, err := evilEdge.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Now opening small step level zero leaves
	smallStepAdder := func(startCommit, endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			honestEdge,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestSmallStepCommit, err := honestStateManager.SmallStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 0 /* from big step */, 1 /* to big step */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Alice creates level zero small step challenge edge")
	smallStepHonest := smallStepAdder(honestStartCommit, honestSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())
	hasRival, err = smallStepHonest.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	t.Log("Alice is presumptive")

	evilSmallStepCommit, err := evilStateManager.SmallStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 0 /* from big step */, 1 /* to big step */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero small step challenge edge")
	smallStepEvil := smallStepAdder(evilStartCommit, evilSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepEvil.GetType())

	hasRival, err = smallStepHonest.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	hasRival, err = smallStepEvil.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	t.Log("Neither is presumptive")

	claimHeight, err = smallStepEvil.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Get the lower-level edge of either vertex we just bisected.
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())

	isAtOneStepFork, err = smallStepHonest.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step proof")
}
