// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package solimpl includes an easy-to-use abstraction
// around the challenge protocol contracts using their Go
// bindings and exposes minimal details of Ethereum's internals.
package solimpl

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/testgen"
)

var (
	ErrNotFound         = errors.New("item not found on-chain")
	ErrBatchNotYetFound = errors.New("batch not yet found")
	ErrAlreadyExists    = errors.New("item already exists on-chain")
	ErrPrevDoesNotExist = errors.New("assertion predecessor does not exist")
	ErrTooLate          = errors.New("too late to create assertion sibling")
)

var assertionCreatedId common.Hash

var defaultBaseGas = int64(500000)

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

// ReceiptFetcher defines the ability to retrieve transactions receipts from the chain.
type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// Transactor defines the ability to send transactions to the chain.
type Transactor interface {
	SendTransaction(ctx context.Context, fn func(opts *bind.TransactOpts) (*types.Transaction, error), opts *bind.TransactOpts, gas uint64) (*types.Transaction, error)
}

type ChainBackendTransactor struct {
	protocol.ChainBackend
	fifo *FIFO
}

func NewChainBackendTransactor(backend protocol.ChainBackend) *ChainBackendTransactor {
	return &ChainBackendTransactor{
		ChainBackend: backend,
		fifo:         NewFIFO(1000),
	}
}

func (d *ChainBackendTransactor) SendTransaction(ctx context.Context, fn func(opts *bind.TransactOpts) (*types.Transaction, error), opts *bind.TransactOpts, gas uint64) (*types.Transaction, error) {
	// Try to acquire lock and if it fails, wait for a bit and try again.
	for !d.fifo.Lock() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	defer d.fifo.Unlock()
	tx, err := fn(opts)
	if err != nil {
		return nil, err
	}
	return tx, d.ChainBackend.SendTransaction(ctx, tx)
}

// AssertionChain is a wrapper around solgen bindings
// that implements the protocol interface.
type AssertionChain struct {
	backend                                  protocol.ChainBackend
	rollup                                   *rollupgen.RollupCore
	userLogic                                *rollupgen.RollupUserLogic
	txOpts                                   *bind.TransactOpts
	rollupAddr                               common.Address
	chalManagerAddr                          common.Address
	confirmedChallengesByParentAssertionHash *threadsafe.LruSet[protocol.AssertionHash]
	specChallengeManager                     protocol.SpecChallengeManager
	averageTimeForBlockCreation              time.Duration
	minAssertionPeriodBlocks                 uint64
	transactor                               Transactor
	withdrawalAddress                        common.Address
	stakeTokenAddr                           common.Address
	autoDeposit                              bool
	enableFastConfirmation                   bool
	fastConfirmSafe                          *FastConfirmSafe
	// rpcHeadBlockNumber is the block number of the latest block on the chain.
	// It is set to rpc.FinalizedBlockNumber by default.
	// WithRpcHeadBlockNumber can be used to set a different block number.
	rpcHeadBlockNumber rpc.BlockNumber
}

type Opt func(*AssertionChain)

func WithTrackedContractBackend() Opt {
	return func(a *AssertionChain) {
		a.backend = NewTrackedContractBackend(a.backend)
	}
}

func WithMetricsContractBackend() Opt {
	return func(a *AssertionChain) {
		a.backend = NewMetricsContractBackend(a.backend)
	}
}

func WithRpcHeadBlockNumber(rpcHeadBlockNumber rpc.BlockNumber) Opt {
	return func(a *AssertionChain) {
		a.rpcHeadBlockNumber = rpcHeadBlockNumber
	}
}

// WithCustomWithdrawalAddress specifies a custom withdrawal address for validators that
// choose to perform a delegated stake to participate in BoLD.
func WithCustomWithdrawalAddress(address common.Address) Opt {
	return func(a *AssertionChain) {
		a.withdrawalAddress = address
	}
}

// WithoutAutoDeposit prevents the assertion chain from automatically depositing stake token
// funds when making stakes on assertions or challenge edges.
func WithoutAutoDeposit() Opt {
	return func(a *AssertionChain) {
		a.autoDeposit = false
	}
}

// WithFastConfirmation enables fast confirmation for the assertion chain.
func WithFastConfirmation() Opt {
	return func(a *AssertionChain) {
		a.enableFastConfirmation = true
	}
}

// WithParentChainBlockCreationTime sets the average time for block creation of the chain where
// assertions are posted to and fetched from.
func WithParentChainBlockCreationTime(d time.Duration) Opt {
	return func(a *AssertionChain) {
		a.averageTimeForBlockCreation = d
	}
}

