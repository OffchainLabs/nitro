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

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	rollup     *rollupgen.RollupCore
	userLogic  *rollupgen.RollupUserLogic
	callOpts   *bind.CallOpts
	txOpts     *bind.TransactOpts
	stakerAddr common.Address
}

// NewAssertionChain instantiates an assertion chain
// instance from a chain backend and provided options.
func NewAssertionChain(
	ctx context.Context,
	rollupAddr common.Address,
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
	coreBinding, err := rollupgen.NewRollupCore(
		rollupAddr, chain.backend,
	)
	if err != nil {
		return nil, err
	}
	assertionChainBinding, err := rollupgen.NewRollupUserLogic(
		rollupAddr, chain.backend,
	)
	if err != nil {
		return nil, err
	}
	chain.rollup = coreBinding
	chain.userLogic = assertionChainBinding
	return chain, nil
}

// ChallengePeriodSeconds
func (ac *AssertionChain) ChallengePeriodSeconds() (time.Duration, error) {
	manager, err := ac.ChallengeManager()
	if err != nil {
		return time.Second, err
	}
	res, err := manager.caller.ChallengePeriodSec(ac.callOpts)
	if err != nil {
		return time.Second, err
	}
	return time.Second * time.Duration(res.Uint64()), nil
}

// AssertionByID --
func (ac *AssertionChain) AssertionByID(assertionNum uint64) (*Assertion, error) {
	res, err := ac.userLogic.GetAssertion(ac.callOpts, assertionNum)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(res.StateHash[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %d",
			assertionNum,
		)
	}
	return &Assertion{
		id:    assertionNum,
		chain: ac,
		inner: res,
		StateCommitment: util.StateCommitment{
			Height:    res.Height.Uint64(),
			StateRoot: res.StateHash,
		},
	}, nil
}

// CreateAssertion makes an on-chain claim given a previous assertion id, execution state,
// and a commitment to a post-state.
func (ac *AssertionChain) CreateAssertion(
	ctx context.Context,
	height uint64,
	prevAssertionId uint64,
	prevAssertionState *ExecutionState,
	postState *ExecutionState,
	prevInboxMaxCount *big.Int,
) (*Assertion, error) {
	prev, err := ac.AssertionByID(prevAssertionId)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get prev assertion with id: %d", prevAssertionId)
	}
	prevHeight := prev.inner.Height.Uint64()
	if prevHeight >= height {
		return nil, errors.Wrapf(ErrInvalidHeight, "prev height %d was >= incoming %d", prevHeight, height)
	}
	stake, err := ac.userLogic.CurrentRequiredStake(ac.callOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current required stake")
	}
	newOpts := copyTxOpts(ac.txOpts)
	newOpts.Value = stake

	receipt, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.userLogic.NewStakeOnNewAssertion(
			newOpts,
			rollupgen.AssertionInputs{
				BeforeState: prevAssertionState.AsSolidityStruct(),
				AfterState:  postState.AsSolidityStruct(),
				NumBlocks:   height - prevHeight,
			},
			common.Hash{}, // Expected hash. TODO: Is this fine as empty?
			prevInboxMaxCount,
		)
	})
	if createErr := handleCreateAssertionError(err, height, postState.GlobalState.BlockHash); createErr != nil {
		return nil, createErr
	}
	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs observed from assertion creation")
	}
	assertionCreated, err := ac.rollup.ParseAssertionCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse assertion creation log")
	}
	return ac.AssertionByID(assertionCreated.AssertionNum)
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(ctx context.Context, assertionId uint64) (*Challenge, error) {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.userLogic.CreateChallenge(
			ac.txOpts,
			assertionId,
		)
	})
	if err2 := handleCreateSuccessionChallengeError(err, assertionId); err2 != nil {
		return nil, err2
	}
	manager, err := ac.ChallengeManager()
	if err != nil {
		return nil, err
	}
	challengeId, err := manager.CalculateChallengeId(ctx, common.Hash{}, BlockChallenge)
	if err != nil {
		return nil, err
	}
	return manager.ChallengeByID(ctx, challengeId)
}

// Confirm creates a confirmation for the given assertion.
func (a *Assertion) Confirm() error {
	_, err := a.chain.userLogic.ConfirmNextAssertion(a.chain.txOpts, common.Hash{}, common.Hash{})
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
func (a *Assertion) Reject() error {
	_, err := a.chain.userLogic.RejectNextAssertion(a.chain.txOpts, a.chain.stakerAddr)
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

func handleCreateSuccessionChallengeError(err error, assertionId uint64) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "Assertion does not exist"):
		return errors.Wrapf(ErrNotFound, "assertion id %d", assertionId)
	case strings.Contains(errS, "Assertion already rejected"):
		return errors.Wrapf(ErrRejectedAssertion, "assertion id %d", assertionId)
	case strings.Contains(errS, "Challenge already created"):
		return errors.Wrapf(ErrAlreadyExists, "assertion id %d", assertionId)
	case strings.Contains(errS, "At least two children not created"):
		return errors.Wrapf(ErrInvalidChildren, "assertion id %d", assertionId)
	case strings.Contains(errS, "Too late to challenge"):
		return errors.Wrapf(ErrTooLate, "assertion id %d", assertionId)
	default:
		return err
	}
}

func handleCreateAssertionError(err error, height uint64, blockHash common.Hash) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "Assertion already exists"):
		return errors.Wrapf(
			ErrAlreadyExists,
			"commit block hash %#x and height %d",
			blockHash,
			height,
		)
	case strings.Contains(errS, "Height not greater than predecessor"):
		return errors.Wrapf(
			ErrInvalidHeight,
			"commit block hash %#x and height %d",
			blockHash,
			height,
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
