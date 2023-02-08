// Package assertionchain includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package assertionchain

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

var (
	ErrRejectedAssertion = errors.New("assertion already rejected")
	ErrInvalidChildren   = errors.New("invalid children")
	ErrNotFound          = errors.New("item not found on-chain")
	ErrAlreadyExists     = errors.New("item already exists on-chain")
	ErrPrevDoesNotExist  = errors.New("assertion predecessor does not exist")
	ErrTooLate           = errors.New("too late to create assertion sibling")
	ErrInvalidHeight     = errors.New("invalid assertion height")
	uint256Ty, _         = abi.NewType("uint256", "", nil)
	hashTy, _            = abi.NewType("bytes32", "", nil)
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

// ChallengePeriodSeconds
func (ac *AssertionChain) ChallengePeriodSeconds() (time.Duration, error) {
	res, err := ac.caller.ChallengePeriodSeconds(ac.callOpts)
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

// CreateAssertion provided a state commitment and the previous
// assertion's ID. Then, if the creation was successful,
// returns the newly created assertion.
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

// Triggers an assertion creation transaction.
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
	return handleCreateAssertionError(err, commitment)
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(assertionId common.Hash) error {
	_, err := ac.writer.CreateSuccessionChallenge(
		ac.txOpts,
		assertionId,
	)
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "Assertion does not exist"):
		return errors.Wrapf(ErrNotFound, "assertion id %#x", assertionId)
	case strings.Contains(err.Error(), "Assertion already rejected"):
		return errors.Wrapf(ErrRejectedAssertion, "assertion id %#x", assertionId)
	case strings.Contains(err.Error(), "Challenge already created"):
		return errors.Wrapf(ErrAlreadyExists, "assertion id %#x", assertionId)
	case strings.Contains(err.Error(), "At least two children not created"):
		return errors.Wrapf(ErrInvalidChildren, "assertion id %#x", assertionId)
	case strings.Contains(err.Error(), "Too late to challenge"):
		return errors.Wrapf(ErrTooLate, "assertion id %#x", assertionId)
	}
	return err
}

func (ac *AssertionChain) UpdateChallengeManager(a common.Address) error {
	_, err := ac.writer.UpdateChallengeManager(ac.txOpts, a)
	return err
}

func handleCreateAssertionError(err error, commitment util.StateCommitment) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "Assertion already exists"):
		return errors.Wrapf(
			ErrAlreadyExists,
			"commit state root %#x and height %d",
			commitment.StateRoot,
			commitment.Height,
		)
	case strings.Contains(errS, "Height not greater than predecessor"):
		return errors.Wrapf(
			ErrInvalidHeight,
			"commit state root %#x and height %d",
			commitment.StateRoot,
			commitment.Height,
		)
	case strings.Contains(errS, "Previous assertion does not exist"):
		return ErrPrevDoesNotExist
	case strings.Contains(errS, "Too late to create sibling"):
		return ErrTooLate
	default:
		return nil
	}
}

// Constructs and assertion ID which is built as
// keccak256(abi.encodePacked(stateRoot,height,prevAssertionId)).
func getAssertionId(
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) common.Hash {
	arguments := abi.Arguments{
		{
			Type: hashTy,
		},
		{
			Type: uint256Ty,
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
