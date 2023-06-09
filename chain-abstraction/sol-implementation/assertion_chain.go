// Package solimpl includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/headerreader"
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

var assertionCreatedId common.Hash

func init() {
	rollupAbi, err := rollupgen.RollupCoreMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	assertionCreatedEvent, ok := rollupAbi.Events["AssertionCreated"]
	if !ok {
		panic("RollupCore ABI missing AssertionCreated event")
	}
	assertionCreatedId = assertionCreatedEvent.ID
}

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
	rollupAddr   common.Address
}

// NewAssertionChain instantiates an assertion chain
// instance from a chain backend and provided options.
func NewAssertionChain(
	_ context.Context,
	rollupAddr common.Address,
	txOpts *bind.TransactOpts,
	backend ChainBackend,
	headerReader *headerreader.HeaderReader,
) (*AssertionChain, error) {
	chain := &AssertionChain{
		backend:      backend,
		txOpts:       txOpts,
		headerReader: headerReader,
		rollupAddr:   rollupAddr,
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

func (a *AssertionChain) GetAssertion(ctx context.Context, assertionId protocol.AssertionId) (protocol.Assertion, error) {
	res, err := a.userLogic.GetAssertion(&bind.CallOpts{Context: ctx}, assertionId)
	if err != nil {
		return nil, err
	}
	if res.Status == uint8(0) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			assertionId,
		)
	}
	return &Assertion{
		id:     assertionId,
		prevId: res.PrevId,
		chain:  a,
	}, nil
}

func (a *AssertionChain) LatestConfirmed(ctx context.Context) (protocol.Assertion, error) {
	res, err := a.rollup.LatestConfirmed(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}
	return a.GetAssertion(ctx, res)
}

func (a *AssertionChain) BaseStake(ctx context.Context) (*big.Int, error) {
	return a.userLogic.BaseStake(&bind.CallOpts{Context: ctx})
}

func (a *AssertionChain) WasmModuleRoot(ctx context.Context) ([32]byte, error) {
	return a.userLogic.WasmModuleRoot(&bind.CallOpts{Context: ctx})
}

// CreateAssertion makes an on-chain claim given a previous assertion id, execution state,
// and a commitment to a post-state.
func (a *AssertionChain) CreateAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	if !assertionCreationInfo.InboxMaxCount.IsUint64() {
		return nil, errors.New("prev assertion creation info inbox max count not a uint64")
	}
	stake, err := a.userLogic.BaseStake(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, errors.Wrap(err, "could not get current required stake")
	}
	newOpts := copyTxOpts(a.txOpts)
	newOpts.Value = stake

	chalManager, err := a.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	chalPeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	if err != nil {
		return nil, err
	}
	wasmModuleRoot, err := a.userLogic.WasmModuleRoot(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, errors.Wrap(err, "could not get current required stake")
	}
	receipt, err := transact(ctx, a.backend, a.headerReader, func() (*types.Transaction, error) {
		return a.userLogic.NewStakeOnNewAssertion(
			newOpts,
			rollupgen.AssertionInputs{
				BeforeStateData: rollupgen.BeforeStateData{
					PrevPrevAssertionHash: assertionCreationInfo.ParentAssertionHash,
					SequencerBatchAcc:     assertionCreationInfo.AfterInboxBatchAcc,
					ConfigData: rollupgen.ConfigData{
						RequiredStake:       stake,
						ChallengeManager:    chalManager.Address(),
						ConfirmPeriodBlocks: chalPeriodBlocks,
						WasmModuleRoot:      wasmModuleRoot,
						NextInboxPosition:   assertionCreationInfo.InboxMaxCount.Uint64(),
					},
				},
				BeforeState: assertionCreationInfo.AfterState,
				AfterState:  postState.AsSolidityStruct(),
			},
			// TODO(RJ): Use the expected assertion hash as a sanity check.
			common.Hash{},
		)
	})
	if createErr := handleCreateAssertionError(err, postState.GlobalState.BlockHash); createErr != nil {
		return nil, fmt.Errorf("failed to create assertion: %w", createErr)
	}
	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs observed from assertion creation")
	}
	var assertionCreated *rollupgen.RollupCoreAssertionCreated
	var found bool
	for _, log := range receipt.Logs {
		creationEvent, err := a.rollup.ParseAssertionCreated(*log)
		if err == nil {
			assertionCreated = creationEvent
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("could not find assertion created event in logs")
	}
	return a.GetAssertion(ctx, assertionCreated.AssertionHash)
}

// SpecChallengeManager creates a new spec challenge manager
func (a *AssertionChain) SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error) {
	challengeManagerAddr, err := a.userLogic.RollupUserLogicCaller.ChallengeManager(
		&bind.CallOpts{Context: ctx},
	)
	if err != nil {
		return nil, err
	}
	return NewSpecChallengeManager(
		ctx,
		challengeManagerAddr,
		a,
		a.backend,
		a.headerReader,
		a.txOpts,
	)
}

