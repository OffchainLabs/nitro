package solimpl_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAssertionStateHash(t *testing.T) {
	ctx := context.Background()

	cfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)

	chain := cfg.Chains[0]
	assertion, err := chain.LatestConfirmed(ctx)
	require.NoError(t, err)

	execState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	computed := protocol.ComputeStateHash(execState, big.NewInt(1))
	stateHash, err := assertion.StateHash()
	require.NoError(t, err)
	require.Equal(t, computed, stateHash)
}

func TestCreateAssertion(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	backend := cfg.Backend

	t.Run("OK", func(t *testing.T) {

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < 100; i++ {
			latestBlockHash = backend.Commit()
		}

		prevState := &protocol.ExecutionState{
			GlobalState:   protocol.GoGlobalState{},
			MachineStatus: protocol.MachineStatusFinished,
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
		prevInboxMaxCount := big.NewInt(1)
		created, err := chain.CreateAssertion(ctx, prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)
		computed := protocol.ComputeStateHash(postState, big.NewInt(2))
		stateHash, err := created.StateHash()
		require.NoError(t, err)
		require.Equal(t, computed, stateHash, "Unequal computed hash")

		_, err = chain.CreateAssertion(ctx, prevState, postState, prevInboxMaxCount)
		require.ErrorContains(t, err, "ALREADY_STAKED")
	})
	t.Run("can create fork", func(t *testing.T) {
		assertionChain := cfg.Chains[1]

		for i := uint64(0); i < 100; i++ {
			backend.Commit()
		}

		prevState := &protocol.ExecutionState{
			GlobalState:   protocol.GoGlobalState{},
			MachineStatus: protocol.MachineStatusFinished,
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
		prevInboxMaxCount := big.NewInt(1)
		forked, err := assertionChain.CreateAssertion(ctx, prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)
		computed := protocol.ComputeStateHash(postState, big.NewInt(2))
		stateHash, err := forked.StateHash()
		require.NoError(t, err)
		require.Equal(t, computed, stateHash, "Unequal computed hash")
	})
}

func TestAssertionBySequenceNum(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]
	resp, err := chain.AssertionBySequenceNum(ctx, 1)
	require.NoError(t, err)

	stateHash, err := resp.StateHash()
	require.NoError(t, err)
	require.Equal(t, true, stateHash != [32]byte{})

	_, err = chain.AssertionBySequenceNum(ctx, 2)
	require.ErrorIs(t, err, solimpl.ErrNotFound)
}

func TestAssertion_Confirm(t *testing.T) {
	ctx := context.Background()
	t.Run("OK", func(t *testing.T) {
		cfg, err := setup.SetupChainsWithEdgeChallengeManager()
		require.NoError(t, err)

		chain := cfg.Chains[0]
		backend := cfg.Backend

		assertionBlockHash := common.Hash{}
		for i := uint64(0); i < 100; i++ {
			assertionBlockHash = backend.Commit()
		}

		prevState := &protocol.ExecutionState{
			GlobalState:   protocol.GoGlobalState{},
			MachineStatus: protocol.MachineStatusFinished,
		}
		postState := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  assertionBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(ctx, prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)

		err = chain.Confirm(ctx, assertionBlockHash, common.Hash{})
		require.ErrorIs(t, err, solimpl.ErrTooSoon)

		for i := uint64(0); i < 100; i++ {
			backend.Commit()
		}
		require.NoError(t, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
		require.ErrorIs(t, solimpl.ErrNoUnresolved, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
	})
}

func TestAssertion_Reject(t *testing.T) {
	ctx := context.Background()

	t.Run("Can reject assertion", func(t *testing.T) {
		t.Skip("TODO: Can't reject assertion. Blocked by one step proof")
	})

	t.Run("Already confirmed assertion", func(t *testing.T) {
		cfg, err := setup.SetupChainsWithEdgeChallengeManager()
		require.NoError(t, err)

		chain := cfg.Chains[0]
		backend := cfg.Backend

		assertionBlockHash := common.Hash{}
		for i := uint64(0); i < 100; i++ {
			assertionBlockHash = backend.Commit()
		}

		prevState := &protocol.ExecutionState{
			GlobalState:   protocol.GoGlobalState{},
			MachineStatus: protocol.MachineStatusFinished,
		}
		postState := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  assertionBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(ctx, prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)

		for i := uint64(0); i < 100; i++ {
			backend.Commit()
		}
		require.NoError(t, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
		require.ErrorIs(t, solimpl.ErrNoUnresolved, chain.Reject(ctx, cfg.Accounts[0].AccountAddr))
	})
}

func TestChallengePeriodBlocks(t *testing.T) {
	ctx := context.Background()
	cfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := cfg.Chains[0]

	manager, err := chain.SpecChallengeManager(ctx)
	require.NoError(t, err)

	chalPeriod, err := manager.ChallengePeriodBlocks(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(20), chalPeriod)
}
