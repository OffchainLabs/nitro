package assertionchain

import (
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

// Self invalidator is an internal interface implemented by common
// types in this package which allows them to invalidate their inner data
// by refreshing it through an on-chain lookup. This is crucial as to have
// data consistent with the chain when making calls and transactions.
type selfInvalidator interface {
	invalidate() error
}

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	StateCommitment util.StateCommitment
	chain           *AssertionChain
	id              [32]byte
	inner           outgen.Assertion
}

// Challenge is a developer-friendly wrapper around
// the protocol struct with the same name.
type Challenge struct {
	manager *ChallengeManager
	id      [32]byte
	inner   outgen.Challenge
}

// ChallengeType defines an enum of the same name
// from the protocol.
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
	inner   outgen.ChallengeVertex
}

func (a *Assertion) invalidate() error {
	inner, err := a.chain.caller.GetAssertion(a.chain.callOpts, a.id)
	if err != nil {
		return err
	}
	a.inner = inner
	return nil
}

func (a *Challenge) invalidate() error {
	inner, err := a.manager.caller.GetChallenge(a.manager.assertionChain.callOpts, a.id)
	if err != nil {
		return err
	}
	a.inner = inner
	return nil
}

func (a *ChallengeVertex) invalidate() error {
	inner, err := a.manager.caller.GetVertex(a.manager.assertionChain.callOpts, a.id)
	if err != nil {
		return err
	}
	a.inner = inner
	return nil
}
