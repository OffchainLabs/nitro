package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
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
	if len(chain.assertions) != 1 {
		t.Fatal()
	}
	if chain.confirmedLatest != 0 {
		t.Fatal()
	}
	genesis := chain.LatestConfirmed()
	if genesis.stateCommitment != (util.HistoryCommitment{Height: 0, Merkle: common.Hash{}}) {
		t.Fatal()
	}

	eventChan := chain.feed.Subscribe(ctx)

	// add an assertion, then confirm it
	comm := util.HistoryCommitment{Height: 1, Merkle: correctBlockHashes[99]}
	newAssertion, err := chain.CreateLeaf(genesis, comm, staker1)
	Require(t, err)
	if len(chain.assertions) != 2 {
		t.Fatal()
	}
	if chain.LatestConfirmed() != genesis {
		t.Fatal()
	}
	verifyCreateLeafEventInFeed(t, eventChan, 1, 0, staker1, comm)

	if err := newAssertion.ConfirmNoRival(); !errors.Is(err, ErrNotYet) {
		t.Fatal(err)
	}
	timeRef.Add(testChallengePeriod + time.Second)
	Require(t, newAssertion.ConfirmNoRival())
	if chain.LatestConfirmed() != newAssertion {
		t.Fatal()
	}
	if newAssertion.status != ConfirmedAssertionState {
		t.Fatal(newAssertion.status)
	}
	verifyConfirmEventInFeed(t, eventChan, 1)

	// try to create a duplicate assertion
	_, err = chain.CreateLeaf(genesis, util.HistoryCommitment{Height: 1, Merkle: correctBlockHashes[99]}, staker1)
	if !errors.Is(err, ErrVertexAlreadyExists) {
		t.Fatal(err)
	}

	// create a fork, let first branch win by timeout
	comm = util.HistoryCommitment{2, correctBlockHashes[199]}
	branch1, err := chain.CreateLeaf(newAssertion, comm, staker1)
	Require(t, err)
	timeRef.Add(5 * time.Second)
	verifyCreateLeafEventInFeed(t, eventChan, 2, 1, staker1, comm)
	comm = util.HistoryCommitment{2, wrongBlockHashes[199]}
	branch2, err := chain.CreateLeaf(newAssertion, comm, staker2)
	verifyCreateLeafEventInFeed(t, eventChan, 3, 1, staker2, comm)
	Require(t, err)
	challenge, err := newAssertion.CreateChallenge(ctx)
	Require(t, err)
	verifyStartChallengeEventInFeed(t, eventChan, newAssertion.sequenceNum)
	chal1, err := challenge.AddLeaf(branch1, util.HistoryCommitment{100, util.ExpansionFromLeaves(correctBlockHashes[99:200]).Root()})
	Require(t, err)
	_, err = challenge.AddLeaf(branch2, util.HistoryCommitment{100, util.ExpansionFromLeaves(wrongBlockHashes[99:200]).Root()})
	Require(t, err)
	err = chal1.ConfirmForPsTimer()
	if !errors.Is(err, ErrNotYet) {
		t.Fatal(err)
	}
	timeRef.Add(testChallengePeriod)
	Require(t, chal1.ConfirmForPsTimer())
	Require(t, branch1.ConfirmForWin())
	if chain.LatestConfirmed() != branch1 {
		t.Fatal()
	}
	verifyConfirmEventInFeed(t, eventChan, 2)
	Require(t, branch2.RejectForLoss())
	verifyRejectEventInFeed(t, eventChan, 3)

	// verify that feed is empty
	time.Sleep(500 * time.Millisecond)
	select {
	case ev := <-eventChan:
		t.Fatal(ev)
	default:
	}
}

func verifyCreateLeafEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum, prevSeqNum uint64, staker common.Address, comm util.HistoryCommitment) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *CreateLeafEvent:
		if e.seqNum != seqNum || e.prevSeqNum != prevSeqNum || e.staker != staker || e.commitment != comm {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyConfirmEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *ConfirmEvent:
		if e.seqNum != seqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyRejectEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *RejectEvent:
		if e.seqNum != seqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyStartChallengeEventInFeed(t *testing.T, c <-chan AssertionChainEvent, parentSeqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *StartChallengeEvent:
		if e.parentSeqNum != parentSeqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func TestChallengeBisections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	correctBranch, err := chain.CreateLeaf(chain.LatestConfirmed(), util.HistoryCommitment{100, correctBlockHashes[100]}, staker1)
	Require(t, err)
	wrongBranch, err := chain.CreateLeaf(chain.LatestConfirmed(), util.HistoryCommitment{100, wrongBlockHashes[100]}, staker2)
	Require(t, err)
	challenge, err := chain.LatestConfirmed().CreateChallenge(ctx)
	Require(t, err)
	correctLeaf, err := challenge.AddLeaf(correctBranch, util.HistoryCommitment{100, util.ExpansionFromLeaves(correctBlockHashes[101:200]).Root()})
	Require(t, err)
	wrongLeaf, err := challenge.AddLeaf(wrongBranch, util.HistoryCommitment{100, util.ExpansionFromLeaves(wrongBlockHashes[101:200]).Root()})
	Require(t, err)

	_ = correctLeaf
	_ = wrongLeaf
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

func Require(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
