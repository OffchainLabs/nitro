package solimpl

import (
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"sync"
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
	lock            sync.Mutex
}

// Challenge is a developer-friendly wrapper around
// the protocol struct with the same name.
type Challenge struct {
	manager *ChallengeManager
	id      [32]byte
	inner   outgen.Challenge
	lock    sync.Mutex
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
	inner   outgen.ChallengeVertex
	lock    sync.Mutex
}

func (a *Assertion) invalidate() error {
	a.lock.Lock()
	defer a.lock.Unlock()
	inner, err := a.chain.caller.GetAssertion(a.chain.callOpts, a.id)
	if err != nil {
		return err
	}
	a.inner = inner
	return nil
}

func (c *Challenge) invalidate() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	inner, err := c.manager.caller.GetChallenge(c.manager.assertionChain.callOpts, c.id)
	if err != nil {
		return err
	}
	c.inner = inner
	return nil
}

func (v *ChallengeVertex) invalidate() error {
	v.lock.Lock()
	defer v.lock.Unlock()
	inner, err := v.manager.caller.GetVertex(v.manager.assertionChain.callOpts, v.id)
	if err != nil {
		return err
	}
	v.inner = inner
	return nil
}
