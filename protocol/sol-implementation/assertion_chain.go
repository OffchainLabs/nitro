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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

var (
	ErrUnconfirmedParent   = errors.New("parent assertion is not confirmed")
	ErrNoUnresolved        = errors.New("no assertion to resolve")
	ErrNonPendingAssertion = errors.New("assertion is not pending")
	ErrRejectedAssertion   = errors.New("assertion already rejected")
	ErrInvalidChildren     = errors.New("invalid children")
	ErrNotFound            = errors.New("item not found on-chain")
	ErrAlreadyExists       = errors.New("item already exists on-chain")
	ErrPrevDoesNotExist    = errors.New("assertion predecessor does not exist")
	ErrTooLate             = errors.New("too late to create assertion sibling")
	ErrTooSoon             = errors.New("too soon to confirm assertion")
	ErrInvalidHeight       = errors.New("invalid assertion height")
)

type activeTx struct {
	readWriteTx bool
	finalized   *big.Int
	head        *big.Int
	sender      common.Address
}

func (a *activeTx) FinalizedBlockNumber() *big.Int {
	return a.finalized
}

func (a *activeTx) HeadBlockNumber() *big.Int {
	return a.head
}

func (a *activeTx) ReadOnly() bool {
	return !a.readWriteTx
}

func (a *activeTx) Sender() common.Address {
	return a.sender
}

// ChainBackend to interact with the underlying blockchain.
type ChainBackend interface {
	bind.ContractBackend
	BlockchainReader
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

// BlockchainReader --
type BlockchainReader interface {
	Blockchain() *core.BlockChain
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

// Tx enables a mutating call to the chain.
func (chain *AssertionChain) Tx(cb func(protocol.ActiveTx) error) error {
	head := chain.backend.Blockchain().CurrentHeader()
	finalized := chain.backend.Blockchain().CurrentFinalizedBlock()
	var headNum *big.Int
	var finalizedNum *big.Int
	if finalized != nil {
		finalizedNum = finalized.Number()
	}
	if head != nil {
		headNum = head.Number
	}
	tx := &activeTx{
		readWriteTx: true,
		head:        headNum,
		finalized:   finalizedNum,
		sender:      chain.stakerAddr,
	}
	return cb(tx)
}

// Call enables a non-mutating call to the chain.
func (chain *AssertionChain) Call(cb func(protocol.ActiveTx) error) error {
	head := chain.backend.Blockchain().CurrentHeader()
	finalized := chain.backend.Blockchain().CurrentFinalizedBlock()
	var headNum *big.Int
	var finalizedNum *big.Int
	if finalized != nil {
		finalizedNum = finalized.Number()
	}
	if head != nil {
		headNum = head.Number
	}
	tx := &activeTx{
		readWriteTx: false,
		head:        headNum,
		finalized:   finalizedNum,
		sender:      chain.stakerAddr,
	}
	return cb(tx)
}

func (ac *AssertionChain) NumAssertions(
	ctx context.Context,
	tx protocol.ActiveTx,
) (uint64, error) {
	return ac.rollup.NumAssertions(ac.callOpts)
}

// AssertionBySequenceNum --
func (ac *AssertionChain) AssertionBySequenceNum(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertionNum protocol.AssertionSequenceNumber,
) (protocol.Assertion, error) {
	res, err := ac.userLogic.GetAssertion(ac.callOpts, uint64(assertionNum))
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
		id:    uint64(assertionNum),
		chain: ac,
		inner: res,
		StateCommitment: util.StateCommitment{
			Height:    res.Height.Uint64(),
			StateRoot: res.StateHash,
		},
	}, nil
}

func (ac *AssertionChain) LatestConfirmed(ctx context.Context, tx protocol.ActiveTx) (protocol.Assertion, error) {
	res, err := ac.rollup.LatestConfirmed(ac.callOpts)
	if err != nil {
		return nil, err
	}
	return ac.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(res))
}

