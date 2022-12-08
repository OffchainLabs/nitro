package protocol

import (
	"math/big"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
)

type AssertionChainEvent interface {
	IsAssertionChainEvent() bool // this method is just a marker that the type intends to be an AssertionChainEvent
}

type genericAssertionChainEvent struct{}

func (ev *genericAssertionChainEvent) IsAssertionChainEvent() bool { return true }

type CreateLeafEvent struct {
	genericAssertionChainEvent
	PrevSeqNum          SequenceNum
	PrevStateCommitment StateCommitment
	SeqNum              SequenceNum
	StateCommitment     StateCommitment
	Validator           common.Address
}

type ConfirmEvent struct {
	genericAssertionChainEvent
	SeqNum SequenceNum
}

type RejectEvent struct {
	genericAssertionChainEvent
	SeqNum SequenceNum
}

type StartChallengeEvent struct {
	genericAssertionChainEvent
	ParentSeqNum          SequenceNum
	ParentStateCommitment StateCommitment
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
	ParentSeqNum      SequenceNum
	SequenceNum       SequenceNum
	WinnerIfConfirmed SequenceNum
	ParentStateCommit StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	FromSequenceNum   SequenceNum // previously existing vertex
	SequenceNum       SequenceNum // newly created vertex
	ParentStateCommit StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	History              util.HistoryCommitment
	ParentStateCommit    StateCommitment
	DeeperSequenceNum    SequenceNum
	ShallowerSequenceNum SequenceNum
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