// TODO: Implement this logic.
func (a *AssertionChain) AssertionUnrivaledTime(_ context.Context, _ protocol.AssertionId) (uint64, error) {
	return 0, nil
}

func (a *AssertionChain) TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionId, error) {
	cm, err := a.SpecChallengeManager(ctx)
	if err != nil {
		return protocol.AssertionId{}, err
	}
	edgeOpt, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return protocol.AssertionId{}, err
	}
	if edgeOpt.IsNone() {
		return protocol.AssertionId{}, errors.New("edge was nil")
	}
	return edgeOpt.Unwrap().AssertionId(ctx)
}

func (a *AssertionChain) TopLevelClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (*protocol.OriginHeights, error) {
	cm, err := a.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	edgeOpt, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return nil, err
	}
	if edgeOpt.IsNone() {
		return nil, errors.New("edge was nil")
	}
	edge := edgeOpt.Unwrap()
	return edge.TopLevelClaimHeight(ctx)
}

func (a *AssertionChain) LatestCreatedAssertion(ctx context.Context) (protocol.Assertion, error) {
	latestConfirmed, err := a.LatestConfirmed(ctx)
	if err != nil {
		return nil, err
	}
	createdAtBlock, err := latestConfirmed.CreatedAtBlock()
	if err != nil {
		return nil, err
	}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(createdAtBlock),
		ToBlock:   nil, // Latest block.
		Addresses: []common.Address{a.rollupAddr},
		Topics:    [][]common.Hash{{assertionCreatedId}},
	}
	logs, err := a.backend.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, errors.New("no assertion creation events found")
	}
	creationEvent, err := a.rollup.ParseAssertionCreated(logs[len(logs)-1])
	if err != nil {
		return nil, err
	}
	return a.GetAssertion(ctx, creationEvent.AssertionHash)
}

// ReadAssertionCreationInfo for an assertion sequence number by looking up its creation
// event from the rollup contracts.
func (a *AssertionChain) ReadAssertionCreationInfo(
	ctx context.Context, id protocol.AssertionId,
) (*protocol.AssertionCreatedInfo, error) {
	var creationBlock uint64
	var topics [][]common.Hash
	if id == (protocol.AssertionId{}) {
		rollupDeploymentBlock, err := a.rollup.RollupDeploymentBlock(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		if !rollupDeploymentBlock.IsUint64() {
			return nil, errors.New("rollup deployment block was not a uint64")
		}
		creationBlock = rollupDeploymentBlock.Uint64()
		topics = [][]common.Hash{{assertionCreatedId}}
	} else {
		node, err := a.rollup.GetAssertion(&bind.CallOpts{Context: ctx}, id)
		if err != nil {
			return nil, err
		}
		creationBlock = node.CreatedAtBlock
		topics = [][]common.Hash{{assertionCreatedId}, {common.Hash(id)}}
	}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(creationBlock),
		ToBlock:   new(big.Int).SetUint64(creationBlock),
		Addresses: []common.Address{a.rollupAddr},
		Topics:    topics,
	}
	logs, err := a.backend.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, errors.New("no assertion creation logs found")
	}
	if len(logs) > 1 {
		return nil, errors.New("found multiple instances of requested node")
	}
	ethLog := logs[0]
	parsedLog, err := a.rollup.ParseAssertionCreated(ethLog)
	if err != nil {
		return nil, err
	}
	afterState := parsedLog.Assertion.AfterState
	return &protocol.AssertionCreatedInfo{
		ConfirmPeriodBlocks: parsedLog.ConfirmPeriodBlocks,
		RequiredStake:       parsedLog.RequiredStake,
		ParentAssertionHash: parsedLog.ParentAssertionHash,
		BeforeState:         parsedLog.Assertion.BeforeState,
		AfterState:          afterState,
		InboxMaxCount:       parsedLog.InboxMaxCount,
		AfterInboxBatchAcc:  parsedLog.AfterInboxBatchAcc,
		AssertionHash:       parsedLog.AssertionHash,
		WasmModuleRoot:      parsedLog.WasmModuleRoot,
	}, nil
}

func handleCreateAssertionError(err error, blockHash common.Hash) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "Assertion already exists"):
		return errors.Wrapf(
			ErrAlreadyExists,
			"commit block hash %#x",
			blockHash,
		)
	case strings.Contains(errS, "Assertion does not exist"):
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
func transact(ctx context.Context, backend ChainBackend, _ *headerreader.HeaderReader, fn func() (*types.Transaction, error)) (*types.Receipt, error) {
	tx, err := fn()
	if err != nil {
		return nil, err
	}
	if commiter, ok := backend.(ChainCommitter); ok {
		commiter.Commit()
	}
	receipt, err := bind.WaitMined(ctx, backend, tx)
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
