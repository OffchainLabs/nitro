package protocol

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var _ = OnChainProtocol(&AssertionChain{})

const testChallengePeriod = 100 * time.Second

func TestAssertionChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	require.Equal(t, 1, len(chain.assertions))
	require.Equal(t, uint64(0), chain.confirmedLatest)
	var eventChan chan AssertionChainEvent
	err := chain.Tx(func(tx *ActiveTx, chain *AssertionChain) error {
		genesis := chain.LatestConfirmed(tx)
		require.Equal(t, StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		}, genesis.StateCommitment)

		bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
		chain.SetBalance(tx, staker1, bigBalance)
		chain.SetBalance(tx, staker2, bigBalance)

		eventChan := make(chan AssertionChainEvent)
		chain.feed.SubscribeWithFilter(ctx, eventChan, func(ev AssertionChainEvent) bool {
			switch ev.(type) {
			case *SetBalanceEvent:
				return false
			default:
				return true
			}
		})

		// add an assertion, then confirm it
		comm := StateCommitment{Height: 1, StateRoot: correctBlockHashes[99]}
		newAssertion, err := chain.CreateLeaf(tx, genesis, comm, staker1)
		require.NoError(t, err)
		require.Equal(t, 2, len(chain.assertions))
		require.Equal(t, genesis, chain.LatestConfirmed(tx))
		verifyCreateLeafEventInFeed(t, eventChan, 1, 0, staker1, comm)

		err = newAssertion.ConfirmNoRival(tx)
		require.ErrorIs(t, err, ErrNotYet)
		timeRef.Add(testChallengePeriod + time.Second)
		require.NoError(t, newAssertion.ConfirmNoRival(tx))

		require.Equal(t, newAssertion, chain.LatestConfirmed(tx))
		require.Equal(t, ConfirmedAssertionState, int(newAssertion.status))
		verifyConfirmEventInFeed(t, eventChan, 1)

		// try to create a duplicate assertion
		_, err = chain.CreateLeaf(tx, genesis, StateCommitment{1, correctBlockHashes[99]}, staker1)
		require.ErrorIs(t, err, ErrVertexAlreadyExists)

		// create a fork, let first branch win by timeout
		comm = StateCommitment{2, correctBlockHashes[199]}
		branch1, err := chain.CreateLeaf(tx, newAssertion, comm, staker1)
		require.NoError(t, err)
		timeRef.Add(5 * time.Second)
		verifyCreateLeafEventInFeed(t, eventChan, 2, 1, staker1, comm)
		comm = StateCommitment{2, wrongBlockHashes[199]}
		branch2, err := chain.CreateLeaf(tx, newAssertion, comm, staker2)
		require.NoError(t, err)
		verifyCreateLeafEventInFeed(t, eventChan, 3, 1, staker2, comm)
		challenge, err := newAssertion.CreateChallenge(tx, ctx)
		require.NoError(t, err)
		verifyStartChallengeEventInFeed(t, eventChan, newAssertion.SequenceNum)
		chal1, err := challenge.AddLeaf(tx, branch1, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(correctBlockHashes[99:200]).Root()})
		require.NoError(t, err)
		_, err = challenge.AddLeaf(tx, branch2, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(wrongBlockHashes[99:200]).Root()})
		require.NoError(t, err)
		err = chal1.ConfirmForPsTimer(tx)
		require.ErrorIs(t, err, ErrNotYet)

		timeRef.Add(testChallengePeriod)
		require.NoError(t, chal1.ConfirmForPsTimer(tx))
		require.NoError(t, branch1.ConfirmForWin(tx))
		require.Equal(t, branch1, chain.LatestConfirmed(tx))

		verifyConfirmEventInFeed(t, eventChan, 2)
		require.NoError(t, branch2.RejectForLoss(tx))
		verifyRejectEventInFeed(t, eventChan, 3)
		return nil
	})
	require.NoError(t, err)

	// verify that feed is empty
	time.Sleep(500 * time.Millisecond)
	select {
	case ev := <-eventChan:
		t.Fatal(ev)
	default:
	}
}

