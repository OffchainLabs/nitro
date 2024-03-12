// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl_test

import (
	"context"
	"fmt"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	stateprovider "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestEdgeChallengeManager_IsUnrivaled(t *testing.T) {
	ctx := context.Background()

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) protocol.SpecEdge {
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startCommit, startErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, startErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight))
		endCommit, endErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endErr)

		prefixProof, proofErr := stateManager.PrefixProof(ctx, req, 0)

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
	challengeLevel := honestEdge.GetChallengeLevel()
	require.Equal(t, protocol.NewBlockChallengeLevel(), challengeLevel)

	t.Run("first leaf is presumptive", func(t *testing.T) {
		hasRival, rivalErr := honestEdge.HasRival(ctx)
		require.NoError(t, rivalErr)
		require.Equal(t, true, !hasRival)
	})

	evilEdge := leafAdder(createdData.EvilStateManager, createdData.Leaf2)
	challengeLevel = evilEdge.GetChallengeLevel()
	require.Equal(t, protocol.NewBlockChallengeLevel(), challengeLevel)

	t.Run("neither is presumptive if rivals", func(t *testing.T) {
		hasRival, err := honestEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)

		hasRival, err = evilEdge.HasRival(ctx)
		require.NoError(t, err)
		require.Equal(t, false, !hasRival)
	})

	t.Run("bisected children are presumptive", func(t *testing.T) {
		var bisectHeight uint64 = challenge_testing.LevelZeroBlockEdgeHeight / 2
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(bisectHeight)),
		}
		honestBisectCommit, err := createdData.HonestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, err)
		req = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight)),
		}
		honestProof, err := createdData.HonestStateManager.PrefixProof(ctx, req, l2stateprovider.Height(bisectHeight))
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
		var height uint64 = challenge_testing.LevelZeroBlockEdgeHeight
		for height > 1 {
			req := &l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              common.Hash{},
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(height / 2)),
			}
			honestBisectCommit, err := honestStateManager.HistoryCommitment(ctx, req)
			require.NoError(t, err)
			prefixCommitReq := &l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              common.Hash{},
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(height)),
			}
			honestProof, err := honestStateManager.PrefixProof(
				ctx,
				prefixCommitReq,
				l2stateprovider.Height(height/2),
			)
			require.NoError(t, err)
			honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
			require.NoError(t, err)

			evilBisectCommit, err := evilStateManager.HistoryCommitment(ctx, req)
			require.NoError(t, err)
			evilProof, err := evilStateManager.PrefixProof(ctx, prefixCommitReq, l2stateprovider.Height(height/2))
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
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	chain1 := createdData.Chains[0]
	challengeManager, err := chain1.SpecChallengeManager(ctx)
	require.NoError(t, err)

	leaves := make([]common.Hash, 4)
	for i := range leaves {
		leaves[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}

	req := &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              common.Hash{},
		FromBatch:                   0,
		ToBatch:                     1,
		UpperChallengeOriginHeights: []l2stateprovider.Height{},
		FromHeight:                  0,
		UpToHeight:                  option.Some(l2stateprovider.Height(0)),
	}
	start, err := createdData.HonestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, err)
	req.UpToHeight = option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight))
	end, err := createdData.HonestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, err)
	prefixProof, err := createdData.HonestStateManager.PrefixProof(ctx, req, l2stateprovider.Height(0))
	require.NoError(t, err)

	t.Run("OK", func(t *testing.T) {
		created, err := challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end, prefixProof)
		require.NoError(t, err)
		existing, err := challengeManager.AddBlockChallengeLevelZeroEdge(ctx, createdData.Leaf1, start, end, prefixProof)
		require.NoError(t, err)
		require.Equal(t, created, existing)
	})
}

func TestEdgeChallengeManager_Bisect(t *testing.T) {
	ctx := context.Background()
	bisectionScenario := setupBisectionScenario(t)
	honestStateManager := bisectionScenario.honestStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge

	t.Run("OK", func(t *testing.T) {
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight / 2)),
		}
		honestBisectCommit, err := honestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, err)
		req.UpToHeight = option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight))
		honestProof, err := honestStateManager.PrefixProof(ctx, req, challenge_testing.LevelZeroBlockEdgeHeight/2)
		require.NoError(t, err)
		lower, upper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		gotLower, gotUpper, err := honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)
		require.Equal(t, lower.Id(), gotLower.Id())
		require.Equal(t, upper.Id(), gotUpper.Id())
	})
}

