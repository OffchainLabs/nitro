package solimpl_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	_ = protocol.SpecEdge(&solimpl.SpecEdge{})
	_ = protocol.SpecChallengeManager(&solimpl.SpecChallengeManager{})
)

//nolint:unused
var genesisOspData = make([]byte, 16)

func TestEdgeChallengeManager_IsUnrivaled(t *testing.T) {
	ctx := context.Background()

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
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	t.Run("first leaf is presumptive", func(t *testing.T) {
		hasRival, err := honestEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, true, !hasRival)
	})

	evilEdge := leafAdder(createdData.EvilStateManager, createdData.Leaf2)
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
		var bisectHeight uint64 = protocol.LevelZeroBlockEdgeHeight / 2
		honestBisectCommit, err := createdData.HonestStateManager.HistoryCommitmentUpToBatch(ctx, 0, bisectHeight, 1)
		require.NoError(t, err)
		honestProof, err := createdData.HonestStateManager.PrefixProofUpToBatch(ctx, 0, bisectHeight, protocol.LevelZeroBlockEdgeHeight, 1)
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
	bisectionScenario := setupBisectionScenario(t)
	honestStateManager := bisectionScenario.honestStateManager
	evilStateManager := bisectionScenario.evilStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge
	evilEdge := bisectionScenario.evilLevelZeroEdge

	t.Run("level zero edge with rivals is not one step fork source", func(t *testing.T) {
		isOSF, err := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
		isOSF, err = evilEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, isOSF)
	})
	t.Run("post bisection, mutual edge is one step fork source", func(t *testing.T) {
		var height uint64 = protocol.LevelZeroBlockEdgeHeight
		for height > 1 {
			honestBisectCommit, err := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, height/2, 1)
			require.NoError(t, err)
			honestProof, err := honestStateManager.PrefixProofUpToBatch(ctx, 0, height/2, height, 1)
			require.NoError(t, err)
			honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
			require.NoError(t, err)

			evilBisectCommit, err := evilStateManager.HistoryCommitmentUpToBatch(ctx, 0, height/2, 1)
			require.NoError(t, err)
			evilProof, err := evilStateManager.PrefixProofUpToBatch(ctx, 0, height/2, height, 1)
			require.NoError(t, err)
			evilEdge, _, err = evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
			require.NoError(t, err)

			height /= 2

			isOSF, err := honestEdge.HasLengthOneRival(ctx)
			require.NoError(t, err)
			require.Equal(t, height == 1, isOSF)
			isOSF, err = evilEdge.HasLengthOneRival(ctx)
			require.NoError(t, err)
			require.Equal(t, height == 1, isOSF)
		}
	})
}

func TestEdgeChallengeManager_BlockChallengeAddLevelZeroEdge(t *testing.T) {
	ctx := context.Background()
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
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

	start, err := createdData.HonestStateManager.HistoryCommitmentUpToBatch(ctx, 0, 0, 1)
	require.NoError(t, err)
	end, err := createdData.HonestStateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)
	prefixProof, err := createdData.HonestStateManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)

	t.Run("OK", func(t *testing.T) {
		_, err = challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end, prefixProof)
		require.NoError(t, err)
	})
	t.Run("already exists", func(t *testing.T) {
		_, err = challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end, prefixProof)
		require.ErrorContains(t, err, "already exists")
	})
}

