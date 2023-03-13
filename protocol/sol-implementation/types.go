package solimpl

import (
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	StateCommitment    util.StateCommitment
	chain              *AssertionChain
	id                 uint64
	createdAtBlockHash common.Hash
	inner              rollupgen.AssertionNode
}

func (a *Assertion) Height() uint64 {
	return a.inner.Height.Uint64()
}

func (a *Assertion) SeqNum() protocol.AssertionSequenceNumber {
	return protocol.AssertionSequenceNumber(a.id)
}

func (a *Assertion) PrevSeqNum() protocol.AssertionSequenceNumber {
	return protocol.AssertionSequenceNumber(a.inner.PrevNum)
}

func (a *Assertion) StateHash() common.Hash {
	return a.inner.StateHash
}

func (a *Assertion) BlockHash() common.Hash {
	return a.createdAtBlockHash
}

// Challenge is a developer-friendly wrapper around
// the protocol struct with the same name.
type Challenge struct {
	manager *ChallengeManager
	id      [32]byte
	inner   challengeV2gen.Challenge
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
	manager *ChallengeManager
	id      [32]byte
	inner   challengeV2gen.ChallengeVertex
}
