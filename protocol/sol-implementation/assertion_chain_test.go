package solimpl_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	_ = protocol.AssertionChain(&solimpl.AssertionChain{})
	_ = protocol.Assertion(&solimpl.Assertion{})
)

func TestAssertionStateHash(t *testing.T) {
	ctx := context.Background()
	setupCfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := setupCfg.Chains[0]

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
	setupCfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := setupCfg.Chains[0]
	accs := setupCfg.Accounts
	addresses := setupCfg.Addrs
	backend := setupCfg.Backend
	headerReader := setupCfg.L1Reader

	t.Run("OK", func(t *testing.T) {
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
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
		created, err := chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)
		computed := protocol.ComputeStateHash(postState, big.NewInt(2))
		stateHash, err := created.StateHash()
		require.NoError(t, err)
		require.Equal(t, computed, stateHash, "Unequal computed hash")

		_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
		require.ErrorContains(t, err, "ALREADY_STAKED")
	})
	t.Run("can create fork", func(t *testing.T) {
		chain, err := solimpl.NewAssertionChain(
			ctx,
			addresses.Rollup,
			accs[2].TxOpts,
			&bind.CallOpts{},
			accs[2].AccountAddr,
			backend,
			headerReader,
			common.Address{},
		)
		require.NoError(t, err)
		height := uint64(1)
		prev := uint64(0)

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
		chain.TxOpts.From = accs[2].AccountAddr
		forked, err := chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)
		computed := protocol.ComputeStateHash(postState, big.NewInt(2))
		stateHash, err := forked.StateHash()
		require.NoError(t, err)
		require.Equal(t, computed, stateHash, "Unequal computed hash")
	})
}

func TestAssertionBySequenceNum(t *testing.T) {
	ctx := context.Background()
	setupCfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := setupCfg.Chains[0]

	resp, err := chain.AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	stateHash, err := resp.StateHash()
	require.NoError(t, err)
	require.Equal(t, true, stateHash != [32]byte{})

	_, err = chain.AssertionBySequenceNum(ctx, 1)
	require.ErrorIs(t, err, solimpl.ErrNotFound)
}

func TestBlockChallenge(t *testing.T) {
	ctx := context.Background()
	setupCfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chain := setupCfg.Chains[0]
	accs := setupCfg.Accounts
	addresses := setupCfg.Addrs
	backend := setupCfg.Backend
	headerReader := setupCfg.L1Reader
	height := uint64(1)
	prev := uint64(0)

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
	_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
	require.NoError(t, err)

	chain, err = solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].TxOpts,
		&bind.CallOpts{},
		accs[2].AccountAddr,
		backend,
		headerReader,
		common.Address{},
	)
	require.NoError(t, err)

	postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
	_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
	require.NoError(t, err)

	_, err = chain.BlockChallenge(ctx, 0)
	require.ErrorContains(t, err, "execution reverted: Challenge does not exist")

	createdChallenge, err := chain.CreateSuccessionChallenge(ctx, 0)
	require.NoError(t, err)

	challenge, err := chain.BlockChallenge(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, createdChallenge.Id(), challenge.Id())
}

func TestAssertion_Confirm(t *testing.T) {
	ctx := context.Background()
	t.Run("OK", func(t *testing.T) {
		chain, _, _, backend, _ := setupAssertionChainWithChallengeManager(t)

		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		assertionBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
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
		_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
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

func TestChallengePeriodSeconds(t *testing.T) {
	ctx := context.Background()
	chain, _, _, _, _ := setupAssertionChainWithChallengeManager(t)
	manager, err := chain.CurrentChallengeManager(ctx)
	require.NoError(t, err)

	chalPeriod, err := manager.ChallengePeriodSeconds(ctx)
	require.NoError(t, err)
	require.Equal(t, time.Second*100, chalPeriod)
}

func TestCreateSuccessionChallenge(t *testing.T) {
	ctx := context.Background()
	t.Run("assertion does not exist", func(t *testing.T) {
		chain, _, _, _, _ := setupAssertionChainWithChallengeManager(t)
		_, err := chain.CreateSuccessionChallenge(ctx, 2)
		require.ErrorIs(t, err, ErrInvalidChildren)
	})
	t.Run("at least two children required", func(t *testing.T) {
		chain, _, _, backend, _ := setupAssertionChainWithChallengeManager(t)
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
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
		_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
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
		chain, accs, addresses, backend, headerReader := setupAssertionChainWithChallengeManager(t)
		height := uint64(1)
		prev := uint64(0)
		minAssertionPeriod, err := chain.userLogic.MinimumAssertionPeriod(chain.callOpts)
		require.NoError(t, err)

		latestBlockHash := common.Hash{}
		for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
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
		_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)

		chain, err = NewAssertionChain(
			ctx,
			addresses.Rollup,
			accs[2].txOpts,
			&bind.CallOpts{},
			accs[2].accountAddr,
			backend,
			headerReader,
			common.Address{},
		)
		require.NoError(t, err)

		postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
		_, err = chain.CreateAssertion(ctx, height, protocol.AssertionSequenceNumber(prev), prevState, postState, prevInboxMaxCount)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(ctx, 0)
		require.NoError(t, err)

		_, err = chain.CreateSuccessionChallenge(ctx, 0)
		require.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func TestCopyTxOpts(t *testing.T) {
	a := &bind.TransactOpts{
		From:      common.BigToAddress(big.NewInt(1)),
		Nonce:     big.NewInt(2),
		Value:     big.NewInt(3),
		GasPrice:  big.NewInt(4),
		GasFeeCap: big.NewInt(5),
		GasTipCap: big.NewInt(6),
		GasLimit:  7,
		Context:   context.TODO(),
		NoSend:    false,
	}

	b := copyTxOpts(a)

	require.Equal(t, a.From, b.From)
	require.Equal(t, a.Nonce, b.Nonce)
	require.Equal(t, a.Value, b.Value)
	require.Equal(t, a.GasPrice, b.GasPrice)
	require.Equal(t, a.GasFeeCap, b.GasFeeCap)
	require.Equal(t, a.GasTipCap, b.GasTipCap)
	require.Equal(t, a.GasLimit, b.GasLimit)
	require.Equal(t, a.Context, b.Context)
	require.Equal(t, a.NoSend, b.NoSend)

	// Make changes like SetBytes which modify the underlying values.

	b.From.SetBytes([]byte("foobar"))
	b.Nonce.SetBytes([]byte("foobar"))
	b.Value.SetBytes([]byte("foobar"))
	b.GasPrice.SetBytes([]byte("foobar"))
	b.GasFeeCap.SetBytes([]byte("foobar"))
	b.GasTipCap.SetBytes([]byte("foobar"))
	b.GasLimit = 123456789
	type foo string // custom type for linter.
	b.Context = context.WithValue(context.TODO(), foo("bar"), foo("baz"))
	b.NoSend = true

	// Everything should be different.
	// Note: signer is not evaluated because function comparison is not possible.
	require.NotEqual(t, a.From, b.From)
	require.NotEqual(t, a.Nonce, b.Nonce)
	require.NotEqual(t, a.Value, b.Value)
	require.NotEqual(t, a.GasPrice, b.GasPrice)
	require.NotEqual(t, a.GasFeeCap, b.GasFeeCap)
	require.NotEqual(t, a.GasTipCap, b.GasTipCap)
	require.NotEqual(t, a.GasLimit, b.GasLimit)
	require.NotEqual(t, a.Context, b.Context)
	require.NotEqual(t, a.NoSend, b.NoSend)
}
