// Package solimpl includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package solimpl

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"time"

	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

var (
	ErrUnconfirmedParent   = errors.New("parent assertion is not confirmed")
	ErrNonPendingAssertion = errors.New("assertion is not pending")
	ErrRejectedAssertion   = errors.New("assertion already rejected")
	ErrInvalidChildren     = errors.New("invalid children")
	ErrNotFound            = errors.New("item not found on-chain")
	ErrAlreadyExists       = errors.New("item already exists on-chain")
	ErrPrevDoesNotExist    = errors.New("assertion predecessor does not exist")
	ErrTooLate             = errors.New("too late to create assertion sibling")
	ErrInvalidHeight       = errors.New("invalid assertion height")
	uint256Ty, _           = abi.NewType("uint256", "", nil)
	hashTy, _              = abi.NewType("bytes32", "", nil)
)

// ChainBackend to interact with the underlying blockchain.
type ChainBackend interface {
	bind.ContractBackend
	ReceiptFetcher
}

// ChainCommitter defines a type of chain backend that supports
// committing changes via a direct method, such as a simulated backend
// for testing purposes.
type ChainCommiter interface {
	Commit() common.Hash
}

// ReceiptFetcher defines the ability to retrieve transactions receipts from the chain.
type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// AssertionChain is a wrapper around solgen bindings
// that implements the protocol interface.
type AssertionChain struct {
	backend    ChainBackend
	caller     *challengeV2gen.AssertionChainCaller
	writer     *challengeV2gen.AssertionChainTransactor
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
	backend ChainBackend,
) (*AssertionChain, error) {
	chain := &AssertionChain{
		backend:    backend,
		callOpts:   callOpts,
		txOpts:     txOpts,
		stakerAddr: stakerAddr,
	}
	assertionChainBinding, err := challengeV2gen.NewAssertionChain(
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
func (ac *AssertionChain) ChallengePeriodSeconds(ctx context.Context) (time.Duration, error) {
	res, err := ac.caller.ChallengePeriodSeconds(ac.callOpts)
	if err != nil {
		return time.Second, err
	}
	return time.Second * time.Duration(res.Uint64()), nil
}

// AssertionByID --
func (ac *AssertionChain) AssertionByID(ctx context.Context, assertionId common.Hash) (*Assertion, error) {
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
		id:    assertionId,
		chain: ac,
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
	ctx context.Context,
	commitment util.StateCommitment,
	prevAssertionId common.Hash,
) (*Assertion, error) {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.writer.CreateNewAssertion(
			ac.txOpts,
			commitment.StateRoot,
			big.NewInt(int64(commitment.Height)),
			prevAssertionId,
		)
	})
	if err2 := handleCreateAssertionError(err, commitment); err2 != nil {
		return nil, err2
	}
	assertionId := getAssertionId(commitment, prevAssertionId)
	return ac.AssertionByID(ctx, assertionId)
}

func (ac *AssertionChain) UpdateChallengeManager(ctx context.Context, a common.Address) error {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.writer.UpdateChallengeManager(ac.txOpts, a)
	})
	return err
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(
	ctx context.Context,
	assertionId common.Hash,
) (*Challenge, error) {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.writer.CreateSuccessionChallenge(
			ac.txOpts,
			assertionId,
		)
	})
	if err3 := handleCreateSuccessionChallengeError(err, assertionId); err3 != nil {
		return nil, err3
	}
	manager, err := ac.ChallengeManager()
	if err != nil {
		return nil, err
	}
	challengeId, err := manager.CalculateChallengeId(ctx, assertionId, BlockChallenge)
	if err != nil {
		return nil, err
	}
	return manager.ChallengeByID(ctx, challengeId)
}

// Confirm creates a confirmation for the given assertion.
func (a *Assertion) Confirm(ctx context.Context) error {
	_, err := a.chain.writer.ConfirmAssertion(a.chain.txOpts, a.id)
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "Assertion does not exist"):
		return errors.Wrapf(ErrNotFound, "assertion with id %#x", a.id)
	case strings.Contains(err.Error(), "Previous assertion not confirmed"):
		return errors.Wrapf(ErrUnconfirmedParent, "previous assertion not confirmed")
	default:
		return err
	}
}

// Reject creates a rejection for the given assertion.
func (a *Assertion) Reject(ctx context.Context) error {
	_, err := a.chain.writer.RejectAssertion(a.chain.txOpts, a.id)
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "Assertion does not exist"):
		return errors.Wrapf(ErrNotFound, "assertion with id %#x", a.id)
	case strings.Contains(err.Error(), "Assertion is not pending"):
		return errors.Wrapf(ErrNonPendingAssertion, "assertion with id %#x", a.id)
	default:
		return err
	}
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
		return err
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
		return err
	}
}

// Runs a callback function meant to write to a chain backend, and if the
// chain backend supports committing directly, we call the commit function before
// returning. This function additionally waits for the transaction to complete and returns
// an optional transaction receipt. It returns an error if the
// transaction had a failed status on-chain, or if the execution of the callback
// failed directly.
// TODO(RJ): Add logic of waiting for transactions to complete.
func transact(ctx context.Context, backend ChainBackend, fn func() (*types.Transaction, error)) (*types.Receipt, error) {
	tx, err := fn()
	if err != nil {
		return nil, err
	}
	if commiter, ok := backend.(ChainCommiter); ok {
		commiter.Commit()
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("receipt status shows failing transaction: %+v", receipt)
	}
	return receipt, nil
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

func copyTxOpts(opts *bind.TransactOpts) *bind.TransactOpts {
	return &bind.TransactOpts{
		From:      opts.From,
		Nonce:     opts.Nonce,
		Signer:    opts.Signer,
		Value:     opts.Value,
		GasPrice:  opts.GasPrice,
		GasFeeCap: opts.GasFeeCap,
		GasTipCap: opts.GasTipCap,
		GasLimit:  opts.GasLimit,
		Context:   opts.Context,
		NoSend:    opts.NoSend,
	}
}