func TestAssertionChain_LeafCreationThroughDiffStakers(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)

	require.NoError(t, chain.Tx(func(tx *ActiveTx, chian *AssertionChain) error {
		oldStaker := common.BytesToAddress([]byte{1})
		staker := common.BytesToAddress([]byte{2})
		require.Equal(t, chain.GetBalance(tx, oldStaker), big.NewInt(0)) // Old staker has 0 because it's already staked.
		chain.SetBalance(tx, staker, AssertionStakeWei)
		require.Equal(t, chain.GetBalance(tx, staker), AssertionStakeWei) // New staker has full balance because it's not yet staked.

		lc := chain.LatestConfirmed(tx)
		lc.Staker = util.FullOption[common.Address](oldStaker)
		_, err := chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.NoError(t, err)

		require.Equal(t, chain.GetBalance(tx, staker), big.NewInt(0))        // New staker has 0 balance after staking.
		require.Equal(t, chain.GetBalance(tx, oldStaker), AssertionStakeWei) // Old staker has full balance after unstaking.
		return nil
	}))
}

func TestAssertionChain_LeafCreationsInsufficientStakes(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)

	require.NoError(t, chain.Tx(func(tx *ActiveTx, chain *AssertionChain) error {
		lc := chain.LatestConfirmed(tx)
		staker := common.BytesToAddress([]byte{1})
		lc.Staker = util.EmptyOption[common.Address]()
		_, err := chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.ErrorIs(t, err, ErrInsufficientBalance)

		diffStaker := common.BytesToAddress([]byte{2})
		lc.Staker = util.FullOption[common.Address](diffStaker)
		_, err = chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.ErrorIs(t, err, ErrInsufficientBalance)
		return nil
	}))
}

func verifyCreateLeafEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum, prevSeqNum uint64, staker common.Address, comm StateCommitment) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *CreateLeafEvent:
		leaf := e.Leaf
		prev := leaf.Prev.OpenKnownFull()
		leafStaker := leaf.Staker.OpenKnownFull()
		if leaf.SequenceNum != seqNum || prev.SequenceNum != prevSeqNum || leafStaker != staker || leaf.StateCommitment != comm {
			t.Fatal(e)
		}
	default:
		t.Fatal(e)
	}
}

func verifyConfirmEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *ConfirmEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyRejectEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *RejectEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyStartChallengeEventInFeed(t *testing.T, c <-chan AssertionChainEvent, parentSeqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *StartChallengeEvent:
		require.Equal(t, parentSeqNum, e.ChallengedAssertion.SequenceNum)
	default:
		t.Fatal()
	}
}

