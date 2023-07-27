// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	chain *AssertionChain
	id    protocol.AssertionHash
}

func (a *Assertion) Id() protocol.AssertionHash {
	return a.id
}

func (a *Assertion) PrevId(ctx context.Context) (protocol.AssertionHash, error) {
	createdAtBlock, err := a.CreatedAtBlock()
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(createdAtBlock),
		ToBlock:   nil, // Latest block.
		Addresses: []common.Address{a.chain.rollupAddr},
		Topics:    [][]common.Hash{{assertionCreatedId}},
	}
	logs, err := a.chain.backend.FilterLogs(ctx, query)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	if len(logs) == 0 {
		return protocol.AssertionHash{}, errors.New("no assertion creation events found")
	}
	creationEvent, err := a.chain.rollup.ParseAssertionCreated(logs[len(logs)-1])
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	return protocol.AssertionHash{Hash: creationEvent.ParentAssertionHash}, nil
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
	assertionNode, err := a.chain.userLogic.GetAssertion(&bind.CallOpts{}, b)
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

func (a *Assertion) CreatedAtBlock() (uint64, error) {
	inner, err := a.inner()
	if err != nil {
		return 0, err
	}
	return inner.CreatedAtBlock, nil
}

type SpecEdge struct {
	id          [32]byte
	mutualId    [32]byte
	manager     *SpecChallengeManager
	miniStaker  option.Option[common.Address]
	inner       challengeV2gen.ChallengeEdge
	startHeight uint64
	endHeight   uint64
}
