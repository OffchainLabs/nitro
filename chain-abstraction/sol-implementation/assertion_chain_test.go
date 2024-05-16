// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl_test

import (
	"context"
	"math/big"
	"strings"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestNewStakeOnNewAssertion(t *testing.T) {
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
		assertion, err := chain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
		require.NoError(t, err)

		existingAssertion, err := chain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
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
		_, err := assertionChain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
		require.NoError(t, err)
	})
}

func TestStakeOnNewAssertion(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	backend := cfg.Backend

	genesisHash, err := chain.GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisInfo, err := chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

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
	assertion, err := chain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
	require.NoError(t, err)

	assertionInfo, err := chain.ReadAssertionCreationInfo(ctx, assertion.Id())
	require.NoError(t, err)

	postState = &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash:  common.BytesToHash([]byte("foo")),
			SendRoot:   common.Hash{},
			Batch:      postState.GlobalState.Batch + 1,
			PosInBatch: 0,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}

	enqueueSequencerMessageAsExecutor(
		t,
		cfg.Accounts[0].TxOpts,
		cfg.Addrs.UpgradeExecutor,
		cfg.Backend,
		cfg.Addrs.Bridge,
		seqMessage{
			dataHash:                 common.BytesToHash([]byte("foo")),
			afterDelayedMessagesRead: big.NewInt(1),
			prevMessageCount:         big.NewInt(1),
			newMessageCount:          big.NewInt(2),
		},
	)

	for i := uint64(0); i < 100; i++ {
		backend.Commit()
	}

	newAssertion, err := chain.StakeOnNewAssertion(ctx, assertionInfo, postState)
	require.NoError(t, err)

	newAssertionCreatedInfo, err := chain.ReadAssertionCreationInfo(ctx, newAssertion.Id())
	require.NoError(t, err)

	// Expect the post state has indeed the number of messages we expect.
	gotPostState := protocol.GoExecutionStateFromSolidity(newAssertionCreatedInfo.AfterState)
	require.Equal(t, postState, gotPostState)
}

type seqMessage struct {
	dataHash                 common.Hash
	afterDelayedMessagesRead *big.Int
	prevMessageCount         *big.Int
	newMessageCount          *big.Int
}

func enqueueSequencerMessageAsExecutor(
	t *testing.T,
	opts *bind.TransactOpts,
	executor common.Address,
	backend *setup.SimulatedBackendWrapper,
	bridge common.Address,
	msg seqMessage,
) {
	execBindings, err := mocksgen.NewUpgradeExecutorMock(executor, backend)
	require.NoError(t, err)
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.AbsBridgeABI))
	require.NoError(t, err)
	data, err := seqInboxABI.Pack(
		"setSequencerInbox",
		executor,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()

	data, err = seqInboxABI.Pack(
		"enqueueSequencerMessage",
		msg.dataHash, msg.afterDelayedMessagesRead, msg.prevMessageCount, msg.newMessageCount,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()
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
	assertion, err := chain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
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
	forkedAssertion, err := assertionChain.NewStakeOnNewAssertion(ctx, genesisInfo, postState)
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
	_, err := setup.ChainsWithEdgeChallengeManager(setup.WithMockOneStepProver())
	require.NoError(t, err)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) protocol.SpecEdge {
		startCommit, startErr := stateManager.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              common.Hash{},
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(0)),
			},
		)
		require.NoError(t, startErr)
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight)),
		}
		endCommit, endErr := stateManager.HistoryCommitment(
			ctx,
			req,
		)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProof(ctx, req, l2stateprovider.Height(0))
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
		_, err = honestEdge.ConfirmByTimer(ctx)
		require.NoError(t, err)

		// Adjust beyond the grace period.
		for i := 0; i < 10; i++ {
			createdData.Backend.Commit()
		}

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

func TestIsChallengeComplete(t *testing.T) {
	ctx := context.Background()
	_, err := setup.ChainsWithEdgeChallengeManager(setup.WithMockOneStepProver())
	require.NoError(t, err)

	createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{}, setup.WithMockOneStepProver())
	require.NoError(t, err)

	challengeManager, err := createdData.Chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(stateManager l2stateprovider.Provider, leaf protocol.Assertion) protocol.SpecEdge {
		startCommit, startErr := stateManager.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              common.Hash{},
				FromBatch:                   0,
				ToBatch:                     1,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(0)),
			},
		)
		require.NoError(t, startErr)
		req := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              common.Hash{},
			FromBatch:                   0,
			ToBatch:                     1,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(challenge_testing.LevelZeroBlockEdgeHeight)),
		}
		endCommit, endErr := stateManager.HistoryCommitment(
			ctx,
			req,
		)
		require.NoError(t, endErr)
		prefixProof, proofErr := stateManager.PrefixProof(ctx, req, l2stateprovider.Height(0))
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

	// We then check if the edge is part of a complete challenge. It should not be.
	chain := createdData.Chains[0]
	challengeParentAssertionHash, err := honestEdge.AssertionHash(ctx)
	require.NoError(t, err)
	chalComplete, err := chain.IsChallengeComplete(ctx, challengeParentAssertionHash)
	require.NoError(t, err)
	require.Equal(t, false, chalComplete)

	// Adjust well beyond a challenge period.
	for i := 0; i < 200; i++ {
		createdData.Backend.Commit()
	}

	_, err = honestEdge.ConfirmByTimer(ctx)
	require.NoError(t, err)

	// Adjust beyond the grace period.
	for i := 0; i < 10; i++ {
		createdData.Backend.Commit()
	}

	// Confirm the assertion by challenge.
	err = chain.ConfirmAssertionByChallengeWinner(
		ctx, createdData.Leaf1.Id(), honestEdge.Id(),
	)
	require.NoError(t, err)

	// Now, the edge should be part of a completed challenge.
	chalComplete, err = chain.IsChallengeComplete(ctx, challengeParentAssertionHash)
	require.NoError(t, err)
	require.Equal(t, true, chalComplete)
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

func TestLatestCreatedAssertionHashes(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]

	abi, err := rollupgen.RollupCoreMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	abiEvt := abi.Events["AssertionCreated"]
	evtID := abiEvt.ID

	// The backend is bad and sent logs in the wrong order and also
	// sent "removed" logs from a nasty reorg.
	logs := []types.Log{
		{
			BlockNumber: 120,
			Index:       0,
			Topics: []common.Hash{
				evtID,
				common.BigToHash(big.NewInt(1)),
			},
		}, {
			BlockNumber: 119,
			Index:       0,
			Topics: []common.Hash{
				evtID,
				common.BigToHash(big.NewInt(0)),
			},
		}, {
			BlockNumber: 122,
			Index:       4,
			Topics: []common.Hash{
				evtID,
				common.BigToHash(big.NewInt(-1)),
			},
			Removed: true,
		},
		{
			BlockNumber: 122,
			Index:       3,
			Topics: []common.Hash{
				evtID,
				common.BigToHash(big.NewInt(3)),
			},
		},
		{
			BlockNumber: 122,
			Index:       2,
			Topics: []common.Hash{
				evtID,
				common.BigToHash(big.NewInt(2)),
			},
		},
	}

	chain.SetBackend(&mockBackend{logs: logs})

	latest, err := chain.LatestCreatedAssertionHashes(ctx)
	require.NoError(t, err)

	// The logs received were in the wrong order, but their IDs indicate their expected position
	// in the return slice.
	require.Equal(t, 4, len(latest))
	for i, id := range latest {
		require.Equal(t, uint64(i), id.Big().Uint64())
	}
}