func TestEdgeChallengeManager_AddSubchallengeLeaf(t *testing.T) {
	// Set up a scenario we can bisect.
	ctx := context.Background()

	heights := &protocol.LayerZeroHeights{
		BlockChallengeHeight:     1 << 5,
		BigStepChallengeHeight:   1 << 5,
		SmallStepChallengeHeight: 1 << 5,
	}
	numBigSteps := uint8(1)
	bisectionScenario := setupBisectionScenario(
		t,
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(heights),
			challenge_testing.WithNumBigStepLevels(numBigSteps),
		),
		setup.WithStateManagerOpts(
			stateprovider.WithLayerZeroHeights(heights, numBigSteps),
		),
	)
	honestStateManager := bisectionScenario.honestStateManager
	evilStateManager := bisectionScenario.evilStateManager
	honestEdge := bisectionScenario.honestLevelZeroEdge
	evilEdge := bisectionScenario.evilLevelZeroEdge

	challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Perform bisections all the way down to a one step fork.
	var blockHeight uint64 = challenge_testing.LevelZeroBlockEdgeHeight
	for blockHeight > 1 {
		bisectTo := l2stateprovider.Height(blockHeight / 2)
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(bisectTo),
		}
		honestBisectCommit, honestErr := honestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, honestErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(blockHeight))
		honestProof, honestProofErr := honestStateManager.PrefixProof(ctx, req, bisectTo)
		require.NoError(t, honestProofErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		req.UpToHeight = option.Some(bisectTo)
		evilBisectCommit, bisectErr := evilStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, bisectErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(blockHeight))
		evilProof, evilErr := evilStateManager.PrefixProof(ctx, req, bisectTo)
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

	req := &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              common.Hash{},
		FromBatch:                   0,
		ToBatch:                     1,
		UpperChallengeOriginHeights: []l2stateprovider.Height{0},
		FromHeight:                  0,
		UpToHeight:                  option.Some(l2stateprovider.Height(0)),
	}
	startCommit, startErr := honestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, startErr)
	req.UpToHeight = option.None[l2stateprovider.Height]()
	endCommit, endErr := honestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, endErr)
	require.Equal(t, startCommit.LastLeaf, endCommit.FirstLeaf)

	req = &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              common.Hash{},
		FromBatch:                   0,
		ToBatch:                     1,
		UpperChallengeOriginHeights: []l2stateprovider.Height{},
		FromHeight:                  0,
		UpToHeight:                  option.Some(l2stateprovider.Height(0)),
	}
	startParentCommitment, parentErr := honestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, parentErr)
	req.UpToHeight = option.Some(l2stateprovider.Height(1))
	endParentCommitment, endParentErr := honestStateManager.HistoryCommitment(ctx, req)
	require.NoError(t, endParentErr)

	req = &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              common.Hash{},
		FromBatch:                   0,
		ToBatch:                     1,
		UpperChallengeOriginHeights: []l2stateprovider.Height{0},
		FromHeight:                  0,
		UpToHeight:                  option.Some(l2stateprovider.Height(endCommit.Height)),
	}
	startEndPrefixProof, proofErr := honestStateManager.PrefixProof(ctx, req, 0)
	require.NoError(t, proofErr)

	leaf, err := challengeManager.AddSubChallengeLevelZeroEdge(
		ctx,
		honestEdge,
		startCommit,
		endCommit,
		startParentCommitment.LastLeafProof,
		endParentCommitment.LastLeafProof,
		startEndPrefixProof,
	)
	require.NoError(t, err)

	lvl := leaf.GetChallengeLevel()
	require.Equal(t, protocol.ChallengeLevel(1), lvl)
}

func TestEdgeChallengeManager_ConfirmByOneStepProof(t *testing.T) {
	ctx := context.Background()
	t.Run("edge does not exist", func(t *testing.T) {
		bisectionScenario := setupBisectionScenario(t)
		challengeManager, err := bisectionScenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
		require.NoError(t, err)
		err = challengeManager.ConfirmEdgeByOneStepProof(
			ctx,
			protocol.EdgeId{Hash: common.BytesToHash([]byte("foo"))},
			&protocol.OneStepData{
				BeforeHash: common.Hash{},
				Proof:      make([]byte, 0),
			},
			make([]common.Hash, 0),
			make([]common.Hash, 0),
		)
		require.ErrorContains(t, err, "execution reverted")
	})
	t.Run("OK", func(t *testing.T) {
		scenario := setupOneStepProofScenario(t)
		honestEdge := scenario.smallStepHonestEdge

		chain := scenario.topLevelFork.Chains[0]
		challengeManager, err := scenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
		require.NoError(t, err)

		honestStateManager := scenario.honestStateManager
		fromBlockChallengeHeight := uint64(0)
		fromBigStep := uint64(0)
		smallStep := uint64(0)

		id, err := honestEdge.AssertionHash(ctx)
		require.NoError(t, err)
		parentAssertionCreationInfo, err := chain.ReadAssertionCreationInfo(ctx, id)
		require.NoError(t, err)

		data, startInclusionProof, endInclusionProof, err := honestStateManager.OneStepProofData(
			ctx,
			parentAssertionCreationInfo.WasmModuleRoot,
			0,
			1,
			[]l2stateprovider.Height{
				l2stateprovider.Height(fromBlockChallengeHeight),
				l2stateprovider.Height(fromBigStep),
			},
			0,
			l2stateprovider.Height(smallStep),
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
		)) // already confirmed should not error.
	})
}

