// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	chain     *AssertionChain
	id        protocol.AssertionHash
	createdAt uint64
}

func (a *Assertion) Id() protocol.AssertionHash {
	return a.id
}

func (a *Assertion) PrevId(ctx context.Context) (protocol.AssertionHash, error) {
	creationInfo, err := a.chain.ReadAssertionCreationInfo(ctx, a.id)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	return protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash}, nil
}

func (a *Assertion) HasSecondChild() (bool, error) {
	inner, err := a.inner()
	if err != nil {
		return false, err
	}
	return inner.SecondChildBlock > 0, nil
}

func (a *Assertion) inner() (*rollupgen.AssertionNode, error) {
	var b [32]byte
	copy(b[:], a.id.Bytes())
	assertionNode, err := a.chain.userLogic.GetAssertion(util.GetFinalizedCallOpts(&bind.CallOpts{}), b)
	if err != nil {
		return nil, err
	}
	if assertionNode.Status == uint8(0) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			a.id,
		)
	}
	return &assertionNode, nil
}
func (a *Assertion) FirstChildCreationBlock() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return inner.FirstChildBlock, nil
}
func (a *Assertion) SecondChildCreationBlock() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return inner.SecondChildBlock, nil
}
func (a *Assertion) IsFirstChild() (bool, error) {
	inner, err := a.inner()
	if err != nil {
		return false, err
	}
	return inner.IsFirstChild, nil
}
func (a *Assertion) CreatedAtBlock() uint64 {
	return a.createdAt
}
func (a *Assertion) Status(ctx context.Context) (protocol.AssertionStatus, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return protocol.AssertionStatus(inner.Status), nil
}

type honestEdge struct {
	protocol.SpecEdge
}

func (h *honestEdge) Honest() {}

type specEdge struct {
	id                   [32]byte
	mutualId             [32]byte
	manager              *specChallengeManager
	miniStaker           option.Option[common.Address]
	inner                challengeV2gen.ChallengeEdge
	startHeight          uint64
	endHeight            uint64
	totalChallengeLevels uint8
	assertionHash        protocol.AssertionHash
}
