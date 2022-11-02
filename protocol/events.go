package protocol

import "github.com/ethereum/go-ethereum/common"

type AssertionChainEvent interface {
	IsAssertionChainEvent() bool // this method is just a marker that the type intends to be an AssertionChainEvent
}

type genericAssertionChainEvent struct{}

func (ev *genericAssertionChainEvent) IsAssertionChainEvent() bool { return true }

type CreateLeafEvent struct {
	genericAssertionChainEvent
	prevSeqNum uint64
	seqNum     uint64
	commitment StateCommitment
	staker     common.Address
}

type ConfirmEvent struct {
	genericAssertionChainEvent
	seqNum uint64
}

type RejectEvent struct {
	genericAssertionChainEvent
	seqNum uint64
}

type StartChallengeEvent struct {
	genericAssertionChainEvent
	parentSeqNum uint64
}

type ChallengeEvent interface {
	IsChallengeEvent() bool // this method is just a marker that the type intends to be a ChallengeEvent
}

type genericChallengeEvent struct{}

func (ev *genericChallengeEvent) IsChallengeEvent() bool { return true }

type ChallengeLeafEvent struct {
	genericChallengeEvent
	sequenceNum       uint64
	winnerIfConfirmed uint64
	history           HistoryCommitment
	becomesPS         bool
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	fromSequenceNum uint64 // previously existing vertex
	sequenceNum     uint64 // newly created vertex
	history         HistoryCommitment
	becomesPS       bool
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	deeperSequenceNum    uint64
	shallowerSequenceNum uint64
	becomesPS            bool
}
