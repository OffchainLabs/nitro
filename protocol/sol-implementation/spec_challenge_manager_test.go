package solimpl_test

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	_ = protocol.SpecEdge(&solimpl.SpecEdge{})
	_ = protocol.SpecChallengeManager(&solimpl.SpecChallengeManager{})
)

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
	genesis, err := createdData.Chains[0].AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			genesis,
			util.HistoryCommitment{Merkle: common.Hash{}},
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	honestEdge := leafAdder(honestEndCommit)
	s0, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s0)

	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)
	evilEdge := leafAdder(evilEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 1, 3)
	require.NoError(t, err)
	honestChildren1, honestChildren2, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	s1, err := honestChildren1.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s1)
	s2, err := honestChildren2.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgePending, s2)

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
	genesis, err := createdData.Chains[0].AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			genesis,
			util.HistoryCommitment{Merkle: common.Hash{}},
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}
	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	honestEdge := leafAdder(honestEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())
	isPs, err := honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)

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

	genesis, err := createdData.Chains[0].AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	// Honest assertion being added.
	startCommit := util.HistoryCommitment{
		Height: 0,
		Merkle: common.Hash{},
	}
	leafAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			genesis,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")
	honestEdge := leafAdder(honestEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())
	isPs, err := honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)
	t.Log("Alice is presumptive")

	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Bob creates level zero block edge")
	evilEdge := leafAdder(evilEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	// Honest and evil edge are rivals, neither is presumptive.
	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	isPs, err = evilEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	t.Log("Neither is presumptive")

	// Attempt bisections down to one step fork.
	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 1, 3)
	require.NoError(t, err)

	t.Log("Alice bisects")
	_, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	evilBisectCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	evilProof, err := evilStateManager.PrefixProof(ctx, 1, 3)
	require.NoError(t, err)

	t.Log("Bob bisects")
	oneStepForkSourceEdge, _, err := evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
	require.NoError(t, err)

	isAtOneStepFork, err := oneStepForkSourceEdge.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Lower child of bisection is at one step fork")

	// Now opening big step level zero leaves
	bigStepAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
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
	honestEdge = bigStepAdder(honestBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, honestEdge.GetType())
	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)

	t.Log("Alice is presumptive")

	evilBigStepCommit, err := evilStateManager.BigStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to big step */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero big step challenge edge")
	evilEdge = bigStepAdder(evilBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, evilEdge.GetType())

	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	isPs, err = evilEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	t.Log("Neither is presumptive")

	isAtOneStepFork, err = honestEdge.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step fork at big step challenge level")

	claimHeight, err := evilEdge.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Now opening small step level zero leaves
	smallStepAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
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
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Alice creates level zero small step challenge edge")
	smallStepHonest := smallStepAdder(honestSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())
	isPs, err = smallStepHonest.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)

	t.Log("Alice is presumptive")

	evilSmallStepCommit, err := evilStateManager.SmallStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero small step challenge edge")
	smallStepEvil := smallStepAdder(evilSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepEvil.GetType())

	isPs, err = smallStepHonest.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	isPs, err = smallStepEvil.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	t.Log("Neither is presumptive")

	claimHeight, err = smallStepEvil.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Get the lower-level edge of either vertex we just bisected.
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())

	isAtOneStepFork, err = smallStepHonest.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step proof!!!")
}