// NewAssertionChain instantiates an assertion chain
// instance from a chain backend and provided options.
func NewAssertionChain(
	ctx context.Context,
	rollupAddr common.Address,
	chalManagerAddr common.Address,
	txOpts *bind.TransactOpts,
	backend protocol.ChainBackend,
	transactor Transactor,
	opts ...Opt,
) (*AssertionChain, error) {
	// We disable sending txs by default, as we will first estimate their gas before
	// we commit them onchain through the transact method in this package.
	copiedOpts := copyTxOpts(txOpts)
	chain := &AssertionChain{
		backend:                                  backend,
		txOpts:                                   copiedOpts,
		rollupAddr:                               rollupAddr,
		chalManagerAddr:                          chalManagerAddr,
		confirmedChallengesByParentAssertionHash: threadsafe.NewLruSet(1000, threadsafe.LruSetWithMetric[protocol.AssertionHash]("confirmedChallengesByParentAssertionHash")),
		averageTimeForBlockCreation:              time.Second * 12,
		transactor:                               transactor,
		rpcHeadBlockNumber:                       rpc.LatestBlockNumber,
		withdrawalAddress:                        copiedOpts.From, // Default to the tx opts' sender.
		autoDeposit:                              true,
	}
	for _, opt := range opts {
		opt(chain)
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
	callOpts := chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
	minPeriod, err := chain.rollup.MinimumAssertionPeriod(callOpts)
	if err != nil {
		return nil, err
	}
	if !minPeriod.IsUint64() {
		return nil, errors.New("minimum assertion period was not a uint64")
	}
	if minPeriod.Uint64() == 0 {
		minPeriod = big.NewInt(1)
	}
	stakeTokenAddr, err := chain.rollup.StakeToken(callOpts)
	if err != nil {
		return nil, err
	}
	code, err := backend.CodeAt(ctx, stakeTokenAddr, nil)
	if err != nil {
		return nil, err
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("stake token address %#x has no code", stakeTokenAddr)
	}
	chain.stakeTokenAddr = stakeTokenAddr
	log.Info("Minimum assertion period", "blocks", minPeriod.Uint64())
	chain.minAssertionPeriodBlocks = minPeriod.Uint64()
	chain.userLogic = assertionChainBinding
	specChallengeManager, err := NewSpecChallengeManager(
		ctx,
		chain.chalManagerAddr,
		chain,
		chain.backend,
		chain.txOpts,
	)
	if err != nil {
		return nil, err
	}
	chain.specChallengeManager = specChallengeManager
	err = chain.setupFastConfirmation(callOpts)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func (a *AssertionChain) setupFastConfirmation(callOpts *bind.CallOpts) error {
	if !a.enableFastConfirmation {
		return nil
	}
	fastConfirmer, err := retry.UntilSucceeds(callOpts.Context, func() (common.Address, error) {
		return a.rollup.AnyTrustFastConfirmer(callOpts)
	})
	if err != nil {
		return fmt.Errorf("getting rollup fast confirmer address: %w", err)
	}
	log.Info("Setting up fast confirmation", "stakerAddress", a.StakerAddress(), "fastConfirmer", fastConfirmer)
	if fastConfirmer == a.StakerAddress() {
		// We can directly fast confirm nodes
		return nil
	} else if fastConfirmer == (common.Address{}) {
		// No fast confirmer enabled
		return errors.New("fast confirmation enabled in config, but no fast confirmer set in rollup contract")
	}
	// The fast confirmer address is a contract address, not sure if it's a safe contract yet.
	fastConfirmSafe, err := NewFastConfirmSafe(callOpts, fastConfirmer, a)
	if err != nil {
		// Unknown while loading the safe contract.
		return fmt.Errorf("loading fast confirm safe: %w", err)
	}
	// Fast confirmer address implements getOwners() and is probably a safe.
	isOwner, err := retry.UntilSucceeds(callOpts.Context, func() (bool, error) {
		return fastConfirmSafe.safe.IsOwner(callOpts, a.StakerAddress())
	})
	if err != nil {
		return fmt.Errorf("checking if wallet is owner of safe: %w", err)
	}
	if !isOwner {
		return fmt.Errorf("staker wallet address %v is not an owner of the fast confirm safe %v", a.StakerAddress(), fastConfirmer)
	}
	a.fastConfirmSafe = fastConfirmSafe
	return nil
}
func (a *AssertionChain) RollupUserLogic() *rollupgen.RollupUserLogic {
	return a.userLogic
}

func (a *AssertionChain) RollupCore() *rollupgen.RollupCore {
	return a.rollup
}

func (a *AssertionChain) Backend() protocol.ChainBackend {
	return a.backend
}

func (a *AssertionChain) DesiredHeaderU64(ctx context.Context) (uint64, error) {
	header, err := a.backend.HeaderByNumber(ctx, big.NewInt(int64(a.rpcHeadBlockNumber)))
	if err != nil {
		return 0, err
	}
	if !header.Number.IsUint64() {
		return 0, errors.New("block number is not uint64")
	}
	return header.Number.Uint64(), nil
}

func (a *AssertionChain) DesiredL1HeaderU64(ctx context.Context) (uint64, error) {
	header, err := a.backend.HeaderByNumber(ctx, big.NewInt(int64(a.rpcHeadBlockNumber)))
	if err != nil {
		return 0, err
	}
	headerInfo := types.DeserializeHeaderExtraInformation(header)
	if headerInfo.ArbOSFormatVersion > 0 {
		return headerInfo.L1BlockNumber, nil
	}
	if !header.Number.IsUint64() {
		return 0, errors.New("block number is not uint64")
	}
	return header.Number.Uint64(), nil
}

func (a *AssertionChain) GetAssertion(ctx context.Context, opts *bind.CallOpts, assertionHash protocol.AssertionHash) (protocol.Assertion, error) {
	var b [32]byte
	copy(b[:], assertionHash.Bytes())
	res, err := a.userLogic.GetAssertion(opts, b)
	if err != nil {
		return nil, err
	}
	if res.Status == uint8(protocol.NoAssertion) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			assertionHash,
		)
	}
	return &Assertion{
		id:        assertionHash,
		chain:     a,
		createdAt: res.CreatedAtBlock,
	}, nil
}

