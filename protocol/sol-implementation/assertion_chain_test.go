package solimpl

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateAssertion(t *testing.T) {
	ctx := context.Background()
	chain, accs, addresses, backend := setupAssertionChainWithChallengeManager(t)

	t.Run("OK", func(t *testing.T) {
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			latestBlockHash = backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  latestBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.ErrorContains(t, err, "ALREADY_STAKED")
	})
	t.Run("can create fork", func(t *testing.T) {
		chain, err := NewAssertionChain(
			ctx,
			addresses.Rollup,
			accs[2].txOpts,
			&bind.CallOpts{},
			accs[2].accountAddr,
			backend,
		)
		require.NoError(t, err)
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  common.BytesToHash([]byte("evil hash")),
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		chain.txOpts.From = accs[2].accountAddr
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)
	})
}

func TestAssertionByID(t *testing.T) {
	chain, _, _, _ := setupAssertionChainWithChallengeManager(t)

	resp, err := chain.AssertionByID(0)
	require.NoError(t, err)

	require.Equal(t, true, resp.inner.StateHash != [32]byte{})

	_, err = chain.AssertionByID(1)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAssertion_Confirm(t *testing.T) {
	ctx := context.Background()
	t.Run("OK", func(t *testing.T) {
		chain, _, _, backend := setupAssertionChainWithChallengeManager(t)

		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		assertionBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			assertionBlockHash = backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  assertionBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		err = chain.Confirm(ctx, assertionBlockHash, common.Hash{})
		require.ErrorIs(t, err, ErrTooSoon)

		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			backend.Commit()
		}
		require.NoError(t, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
		require.ErrorIs(t, ErrNoUnresolved, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
	})
}

func TestAssertion_Reject(t *testing.T) {
	ctx := context.Background()

	t.Run("Can reject assertion", func(t *testing.T) {
		t.Skip("TODO: Can't reject assertion. Blocked by one step proof")
	})

	t.Run("Already confirmed assertion", func(t *testing.T) {
		chain, _, _, backend := setupAssertionChainWithChallengeManager(t)

		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		assertionBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			assertionBlockHash = backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  assertionBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			backend.Commit()
		}
		require.NoError(t, chain.Confirm(ctx, assertionBlockHash, common.Hash{}))
		require.ErrorIs(t, ErrNoUnresolved, chain.Reject(ctx, chain.stakerAddr))
	})
}

func TestChallengePeriodSeconds(t *testing.T) {
	chain, _, _, _ := setupAssertionChainWithChallengeManager(t)
	chalPeriod, err := chain.ChallengePeriodSeconds()
	require.NoError(t, err)
	require.Equal(t, time.Second*100, chalPeriod)
}

func TestCreateSuccessionChallenge(t *testing.T) {
	ctx := context.Background()
	t.Run("assertion does not exist", func(t *testing.T) {
		chain, _, _, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge(ctx, 2)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})
	t.Run("at least two children required", func(t *testing.T) {
		chain, _, _, backend := setupAssertionChainWithChallengeManager(t)
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			latestBlockHash = backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  latestBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(ctx, 0)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})
	t.Run("assertion already rejected", func(t *testing.T) {
		t.Skip(
			"Needs a challenge manager to provide a winning claim first",
		)
	})
	t.Run("OK", func(t *testing.T) {
		chain, accs, addresses, backend := setupAssertionChainWithChallengeManager(t)
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			latestBlockHash = backend.Commit()
		}

		prevState := &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}
		postState := &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  latestBlockHash,
				SendRoot:   common.Hash{},
				Batch:      1,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		}
		prevInboxMaxCount := big.NewInt(1)
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		chain, err = NewAssertionChain(
			ctx,
			addresses.Rollup,
			accs[2].txOpts,
			&bind.CallOpts{},
			accs[2].accountAddr,
			backend,
		)
		require.NoError(t, err)

		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
			backend.Commit()
		}

		postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
		_, err = chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(ctx, 0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(ctx, 0)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func setupAssertionChainWithChallengeManager(t *testing.T) (*AssertionChain, []*testAccount, *rollupAddresses, *backends.SimulatedBackend) {
	t.Helper()
	ctx := context.Background()
	accs, backend := setupAccounts(t, 3)
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].accountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	challengePeriodSeconds := big.NewInt(100)
	miniStake := big.NewInt(1)
	cfg := generateRollupConfig(prod, wasmModuleRoot, rollupOwner, chainId, loserStakeEscrow, challengePeriodSeconds, miniStake)
	addresses := deployFullRollupStack(
		t,
		ctx,
		backend,
		accs[0].txOpts,
		common.Address{}, // Sequencer addr.
		cfg,
	)
	chain, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].txOpts,
		&bind.CallOpts{},
		accs[1].accountAddr,
		backend,
	)
	require.NoError(t, err)
	return chain, accs, addresses, backend
}
