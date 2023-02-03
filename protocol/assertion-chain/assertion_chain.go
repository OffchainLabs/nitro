// Package assertionchain includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package assertionchain

import (
	"context"
	"time"

	"math/big"

	"bytes"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"strings"
)

var (
	ErrNotFound      = errors.New("item not found on-chain")
	ErrAlreadyExists = errors.New("item already exists on-chain")
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
	caller     *outgen.AssertionChainCaller
	writer     *outgen.AssertionChainTransactor
	callOpts   *bind.CallOpts
	txOpts     *bind.TransactOpts
	stakerAddr common.Address
}

// CreateAssertion provided a state commitment and the previous
// assertion's ID.
func (ac *AssertionChain) CreateAssertion(
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) (*Assertion, error) {
	if err := ac.createAssertion(
		commitment,
		prevAssertionId,
	); err != nil {
		return nil, err
	}
	assertionId := getAssertionId(commitment, prevAssertionId)
	return ac.AssertionByID(assertionId)
}

func (ac *AssertionChain) createAssertion(
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) error {
	_, err := ac.writer.CreateNewAssertion(
		ac.txOpts,
		commitment.StateRoot,
		big.NewInt(int64(commitment.Height)),
		prevAssertionId,
	)
	if err != nil {
		errS := err.Error()
		switch {
		case strings.Contains(errS, "Assertion already exists"):
			return errors.Wrapf(
				ErrAlreadyExists,
				"commit state root %#x and height %d",
				commitment.StateRoot,
				commitment.Height,
			)
		default:
			return err
		}
	}
	return nil
}

// ChallengePeriod length in seconds.
func (ac *AssertionChain) ChallengePeriodLength() (time.Duration, error) {
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
	if bytes.Equal(res.StateHash[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			assertionId,
		)
	}
	return &Assertion{
		inner: res,
	}, nil
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
	assertionChainBinding, err := outgen.NewAssertionChain(
		contractAddr, chain.backend,
	)
	if err != nil {
		return nil, err
	}
	chain.caller = &assertionChainBinding.AssertionChainCaller
	chain.writer = &assertionChainBinding.AssertionChainTransactor
	return chain, nil
}

func getAssertionId(
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) common.Hash {
	heightTy, _ := abi.NewType("uint256", "", nil)
	hashTy, _ := abi.NewType("bytes32", "", nil)
	arguments := abi.Arguments{
		{
			Type: hashTy,
		},
		{
			Type: heightTy,
		},
		{
			Type: hashTy,
		},
	}

	height := big.NewInt(int64(commitment.Height))
	packed, _ := arguments.Pack(
		commitment.StateRoot,
		height,
		prevAssertionId,
	)
	return crypto.Keccak256Hash(packed)
}