func (a *AssertionChain) AssertionStatus(ctx context.Context, assertionHash protocol.AssertionHash) (protocol.AssertionStatus, error) {
	res, err := a.rollup.GetAssertion(&bind.CallOpts{Context: ctx}, assertionHash.Hash)
	if err != nil {
		return protocol.NoAssertion, err
	}
	return protocol.AssertionStatus(res.Status), nil
}

func (a *AssertionChain) LatestConfirmed(ctx context.Context, opts *bind.CallOpts) (protocol.Assertion, error) {
	res, err := a.rollup.LatestConfirmed(opts)
	if err != nil {
		return nil, err
	}
	return a.GetAssertion(ctx, opts, protocol.AssertionHash{Hash: res})
}

// Returns true if the staker's address is currently staked in the assertion chain.
func (a *AssertionChain) IsStaked(ctx context.Context) (bool, error) {
	return a.rollup.IsStaked(&bind.CallOpts{Context: ctx}, a.StakerAddress())
}

// RollupAddress for the assertion chain.
func (a *AssertionChain) RollupAddress() common.Address {
	return a.rollupAddr
}

// StakerAddress for the staker which initialized this chain interface.
func (a *AssertionChain) StakerAddress() common.Address {
	return a.txOpts.From
}

// IsChallengeComplete checks if a challenge is complete by using the challenge's parent assertion hash.
func (a *AssertionChain) IsChallengeComplete(
	ctx context.Context,
	challengeParentAssertionHash protocol.AssertionHash,
) (bool, error) {
	if a.confirmedChallengesByParentAssertionHash.Has(challengeParentAssertionHash) {
		return true, nil
	}
	parentAssertionStatus, err := a.AssertionStatus(ctx, challengeParentAssertionHash)
	if err != nil {
		return false, err
	}
	// Parent must be confirmed for a challenge to be considered complete, so we can
	// short-circuit early here.
	parentIsConfirmed := parentAssertionStatus == protocol.AssertionConfirmed
	if !parentIsConfirmed {
		return false, nil
	}
	latestConfirmed, err := a.LatestConfirmed(ctx, a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return false, err
	}
	// A challenge is complete if the parent assertion of the challenge is confirmed
	// and the latest confirmed assertion hash is not equal to the challenge's parent assertion hash.
	challengeConfirmed := latestConfirmed.Id() != challengeParentAssertionHash
	if challengeConfirmed {
		a.confirmedChallengesByParentAssertionHash.Insert(challengeParentAssertionHash)
	}
	return challengeConfirmed, nil
}

// AutoDepositTokenForStaking ensures that the validator has enough funds to stake
// on assertions if not already staked, and then deposits the difference required to participate.
func (a *AssertionChain) AutoDepositTokenForStaking(
	ctx context.Context,
	amount *big.Int,
) error {
	staked, err := a.IsStaked(ctx)
	if err != nil {
		return err
	}
	if staked {
		return nil
	}
	return a.autoDepositFunds(ctx, amount)
}

