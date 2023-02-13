package goimpl

import (
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
)

type AssertionChainEvent interface {
	IsAssertionChainEvent() bool // this method is just a marker that the type intends to be an AssertionChainEvent
}

type genericAssertionChainEvent struct{}

func (ev *genericAssertionChainEvent) IsAssertionChainEvent() bool { return true }

type CreateLeafEvent struct {
	genericAssertionChainEvent
	PrevSeqNum          AssertionSequenceNumber
	PrevStateCommitment util.StateCommitment
	SeqNum              AssertionSequenceNumber
	StateCommitment     util.StateCommitment
	Validator           common.Address
}

type ConfirmEvent struct {
	genericAssertionChainEvent
	SeqNum AssertionSequenceNumber
}

type RejectEvent struct {
	genericAssertionChainEvent
	SeqNum AssertionSequenceNumber
}

type StartChallengeEvent struct {
	genericAssertionChainEvent
	ParentSeqNum          AssertionSequenceNumber
	ParentStateCommitment util.StateCommitment
	ParentStaker          common.Address
	Validator             common.Address
}

type SetBalanceEvent struct {
	genericAssertionChainEvent
	Addr       common.Address
	OldBalance *big.Int
	NewBalance *big.Int
}

type ChallengeEvent interface {
	IsChallengeEvent() bool // this method is just a marker that the type intends to be a ChallengeEvent
	ParentStateCommitmentHash() common.Hash
	ValidatorAddress() common.Address
}

type genericChallengeEvent struct{}

func (ev *genericChallengeEvent) IsChallengeEvent() bool { return true }

type ChallengeLeafEvent struct {
	genericChallengeEvent
	ParentSeqNum      VertexSequenceNumber
	SequenceNum       VertexSequenceNumber
	WinnerIfConfirmed AssertionSequenceNumber
	ParentStateCommit util.StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	FromSequenceNum   VertexSequenceNumber // previously existing vertex
	SequenceNum       VertexSequenceNumber // newly created vertex
	ParentStateCommit util.StateCommitment
	ToHistory         util.HistoryCommitment
	FromHistory       util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	ToHistory            util.HistoryCommitment
	FromHistory          util.HistoryCommitment
	ParentStateCommit    util.StateCommitment
	DeeperSequenceNum    VertexSequenceNumber
	ShallowerSequenceNum VertexSequenceNumber
	BecomesPS            bool
	Validator            common.Address
}

func (c *ChallengeLeafEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeBisectEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeMergeEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeLeafEvent) ValidatorAddress() common.Address {
	return c.Validator
}

func (c *ChallengeBisectEvent) ValidatorAddress() common.Address {
	return c.Validator
}

func (c *ChallengeMergeEvent) ValidatorAddress() common.Address {
	return c.Validator
}
