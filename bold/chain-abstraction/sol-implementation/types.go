// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package solimpl

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	chain     *AssertionChain
	id        protocol.AssertionHash
	createdAt uint64

	// Fields that are eventually constant like status, firstChildBlock etc.
	// These are set to option.None until they are in the final state, after which they are set
	// to the final value and never changed again (this saves us the on-chain call)
	prevId           option.Option[protocol.AssertionHash] // This is set the first time prevId is called
	firstChildBlock  option.Option[uint64]                 // Once the assertion has a first child, this is set
	secondChildBlock option.Option[uint64]                 // Once the assertion has a second child, this is set
	isFirstChild     bool                                  // Once the assertion is determined to be a first child, this is set
	isConfirmed      bool                                  // Once the assertion is confirmed, this is set
}

func (a *Assertion) Id() protocol.AssertionHash {
	return a.id
}

func (a *Assertion) PrevId(ctx context.Context) (protocol.AssertionHash, error) {
	if a.prevId.IsSome() {
		return a.prevId.Unwrap(), nil
	}
	creationInfo, err := a.chain.ReadAssertionCreationInfo(ctx, a.id)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	a.prevId = option.Some(creationInfo.ParentAssertionHash)
	return a.prevId.Unwrap(), nil
}

func (a *Assertion) HasSecondChild(ctx context.Context, opts *bind.CallOpts) (bool, error) {
	if a.secondChildBlock.IsSome() {
		return a.secondChildBlock.Unwrap() > 0, nil
	}
	inner, err := a.inner(ctx, opts)
	if err != nil {
		return false, err
	}
	return inner.SecondChildBlock > 0, nil
}

func (a *Assertion) inner(ctx context.Context, opts *bind.CallOpts) (*rollupgen.AssertionNode, error) {
	var b [32]byte
	copy(b[:], a.id.Bytes())
	assertionNode, err := a.chain.userLogic.GetAssertion(opts, b)
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
	// Update the assertion with the latest data, if they are in now in constant state.
	if assertionNode.FirstChildBlock > 0 {
		a.firstChildBlock = option.Some(assertionNode.FirstChildBlock)
	}
	if assertionNode.SecondChildBlock > 0 {
		a.secondChildBlock = option.Some(assertionNode.SecondChildBlock)
	}
	if assertionNode.IsFirstChild {
		a.isFirstChild = true
	}
	assertionStatus := protocol.AssertionStatus(assertionNode.Status)
	if assertionStatus == protocol.AssertionConfirmed {
		a.isConfirmed = true
	}
	return &assertionNode, nil
}
func (a *Assertion) FirstChildCreationBlock(ctx context.Context, opts *bind.CallOpts) (uint64, error) {
	if a.firstChildBlock.IsSome() {
		return a.firstChildBlock.Unwrap(), nil
	}
	inner, err := a.inner(ctx, opts)
	if err != nil {
		return 0, err
	}
	return inner.FirstChildBlock, nil
}
func (a *Assertion) SecondChildCreationBlock(ctx context.Context, opts *bind.CallOpts) (uint64, error) {
	if a.secondChildBlock.IsSome() {
		return a.secondChildBlock.Unwrap(), nil
	}
	inner, err := a.inner(ctx, opts)
	if err != nil {
		return 0, err
	}
	return inner.SecondChildBlock, nil
}
func (a *Assertion) IsFirstChild(ctx context.Context, opts *bind.CallOpts) (bool, error) {
	if a.isFirstChild {
		return a.isFirstChild, nil
	}
	inner, err := a.inner(ctx, opts)
	if err != nil {
		return false, err
	}
	return inner.IsFirstChild, nil
}
func (a *Assertion) CreatedAtBlock() uint64 {
	return a.createdAt
}
func (a *Assertion) Status(ctx context.Context, opts *bind.CallOpts) (protocol.AssertionStatus, error) {
	if a.isConfirmed {
		return protocol.AssertionConfirmed, nil
	}
	inner, err := a.inner(ctx, opts)
	if err != nil {
		return 0, err
	}
	return protocol.AssertionStatus(inner.Status), nil
}

type honestEdge struct {
	*specEdge
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

	// Fields that are eventually constant like status, hasRival etc.
	// These are set to option.None until they are in the final state, after which they are set
	// to the final value and never changed again (this saves us the on-chain call)
	timeUnrivaled     option.Option[uint64]          // Once edge has a rival, this is set
	hasRival          bool                           // Once edge has a rival, this is set
	isConfirmed       bool                           // Once the edge is confirmed, this is set
	confirmedAtBlock  option.Option[uint64]          // Once the edge is confirmed, this is set
	lowerChild        option.Option[protocol.EdgeId] // Once the edge has a lower child, this is set
	upperChild        option.Option[protocol.EdgeId] // Once the edge has an upper child, this is set
	hasLengthOneRival bool                           // Once the edge has a rival of length 1, this is set
	verifiedHonest    bool                           // Whether or not the edge has been verified as honest.
}