// Attempts to auto-wrap ETH to WETH with the required amount that is specified to the function.
// This function uses `latest` onchain data to determine the current balance of the staker
// and deposits the difference between the required amount and the current balance.
func (a *AssertionChain) autoDepositFunds(ctx context.Context, amount *big.Int) error {
	if !a.autoDeposit {
		return nil
	}
	// The validity of the stake token address containing code is checked in the constructor
	// of the assertion chain.
	erc20, err := testgen.NewERC20Token(a.stakeTokenAddr, a.backend)
	if err != nil {
		return err
	}
	balance, err := erc20.BalanceOf(&bind.CallOpts{Context: ctx}, a.txOpts.From)
	if err != nil {
		return err
	}
	// Get the difference between the required amount and the current balance.
	// If we have more than enough balance, we exit early.
	if balance.Cmp(amount) >= 0 {
		return nil
	}
	diff := new(big.Int).Sub(amount, balance)
	weth, err := mocksgen.NewIWETH9(a.stakeTokenAddr, a.backend)
	if err != nil {
		return err
	}
	// Otherwise, we deposit the difference.
	receipt, err := a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = diff
		return weth.Deposit(opts)
	})
	if err != nil {
		return err
	}
	_ = receipt
	return nil
}

func (a *AssertionChain) ApproveAllowances(
	ctx context.Context,
) error {
	// The validity of the stake token address containing code is checked in the constructor
	// of the assertion chain.
	erc20, err := testgen.NewERC20Token(a.stakeTokenAddr, a.backend)
	if err != nil {
		return err
	}
	rollupAllowance, err := erc20.Allowance(&bind.CallOpts{Context: ctx}, a.txOpts.From, a.rollupAddr)
	if err != nil {
		return err
	}
	chalManagerAllowance, err := erc20.Allowance(&bind.CallOpts{Context: ctx}, a.txOpts.From, a.chalManagerAddr)
	if err != nil {
		return err
	}
	maxUint256 := new(big.Int)
	maxUint256.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	if rollupAllowance.Cmp(maxUint256) == 0 {
		return nil
	}
	// Approve the rollup and challenge manager spending the user's stake token.
	if _, err = a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return erc20.Approve(opts, a.rollupAddr, maxUint256)
	}); err != nil {
		return err
	}
	if chalManagerAllowance.Cmp(maxUint256) == 0 {
		return nil
	}
	if _, err = a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return erc20.Approve(opts, a.chalManagerAddr, maxUint256)
	}); err != nil {
		return err
	}
	return nil
}

// NewStake is a function made for stakers that are delegated. It allows them to mark themselves as a "pending"
// staker in the rollup contracts with some required stake and allows another party to fund the staker onchain
// to proceed with its activities.
func (a *AssertionChain) NewStake(
	ctx context.Context,
) error {
	staked, err := a.IsStaked(ctx)
	if err != nil {
		return err
	}
	if staked {
		return nil
	}
	_, err = a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return a.userLogic.NewStake(opts, new(big.Int), a.withdrawalAddress)
	})
	return err
}

// NewStakeOnNewAssertion makes an onchain claim given a previous assertion hash, execution state,
// and a commitment to a post-state. It also adds a new stake to the newly created assertion.
// if the validator is already staked, use StakeOnNewAssertion instead.
func (a *AssertionChain) NewStakeOnNewAssertion(
	ctx context.Context,
	parentAssertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	stakeFn := func(
		opts *bind.TransactOpts,
		tokenAmount *big.Int,
		assertionInputs rollupgen.AssertionInputs,
		expectedAssertionHash [32]byte,
	) (*types.Transaction, error) {
		return a.userLogic.NewStakeOnNewAssertion50f32f68(
			opts,
			tokenAmount,
			assertionInputs,
			expectedAssertionHash,
			a.withdrawalAddress,
		)
	}
	return a.createAndStakeOnAssertion(
		ctx,
		parentAssertionCreationInfo,
		postState,
		stakeFn,
	)
}

// StakeOnNewAssertion makes an onchain claim given a previous assertion hash, execution state,
// and a commitment to a post-state. It also adds moves an existing stake to the newly created assertion.
// if the validator is not staked, use NewStakeOnNewAssertion instead.
func (a *AssertionChain) StakeOnNewAssertion(
	ctx context.Context,
	parentAssertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
) (protocol.Assertion, error) {
	stakeFn := func(opts *bind.TransactOpts, _ *big.Int, assertionInputs rollupgen.AssertionInputs, assertionHash [32]byte) (*types.Transaction, error) {
		return a.userLogic.StakeOnNewAssertion(
			opts,
			assertionInputs,
			assertionHash,
		)
	}
	return a.createAndStakeOnAssertion(
		ctx,
		parentAssertionCreationInfo,
		postState,
		stakeFn,
	)
}