func TestEdgeChallengeManager_ConfirmByTimer(t *testing.T) {
	ctx := context.Background()

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) protocol.SpecEdge {
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startCommit, startErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, startErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight))
		endCommit, endErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProof(ctx, req, 0)
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

	t.Run("edge not found", func(t *testing.T) {
		require.ErrorContains(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{{Hash: common.Hash{1}}}), "execution reverted")
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
		require.NoError(t, honestEdge.ConfirmByTimer(ctx, []protocol.EdgeId{})) // already confirmed should not error.
	})
}

func TestUpgradingConfigMidChallenge(t *testing.T) {
	ctx := context.Background()
	scenario := setupOneStepProofScenario(t)

	rollupAddr := scenario.topLevelFork.Addrs.Rollup
	backend := scenario.topLevelFork.Backend
	adminAccount := scenario.topLevelFork.Accounts[0].TxOpts

	// We upgrade the Rollup's config values.
	adminLogic, err := rollupgen.NewRollupAdminLogic(rollupAddr, backend)
	require.NoError(t, err)

	newWasmModuleRoot := common.BytesToHash([]byte("nyannyannyan"))
	tx, err := adminLogic.SetWasmModuleRoot(adminAccount, newWasmModuleRoot)
	require.NoError(t, err)
	err = challenge_testing.WaitForTx(ctx, backend, tx)
	require.NoError(t, err)
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	tx, err = adminLogic.SetConfirmPeriodBlocks(adminAccount, uint64(329094))
	require.NoError(t, err)
	err = challenge_testing.WaitForTx(ctx, backend, tx)
	require.NoError(t, err)
	receipt, err = backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// We confirm the edge by one-step-proof.
	honestEdge := scenario.smallStepHonestEdge
	chain := scenario.topLevelFork.Chains[0]
	challengeManager, err := scenario.topLevelFork.Chains[1].SpecChallengeManager(ctx)
	require.NoError(t, err)

	honestStateManager := scenario.honestStateManager
	fromBlockChallengeHeight := uint64(0)
	fromBigStep := uint64(0)
	smallStep := uint64(0)

	id, err := honestEdge.AssertionHash(ctx)
	require.NoError(t, err)
	parentAssertionCreationInfo, err := chain.ReadAssertionCreationInfo(ctx, id)
	require.NoError(t, err)

	// We check the config snapshot used for the one step proof is different than what
	// is now onchain, as these values changed mid-challenge.
	gotWasmModuleRoot, err := adminLogic.WasmModuleRoot(util.GetSafeCallOpts(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, newWasmModuleRoot[:], gotWasmModuleRoot[:])
	require.NotEqual(t, parentAssertionCreationInfo.WasmModuleRoot[:], gotWasmModuleRoot)

	data, startInclusionProof, endInclusionProof, err := honestStateManager.OneStepProofData(
		ctx,
		parentAssertionCreationInfo.WasmModuleRoot,
		0,
		1,
		[]l2stateprovider.Height{
			l2stateprovider.Height(fromBlockChallengeHeight),
			l2stateprovider.Height(fromBigStep),
		},
		0,
		l2stateprovider.Height(smallStep),
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

	// Check the edge was confirmed.
	edgeStatus, err := honestEdge.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, protocol.EdgeConfirmed, edgeStatus)
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
	opts ...setup.Opt,
) *bisectionScenario {
	ctx := context.Background()

	opts = append(opts, setup.WithMockOneStepProver())
	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, opts...)
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) (commitments.History, protocol.SpecEdge) {
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startCommit, startErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, startErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight))
		endCommit, endErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProof(ctx, req, 0)
		require.NoError(t, proofErr)
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
	chalLevel := honestEdge.GetChallengeLevel()
	require.Equal(t, true, chalLevel.IsBlockChallengeLevel())
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	isOSF, err := honestEdge.HasLengthOneRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isOSF)

	evilStartCommit, evilEdge := leafAdder(createdData.EvilStateManager, createdData.Leaf2)
	chalLevel = evilEdge.GetChallengeLevel()
	require.Equal(t, true, chalLevel.IsBlockChallengeLevel())

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

	var blockHeight uint64 = challenge_testing.LevelZeroBlockEdgeHeight
	for blockHeight > 1 {
		bisectTo := l2stateprovider.Height(blockHeight / 2)
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(bisectTo),
		}
		honestBisectCommit, honestErr := honestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, honestErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(blockHeight))
		honestProof, honestProofErr := honestStateManager.PrefixProof(ctx, req, bisectTo)
		require.NoError(t, honestProofErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		req.UpToHeight = option.Some(bisectTo)
		evilBisectCommit, bisectErr := evilStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, bisectErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(blockHeight))
		evilProof, evilErr := evilStateManager.PrefixProof(ctx, req, bisectTo)
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
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startCommit, startErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, startErr)
		req.UpToHeight = option.None[l2stateprovider.Height]()
		endCommit, endErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endErr)
		require.Equal(t, startCommit.LastLeaf, endCommit.FirstLeaf)

		req = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startParentCommitment, parentErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, parentErr)
		req.UpToHeight = option.Some(l2stateprovider.Height(1))
		endParentCommitment, endParentErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endParentErr)

		req = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(endCommit.Height)),
		}
		startEndPrefixProof, proofErr := stateManager.PrefixProof(ctx, req, 0)
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
	challengeLevel := honestEdge.GetChallengeLevel()
	totalChallengeLevels := honestEdge.GetTotalChallengeLevels(ctx)
	require.Equal(t, true, uint8(challengeLevel) < totalChallengeLevels-1)
	require.Equal(t, true, challengeLevel > 0)
	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilEdge = bigStepAdder(evilStateManager, evilEdge)
	challengeLevel = evilEdge.GetChallengeLevel()
	totalChallengeLevels = evilEdge.GetTotalChallengeLevels(ctx)
	require.Equal(t, true, uint8(challengeLevel) < totalChallengeLevels-1)
	require.Equal(t, true, challengeLevel > 0)

	var bigStepHeight uint64 = challenge_testing.LevelZeroBigStepEdgeHeight
	for bigStepHeight > 1 {
		bisectTo := l2stateprovider.Height(bigStepHeight / 2)

		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(bisectTo),
		}
		honestBisectCommit, bisectErr := honestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, bisectErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(bigStepHeight))
		honestProof, honestErr := honestStateManager.PrefixProof(ctx, req, bisectTo)
		require.NoError(t, honestErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		req.UpToHeight = option.Some(bisectTo)
		evilBisectCommit, bisectErr := evilStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, bisectErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(bigStepHeight))
		evilProof, evilErr := evilStateManager.PrefixProof(ctx, req, bisectTo)
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
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0, 0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startCommit, startErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, startErr)

		req.UpToHeight = option.None[l2stateprovider.Height]()
		endCommit, endErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endErr)

		req = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startParentCommitment, parentErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, parentErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(1))
		endParentCommitment, endParentErr := stateManager.HistoryCommitment(ctx, req)
		require.NoError(t, endParentErr)

		req = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0, 0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(endCommit.Height)),
		}
		startEndPrefixProof, prefixErr := stateManager.PrefixProof(ctx, req, 0)
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
	challengeLevel = honestEdge.GetChallengeLevel()
	totalChallengeLevels = honestEdge.GetTotalChallengeLevels(ctx)
	require.Equal(t, true, uint8(challengeLevel) == totalChallengeLevels-1)
	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilEdge = smallStepAdder(evilStateManager, evilEdge)
	challengeLevel = honestEdge.GetChallengeLevel()
	totalChallengeLevels = honestEdge.GetTotalChallengeLevels(ctx)
	require.Equal(t, true, uint8(challengeLevel) == totalChallengeLevels-1)

	hasRival, err = honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)
	hasRival, err = evilEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, false, !hasRival)

	// Get the lower-level edge of either edge we just bisected.
	challengeLevel = honestEdge.GetChallengeLevel()
	totalChallengeLevels = honestEdge.GetTotalChallengeLevels(ctx)
	require.Equal(t, true, uint8(challengeLevel) == totalChallengeLevels-1)

	var smallStepHeight uint64 = challenge_testing.LevelZeroBigStepEdgeHeight
	for smallStepHeight > 1 {
		bisectTo := l2stateprovider.Height(smallStepHeight / 2)
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{0, 0},
			FromHeight:                  0,
			UpToHeight:                  option.Some(bisectTo),
		}

		honestBisectCommit, bisectErr := honestStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, bisectErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(smallStepHeight))
		honestProof, proofErr := honestStateManager.PrefixProof(ctx, req, bisectTo)
		require.NoError(t, proofErr)
		honestEdge, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
		require.NoError(t, err)

		req.UpToHeight = option.Some(bisectTo)
		evilBisectCommit, evilBisectErr := evilStateManager.HistoryCommitment(ctx, req)
		require.NoError(t, evilBisectErr)

		req.UpToHeight = option.Some(l2stateprovider.Height(smallStepHeight))
		evilProof, evilProofErr := evilStateManager.PrefixProof(ctx, req, bisectTo)
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
