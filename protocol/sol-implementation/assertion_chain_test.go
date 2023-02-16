package solimpl

import (
	"context"
	"math/big"
	"testing"
	"time"

	//"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	// "github.com/OffchainLabs/challenge-protocol-v2/util"
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
		created, err := chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)
		t.Logf("%+v", created)

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
		created, err := chain.CreateAssertion(
			ctx,
			height,
			prev,
			prevState,
			postState,
			prevInboxMaxCount,
		)
		require.NoError(t, err)
		t.Logf("%+v", created)
		// require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
	})
	// t.Run("previous assertion does not exist", func(t *testing.T) {
	// 	commit := util.StateCommitment{
	// 		Height:    2,
	// 		StateRoot: common.BytesToHash([]byte{2}),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 1)
	// 	require.ErrorIs(t, err, ErrPrevDoesNotExist)
	// })
	// t.Run("invalid height", func(t *testing.T) {
	// 	commit := util.StateCommitment{
	// 		Height:    0,
	// 		StateRoot: common.BytesToHash([]byte{3}),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 0)
	// 	require.ErrorIs(t, err, ErrInvalidHeight)
	// })
	// t.Run("too late to create sibling", func(t *testing.T) {
	// 	Adds two challenge periods to the chain timestamp.
	// 	err = acc.backend.AdjustTime(time.Second * 20)
	// 	require.NoError(t, err)
	// 	commit := util.StateCommitment{
	// 		Height:    1,
	// 		StateRoot: common.BytesToHash([]byte("forked")),
	// 	}
	// 	_, err = chain.CreateAssertion(commit, 0)
	// 	require.ErrorIs(t, err, ErrTooLate)
	// })
}

func TestAssertionByID(t *testing.T) {
	chain, _, _, _ := setupAssertionChainWithChallengeManager(t)

	resp, err := chain.AssertionByID(0)
	require.NoError(t, err)

	require.Equal(t, true, resp.inner.StateHash != [32]byte{})

	_, err = chain.AssertionByID(1)
	require.ErrorIs(t, err, ErrNotFound)
}

// func TestAssertion_Confirm(t *testing.T) {
// 	ctx := context.Background()
// 	acc, err := setupAccount()
// 	require.NoError(t, err)

// 	genesisStateRoot := common.BytesToHash([]byte("foo"))
// 	addr, _, _, err := challengeV2gen.DeployAssertionChain(
// 		acc.txOpts,
// 		acc.backend,
// 		genesisStateRoot,
// 		big.NewInt(10), // 10 second challenge period.
// 	)
// 	require.NoError(t, err)
// 	acc.backend.Commit()

// 	chain, err := NewAssertionChain(
// 		ctx, addr, acc.txOpts, &bind.CallOpts{}, acc.accountAddr, acc.backend,
// 	)
// 	require.NoError(t, err)

// 	commit := util.StateCommitment{
// 		Height:    1,
// 		StateRoot: common.BytesToHash([]byte{1}),
// 	}
// 	genesisId := common.Hash{}

// 	created, err := chain.CreateAssertion(commit, genesisId)
// 	require.NoError(t, err)
// 	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
// 	acc.backend.Commit()

// 	t.Run("Can confirm assertion", func(t *testing.T) {
// 		require.Equal(t, uint8(0), created.inner.Status) // Pending.
// 		require.NoError(t, created.Confirm())
// 		acc.backend.Commit()
// 		created, err = chain.AssertionByID(created.id)
// 		require.NoError(t, err)
// 		require.Equal(t, uint8(1), created.inner.Status) // Confirmed.
// 	})

// 	t.Run("Unknown assertion", func(t *testing.T) {
// 		created.id = common.BytesToHash([]byte("meow"))
// 		require.ErrorIs(t, created.Confirm(), ErrNotFound)
// 	})
// }

// func TestAssertion_Reject(t *testing.T) {
// 	acc, err := setupAccount()
// 	require.NoError(t, err)

// 	chain, _ := setupAssertionChainWithChallengeManager(t)
// 	commit := util.StateCommitment{
// 		Height:    1,
// 		StateRoot: common.BytesToHash([]byte{1}),
// 	}
// 	created, err := chain.CreateAssertion(commit, 0)
// 	require.NoError(t, err)
// 	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
// 	acc.backend.Commit()

// 	commit = util.StateCommitment{
// 		Height:    1,
// 		StateRoot: common.BytesToHash([]byte{2}),
// 	}
// 	created, err = chain.CreateAssertion(commit, 0)
// 	require.NoError(t, err)
// 	require.Equal(t, commit.StateRoot[:], created.inner.StateHash[:])
// 	acc.backend.Commit()

// 	_, err = chain.CreateSuccessionChallenge(0)
// 	require.NoError(t, err)

// 	// t.Run("Can reject assertion", func(t *testing.T) {
// 	// 	t.Skip("TODO: Can't reject assertion. Blocked by one step proof")
// 	// 	require.Equal(t, uint8(0), created.inner.Status) // Pending.
// 	// 	require.NoError(t, created.Reject())
// 	// 	acc.backend.Commit()
// 	// 	created, err = chain.AssertionByID(created.id)
// 	// 	require.NoError(t, err)
// 	// 	require.Equal(t, uint8(2), created.inner.Status) // Confirmed.
// 	// })

// 	t.Run("Unknown assertion", func(t *testing.T) {
// 		created.id = 1
// 		require.ErrorIs(t, created.Reject(), ErrNotFound)
// 	})

// 	t.Run("Already confirmed assertion", func(t *testing.T) {
// 		ga, err := chain.AssertionByID(0)
// 		require.NoError(t, err)
// 		require.ErrorIs(t, ga.Reject(), ErrNonPendingAssertion)
// 	})
// }

func TestChallengePeriodSeconds(t *testing.T) {
	chain, _, _, _ := setupAssertionChainWithChallengeManager(t)
	chalPeriod, err := chain.ChallengePeriodSeconds()
	require.NoError(t, err)
	require.Equal(t, time.Second, chalPeriod)
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
	cfg := generateRollupConfig(prod, wasmModuleRoot, rollupOwner, chainId, loserStakeEscrow)
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
