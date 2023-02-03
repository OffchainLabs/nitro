// Package assertionchain includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package assertionchain

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	inner outgen.Assertion
}

// AssertionChain is a wrapper around solgen bindings
// that implements the protocol interface.
type AssertionChain struct {
	backend    bind.ContractBackend
	caller     *outgen.AssertionChainV2Caller
	writer     *outgen.AssertionChainV2Transactor
	callOpts   *bind.CallOpts
	txOpts     *bind.TransactOpts
	stakerAddr common.Address
}

// NewAssertionChain instantiates an assertion chain
// instance from a chain backend and provided options.
func NewAssertionChain(
	ctx context.Context,
	contractAddr common.Address,
	txOpts *bind.TransactOpts,
	callOpts *bind.CallOpts,
	stakerAddr common.Address,
	backend bind.ContractBackend,
) (*AssertionChain, error) {
	chain := &AssertionChain{
		backend:    backend,
		callOpts:   callOpts,
		txOpts:     txOpts,
		stakerAddr: stakerAddr,
	}
	assertionChainBinding, err := outgen.NewAssertionChainV2(
		contractAddr, chain.backend,
	)

	if err != nil {
		return nil, err
	}
	chain.caller = &assertionChainBinding.AssertionChainV2Caller
	chain.writer = &assertionChainBinding.AssertionChainV2Transactor
	return chain, nil
}

// ChallengePeriod length in seconds.
func (ac *AssertionChain) ChalengePeriodLength() (time.Duration, error) {
	res, err := ac.caller.ChallengePeriod(ac.callOpts)
	if err != nil {
		return time.Second, err
	}
	return time.Second * time.Duration(res.Uint64()), nil
}

// AssertionByID --
func (ac *AssertionChain) AssertionByID(assertionId common.Hash) (*Assertion, error) {
	res, err := ac.caller.GetAssertion(ac.callOpts, assertionId)
	if err != nil {
		return nil, err
	}
	return &Assertion{
		inner: res,
	}, nil
}