func TestBisectionChallengeGame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(10)
	wrongBlockHashes := wrongBlockHashesForTest(10)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod)

	err := chain.Tx(func(tx *ActiveTx, chain *AssertionChain) error {
		// We create a fork with genesis as the parent, where one branch is a higher depth than the other.
		genesis := chain.LatestConfirmed(tx)
		bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
		chain.SetBalance(tx, staker1, bigBalance)
		chain.SetBalance(tx, staker2, bigBalance)

		correctBranch, err := chain.CreateLeaf(tx, genesis, StateCommitment{6, correctBlockHashes[6]}, staker1)
		require.NoError(t, err)
		wrongBranch, err := chain.CreateLeaf(tx, genesis, StateCommitment{7, wrongBlockHashes[7]}, staker2)
		require.NoError(t, err)

		challenge, err := genesis.CreateChallenge(tx, ctx)
		require.NoError(t, err)

		// Add some leaves to the mix...
		expectedBisectionHeight := uint64(4)
		lo := expectedBisectionHeight
		hi := uint64(7)
		loExp := util.ExpansionFromLeaves(wrongBlockHashes[:lo])
		hiExp := util.ExpansionFromLeaves(wrongBlockHashes[:hi])

		cl1, err := challenge.AddLeaf(
			tx,
			wrongBranch,
			util.HistoryCommitment{
				Height: 6,
				Merkle: util.ExpansionFromLeaves(correctBlockHashes[:7]).Root(),
			},
		)
		require.NoError(t, err)
		cl2, err := challenge.AddLeaf(
			tx,
			correctBranch,
			util.HistoryCommitment{
				Height: 7,
				Merkle: hiExp.Root(),
			},
		)
		require.NoError(t, err)

		// Ensure the lower height challenge vertex is the ps.
		require.Equal(t, true, cl1.isPresumptiveSuccessor())
		require.Equal(t, false, cl2.isPresumptiveSuccessor())

		// Next, only the vertex that is not the presumptive successor can start a bisection move.
		bisectionHeight, err := cl2.requiredBisectionHeight()
		require.NoError(t, err)
		require.Equal(t, expectedBisectionHeight, bisectionHeight)

		proof := util.GeneratePrefixProof(lo, loExp, correctBlockHashes[lo:6])
		_, err = cl1.Bisect(
			tx,
			util.HistoryCommitment{
				Height: lo,
				Merkle: loExp.Root(),
			},
			proof,
		)
		require.ErrorIs(t, err, ErrWrongState)

		// Generate a prefix proof for the associated history commitments from the bisection
		// height up to the height of the state commitment for the non-presumptive challenge leaf.
		proof = util.GeneratePrefixProof(lo, loExp, wrongBlockHashes[lo:hi])
		bisection, err := cl2.Bisect(
			tx,
			util.HistoryCommitment{
				Height: lo,
				Merkle: loExp.Root(),
			},
			proof,
		)
		require.NoError(t, err)

		// The parent of the bisectoin should be the root of this challenge and the bisection
		// should be the new presumptive successor.
		require.Equal(t, challenge.root.commitment.Merkle, bisection.prev.commitment.Merkle)
		require.Equal(t, true, bisection.prev.isPresumptiveSuccessor())
		return nil
	})

	require.NoError(t, err)
}

func correctBlockHashesForTest(numBlocks uint64) []common.Hash {
	ret := []common.Hash{}
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(i))
	}
	return ret
}

func wrongBlockHashesForTest(numBlocks uint64) []common.Hash {
	ret := []common.Hash{}
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(71285937102384-i))
	}
	return ret
}

func TestAssertionChain_StakerInsufficientBalance(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	require.Equal(t, chain.DeductFromBalance(
		&ActiveTx{txStatus: readWriteTxStatus},
		common.BytesToAddress([]byte{1}),
		AssertionStakeWei,
	), ErrInsufficientBalance)
}

func TestAssertionChain_ChallengePeriodLength(t *testing.T) {
	ctx := context.Background()
	cp := 123 * time.Second
	tx := &ActiveTx{txStatus: readOnlyTxStatus}
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), cp)
	require.Equal(t, chain.ChallengePeriodLength(tx), cp)
}

func TestAssertionChain_LeafCreationErrors(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	badChain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod+1)
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	lc := chain.LatestConfirmed(tx)
	_, err := badChain.CreateLeaf(tx, lc, StateCommitment{}, common.BytesToAddress([]byte{}))
	require.ErrorIs(t, err, ErrWrongChain)
	_, err = chain.CreateLeaf(tx, lc, StateCommitment{}, common.BytesToAddress([]byte{}))
	require.ErrorIs(t, err, ErrInvalid)
}

func TestAssertion_ErrWrongState(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	a := chain.LatestConfirmed(tx)
	require.ErrorIs(t, a.RejectForPrev(tx), ErrWrongState)
	require.ErrorIs(t, a.RejectForLoss(tx), ErrWrongState)
	require.ErrorIs(t, a.ConfirmForWin(tx), ErrWrongState)
}

func TestAssertion_ErrWrongPredecessorState(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	newA, err := chain.CreateLeaf(tx, chain.LatestConfirmed(tx), StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	require.ErrorIs(t, newA.RejectForPrev(tx), ErrWrongPredecessorState)
	require.ErrorIs(t, newA.ConfirmForWin(tx), ErrWrongPredecessorState)
}

func TestAssertion_ErrNotYet(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	newA, err := chain.CreateLeaf(tx, chain.LatestConfirmed(tx), StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	require.ErrorIs(t, newA.ConfirmNoRival(tx), ErrNotYet)
}
