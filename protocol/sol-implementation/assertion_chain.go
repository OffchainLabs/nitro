// Package solimpl includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package solimpl

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"

	"github.com/offchainlabs/nitro/util/headerreader"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum"
	"github.com/pkg/errors"
)

var (
	ErrUnconfirmedParent = errors.New("parent assertion is not confirmed")
	ErrNoUnresolved      = errors.New("no assertion to resolve")

	ErrNotFound         = errors.New("item not found on-chain")
	ErrAlreadyExists    = errors.New("item already exists on-chain")
	ErrPrevDoesNotExist = errors.New("assertion predecessor does not exist")
	ErrTooLate          = errors.New("too late to create assertion sibling")
	ErrTooSoon          = errors.New("too soon to confirm assertion")
	ErrInvalidHeight    = errors.New("invalid assertion height")
)

// ChainBackend to interact with the underlying blockchain.
type ChainBackend interface {
	bind.ContractBackend
	ReceiptFetcher
}

// ChainCommitter defines a type of chain backend that supports
// committing changes via a direct method, such as a simulated backend
// for testing purposes.
type ChainCommitter interface {
	Commit() common.Hash
}

// ReceiptFetcher defines the ability to retrieve transactions receipts from the chain.
type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// AssertionChain is a wrapper around solgen bindings
// that implements the protocol interface.
type AssertionChain struct {
	backend      ChainBackend
	rollup       *rollupgen.RollupCore
	userLogic    *rollupgen.RollupUserLogic
	txOpts       *bind.TransactOpts
	headerReader *headerreader.HeaderReader
}

