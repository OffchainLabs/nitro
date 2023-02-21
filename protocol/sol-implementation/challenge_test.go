package solimpl

import (
	"context"
	"math/big"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallenge_BlockChallenge_AddLeaf(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(1)
	height2 := uint64(1)
	a1, _, challenge, chain1, _ := setupTopLevelFork(t, ctx, height1, height2)
	t.Run("claim predecessor not linked to challenge", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			&Assertion{
				chain: chain1,
				id:    20,
				inner: rollupgen.AssertionNode{
					Height: big.NewInt(1),
				},
			},
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("bar")),
			},
		)
		require.ErrorContains(t, err, "INVALID_ASSERTION_NUM")
	})
	t.Run("invalid height", func(t *testing.T) {
		// Pass in a junk assertion that has no predecessor.
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			&Assertion{
				chain: chain1,
				id:    1,
				inner: rollupgen.AssertionNode{
					Height: big.NewInt(0),
				},
			},
			util.HistoryCommitment{
				Height: 0,
				Merkle: common.BytesToHash([]byte("bar")),
			},
		)
		require.ErrorContains(t, err, "Invalid height")
	})
	t.Run("last state is not assertion claim block hash", func(t *testing.T) {
		t.Skip("Needs proofs implemented in solidity")
	})
	t.Run("empty history commitment", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.Hash{},
			},
		)
		require.ErrorContains(t, err, "Empty historyRoot")
	})
	t.Run("winner already declared", func(t *testing.T) {
		t.Skip("Needs winner declaration logic implemented in solidity")
	})
	t.Run("last state not in history", func(t *testing.T) {
		t.Skip()
	})
	t.Run("first state not in history", func(t *testing.T) {
		t.Skip()
	})
	t.Run("OK", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height1,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.NoError(t, err)
	})
	t.Run("already exists", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			a1,
			util.HistoryCommitment{
				Height: height2,
				Merkle: common.BytesToHash([]byte("nyan")),
			},
		)
		require.ErrorContains(t, err, "already exists")
	})
}

func setupTopLevelFork(
	t *testing.T,
	ctx context.Context,
	height1,
	height2 uint64,
) (*Assertion, *Assertion, *Challenge, *AssertionChain, *AssertionChain) {
	t.Helper()
	chain1, accs, addresses, backend := setupAssertionChainWithChallengeManager(t)
	prev := uint64(0)

	minAssertionPeriod, err := chain1.userLogic.MinimumAssertionPeriod(chain1.callOpts)
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
	a1, err := chain1.CreateAssertion(
		ctx,
		height1,
		prev,
		prevState,
		postState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	chain2, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].txOpts,
		&bind.CallOpts{},
		accs[2].accountAddr,
		backend,
	)
	require.NoError(t, err)

	postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
	a2, err := chain2.CreateAssertion(
		ctx,
		height2,
		prev,
		prevState,
		postState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	challenge, err := chain2.CreateSuccessionChallenge(ctx, 0, common.Hash{})
	require.NoError(t, err)
	return a1, a2, challenge, chain1, chain2
}
