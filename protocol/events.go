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
	PrevSeqNum          uint64
	PrevStateCommitment StateCommitment
	SeqNum              uint64
	StateCommitment     StateCommitment
	Staker              common.Address
}

type ConfirmEvent struct {
	genericAssertionChainEvent
	SeqNum uint64
}

type RejectEvent struct {
	genericAssertionChainEvent
	SeqNum uint64
}

type StartChallengeEvent struct {
	genericAssertionChainEvent
	ParentSeqNum          uint64
	ParentStateCommitment StateCommitment
	ParentStaker          common.Address
	Challenger            common.Address
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
	ActorAddress() common.Address
}

type genericChallengeEvent struct{}

func (ev *genericChallengeEvent) IsChallengeEvent() bool { return true }

type ChallengeLeafEvent struct {
	genericChallengeEvent
	ParentSeqNum      uint64
	SequenceNum       uint64
	WinnerIfConfirmed uint64
	ParentStateCommit StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Actor             common.Address
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	FromSequenceNum   uint64 // previously existing vertex
	SequenceNum       uint64 // newly created vertex
	ParentStateCommit StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Actor             common.Address
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	History              util.HistoryCommitment
	ParentStateCommit    StateCommitment
	DeeperSequenceNum    uint64
	ShallowerSequenceNum uint64
	BecomesPS            bool
	Actor                common.Address
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

func (c *ChallengeLeafEvent) ActorAddress() common.Address {
	return c.Actor
}

func (c *ChallengeBisectEvent) ActorAddress() common.Address {
	return c.Actor
}

func (c *ChallengeMergeEvent) ActorAddress() common.Address {
	return c.Actor
}
