// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl_test

import (
	"context"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestCreateAssertion(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	backend := cfg.Backend

	genesisHash, err := chain.GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisInfo, err := chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	t.Run("OK", func(t *testing.T) {
		latestBlockHash := common.Hash{}
		for i := uint64(0); i < 100; i++ {
			latestBlockHash = backend.Commit()
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
		assertion, err := chain.CreateAssertion(ctx, genesisInfo, postState)
		require.NoError(t, err)

		existingAssertion, err := chain.CreateAssertion(ctx, genesisInfo, postState)
		require.NoError(t, err)
		require.Equal(t, assertion.Id(), existingAssertion.Id())
	})
	t.Run("can create fork", func(t *testing.T) {
		assertionChain := cfg.Chains[1]

		for i := uint64(0); i < 100; i++ {
			backend.Commit()
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
		_, err := assertionChain.CreateAssertion(ctx, genesisInfo, postState)
		require.NoError(t, err)
	})
}

func TestAssertionUnrivaledBlocks(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	backend := cfg.Backend

	latestBlockHash := common.Hash{}
	for i := uint64(0); i < 100; i++ {
		latestBlockHash = backend.Commit()
	}
	genesisHash, err := chain.GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisInfo, err := chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	postState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash:  latestBlockHash,
			SendRoot:   common.Hash{},
			Batch:      1,
			PosInBatch: 0,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	assertion, err := chain.CreateAssertion(ctx, genesisInfo, postState)
	require.NoError(t, err)

	unrivaledBlocks, err := chain.AssertionUnrivaledBlocks(ctx, assertion.Id())
	require.NoError(t, err)

	// Should have been zero blocks since creation.
	require.Equal(t, uint64(0), unrivaledBlocks)

	backend.Commit()
	backend.Commit()
	backend.Commit()

	unrivaledBlocks, err = chain.AssertionUnrivaledBlocks(ctx, assertion.Id())
	require.NoError(t, err)

	// Three blocks since creation.
	require.Equal(t, uint64(3), unrivaledBlocks)

	// We then post a second child assertion.
	assertionChain := cfg.Chains[1]

	postState = &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash:  common.BytesToHash([]byte("evil hash")),
			SendRoot:   common.Hash{},
			Batch:      1,
			PosInBatch: 0,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	forkedAssertion, err := assertionChain.CreateAssertion(ctx, genesisInfo, postState)
	require.NoError(t, err)

	// We advance the chain by three blocks and check the assertion unrivaled times
	// of both created assertions.
	backend.Commit()
	backend.Commit()
	backend.Commit()

	unrivaledFirstChild, err := assertionChain.AssertionUnrivaledBlocks(ctx, assertion.Id())
	require.NoError(t, err)
	unrivaledSecondChild, err := assertionChain.AssertionUnrivaledBlocks(ctx, forkedAssertion.Id())
	require.NoError(t, err)

	// The amount of blocks unrivaled should not change for the first child (except for
	// the addition of one more block to account for the creation of its rival) and should
	// be zero for the second child block.
	require.Equal(t, uint64(4), unrivaledFirstChild)
	require.Equal(t, uint64(0), unrivaledSecondChild)

	// 100 blocks later, results should be unchanged.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	unrivaledFirstChild, err = assertionChain.AssertionUnrivaledBlocks(ctx, assertion.Id())
	require.NoError(t, err)
	unrivaledSecondChild, err = assertionChain.AssertionUnrivaledBlocks(ctx, forkedAssertion.Id())
	require.NoError(t, err)

	// The amount of blocks unrivaled should not change for the first child (except for
	// the addition of one more block to account for the creation of its rival) and should
	// be zero for the second child block.
	require.Equal(t, uint64(4), unrivaledFirstChild)
	require.Equal(t, uint64(0), unrivaledSecondChild)
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
		endCommit, endErr := stateManager.HistoryCommitmentUpToBatch(ctx, 0, challenge_testing.LevelZeroBlockEdgeHeight, 1)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProofUpToBatch(ctx, 0, 0, challenge_testing.LevelZeroBlockEdgeHeight, 1)
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

	_, err = chain.GetAssertion(ctx, protocol.AssertionHash{Hash: common.BytesToHash([]byte("foo"))})
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

type mockBackend struct {
	*backends.SimulatedBackend

	logs []types.Log
}

func (mb *mockBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return mb.logs, nil
}

func TestLatestCreatedAssertion(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]

	abi, err := rollupgen.RollupCoreMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	abiEvt := abi.Events["AssertionCreated"]

	packLog := func(evt *rollupgen.RollupCoreAssertionCreated) []byte {
		// event AssertionCreated(
		// 	bytes32 indexed assertionHash,
		// 	bytes32 indexed parentAssertionHash,
		// 	AssertionInputs assertion,
		// 	bytes32 afterInboxBatchAcc,
		// 	uint256 inboxMaxCount,
		// 	bytes32 wasmModuleRoot,
		// 	uint256 requiredStake,
		// 	address challengeManager,
		// 	uint64 confirmPeriodBlocks
		// );
		d, packErr := abiEvt.Inputs.Pack(
			evt.AssertionHash,
			evt.ParentAssertionHash,
			// Non-indexed fields.
			evt.Assertion,
			evt.AfterInboxBatchAcc,
			evt.InboxMaxCount,
			evt.WasmModuleRoot,
			evt.RequiredStake,
			evt.ChallengeManager,
			evt.ConfirmPeriodBlocks,
		)

		if packErr != nil {
			t.Fatal(packErr)
		}

		return d
	}

	// Minimal event data.
	// Note: *big.Int values cannot be nil.
	latest := &rollupgen.RollupCoreAssertionCreated{
		Assertion: rollupgen.AssertionInputs{
			BeforeStateData: rollupgen.BeforeStateData{
				ConfigData: rollupgen.ConfigData{RequiredStake: big.NewInt(0)},
			},
		},
		InboxMaxCount: big.NewInt(0),
		RequiredStake: big.NewInt(0),
	}

	// Use the latest confirmed assertion as the last assertion.
	expected, err := chain.LatestConfirmed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var latestAssertionID [32]byte
	copy(latestAssertionID[:], expected.Id().Bytes())
	var fakeAssertionID [32]byte
	copy(fakeAssertionID[:], []byte("fake assertion id as parent"))

	evtID := abiEvt.ID
	validTopics := []common.Hash{evtID, latestAssertionID, fakeAssertionID}
	// Invalid topics will return an error when trying to lookup an assertion with the fake ID.
	invalidTopics := []common.Hash{evtID, fakeAssertionID, fakeAssertionID}

	// The backend is bad and sent logs in the wrong order and also
	// sent "removed" logs from a nasty reorg.
	logs := []types.Log{
		{
			BlockNumber: 120,
			Index:       0,
			Topics:      invalidTopics,
		}, {
			BlockNumber: 119,
			Index:       0,
			Topics:      invalidTopics,
		}, {
			BlockNumber: 122,
			Index:       4,
			Topics:      invalidTopics,
			Removed:     true,
		},
		{ // This is the latest created assertion.
			BlockNumber: 122,
			Index:       3,
			Topics:      validTopics,
			Data:        packLog(latest),
		},
		{
			BlockNumber: 122,
			Index:       2,
			Topics:      invalidTopics,
		}, {
			BlockNumber: 120,
			Index:       0,
			Topics:      invalidTopics,
		},
	}

	chain.SetBackend(&mockBackend{logs: logs})

	latestCreated, err := chain.LatestCreatedAssertion(ctx)
	require.NoError(t, err)

	require.Equal(t, expected.Id().Hash, latestCreated.Id().Hash)
}
