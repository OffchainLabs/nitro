package solimpl

import (
	"context"
	"math/big"
	"testing"

	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var _ = protocol.Challenge(&Challenge{})

func TestChallenge_BlockChallenge_AddLeaf(t *testing.T) {
	ctx := context.Background()
	tx := &activeTx{readWriteTx: true}
	genesisHeight := uint64(0)
	height1 := uint64(4)
	height2 := uint64(4)
	heightDiff := height1 - genesisHeight
	a1, _, challenge, chain1, _ := setupTopLevelFork(t, ctx, height1, height2)
	t.Run("claim predecessor not linked to challenge", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			&Assertion{
				chain: chain1,
				id:    20,
				inner: rollupgen.AssertionNode{
					Height: big.NewInt(1),
				},
			},
			util.HistoryCommitment{
				Height: heightDiff,
				Merkle: common.BytesToHash([]byte("bar")),
			},
		)
		require.ErrorContains(t, err, "INVALID_ASSERTION_NUM")
	})
	t.Run("invalid height", func(t *testing.T) {
		// Pass in a junk assertion that has no predecessor.
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
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
		require.ErrorContains(t, err, "Invalid leaf height")
	})
	t.Run("last state is not assertion claim block hash", func(t *testing.T) {
		t.Skip("Needs proofs implemented in solidity")
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
	leaves := make([]common.Hash, 4)
	for i := range leaves {
		leaves[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	history, err := util.NewHistoryCommitment(heightDiff, leaves)
	require.NoError(t, err)
	t.Run("OK", func(t *testing.T) {
		_, err = challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			history,
		)
		require.NoError(t, err)

		v, err := challenge.RootVertex(ctx, tx)
		require.NoError(t, err)
		want, err := challenge.manager.GetVertex(ctx, tx, v.Id())
		require.NoError(t, err)
		require.Equal(t, want.Unwrap(), v)
	})
	t.Run("already exists", func(t *testing.T) {
		_, err := challenge.AddBlockChallengeLeaf(
			ctx,
			tx,
			a1,
			history,
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
	tx := &activeTx{readWriteTx: true}
	chain1, accs, addresses, backend, headerReader := setupAssertionChainWithChallengeManager(t)
	prev := uint64(0)

	minAssertionPeriod, err := chain1.userLogic.MinimumAssertionPeriod(chain1.callOpts)
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
	a1, err := chain1.CreateAssertion(
		ctx,
		tx,
		height1,
		protocol.AssertionSequenceNumber(prev),
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
		headerReader,
	)
	require.NoError(t, err)

	postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
	a2, err := chain2.CreateAssertion(
		ctx,
		tx,
		height2,
		protocol.AssertionSequenceNumber(prev),
		prevState,
		postState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	challenge, err := chain2.CreateSuccessionChallenge(ctx, tx, 0)
	require.NoError(t, err)
	return a1.(*Assertion), a2.(*Assertion), challenge.(*Challenge), chain1, chain2
}