// NewAssertionChain instantiates an assertion chain
// instance from a chain backend and provided options.
func NewAssertionChain(
	ctx context.Context,
	rollupAddr common.Address,
	txOpts *bind.TransactOpts,
	backend ChainBackend,
	headerReader *headerreader.HeaderReader,
) (*AssertionChain, error) {
	chain := &AssertionChain{
		backend:      backend,
		txOpts:       txOpts,
		headerReader: headerReader,
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

func (ac *AssertionChain) NumAssertions(ctx context.Context) (uint64, error) {
	return ac.rollup.NumAssertions(&bind.CallOpts{Context: ctx})
}

// AssertionBySequenceNum --
func (ac *AssertionChain) AssertionBySequenceNum(ctx context.Context, seqNum protocol.AssertionSequenceNumber) (protocol.Assertion, error) {
	res, err := ac.userLogic.GetAssertion(&bind.CallOpts{Context: ctx}, uint64(seqNum))
	if err != nil {
		return nil, err
	}
	if bytes.Equal(res.StateHash[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %d",
			seqNum,
		)
	}
	return &Assertion{
		id:    uint64(seqNum),
		chain: ac,
		StateCommitment: util.StateCommitment{
			Height:    res.Height.Uint64(),
			StateRoot: res.StateHash,
		},
	}, nil
}

func (ac *AssertionChain) LatestConfirmed(ctx context.Context) (protocol.Assertion, error) {
	res, err := ac.rollup.LatestConfirmed(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}
	return ac.AssertionBySequenceNum(ctx, protocol.AssertionSequenceNumber(res))
}

// CreateAssertion makes an on-chain claim given a previous assertion id, execution state,
// and a commitment to a post-state.
func (ac *AssertionChain) CreateAssertion(
	ctx context.Context,
	height uint64,
	prevAssertionId protocol.AssertionSequenceNumber,
	prevAssertionState *protocol.ExecutionState,
	postState *protocol.ExecutionState,
	prevInboxMaxCount *big.Int,
) (protocol.Assertion, error) {
	prev, err := ac.AssertionBySequenceNum(ctx, prevAssertionId)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get prev assertion with id: %d", prevAssertionId)
	}
	prevHeight, err := prev.Height()
	if err != nil {
		return nil, err
	}
	if prevHeight >= height {
		return nil, errors.Wrapf(ErrInvalidHeight, "prev height %d was >= incoming %d", prevHeight, height)
	}
	stake, err := ac.userLogic.CurrentRequiredStake(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, errors.Wrap(err, "could not get current required stake")
	}
	newOpts := copyTxOpts(ac.txOpts)
	newOpts.Value = stake

	receipt, err := transact(ctx, ac.backend, ac.headerReader, func() (*types.Transaction, error) {
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
	return ac.AssertionBySequenceNum(ctx, protocol.AssertionSequenceNumber(assertionCreated.AssertionNum))
}

func (ac *AssertionChain) GetAssertionId(ctx context.Context, seqNum protocol.AssertionSequenceNumber) (protocol.AssertionHash, error) {
	return ac.userLogic.GetAssertionId(&bind.CallOpts{Context: ctx}, uint64(seqNum))
}

func (ac *AssertionChain) GetAssertionNum(ctx context.Context, assertionHash protocol.AssertionHash) (protocol.AssertionSequenceNumber, error) {
	res, err := ac.userLogic.GetAssertionNum(&bind.CallOpts{Context: ctx}, assertionHash)
	if err != nil {
		return 0, err
	}
	return protocol.AssertionSequenceNumber(res), nil
}
func (ac *AssertionChain) BlockChallenge(_ context.Context, _ protocol.AssertionSequenceNumber) (protocol.Challenge, error) {
	return nil, errors.New("unimplemented")
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(_ context.Context, _ protocol.AssertionSequenceNumber) (protocol.Challenge, error) {
	return nil, errors.New("unimplemented")
}

// SpecChallengeManager creates a new spec challenge manager
func (ac *AssertionChain) SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error) {
	challengeManagerAddr, err := ac.userLogic.RollupUserLogicCaller.ChallengeManager(
		&bind.CallOpts{Context: ctx},
	)
	if err != nil {
		return nil, err
	}
	return NewSpecChallengeManager(
		ctx,
		challengeManagerAddr,
		ac,
		ac.backend,
		ac.headerReader,
		ac.txOpts,
	)
}

// Confirm creates a confirmation for an assertion at the block hash and send root.
func (ac *AssertionChain) Confirm(ctx context.Context, blockHash, sendRoot common.Hash) error {
	receipt, err := transact(ctx, ac.backend, ac.headerReader, func() (*types.Transaction, error) {
		return ac.userLogic.ConfirmNextAssertion(ac.txOpts, blockHash, sendRoot, [32]byte{}) // TODO(RJ): Add winning edge.
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "Assertion does not exist"):
			return errors.Wrapf(ErrNotFound, "block hash %#x", blockHash)
		case strings.Contains(err.Error(), "Previous assertion not confirmed"):
			return errors.Wrapf(ErrUnconfirmedParent, "previous assertion not confirmed")
		case strings.Contains(err.Error(), "NO_UNRESOLVED"):
			return ErrNoUnresolved
		case strings.Contains(err.Error(), "CHILD_TOO_RECENT"):
			return ErrTooSoon
		default:
			return err
		}
	}
	if len(receipt.Logs) == 0 {
		return errors.New("no logs observed from assertion confirmation")
	}
	confirmed, err := ac.rollup.ParseAssertionConfirmed(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return errors.Wrap(err, "could not parse assertion confirmation log")
	}
	if confirmed.BlockHash != blockHash {
		return fmt.Errorf(
			"wanted assertion at block hash %#x confirmed, but block hash was %#x",
			blockHash,
			confirmed.BlockHash,
		)
	}
	if confirmed.SendRoot != sendRoot {
		return fmt.Errorf(
			"wanted assertion at send root %#x confirmed, but send root was %#x",
			sendRoot,
			confirmed.SendRoot,
		)
	}
	return nil
}

// Reject creates a rejection for the given assertion.
func (ac *AssertionChain) Reject(ctx context.Context, staker common.Address) error {
	_, err := transact(ctx, ac.backend, ac.headerReader, func() (*types.Transaction, error) {
		return ac.userLogic.RejectNextAssertion(ac.txOpts, staker)
	})
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "NO_UNRESOLVED"):
		return ErrNoUnresolved
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
func transact(ctx context.Context, backend ChainBackend, l1Reader *headerreader.HeaderReader, fn func() (*types.Transaction, error)) (*types.Receipt, error) {
	tx, err := fn()
	if err != nil {
		return nil, err
	}
	if commiter, ok := backend.(ChainCommitter); ok {
		commiter.Commit()
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		callMsg := ethereum.CallMsg{
			From:       common.Address{},
			To:         tx.To(),
			Gas:        0,
			GasPrice:   nil,
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		}
		if _, err := backend.CallContract(ctx, callMsg, nil); err != nil {
			return nil, errors.Wrap(err, "failed transaction")
		}
	}
	return receipt, nil
}

// copyTxOpts creates a deep copy of the given transaction options.
func copyTxOpts(opts *bind.TransactOpts) *bind.TransactOpts {
	copied := &bind.TransactOpts{
		From:     opts.From,
		Context:  opts.Context,
		NoSend:   opts.NoSend,
		Signer:   opts.Signer,
		GasLimit: opts.GasLimit,
	}

	if opts.Nonce != nil {
		copied.Nonce = new(big.Int).Set(opts.Nonce)
	}
	if opts.Value != nil {
		copied.Value = new(big.Int).Set(opts.Value)
	}
	if opts.GasPrice != nil {
		copied.GasPrice = new(big.Int).Set(opts.GasPrice)
	}
	if opts.GasFeeCap != nil {
		copied.GasFeeCap = new(big.Int).Set(opts.GasFeeCap)
	}
	if opts.GasTipCap != nil {
		copied.GasTipCap = new(big.Int).Set(opts.GasTipCap)
	}
	return copied
}
