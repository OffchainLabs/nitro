package solimpl

import (
	"bytes"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
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

func (a *Assertion) Height() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return inner.Height.Uint64(), nil
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

func (a *Assertion) StateHash() (common.Hash, error) {
	inner, err := a.inner()
	if err != nil {
		return common.Hash{}, err
	}
	return inner.StateHash, nil
}

func (a *Assertion) InboxMsgCountSeen() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	if !inner.InboxMsgCountSeen.IsUint64() {
		return 0, fmt.Errorf("assertion %v inbox msg count %v is not a uint64", a.id, inner.InboxMsgCountSeen)
	}
	return inner.InboxMsgCountSeen.Uint64(), nil
}

func (a *Assertion) inner() (*rollupgen.AssertionNode, error) {
	assertionNode, err := a.chain.userLogic.GetAssertion(&bind.CallOpts{}, a.id)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(assertionNode.StateHash[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %d",
			a.id,
		)
	}
	return &assertionNode, nil
}

type SpecEdge struct {
	id         [32]byte
	manager    *SpecChallengeManager
	miniStaker util.Option[common.Address]
	inner      challengeV2gen.ChallengeEdge
}