func TestEdgeChallengeManager_Bisect(t *testing.T) {
	ctx := context.Background()
	bisectionScenario := setupBisectionScenario(t)
	honestStateManager := bisectionScenario.honestStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge

	t.Run("cannot bisect unrivaled", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("invalid prefix proof", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("edge has children", func(t *testing.T) {
		t.Skip("TODO(RJ): Implement")
	})
	t.Run("OK", func(t *testing.T) {
		honestBisectCommit, err := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, 1)
		require.NoError(t, err)
		honestProof, err := honestStateManager.PrefixProofUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, protocol.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, err)
		lower, upper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		gotLower, gotUpper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)
		require.Equal(t, lower.Id(), gotLower.Id())
		require.Equal(t, upper.Id(), gotUpper.Id())
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

func TestEdgeChallengeManager_ConfirmByOneStepProof(t *testing.T) {
	ctx := context.Background()
	t.Run("edge does not exist", func(t *testing.T) {
		bisectionScenario := setupBisectionScenario(t)
		challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
		require.NoError(t, err)
		err = challengeManager.ConfirmEdgeByOneStepProof(
			ctx,
			protocol.EdgeId(common.BytesToHash([]byte("foo"))),
			&protocol.OneStepData{
				BeforeHash:        common.Hash{},
				Proof:             make([]byte, 0),
				InboxMsgCountSeen: big.NewInt(0),
			},
			make([]common.Hash, 0),
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "Edge does not exist")
	})
	// t.Run("edge not pending", func(t *testing.T) {
	// 	bisectionScenario := setupBisectionScenario(t)
	// 	honestStateManager := bisectionScenario.honestStateManager
	// 	honestEdge := bisectionScenario.honestLevelZeroEdge
	// 	challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	// 	require.NoError(t, err)

	// 	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, 1)
	// 	require.NoError(t, err)
	// 	honestProof, err := honestStateManager.PrefixProofUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, protocol.LevelZeroBlockEdgeHeight, 1)
	// 	require.NoError(t, err)
	// 	honestChildren1, honestChildren2, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	// 	require.NoError(t, err)

	// 	s1, err := honestChildren1.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgePending, s1)
	// 	s2, err := honestChildren2.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgePending, s2)

	// 	// Adjust well beyond a challenge period.
	// 	for i := 0; i < 200; i++ {
	// 		bisectionScenario.topLevelFork.Backend.Commit()
	// 	}

	// 	require.NoError(t, honestChildren1.ConfirmByTimer(ctx, []protocol.EdgeId{honestEdge.Id()}))
	// 	require.NoError(t, honestChildren2.ConfirmByTimer(ctx, []protocol.EdgeId{honestEdge.Id()}))
	// 	s1, err = honestChildren1.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgeConfirmed, s1)
	// 	s2, err = honestChildren2.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgeConfirmed, s2)

	// 	executionHash, _, wasmModuleRoot, err := bisectionScenario.topLevelFork.Chains[0].GenesisAssertionHashes(ctx)
	// 	require.NoError(t, err)
	// 	wasmModuleRootProof, err := statemanager.WasmModuleProofAbi.Pack(common.Hash{}, executionHash, common.Hash{})
	// 	require.NoError(t, err)

	// 	inboxMaxCountProof, err := statemanager.ExecutionStateAbi.Pack(
	// 		common.Hash{},
	// 		common.Hash{},
	// 		uint64(0),
	// 		uint64(0),
	// 		protocol.MachineStatusFinished,
	// 	)
	// 	require.NoError(t, err)

	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		honestChildren1.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             common.Hash{},
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		make([]common.Hash, 0),
	// 		make([]common.Hash, 0),
	// 	)
	// 	require.ErrorContains(t, err, "Edge not pending")
	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		honestChildren2.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             common.Hash{},
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		make([]common.Hash, 0),
	// 		make([]common.Hash, 0),
	// 	)
	// 	require.ErrorContains(t, err, "Edge not pending")
	// })
	// t.Run("edge not small step type", func(t *testing.T) {
	// 	bisectionScenario := setupBisectionScenario(t)
	// 	honestStateManager := bisectionScenario.honestStateManager
	// 	honestEdge := bisectionScenario.honestLevelZeroEdge
	// 	challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	// 	require.NoError(t, err)

	// 	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, 1)
	// 	require.NoError(t, err)
	// 	honestProof, err := honestStateManager.PrefixProofUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, protocol.LevelZeroBlockEdgeHeight, 1)
	// 	require.NoError(t, err)
	// 	honestChildren1, honestChildren2, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	// 	require.NoError(t, err)

	// 	s1, err := honestChildren1.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgePending, s1)
	// 	s2, err := honestChildren2.Status(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, protocol.EdgePending, s2)

	// 	executionHash, _, wasmModuleRoot, err := bisectionScenario.topLevelFork.Chains[0].GenesisAssertionHashes(ctx)
	// 	require.NoError(t, err)
	// 	wasmModuleRootProof, err := statemanager.WasmModuleProofAbi.Pack(common.Hash{}, executionHash, common.Hash{})
	// 	require.NoError(t, err)

	// 	inboxMaxCountProof, err := statemanager.ExecutionStateAbi.Pack(
	// 		common.Hash{},
	// 		common.Hash{},
	// 		uint64(0),
	// 		uint64(0),
	// 		protocol.MachineStatusFinished,
	// 	)

	// 	require.NoError(t, err)
	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		honestChildren1.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             common.Hash{},
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		make([]common.Hash, 0),
	// 		make([]common.Hash, 0),
	// 	)
	// 	require.ErrorContains(t, err, "Edge is not a small step")
	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		honestChildren2.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             common.Hash{},
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		make([]common.Hash, 0),
	// 		make([]common.Hash, 0),
	// 	)
	// 	require.ErrorContains(t, err, "Edge is not a small step")
	// })
	// t.Run("before state not in history", func(t *testing.T) {
	// 	scenario := setupOneStepProofScenario(t)
	// 	honestEdge := scenario.smallStepHonestEdge

	// 	challengeManager, err := scenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	// 	require.NoError(t, err)

	// 	honestStateManager := scenario.honestStateManager
	// 	fromAssertion := uint64(0)
	// 	toAssertion := uint64(1)
	// 	fromBigStep := uint64(0)
	// 	toBigStep := fromBigStep + 1
	// 	toSmallStep := uint64(0)
	// 	honestCommit, err := honestStateManager.SmallStepCommitmentUpTo(
	// 		ctx,
	// 		fromAssertion,
	// 		toAssertion,
	// 		fromBigStep,
	// 		toBigStep,
	// 		toSmallStep,
	// 	)
	// 	require.NoError(t, err)

	// 	executionHash, _, wasmModuleRoot, err := scenario.topLevelFork.Chains[0].GenesisAssertionHashes(ctx)
	// 	require.NoError(t, err)
	// 	wasmModuleRootProof, err := statemanager.WasmModuleProofAbi.Pack(common.Hash{}, executionHash, common.Hash{})
	// 	require.NoError(t, err)

	// 	inboxMaxCountProof, err := statemanager.ExecutionStateAbi.Pack(
	// 		common.Hash{},
	// 		common.Hash{},
	// 		uint64(0),
	// 		uint64(0),
	// 		protocol.MachineStatusFinished,
	// 	)
	// 	require.NoError(t, err)

	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		honestEdge.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             common.BytesToHash([]byte("foo")),
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		honestCommit.FirstLeafProof,
	// 		honestCommit.LastLeafProof,
	// 	)
	// 	require.ErrorContains(t, err, "Invalid inclusion proof")
	// })
	// t.Run("one step proof fails", func(t *testing.T) {
	// 	scenario := setupOneStepProofScenario(t)
	// 	evilEdge := scenario.smallStepEvilEdge

	// 	challengeManager, err := scenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	// 	require.NoError(t, err)

	// 	evilStateManager := scenario.evilStateManager
	// 	fromAssertion := uint64(0)
	// 	toAssertion := uint64(1)
	// 	fromBigStep := uint64(0)
	// 	toBigStep := fromBigStep + 1
	// 	toSmallStep := uint64(0)
	// 	startCommit, err := evilStateManager.SmallStepCommitmentUpTo(
	// 		ctx,
	// 		fromAssertion,
	// 		toAssertion,
	// 		fromBigStep,
	// 		toBigStep,
	// 		toSmallStep,
	// 	)
	// 	require.NoError(t, err)
	// 	endCommit, err := evilStateManager.SmallStepCommitmentUpTo(
	// 		ctx,
	// 		fromAssertion,
	// 		toAssertion,
	// 		fromBigStep,
	// 		toBigStep,
	// 		toSmallStep,
	// 	)
	// 	require.NoError(t, err)

	// 	executionHash, _, wasmModuleRoot, err := scenario.topLevelFork.Chains[0].GenesisAssertionHashes(ctx)
	// 	require.NoError(t, err)
	// 	wasmModuleRootProof, err := statemanager.WasmModuleProofAbi.Pack(common.Hash{}, executionHash, common.Hash{})
	// 	require.NoError(t, err)

	// 	inboxMaxCountProof, err := statemanager.ExecutionStateAbi.Pack(
	// 		common.Hash{},
	// 		common.Hash{},
	// 		uint64(0),
	// 		uint64(0),
	// 		protocol.MachineStatusFinished,
	// 	)
	// 	require.NoError(t, err)

	// 	err = challengeManager.ConfirmEdgeByOneStepProof(
	// 		ctx,
	// 		evilEdge.Id(),
	// 		&protocol.OneStepData{
	// 			BeforeHash:             startCommit.LastLeaf,
	// 			Proof:                  genesisOspData,
	// 			InboxMsgCountSeen:      big.NewInt(1),
	// 			InboxMsgCountSeenProof: inboxMaxCountProof,
	// 			WasmModuleRoot:         wasmModuleRoot,
	// 			WasmModuleRootProof:    wasmModuleRootProof,
	// 		},
	// 		startCommit.LastLeafProof,
	// 		endCommit.LastLeafProof,
	// 	)
	// 	require.ErrorContains(t, err, "Invalid inclusion proof")
	// })
	t.Run("OK", func(t *testing.T) {
		scenario := setupOneStepProofScenario(t)
		honestEdge := scenario.smallStepHonestEdge

		chain := scenario.topLevelFork.Chains[0]
		challengeManager, err := scenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
		require.NoError(t, err)

		honestStateManager := scenario.honestStateManager
		fromBlockChallengeHeight := uint64(0)
		toBlockChallengeHeight := uint64(1)
		fromBigStep := uint64(0)
		toBigStep := uint64(1)
		fromSmallStep := uint64(0)
		toSmallStep := uint64(1)

		prevId, err := honestEdge.PrevAssertionId(ctx)
		require.NoError(t, err)
		parentAssertionCreationInfo, err := chain.ReadAssertionCreationInfo(ctx, prevId)
		require.NoError(t, err)

		requiredStake, err := chain.BaseStake(ctx)
		require.NoError(t, err)

		challengePeriod, err := challengeManager.ChallengePeriodBlocks(ctx)
		require.NoError(t, err)

		wasmRoot, err := chain.WasmModuleRoot(ctx)
		require.NoError(t, err)

		cfgSnapshot := &l2stateprovider.ConfigSnapshot{
			RequiredStake:           requiredStake,
			ChallengeManagerAddress: challengeManager.Address(),
			ConfirmPeriodBlocks:     challengePeriod,
			WasmModuleRoot:          wasmRoot,
			InboxMaxCount:           big.NewInt(1),
		}

		data, startInclusionProof, endInclusionProof, err := honestStateManager.OneStepProofData(
			ctx,
			cfgSnapshot,
			parentAssertionCreationInfo.AfterState,
			fromBlockChallengeHeight,
			toBlockChallengeHeight,
			fromBigStep,
			toBigStep,
			fromSmallStep,
			toSmallStep,
		)
		require.NoError(t, err)

		err = challengeManager.ConfirmEdgeByOneStepProof(
			ctx,
			honestEdge.Id(),
			data,
			startInclusionProof,
			endInclusionProof,
		)
		require.NoError(t, err)
		edgeStatus, err := honestEdge.Status(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.EdgeConfirmed, edgeStatus)

		require.NoError(t, challengeManager.ConfirmEdgeByOneStepProof(
			ctx,
			honestEdge.Id(),
			data,
			startInclusionProof,
			endInclusionProof,
		)) // already confirmed should not fail.
	})
}

func TestEdgeChallengeManager_ConfirmByTimerAndChildren(t *testing.T) {
	ctx := context.Background()
	bisectionScenario := setupBisectionScenario(t)
	honestStateManager := bisectionScenario.honestStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge

	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, 1)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProofUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight/2, protocol.LevelZeroBlockEdgeHeight, 1)
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
	for i := 0; i < 200; i++ {
		bisectionScenario.topLevelFork.Backend.Commit()
	}

	require.NoError(t, honestChildren1.ConfirmByTimer(ctx, []protocol.EdgeId{honestEdge.Id()}))
	require.NoError(t, honestChildren2.ConfirmByTimer(ctx, []protocol.EdgeId{honestEdge.Id()}))
	s1, err = honestChildren1.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s1)
	s2, err = honestChildren2.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s2)

	require.NoError(t, honestEdge.ConfirmByChildren(ctx))
	s0, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, s0)

	require.NoError(t, honestEdge.ConfirmByChildren(ctx)) // already confirmed should not fail.
}

