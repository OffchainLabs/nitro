package solimpl

import (
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	StateCommitment util.StateCommitment
	chain           *AssertionChain
	id              uint64
}

func (a *Assertion) Height() uint64 {
	return a.inner().Height.Uint64()
}

func (a *Assertion) SeqNum() protocol.AssertionSequenceNumber {
	return protocol.AssertionSequenceNumber(a.id)
}

func (a *Assertion) PrevSeqNum() protocol.AssertionSequenceNumber {
	return protocol.AssertionSequenceNumber(a.inner().PrevNum)
}

func (a *Assertion) StateHash() common.Hash {
	return a.inner().StateHash
}

func (a *Assertion) inner() *rollupgen.AssertionNode {
	return nil
}

// Challenge is a developer-friendly wrapper around
// the protocol struct with the same name.
type Challenge struct {
	chain *AssertionChain
	id    [32]byte
}

// ChallengeType defines an enum of the same name
// from the goimpl.
type ChallengeType uint

const (
	BlockChallenge ChallengeType = iota
	BigStepChallenge
	SmallStepChallenge
	OneStepChallenge
)

// ChallengeVertex is a developer-friendly wrapper around
// the protocol struct with the same name.
type ChallengeVertex struct {
	chain *AssertionChain
	id    [32]byte
}