func (a *AssertionChain) createAndStakeOnAssertion(
	ctx context.Context,
	parentAssertionCreationInfo *protocol.AssertionCreatedInfo,
	postState *protocol.ExecutionState,
	stakeFn func(opts *bind.TransactOpts, requiredStake *big.Int, assertionInputs rollupgen.AssertionInputs, assertionHash [32]byte) (*types.Transaction, error),
) (protocol.Assertion, error) {
	if !parentAssertionCreationInfo.InboxMaxCount.IsUint64() {
		return nil, errors.New("prev assertion creation info inbox max count not a uint64")
	}
	if postState.GlobalState.Batch == 0 {
		return nil, errors.New("assertion post state cannot have a batch count of 0, as only genesis can")
	}
	bridgeAddr, err := a.userLogic.Bridge(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve bridge address for user rollup logic contract")
	}
	bridge, err := bridgegen.NewIBridgeCaller(bridgeAddr, a.backend)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize bridge at address %#x", bridgeAddr)
	}
	inboxBatchAcc, err := bridge.SequencerInboxAccs(
		a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}),
		new(big.Int).SetUint64(postState.GlobalState.Batch-1),
	)
	if err != nil {
		return nil, ErrBatchNotYetFound
	}
	computedHash, err := a.userLogic.ComputeAssertionHash(
		a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}),
		parentAssertionCreationInfo.AssertionHash.Hash,
		postState.AsSolidityStruct(),
		inboxBatchAcc,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute assertion hash")
	}
	// Check if the assertion already exists based on latest head, which means we should not make a mutating call.
	existingAssertion, err := a.GetAssertion(ctx, &bind.CallOpts{Context: ctx}, protocol.AssertionHash{Hash: computedHash})
	switch {
	case err == nil:
		return existingAssertion, nil
	case !errors.Is(err, ErrNotFound):
		return nil, errors.Wrapf(err, "could not fetch assertion with computed hash %#x", computedHash)
	default:
	}
	staked, err := a.IsStaked(ctx)
	if err != nil {
		return nil, err
	}
	if !staked {
		if err = a.autoDepositFunds(ctx, parentAssertionCreationInfo.RequiredStake); err != nil {
			return nil, errors.Wrapf(err, "could not auto-deposit funds for assertion creation")
		}
	}
	receipt, err := a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return stakeFn(
			opts,
			parentAssertionCreationInfo.RequiredStake,
			rollupgen.AssertionInputs{
				BeforeStateData: rollupgen.BeforeStateData{
					PrevPrevAssertionHash: parentAssertionCreationInfo.ParentAssertionHash.Hash,
					SequencerBatchAcc:     parentAssertionCreationInfo.AfterInboxBatchAcc,
					ConfigData: rollupgen.ConfigData{
						RequiredStake:       parentAssertionCreationInfo.RequiredStake,
						ChallengeManager:    parentAssertionCreationInfo.ChallengeManager,
						ConfirmPeriodBlocks: parentAssertionCreationInfo.ConfirmPeriodBlocks,
						WasmModuleRoot:      parentAssertionCreationInfo.WasmModuleRoot,
						NextInboxPosition:   parentAssertionCreationInfo.InboxMaxCount.Uint64(),
					},
				},
				BeforeState: parentAssertionCreationInfo.AfterState,
				AfterState:  postState.AsSolidityStruct(),
			},
			computedHash,
		)
	})
	opts := a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
	if createErr := handleCreateAssertionError(err, postState.GlobalState.BlockHash); createErr != nil {
		if strings.Contains(err.Error(), "already exists") {
			assertionItem, err2 := a.GetAssertion(ctx, opts, protocol.AssertionHash{Hash: computedHash})
			if err2 != nil {
				return nil, err2
			}
			return assertionItem, nil
		}
		return nil, fmt.Errorf("could not create assertion: %w", createErr)
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
	return a.GetAssertion(ctx, opts, protocol.AssertionHash{Hash: assertionCreated.AssertionHash})
}