func TestEdgeChallengeManager_ConfirmByTimer(t *testing.T) {
	ctx := context.Background()

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

	t.Run("confirmed by timer", func(t *testing.T) {
		require.ErrorContains(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{protocol.EdgeId(common.Hash{1})}), "execution reverted: Edge does not exist")
	})
	t.Run("confirmed by timer", func(t *testing.T) {
		require.NoError(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{}))
		status, err := honestEdge.Status(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.EdgeConfirmed, status)
	})
	t.Run("double confirm is a no-op", func(t *testing.T) {
		status, err := honestEdge.Status(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.EdgeConfirmed, status)
		require.NoError(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{})) // already confirmed should not fail.
	})
}

// Returns a snapshot of the data for a scenario in which both honest
// and evil validator validators have created level zero edges in a top-level
// challenge and are ready to bisect.
type bisectionScenario struct {
	topLevelFork        *setup.CreatedValidatorFork
	honestStateManager  l2stateprovider.Provider
	evilStateManager    l2stateprovider.Provider
	honestLevelZeroEdge protocol.SpecEdge
	evilLevelZeroEdge   protocol.SpecEdge
	honestStartCommit   commitments.History
	evilStartCommit     commitments.History
}

func setupBisectionScenario(
	t *testing.T,
) *bisectionScenario {
	ctx := context.Background()

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{})
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) (commitments.History, protocol.SpecEdge) {
		startCommit, startErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, 0, 1)
		require.NoError(t, startErr)
		endCommit, endErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, endErr)
		prefixProof, prefixErr := stateManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, prefixErr)

		edge, edgeErr := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
			prefixProof,
		)
		require.NoError(t, edgeErr)
		return startCommit, edge
	}

	honestStartCommit, honestEdge := leafAdder(createdData.HonestStateManager, createdData.Leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	isOSF, err := honestEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isOSF)

	evilStartCommit, evilEdge := leafAdder(createdData.EvilStateManager, createdData.Leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	// Honest and evil edge are rivals, neither is presumptive.
	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	return &bisectionScenario{
		topLevelFork:        createdData,
		honestStateManager:  createdData.HonestStateManager,
		evilStateManager:    createdData.EvilStateManager,
		honestLevelZeroEdge: honestEdge,
		evilLevelZeroEdge:   evilEdge,
		honestStartCommit:   honestStartCommit,
		evilStartCommit:     evilStartCommit,
	}
}

