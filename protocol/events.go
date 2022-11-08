package protocol

import (
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
	PrevSeqNum uint64
	SeqNum     uint64
	Commitment util.HistoryCommitment
	Staker     common.Address
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
	ParentSeqNum uint64
}

type ChallengeEvent interface {
	IsChallengeEvent() bool // this method is just a marker that the type intends to be a ChallengeEvent
}

type genericChallengeEvent struct{}

func (ev *genericChallengeEvent) IsChallengeEvent() bool { return true }

type ChallengeLeafEvent struct {
	genericChallengeEvent
	SequenceNum       uint64
	WinnerIfConfirmed uint64
	History           util.HistoryCommitment
	BecomesPS         bool
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	FromSequenceNum uint64 // previously existing vertex
	SequenceNum     uint64 // newly created vertex
	History         util.HistoryCommitment
	BecomesPS       bool
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	DeeperSequenceNum    uint64
	ShallowerSequenceNum uint64
	BecomesPS            bool
}
