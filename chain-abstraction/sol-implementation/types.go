package solimpl

import (
	"bytes"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	StateCommitment commitments.State
	chain           *AssertionChain
	id              uint64
}

func (a *Assertion) SeqNum() protocol.AssertionSequenceNumber {
	return protocol.AssertionSequenceNumber(a.id)
}

func (a *Assertion) PrevSeqNum() (protocol.AssertionSequenceNumber, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	if inner.PrevNum == 0 {
		return protocol.AssertionSequenceNumber(1), nil
	}
	return protocol.AssertionSequenceNumber(inner.PrevNum), nil
}

func (a *Assertion) IsFirstChild() (bool, error) {
	inner, err := a.inner()
	if err != nil {
		return false, err
	}
	return inner.IsFirstChild, nil
}

func (a *Assertion) inner() (*rollupgen.AssertionNode, error) {
	assertionNode, err := a.chain.userLogic.GetAssertion(&bind.CallOpts{}, a.id)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(assertionNode.AssertionHash[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %d",
			a.id,
		)
	}
	return &assertionNode, nil
}

func (a *Assertion) CreatedAtBlock() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return inner.CreatedAtBlock, nil
}

type SpecEdge struct {
	id         [32]byte
	mutualId   [32]byte
	manager    *SpecChallengeManager
	miniStaker option.Option[common.Address]
	inner      challengeV2gen.ChallengeEdge
}