func (a *AssertionChain) GenesisAssertionHash(ctx context.Context) (common.Hash, error) {
	return a.userLogic.GenesisAssertionHash(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
}

func (a *AssertionChain) MinAssertionPeriodBlocks() uint64 {
	return a.minAssertionPeriodBlocks
}

// MaxAssertionsPerChallenge period returns maximum number of assertions that
// may need to be processed during a challenge period of blocks.
func (a *AssertionChain) MaxAssertionsPerChallengePeriod() uint64 {
	cb := a.SpecChallengeManager().ChallengePeriodBlocks()
	return cb / a.minAssertionPeriodBlocks
}

func TryConfirmingAssertion(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
	confirmableAfterBlock uint64,
	chain protocol.AssertionChain,
	averageTimeForBlockCreation time.Duration,
	winningEdgeId option.Option[protocol.EdgeId],
) (bool, error) {
	status, err := chain.AssertionStatus(ctx, assertionHash)
	if err != nil {
		return false, fmt.Errorf("could not get assertion by hash: %#x: %w", assertionHash, err)
	}
	if status == protocol.NoAssertion {
		return false, fmt.Errorf("no assertion found by hash: %#x", assertionHash)
	}
	if status == protocol.AssertionConfirmed {
		return true, nil
	}
	for {
		var latestL1HeaderNumber uint64
		latestL1HeaderNumber, err = chain.DesiredL1HeaderU64(ctx)
		if err != nil {
			return false, err
		}
		confirmable := latestL1HeaderNumber >= confirmableAfterBlock

		// If the assertion is not yet confirmable, we can simply wait.
		if !confirmable {
			var blocksLeftForConfirmation int64
			if latestL1HeaderNumber > confirmableAfterBlock {
				blocksLeftForConfirmation = 0
			} else {
				blocksLeftForConfirmation, err = safecast.ToInt64(confirmableAfterBlock - latestL1HeaderNumber)
				if err != nil {
					return false, err
				}
			}
			timeToWait := averageTimeForBlockCreation * time.Duration(blocksLeftForConfirmation)
			log.Info(
				fmt.Sprintf(
					"Assertion with hash %s needs at least %d blocks before being confirmable, waiting for %s",
					containers.Trunc(assertionHash.Bytes()),
					blocksLeftForConfirmation,
					timeToWait,
				),
			)
			select {
			case <-time.After(timeToWait):
			case <-ctx.Done():
				return false, ctx.Err()
			}
		} else {
			break
		}
	}

	if winningEdgeId.IsSome() {
		err = chain.ConfirmAssertionByChallengeWinner(ctx, assertionHash, winningEdgeId.Unwrap())
		if err != nil {
			if strings.Contains(err.Error(), protocol.ChallengeGracePeriodNotPassedAssertionConfirmationError) {
				return false, nil
			}
			if strings.Contains(err.Error(), "is not the latest confirmed assertion") {
				return false, nil
			}
			return false, err

		}
	} else {
		err = chain.ConfirmAssertionByTime(ctx, assertionHash)
		if err != nil {
			if strings.Contains(err.Error(), protocol.BeforeDeadlineAssertionConfirmationError) {
				return false, nil
			}
			if strings.Contains(err.Error(), "is not the latest confirmed assertion") {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

func (a *AssertionChain) ConfirmAssertionByTime(ctx context.Context, assertionHash protocol.AssertionHash) error {
	return a.ConfirmAssertionByChallengeWinner(ctx, assertionHash, protocol.EdgeId{})
}

// ConfirmAssertionByChallengeWinner attempts to confirm an assertion onchain
// if there is a winning, level zero, block challenge edge that claims it.
func (a *AssertionChain) ConfirmAssertionByChallengeWinner(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
	winningEdgeId protocol.EdgeId,
) error {
	var b [32]byte
	copy(b[:], assertionHash.Bytes())
	node, err := a.userLogic.GetAssertion(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), b)
	if err != nil {
		return err
	}
	if node.Status == uint8(protocol.AssertionConfirmed) {
		return nil
	}
	creationInfo, err := a.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	// If the assertion is genesis, return nil.
	if creationInfo.ParentAssertionHash.Hash == [32]byte{} {
		return nil
	}
	prevCreationInfo, err := a.ReadAssertionCreationInfo(ctx, creationInfo.ParentAssertionHash)
	if err != nil {
		return err
	}
	latestConfirmed, err := a.LatestConfirmed(ctx, &bind.CallOpts{Context: ctx})
	if err != nil {
		return err
	}
	if creationInfo.ParentAssertionHash != latestConfirmed.Id() {
		return fmt.Errorf(
			"parent id %#x is not the latest confirmed assertion %#x",
			creationInfo.ParentAssertionHash,
			latestConfirmed.Id(),
		)
	}
	if !prevCreationInfo.InboxMaxCount.IsUint64() {
		return errors.New("assertion prev creation info inbox max count was not a uint64")
	}
	receipt, err := a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return a.userLogic.ConfirmAssertion(
			opts,
			b,
			creationInfo.ParentAssertionHash.Hash,
			creationInfo.AfterState,
			winningEdgeId.Hash,
			rollupgen.ConfigData{
				WasmModuleRoot:      prevCreationInfo.WasmModuleRoot,
				ConfirmPeriodBlocks: prevCreationInfo.ConfirmPeriodBlocks,
				RequiredStake:       prevCreationInfo.RequiredStake,
				ChallengeManager:    prevCreationInfo.ChallengeManager,
				NextInboxPosition:   prevCreationInfo.InboxMaxCount.Uint64(),
			},
			creationInfo.AfterInboxBatchAcc,
		)
	})
	if err != nil {
		return err
	}
	if len(receipt.Logs) == 0 {
		return errors.New("no logs observed from assertion confirmation")
	}
	return nil
}

// FastConfirmAssertion attempts to fast confirm an assertion onchain.
func (a *AssertionChain) FastConfirmAssertion(
	ctx context.Context,
	assertionCreationInfo *protocol.AssertionCreatedInfo,
) (bool, error) {
	if a.fastConfirmSafe != nil {
		return a.fastConfirmSafe.fastConfirmAssertion(ctx, assertionCreationInfo)
	}
	receipt, err := a.transact(ctx, a.backend, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return a.userLogic.FastConfirmAssertion(
			opts,
			assertionCreationInfo.AssertionHash.Hash,
			assertionCreationInfo.ParentAssertionHash.Hash,
			assertionCreationInfo.AfterState,
			assertionCreationInfo.AfterInboxBatchAcc,
		)
	})
	if err != nil {
		return false, err
	}
	if len(receipt.Logs) == 0 {
		return false, errors.New("no logs observed from assertion confirmation")
	}
	return true, nil
}

// SpecChallengeManager returns the assertions chain's spec challenge manager.
func (a *AssertionChain) SpecChallengeManager() protocol.SpecChallengeManager {
	return a.specChallengeManager
}

// AssertionUnrivaledBlocks gets the number of blocks an assertion was unrivaled. That is, it looks up the
// assertion's parent, and from that parent, computes second_child_creation_block - first_child_creation_block.
// If an assertion is a second child, this function will return 0.
func (a *AssertionChain) AssertionUnrivaledBlocks(ctx context.Context, assertionHash protocol.AssertionHash) (uint64, error) {
	var b [32]byte
	copy(b[:], assertionHash.Bytes())
	wantNode, err := a.rollup.GetAssertion(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), b)
	if err != nil {
		return 0, err
	}
	if wantNode.Status == uint8(protocol.NoAssertion) {
		return 0, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			assertionHash,
		)
	}
	// If the assertion requested is not the first child, it was never unrivaled.
	if !wantNode.IsFirstChild {
		return 0, nil
	}
	assertion := &Assertion{
		id:        assertionHash,
		chain:     a,
		createdAt: wantNode.CreatedAtBlock,
	}
	prevId, err := assertion.PrevId(ctx)
	if err != nil {
		return 0, err
	}
	copy(b[:], prevId.Bytes())
	prevNode, err := a.rollup.GetAssertion(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), b)
	if err != nil {
		return 0, err
	}
	if prevNode.Status == uint8(protocol.NoAssertion) {
		return 0, errors.Wrapf(
			ErrNotFound,
			"assertion with id %#x",
			assertionHash,
		)
	}
	// If there is no second child, we simply return the number of blocks
	// since the assertion was created and its parent.
	if prevNode.SecondChildBlock == 0 {
		l1BlockNum, err := a.DesiredL1HeaderU64(ctx)
		if err != nil {
			return 0, err
		}

		// Should never happen.
		if assertion.CreatedAtBlock() > l1BlockNum {
			return 0, fmt.Errorf(
				"assertion creation block %d > latest block number %d for assertion hash %#x",
				assertion.CreatedAtBlock(),
				l1BlockNum,
				assertionHash,
			)
		}
		return l1BlockNum - assertion.CreatedAtBlock(), nil
	}
	// Should never happen.
	if prevNode.FirstChildBlock > prevNode.SecondChildBlock {
		return 0, fmt.Errorf(
			"first child creation block %d > second child creation block %d for assertion hash %#x",
			prevNode.FirstChildBlock,
			prevNode.SecondChildBlock,
			prevId,
		)
	}
	return prevNode.SecondChildBlock - prevNode.FirstChildBlock, nil
}