// CreateAssertion makes an on-chain claim given a previous assertion id, execution state,
// and a commitment to a post-state.
func (ac *AssertionChain) CreateAssertion(
	ctx context.Context,
	tx protocol.ActiveTx,
	height uint64,
	prevAssertionId protocol.AssertionSequenceNumber,
	prevAssertionState *protocol.ExecutionState,
	postState *protocol.ExecutionState,
	prevInboxMaxCount *big.Int,
) (protocol.Assertion, error) {
	prev, err := ac.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(prevAssertionId))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get prev assertion with id: %d", prevAssertionId)
	}
	prevHeight := prev.Height()
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
	return ac.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(assertionCreated.AssertionNum))
}

func (ac *AssertionChain) GetAssertionId(
	ctx context.Context,
	tx protocol.ActiveTx,
	seqNum protocol.AssertionSequenceNumber,
) (protocol.AssertionHash, error) {
	return ac.userLogic.GetAssertionId(ac.callOpts, uint64(seqNum))
}

func (ac *AssertionChain) GetAssertionNum(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertionHash protocol.AssertionHash,
) (protocol.AssertionSequenceNumber, error) {
	res, err := ac.userLogic.GetAssertionNum(ac.callOpts, assertionHash)
	if err != nil {
		return 0, err
	}
	return protocol.AssertionSequenceNumber(res), nil
}

// CreateSuccessionChallenge creates a succession challenge
func (ac *AssertionChain) CreateSuccessionChallenge(
	ctx context.Context,
	tx protocol.ActiveTx,
	seqNum protocol.AssertionSequenceNumber,
) (protocol.Challenge, error) {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.userLogic.CreateChallenge(
			ac.txOpts,
			uint64(seqNum),
		)
	})
	if err2 := handleCreateSuccessionChallengeError(err, uint64(seqNum)); err2 != nil {
		return nil, err2
	}
	manager, err := ac.CurrentChallengeManager(ctx, tx)
	if err != nil {
		return nil, err
	}
	assertionId, err := ac.rollup.GetAssertionId(ac.callOpts, uint64(seqNum))
	if err != nil {
		return nil, err
	}
	challengeId, err := manager.CalculateChallengeHash(ctx, tx, assertionId, protocol.BlockChallenge)
	if err != nil {
		return nil, err
	}
	chal, err := manager.GetChallenge(ctx, tx, challengeId)
	if err != nil {
		return nil, err
	}
	return chal.Unwrap(), nil
}

// Confirm creates a confirmation for an assertion at the block hash and send root.
func (ac *AssertionChain) Confirm(
	ctx context.Context, tx protocol.ActiveTx, blockHash, sendRoot common.Hash,
) error {
	receipt, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
		return ac.userLogic.ConfirmNextAssertion(ac.txOpts, blockHash, sendRoot)
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
			"Wanted assertion at block hash %#x confirmed, but block hash was %#x",
			blockHash,
			confirmed.BlockHash,
		)
	}
	if confirmed.SendRoot != sendRoot {
		return fmt.Errorf(
			"Wanted assertion at send root %#x confirmed, but send root was %#x",
			sendRoot,
			confirmed.SendRoot,
		)
	}
	return nil
}

// Reject creates a rejection for the given assertion.
func (ac *AssertionChain) Reject(
	ctx context.Context, tx protocol.ActiveTx, staker common.Address,
) error {
	_, err := transact(ctx, ac.backend, func() (*types.Transaction, error) {
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
	case strings.Contains(errS, "ALREADY_CHALLENGED"):
		return errors.Wrapf(ErrAlreadyExists, "assertion id %d", assertionId)
	case strings.Contains(errS, "At least two children not created"):
		return errors.Wrapf(ErrInvalidChildren, "assertion id %d", assertionId)
	case strings.Contains(errS, "NO_SECOND_CHILD"):
		return errors.Wrapf(ErrInvalidChildren, "assertion id %d", assertionId)
	case strings.Contains(errS, "too late"):
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
