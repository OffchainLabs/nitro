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
	uint8Ty, _           = abi.NewType("uint8", "", nil)
	hashTy, _            = abi.NewType("bytes32", "", nil)
)

// ChainCommitter defines a type of chain backend that supports
// committing changes via a direct method, such as a simulated backend
// for testing purposes.
type ChainCommiter interface {
	Commit() common.Hash
}

// Assertion is a wrapper around the binding to the type
// of the same name in the protocol contracts. This allows us
// to have a smaller API surface area and attach useful
// methods that callers can use directly.
type Assertion struct {
	StateCommitment util.StateCommitment
	inner           outgen.Assertion
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
		StateCommitment: util.StateCommitment{
			Height:    res.Height.Uint64(),
			StateRoot: res.StateHash,
		},
	}, nil
}

// CreateAssertion provided a state commitment and the previous
// assertion's ID. Then, if the creation was successful,
// returns the newly created assertion.
func (ac *AssertionChain) CreateAssertion(
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) (*Assertion, error) {
	err := withChainCommitment(ac.backend, func() error {
		_, err := ac.writer.CreateNewAssertion(
			ac.txOpts,
			commitment.StateRoot,
			big.NewInt(int64(commitment.Height)),
			prevAssertionId,
		)
		return err
	})
	if err := handleCreateAssertionError(err, commitment); err != nil {
		return nil, err
	}
	assertionId := getAssertionId(commitment, prevAssertionId)
	return ac.AssertionByID(assertionId)
}

func (ac *AssertionChain) UpdateChallengeManager(a common.Address) error {
	return withChainCommitment(ac.backend, func() error {
		_, err := ac.writer.UpdateChallengeManager(ac.txOpts, a)
		return err
	})
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(assertionId common.Hash) (*Challenge, error) {
	err := withChainCommitment(ac.backend, func() error {
		_, err := ac.writer.CreateSuccessionChallenge(
			ac.txOpts,
			assertionId,
		)
		return err
	})
	if err := handleCreateSuccessionChallengeError(err, assertionId); err != nil {
		return nil, err
	}
	manager, err := ac.ChallengeManager()
	if err != nil {
		return nil, err
	}
	challengeId, err := manager.CalculateChallengeId(assertionId, uint8(BlockChallenge))
	if err != nil {
		return nil, err
	}
	return manager.ChallengeByID(challengeId)
}

func handleCreateSuccessionChallengeError(err error, assertionId common.Hash) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "Assertion does not exist"):
		return errors.Wrapf(ErrNotFound, "assertion id %#x", assertionId)
	case strings.Contains(errS, "Assertion already rejected"):
		return errors.Wrapf(ErrRejectedAssertion, "assertion id %#x", assertionId)
	case strings.Contains(errS, "Challenge already created"):
		return errors.Wrapf(ErrAlreadyExists, "assertion id %#x", assertionId)
	case strings.Contains(errS, "At least two children not created"):
		return errors.Wrapf(ErrInvalidChildren, "assertion id %#x", assertionId)
	case strings.Contains(errS, "Too late to challenge"):
		return errors.Wrapf(ErrTooLate, "assertion id %#x", assertionId)
	default:
		return nil
	}
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

// Runs a callback function meant to write to a contract backend, and if the
// chain backend supports committing directly, we call the commit function before
// returning.
func withChainCommitment(backend bind.ContractBackend, fn func() error) error {
	if err := fn(); err != nil {
		return err
	}
	if commiter, ok := backend.(ChainCommiter); ok {
		commiter.Commit()
	}
	return nil
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