// Returns a snapshot of the data for a one-step-proof scenario in which
// an evil validator has reached a one step fork against an honest validator
// in a small step subchallenge. Their disagreement must then be resolved via
// a one-step-proof to declare a winner.
type oneStepProofScenario struct {
	topLevelFork        *setup.CreatedValidatorFork
	honestStateManager  l2stateprovider.Provider
	evilStateManager    l2stateprovider.Provider
	smallStepHonestEdge protocol.SpecEdge
	smallStepEvilEdge   protocol.SpecEdge
}

// Sets up a challenge between two validators in which they make challenge moves
// to reach a one-step-proof in a small step subchallenge. It returns the data needed
// to then confirm the winner by one-step-proof execution.
func setupOneStepProofScenario(
	t *testing.T,
) *oneStepProofScenario {
	ctx := context.Background()
	bisectionScenario := setupBisectionScenario(t)
	honestStateManager := bisectionScenario.honestStateManager
	evilStateManager := bisectionScenario.evilStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge
	evilEdge := bisectionScenario.evilLevelZeroEdge

	challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	require.NoError(t, err)

	var blockHeight uint64 = protocol.LevelZeroBlockEdgeHeight
	for blockHeight > 1 {
		honestBisectCommit, honestErr := honestStateManager.HistoryCommitmentUpToBatch(ctx, 0, blockHeight/2, 1)
		require.NoError(t, honestErr)
		honestProof, honestProofErr := honestStateManager.PrefixProofUpToBatch(ctx, 0, blockHeight/2, blockHeight, 1)
		require.NoError(t, honestProofErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		evilBisectCommit, bisectErr := evilStateManager.HistoryCommitmentUpToBatch(ctx, 0, blockHeight/2, 1)
		require.NoError(t, bisectErr)
		evilProof, evilErr := evilStateManager.PrefixProofUpToBatch(ctx, 0, blockHeight/2, blockHeight, 1)
		require.NoError(t, evilErr)
		evilEdge, _, err = evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
		require.NoError(t, err)

		blockHeight /= 2

		isOSF, osfErr := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, osfErr)
		require.Equal(t, blockHeight == 1, isOSF)
		isOSF, osfErr = evilEdge.HasLengthOneRival(ctx)
		require.NoError(t, osfErr)
		require.Equal(t, blockHeight == 1, isOSF)
	}

	// Now opening big step level zero leaves at index 0
	bigStepAdder := func(stateManager l2stateprovider.Provider, sourceEdge protocol.SpecEdge) protocol.SpecEdge {
		startCommit, startErr := stateManager.BigStepCommitmentUpTo(ctx, 0, 1, 0)
		require.NoError(t, startErr)
		endCommit, endErr := stateManager.BigStepLeafCommitment(ctx, 0, 1)
		require.NoError(t, endErr)
		require.Equal(t, startCommit.LastLeaf, endCommit.FirstLeaf)
		startParentCommitment, parentErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, 0, 1)
		require.NoError(t, parentErr)
		endParentCommitment, endParentErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, 1, 1)
		require.NoError(t, endParentErr)
		startEndPrefixProof, proofErr := stateManager.BigStepPrefixProof(ctx, 0, 1, 0, endCommit.Height)
		require.NoError(t, proofErr)
		leaf, leafErr := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			sourceEdge,
			startCommit,
			endCommit,
			startParentCommitment.LastLeafProof,
			endParentCommitment.LastLeafProof,
			startEndPrefixProof,
		)
		require.NoError(t, leafErr)
		return leaf
	}

	honestEdge = bigStepAdder(honestStateManager, honestEdge)
	require.Equal(t, protocol.BigStepChallengeEdge, honestEdge.GetType())
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilEdge = bigStepAdder(evilStateManager, evilEdge)
	require.Equal(t, protocol.BigStepChallengeEdge, evilEdge.GetType())

	var bigStepHeight uint64 = protocol.LevelZeroBigStepEdgeHeight
	for bigStepHeight > 1 {
		honestBisectCommit, bisectErr := honestStateManager.BigStepCommitmentUpTo(ctx, 0, 1, bigStepHeight/2)
		require.NoError(t, bisectErr)
		honestProof, honestErr := honestStateManager.BigStepPrefixProof(ctx, 0, 1, bigStepHeight/2, bigStepHeight)
		require.NoError(t, honestErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		evilBisectCommit, bisectErr := evilStateManager.BigStepCommitmentUpTo(ctx, 0, 1, bigStepHeight/2)
		require.NoError(t, bisectErr)
		evilProof, evilErr := evilStateManager.BigStepPrefixProof(ctx, 0, 1, bigStepHeight/2, bigStepHeight)
		require.NoError(t, evilErr)
		evilEdge, _, err = evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
		require.NoError(t, err)

		bigStepHeight /= 2

		isOSF, osfErr := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, osfErr)
		require.Equal(t, bigStepHeight == 1, isOSF)
		isOSF, osfErr = evilEdge.HasLengthOneRival(ctx)
		require.NoError(t, osfErr)
		require.Equal(t, bigStepHeight == 1, isOSF)
	}

	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	isAtOneStepFork, err := honestEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	// Now opening small step level zero leaves at index 0
	smallStepAdder := func(stateManager l2stateprovider.Provider, edge protocol.SpecEdge) protocol.SpecEdge {
		startCommit, startErr := stateManager.SmallStepCommitmentUpTo(ctx, 0, 1, 0, 1, 0)
		require.NoError(t, startErr)
		endCommit, endErr := stateManager.SmallStepLeafCommitment(ctx, 0, 1, 0, 1)
		require.NoError(t, endErr)
		startParentCommitment, parentErr := stateManager.BigStepCommitmentUpTo(ctx, 0, 1, 0)
		require.NoError(t, parentErr)
		endParentCommitment, endParentErr := stateManager.BigStepCommitmentUpTo(ctx, 0, 1, 1)
		require.NoError(t, endParentErr)
		startEndPrefixProof, prefixErr := stateManager.SmallStepPrefixProof(ctx, 0, 1, 0, 1, 0, endCommit.Height)
		require.NoError(t, prefixErr)
		leaf, leafErr := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			edge,
			startCommit,
			endCommit,
			startParentCommitment.LastLeafProof,
			endParentCommitment.LastLeafProof,
			startEndPrefixProof,
		)
		require.NoError(t, leafErr)

		_, leafErr = challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			edge,
			startCommit,
			endCommit,
			startParentCommitment.LastLeafProof,
			endParentCommitment.LastLeafProof,
			startEndPrefixProof,
		)
		require.NoError(t, leafErr) // Already submitted, should be a no-op.

		return leaf
	}

	honestEdge = smallStepAdder(honestStateManager, honestEdge)
	require.Equal(t, protocol.SmallStepChallengeEdge, honestEdge.GetType())
	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilEdge = smallStepAdder(evilStateManager, evilEdge)
	require.Equal(t, protocol.SmallStepChallengeEdge, evilEdge.GetType())

	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	// Get the lower-level edge of either edge we just bisected.
	require.Equal(t, protocol.SmallStepChallengeEdge, honestEdge.GetType())

	var smallStepHeight uint64 = protocol.LevelZeroBigStepEdgeHeight
	for smallStepHeight > 1 {
		honestBisectCommit, bisectErr := honestStateManager.SmallStepCommitmentUpTo(ctx, 0, 1, 0, 1, smallStepHeight/2)
		require.NoError(t, bisectErr)
		honestProof, proofErr := honestStateManager.SmallStepPrefixProof(ctx, 0, 1, 0, 1, smallStepHeight/2, smallStepHeight)
		require.NoError(t, proofErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		evilBisectCommit, evilBisectErr := evilStateManager.SmallStepCommitmentUpTo(ctx, 0, 1, 0, 1, smallStepHeight/2)
		require.NoError(t, evilBisectErr)
		evilProof, evilProofErr := evilStateManager.SmallStepPrefixProof(ctx, 0, 1, 0, 1, smallStepHeight/2, smallStepHeight)
		require.NoError(t, evilProofErr)
		evilEdge, _, err = evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
		require.NoError(t, err)

		smallStepHeight /= 2

		isOSF, osfErr := honestEdge.HasLengthOneRival(ctx)
		require.NoError(t, osfErr)
		require.Equal(t, smallStepHeight == 1, isOSF)
		isOSF, err = evilEdge.HasLengthOneRival(ctx)
		require.NoError(t, err)
		require.Equal(t, smallStepHeight == 1, isOSF)
	}

	isAtOneStepFork, err = honestEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)
	isAtOneStepFork, err = evilEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	return &oneStepProofScenario{
		topLevelFork:        bisectionScenario.topLevelFork,
		honestStateManager:  honestStateManager,
		evilStateManager:    evilStateManager,
		smallStepHonestEdge: honestEdge,
		smallStepEvilEdge:   evilEdge,
	}
}