// GetAssertionCreationParentBlock returns parent chain block number when the assertion was created.
// assertion.CreatedAtBlock is the block number when the assertion was created on L1.
// But in case of L3, we need to look up the block number when the assertion was created on L2.
// To do this, we use getAssertionCreationBlockForLogLookup which returns the block number when the assertion was created
// on parent chain be it L2 or L1.
func (a *AssertionChain) GetAssertionCreationParentBlock(ctx context.Context, assertionHash common.Hash) (uint64, error) {
	callOpts := a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
	createdAtBlock, err := a.userLogic.GetAssertionCreationBlockForLogLookup(callOpts, assertionHash)
	if err != nil {
		return 0, errors.Wrapf(err, "could not get assertion creation block for assertion hash %#x", assertionHash)
	}
	if !createdAtBlock.IsUint64() {
		return 0, errors.New(fmt.Sprintf("for assertion hash %#x, createdAtBlock was not a uint64", assertionHash))

	}
	return createdAtBlock.Uint64(), nil
}

func (a *AssertionChain) TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionHash, error) {
	cm := a.SpecChallengeManager()
	edgeOpt, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	if edgeOpt.IsNone() {
		return protocol.AssertionHash{}, errors.New("edge was nil")
	}
	return edgeOpt.Unwrap().AssertionHash(ctx)
}

func (a *AssertionChain) TopLevelClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (protocol.OriginHeights, error) {
	cm := a.SpecChallengeManager()
	edgeOpt, err := cm.GetEdge(ctx, edgeId)
	if err != nil {
		return protocol.OriginHeights{}, err
	}
	if edgeOpt.IsNone() {
		return protocol.OriginHeights{}, errors.New("edge was nil")
	}
	edge := edgeOpt.Unwrap()
	return edge.TopLevelClaimHeight(ctx)
}

// ReadAssertionCreationInfo for an assertion sequence number by looking up its creation
// event from the rollup contracts.
func (a *AssertionChain) ReadAssertionCreationInfo(
	ctx context.Context, id protocol.AssertionHash,
) (*protocol.AssertionCreatedInfo, error) {
	var assertionCreationBlock uint64
	var topics [][]common.Hash
	if id == (protocol.AssertionHash{}) {
		rollupDeploymentBlock, err := a.rollup.RollupDeploymentBlock(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		if !rollupDeploymentBlock.IsUint64() {
			return nil, errors.New("rollup deployment block was not a uint64")
		}
		assertionCreationBlock = rollupDeploymentBlock.Uint64()
		topics = [][]common.Hash{{assertionCreatedId}}
	} else {
		var b [32]byte
		copy(b[:], id.Bytes())
		var err error
		assertionCreationBlock, err = a.GetAssertionCreationParentBlock(ctx, b)
		if err != nil {
			return nil, err
		}
		topics = [][]common.Hash{{assertionCreatedId}, {id.Hash}}
	}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(assertionCreationBlock),
		ToBlock:   new(big.Int).SetUint64(assertionCreationBlock),
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
	res, err := a.rollup.GetAssertion(a.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}), parsedLog.AssertionHash)
	if err != nil {
		return nil, err
	}
	creationL1Block := res.CreatedAtBlock
	return &protocol.AssertionCreatedInfo{
		ConfirmPeriodBlocks: parsedLog.ConfirmPeriodBlocks,
		RequiredStake:       parsedLog.RequiredStake,
		ParentAssertionHash: protocol.AssertionHash{Hash: parsedLog.ParentAssertionHash},
		BeforeState:         parsedLog.Assertion.BeforeState,
		AfterState:          afterState,
		InboxMaxCount:       parsedLog.InboxMaxCount,
		AfterInboxBatchAcc:  parsedLog.AfterInboxBatchAcc,
		AssertionHash:       protocol.AssertionHash{Hash: parsedLog.AssertionHash},
		WasmModuleRoot:      parsedLog.WasmModuleRoot,
		ChallengeManager:    parsedLog.ChallengeManager,
		TransactionHash:     ethLog.TxHash,
		CreationParentBlock: ethLog.BlockNumber,
		CreationL1Block:     creationL1Block,
	}, nil
}

func handleCreateAssertionError(err error, blockHash common.Hash) error {
	if err == nil {
		return nil
	}
	errS := err.Error()
	switch {
	case strings.Contains(errS, "EXPECTED_ASSERTION_SEEN"):
		return errors.Wrapf(
			ErrAlreadyExists,
			"commit block hash %#x",
			blockHash,
		)
	case strings.Contains(errS, "already known"):
		return errors.Wrapf(
			ErrAlreadyExists,
			"commit block hash %#x",
			blockHash,
		)
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

func (a *AssertionChain) GetCallOptsWithDesiredRpcHeadBlockNumber(opts *bind.CallOpts) *bind.CallOpts {
	if opts == nil {
		opts = &bind.CallOpts{}
	}
	// If we are running tests, we want to use the latest block number since
	// simulated backends only support the latest block number.
	if flag.Lookup("test.v") != nil {
		return opts
	}
	opts.BlockNumber = big.NewInt(int64(a.rpcHeadBlockNumber))
	return opts
}

func (a *AssertionChain) GetCallOptsWithSafeBlockNumber(opts *bind.CallOpts) *bind.CallOpts {
	if opts == nil {
		opts = &bind.CallOpts{}
	}
	// If we are running tests, we want to use the latest block number since
	// simulated backends only support the latest block number.
	if flag.Lookup("test.v") != nil {
		return nil
	}
	opts.BlockNumber = big.NewInt(int64(rpc.SafeBlockNumber))
	return opts
}

func (a *AssertionChain) GetDesiredRpcHeadBlockNumber() rpc.BlockNumber {
	return a.rpcHeadBlockNumber
}
