// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package bold

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/util"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/proofenhancement"
)

var (
	boldStakerBalanceGauge      = metrics.GetOrRegisterGaugeFloat64("arb/staker/balance", nil)
	boldStakerAmountStakedGauge = metrics.GetOrRegisterGauge("arb/staker/amount_staked", nil)
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

type BoldConfig struct {
	// How often to post assertions onchain.
	AssertionPostingInterval time.Duration `koanf:"assertion-posting-interval"`
	// How often to scan for newly created assertions onchain.
	AssertionScanningInterval time.Duration `koanf:"assertion-scanning-interval"`
	// How often to confirm assertions onchain.
	AssertionConfirmingInterval         time.Duration          `koanf:"assertion-confirming-interval"`
	API                                 bool                   `koanf:"api"`
	APIHost                             string                 `koanf:"api-host"`
	APIPort                             uint16                 `koanf:"api-port"`
	APIDBPath                           string                 `koanf:"api-db-path"`
	TrackChallengeParentAssertionHashes []string               `koanf:"track-challenge-parent-assertion-hashes"`
	CheckStakerSwitchInterval           time.Duration          `koanf:"check-staker-switch-interval"`
	MaxGetLogBlocks                     int64                  `koanf:"max-get-log-blocks"`
	StateProviderConfig                 StateProviderConfig    `koanf:"state-provider-config"`
	StartValidationFromStaked           bool                   `koanf:"start-validation-from-staked"`
	AutoDeposit                         bool                   `koanf:"auto-deposit"`
	AutoIncreaseAllowance               bool                   `koanf:"auto-increase-allowance"`
	DelegatedStaking                    DelegatedStakingConfig `koanf:"delegated-staking"`
	RPCBlockNumber                      string                 `koanf:"rpc-block-number"`
	EnableFastConfirmation              bool                   `koanf:"enable-fast-confirmation"`
	ParentChainBlockTime                time.Duration          `koanf:"parent-chain-block-time"`
	// How long to wait since parent assertion was created to post a new assertion
	MinimumGapToParentAssertion time.Duration `koanf:"minimum-gap-to-parent-assertion"`
	blockNum                    rpc.BlockNumber
}

func (c *BoldConfig) Validate() error {
	var blockNum rpc.BlockNumber
	switch strings.ToLower(c.RPCBlockNumber) {
	case "safe":
		blockNum = rpc.SafeBlockNumber
	case "finalized":
		blockNum = rpc.FinalizedBlockNumber
	case "latest":
		blockNum = rpc.LatestBlockNumber
	default:
		return fmt.Errorf("unknown rpc block number \"%v\", expected either latest, safe, or finalized", c.RPCBlockNumber)
	}
	c.blockNum = blockNum
	return nil
}

type DelegatedStakingConfig struct {
	Enable                  bool   `koanf:"enable"`
	CustomWithdrawalAddress string `koanf:"custom-withdrawal-address"`
}

var DefaultDelegatedStakingConfig = DelegatedStakingConfig{
	Enable:                  false,
	CustomWithdrawalAddress: "",
}

type StateProviderConfig struct {
	// A name identifier for the validator for cosmetic purposes.
	ValidatorName      string `koanf:"validator-name"`
	CheckBatchFinality bool   `koanf:"check-batch-finality"`
	// Path to a filesystem directory that will cache machine hashes for BOLD.
	MachineLeavesCachePath string `koanf:"machine-leaves-cache-path"`
}

var DefaultStateProviderConfig = StateProviderConfig{
	ValidatorName:          "default-validator",
	CheckBatchFinality:     true,
	MachineLeavesCachePath: "machine-hashes-cache",
}

var DefaultBoldConfig = BoldConfig{
	AssertionPostingInterval:            time.Minute * 15,
	AssertionScanningInterval:           time.Minute,
	AssertionConfirmingInterval:         time.Minute,
	MinimumGapToParentAssertion:         time.Minute, // Correct default?
	API:                                 false,
	APIHost:                             "127.0.0.1",
	APIPort:                             9393,
	APIDBPath:                           "bold-api-db",
	TrackChallengeParentAssertionHashes: []string{},
	CheckStakerSwitchInterval:           time.Minute, // Every minute, check if the Nitro node staker should switch to using BOLD.
	StateProviderConfig:                 DefaultStateProviderConfig,
	StartValidationFromStaked:           true,
	AutoDeposit:                         true,
	AutoIncreaseAllowance:               true,
	DelegatedStaking:                    DefaultDelegatedStakingConfig,
	ParentChainBlockTime:                time.Second * 12,
	RPCBlockNumber:                      "finalized",
	EnableFastConfirmation:              false,
	MaxGetLogBlocks:                     5000,
}

var BoldModes = map[legacystaker.StakerStrategy]types.Mode{
	legacystaker.WatchtowerStrategy:   types.WatchTowerMode,
	legacystaker.DefensiveStrategy:    types.DefensiveMode,
	legacystaker.ResolveNodesStrategy: types.ResolveMode,
	legacystaker.MakeNodesStrategy:    types.MakeMode,
}

func BoldConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".rpc-block-number", DefaultBoldConfig.RPCBlockNumber, "define the block number to use for reading data onchain, either latest, safe, or finalized")
	f.Int64(prefix+".max-get-log-blocks", DefaultBoldConfig.MaxGetLogBlocks, "maximum size for chunk of blocks when using get logs rpc")
	f.Duration(prefix+".assertion-posting-interval", DefaultBoldConfig.AssertionPostingInterval, "assertion posting interval")
	f.Duration(prefix+".assertion-scanning-interval", DefaultBoldConfig.AssertionScanningInterval, "scan assertion interval")
	f.Duration(prefix+".assertion-confirming-interval", DefaultBoldConfig.AssertionConfirmingInterval, "confirm assertion interval")
	f.Duration(prefix+".minimum-gap-to-parent-assertion", DefaultBoldConfig.MinimumGapToParentAssertion, "minimum duration to wait since the parent assertion was created to post a new assertion")
	f.Duration(prefix+".check-staker-switch-interval", DefaultBoldConfig.CheckStakerSwitchInterval, "how often to check if staker can switch to bold")
	f.Duration(prefix+".parent-chain-block-time", DefaultBoldConfig.ParentChainBlockTime, "the average block time of the parent chain where assertions are posted")
	f.Bool(prefix+".api", DefaultBoldConfig.API, "enable api")
	f.String(prefix+".api-host", DefaultBoldConfig.APIHost, "bold api host")
	f.Uint16(prefix+".api-port", DefaultBoldConfig.APIPort, "bold api port")
	f.String(prefix+".api-db-path", DefaultBoldConfig.APIDBPath, "bold api db path")
	f.StringSlice(prefix+".track-challenge-parent-assertion-hashes", DefaultBoldConfig.TrackChallengeParentAssertionHashes, "only track challenges/edges with these parent assertion hashes")
	StateProviderConfigAddOptions(prefix+".state-provider-config", f)
	f.Bool(prefix+".start-validation-from-staked", DefaultBoldConfig.StartValidationFromStaked, "assume staked nodes are valid")
	f.Bool(prefix+".auto-deposit", DefaultBoldConfig.AutoDeposit, "auto-deposit stake token whenever making a move in BoLD that does not have enough stake token balance")
	f.Bool(prefix+".auto-increase-allowance", DefaultBoldConfig.AutoIncreaseAllowance, "auto-increase spending allowance of the stake token by the rollup and challenge manager contracts")
	DelegatedStakingConfigAddOptions(prefix+".delegated-staking", f)
	f.Bool(prefix+".enable-fast-confirmation", DefaultBoldConfig.EnableFastConfirmation, "enable fast confirmation")
}

func StateProviderConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".validator-name", DefaultStateProviderConfig.ValidatorName, "name identifier for cosmetic purposes")
	f.Bool(prefix+".check-batch-finality", DefaultStateProviderConfig.CheckBatchFinality, "check batch finality")
	f.String(prefix+".machine-leaves-cache-path", DefaultStateProviderConfig.MachineLeavesCachePath, "path to machine cache")
}

func DelegatedStakingConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultDelegatedStakingConfig.Enable, "enable delegated staking by having the validator call newStake on startup")
	f.String(prefix+".custom-withdrawal-address", DefaultDelegatedStakingConfig.CustomWithdrawalAddress, "enable a custom withdrawal address for staking on the rollup contract, useful for delegated stakers")
}

type BOLDStaker struct {
	stopwaiter.StopWaiter
	config             *BoldConfig
	strategy           legacystaker.StakerStrategy
	chalManager        *challengemanager.Manager
	blockValidator     *staker.BlockValidator
	rollupAddress      common.Address
	l1Reader           *headerreader.HeaderReader
	client             *util.BackendWrapper
	callOpts           bind.CallOpts
	wallet             legacystaker.ValidatorWalletInterface
	stakedNotifiers    []legacystaker.LatestStakedNotifier
	confirmedNotifiers []legacystaker.LatestConfirmedNotifier
	inboxTracker       staker.InboxTrackerInterface
	inboxStreamer      staker.TransactionStreamerInterface
	fatalErr           chan<- error
}

func NewBOLDStaker(
	ctx context.Context,
	stack *node.Node,
	rollupAddress common.Address,
	callOpts bind.CallOpts,
	txOpts *bind.TransactOpts,
	l1Reader *headerreader.HeaderReader,
	blockValidator *staker.BlockValidator,
	statelessBlockValidator *staker.StatelessBlockValidator,
	config *BoldConfig,
	strategy legacystaker.StakerStrategy,
	dataPoster *dataposter.DataPoster,
	wallet legacystaker.ValidatorWalletInterface,
	stakedNotifiers []legacystaker.LatestStakedNotifier,
	confirmedNotifiers []legacystaker.LatestConfirmedNotifier,
	inboxTracker staker.InboxTrackerInterface,
	inboxStreamer staker.TransactionStreamerInterface,
	inboxReader staker.InboxReaderInterface,
	dapRegistry *daprovider.DAProviderRegistry,
	fatalErr chan<- error,
) (*BOLDStaker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create proof enhancer if registry is available
	var proofEnhancer proofenhancement.ProofEnhancer
	if dapRegistry != nil {
		proofEnhancer = proofenhancement.NewCustomDAProofEnhancer(dapRegistry, inboxTracker, inboxReader)
	}

	wrappedClient := util.NewBackendWrapper(l1Reader.Client(), rpc.LatestBlockNumber)
	manager, err := newBOLDChallengeManager(ctx, stack, rollupAddress, txOpts, l1Reader, wrappedClient, blockValidator, statelessBlockValidator, config, strategy, dataPoster, inboxTracker, inboxStreamer, inboxReader, proofEnhancer)
	if err != nil {
		return nil, err
	}
	return &BOLDStaker{
		config:             config,
		strategy:           strategy,
		chalManager:        manager,
		blockValidator:     blockValidator,
		rollupAddress:      rollupAddress,
		l1Reader:           l1Reader,
		client:             wrappedClient,
		callOpts:           callOpts,
		wallet:             wallet,
		stakedNotifiers:    stakedNotifiers,
		confirmedNotifiers: confirmedNotifiers,
		inboxTracker:       inboxTracker,
		inboxStreamer:      inboxStreamer,
		fatalErr:           fatalErr,
	}, nil
}

// Initialize Updates the block validator module root.
// And updates the init state of the block validator if block validator has not started yet.
func (b *BOLDStaker) Initialize(ctx context.Context) error {
	err := b.updateBlockValidatorModuleRoot(ctx)
	if err != nil {
		log.Warn("error updating latest wasm module root", "err", err)
	}
	walletAddressOrZero := b.wallet.AddressOrZero()
	var stakerAddr common.Address
	if b.wallet.DataPoster() != nil {
		stakerAddr = b.wallet.DataPoster().Sender()
	}
	log.Info("running as validator", "txSender", stakerAddr, "actingAsWallet", walletAddressOrZero, "strategy", b.strategy.ToString())

	if b.blockValidator != nil && b.config.StartValidationFromStaked && !b.blockValidator.Started() {
		rollupUserLogic, err := rollupgen.NewRollupUserLogic(b.rollupAddress, b.client)
		if err != nil {
			return err
		}
		latestStaked, err := rollupUserLogic.LatestStakedAssertion(b.getCallOpts(ctx), walletAddressOrZero)
		if err != nil {
			return err
		}
		if latestStaked == [32]byte{} {
			latestConfirmed, err := rollupUserLogic.LatestConfirmed(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}
			latestStaked = latestConfirmed
		}
		assertion, err := ReadBoldAssertionCreationInfo(
			ctx,
			rollupUserLogic,
			b.client,
			b.rollupAddress,
			latestStaked,
		)
		if err != nil {
			return err
		}
		afterState := protocol.GoGlobalStateFromSolidity(assertion.AfterState.GlobalState)
		return b.blockValidator.InitAssumeValid(validator.GoGlobalState(afterState))
	}
	return nil
}

func (b *BOLDStaker) Start(ctxIn context.Context) {
	b.StopWaiter.Start(ctxIn, b)
	b.chalManager.Start(ctxIn)
	b.CallIteratively(func(ctx context.Context) time.Duration {
		err := b.updateBlockValidatorModuleRoot(ctx)
		if err != nil {
			log.Warn("error updating latest wasm module root", "err", err)
		}
		confirmedMsgCount, confirmedGlobalState, err := b.getLatestState(ctx, true)
		if err != nil {
			log.Error("staker: error checking latest confirmed", "err", err)
			if errors.Is(err, staker.ErrGlobalStateNotInChain) {
				b.fatalErr <- err
			}
			return b.config.AssertionPostingInterval
		}

		agreedMsgCount, agreedGlobalState, err := b.getLatestState(ctx, false)
		if err != nil {
			log.Error("staker: error checking latest agreed", "err", err)
		}

		if agreedGlobalState == nil {
			// If we don't have a latest agreed global state, we should fall back to
			// using the latest confirmed global state.
			agreedGlobalState = confirmedGlobalState
			agreedMsgCount = confirmedMsgCount
		}
		if agreedGlobalState != nil {
			for _, notifier := range b.stakedNotifiers {
				notifier.UpdateLatestStaked(agreedMsgCount, *agreedGlobalState)
			}
		}

		if confirmedGlobalState != nil {
			for _, notifier := range b.confirmedNotifiers {
				notifier.UpdateLatestConfirmed(confirmedMsgCount, *confirmedGlobalState)
			}
		}
		err = b.updateStakerBalanceMetric(ctx)
		if err != nil {
			log.Warn("error updating staker balance metric", "err", err)
		}
		return b.config.AssertionPostingInterval
	})
}

func (b *BOLDStaker) updateStakerBalanceMetric(ctx context.Context) error {
	walletAddressOrZero := b.wallet.AddressOrZero()
	if walletAddressOrZero != (common.Address{}) {
		rollupUserLogic, err := rollupgen.NewRollupUserLogic(b.rollupAddress, b.client)
		if err != nil {
			return fmt.Errorf("error creating rollup user logic: %w", err)
		}
		amountStaked, err := rollupUserLogic.AmountStaked(&bind.CallOpts{Context: ctx}, walletAddressOrZero)
		if err != nil {
			return fmt.Errorf("error getting amount staked: %w", err)
		}
		boldStakerAmountStakedGauge.Update(arbmath.BigDivByUint(amountStaked, params.Ether).Int64())
	} else {
		boldStakerAmountStakedGauge.Update(0)
	}

	txSenderAddress := b.wallet.TxSenderAddress()
	if txSenderAddress != nil {
		balance, err := b.client.BalanceAt(ctx, *txSenderAddress, nil)
		if err != nil {
			return fmt.Errorf("error getting balance for %v: %w", txSenderAddress, err)
		}
		boldStakerBalanceGauge.Update(arbmath.BalancePerEther(balance))
	} else {
		boldStakerBalanceGauge.Update(0)
	}
	return nil
}

func (b *BOLDStaker) getLatestState(ctx context.Context, confirmed bool) (arbutil.MessageIndex, *validator.GoGlobalState, error) {
	var globalState protocol.GoGlobalState
	var err error
	if confirmed {
		globalState, err = b.chalManager.LatestConfirmedState(ctx)
	} else {
		globalState, err = b.chalManager.LatestAgreedState(ctx)
	}
	var assertionType string
	if confirmed {
		assertionType = "confirmed"
	} else {
		assertionType = "agreed"
	}
	if err != nil {
		return 0, nil, fmt.Errorf("error getting latest %s: %w", assertionType, err)
	}
	caughtUp, count, err := staker.GlobalStateToMsgCount(b.inboxTracker, b.inboxStreamer, validator.GoGlobalState(globalState))
	if err != nil {
		if errors.Is(err, staker.ErrGlobalStateNotInChain) {
			return 0, nil, fmt.Errorf("latest %s assertion of %v not yet in our node: %w", assertionType, globalState, err)
		}
		log.Error("error getting message count", "err", err)
		return 0, nil, nil
	}

	if !caughtUp {
		log.Info(fmt.Sprintf("latest %s assertion not yet in our node", assertionType), "state", globalState)
		return 0, nil, nil
	}

	processedCount, err := b.inboxStreamer.GetProcessedMessageCount()
	if err != nil {
		log.Error("error getting processed message count", "err", err)
		return 0, nil, nil
	}

	if processedCount < count {
		log.Info("execution catching up to rollup", "rollupCount", count, "processedCount", processedCount)
		return 0, nil, nil
	}

	return count, (*validator.GoGlobalState)(&globalState), nil
}

func (b *BOLDStaker) StopAndWait() {
	b.chalManager.StopAndWait()
	b.StopWaiter.StopAndWait()
}

func (b *BOLDStaker) updateBlockValidatorModuleRoot(ctx context.Context) error {
	if b.blockValidator == nil {
		return nil
	}
	boldRollup, err := rollupgen.NewRollupUserLogic(b.rollupAddress, b.client)
	if err != nil {
		return err
	}
	moduleRoot, err := boldRollup.WasmModuleRoot(b.getCallOpts(ctx))
	if err != nil {
		return err
	}
	return b.blockValidator.SetCurrentWasmModuleRoot(moduleRoot)
}

func (b *BOLDStaker) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := b.callOpts
	opts.Context = ctx
	return &opts
}

// Sets up a BOLD challenge manager implementation by providing it with
// its necessary dependencies and configuration. The challenge manager can then be started, as it
// implements the StopWaiter pattern as part of the Nitro validator.
func newBOLDChallengeManager(
	ctx context.Context,
	stack *node.Node,
	rollupAddress common.Address,
	txOpts *bind.TransactOpts,
	l1Reader *headerreader.HeaderReader,
	client protocol.ChainBackend,
	blockValidator *staker.BlockValidator,
	statelessBlockValidator *staker.StatelessBlockValidator,
	config *BoldConfig,
	strategy legacystaker.StakerStrategy,
	dataPoster *dataposter.DataPoster,
	inboxTracker staker.InboxTrackerInterface,
	inboxStreamer staker.TransactionStreamerInterface,
	inboxReader staker.InboxReaderInterface,
	proofEnhancer proofenhancement.ProofEnhancer,
) (*challengemanager.Manager, error) {
	// Initializes the BOLD contract bindings and the assertion chain abstraction.
	rollupBindings, err := rollupgen.NewRollupUserLogic(rollupAddress, client)
	if err != nil {
		return nil, fmt.Errorf("could not create rollup bindings: %w", err)
	}
	chalManager, err := rollupBindings.ChallengeManager(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get challenge manager: %w", err)
	}
	chalManagerBindings, err := challengeV2gen.NewEdgeChallengeManager(chalManager, client)
	if err != nil {
		return nil, fmt.Errorf("could not create challenge manager bindings: %w", err)
	}
	assertionChainOpts := []solimpl.Opt{
		solimpl.WithRpcHeadBlockNumber(config.blockNum),
		solimpl.WithParentChainBlockCreationTime(config.ParentChainBlockTime),
	}
	if config.DelegatedStaking.Enable && config.DelegatedStaking.CustomWithdrawalAddress != "" {
		withdrawalAddr := common.HexToAddress(config.DelegatedStaking.CustomWithdrawalAddress)
		assertionChainOpts = append(assertionChainOpts, solimpl.WithCustomWithdrawalAddress(withdrawalAddr))
	}
	if !config.AutoDeposit {
		assertionChainOpts = append(assertionChainOpts, solimpl.WithoutAutoDeposit())
	}

	if config.EnableFastConfirmation {
		assertionChainOpts = append(assertionChainOpts, solimpl.WithFastConfirmation())
	}
	assertionChain, err := solimpl.NewAssertionChain(
		ctx,
		rollupAddress,
		chalManager,
		txOpts,
		client,
		NewDataPosterTransactor(dataPoster),
		assertionChainOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create assertion chain: %w", err)
	}

	blockChallengeHeightBig, err := chalManagerBindings.LAYERZEROBLOCKEDGEHEIGHT(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get block challenge height: %w", err)
	}
	if !blockChallengeHeightBig.IsUint64() {
		return nil, errors.New("block challenge height was not a uint64")
	}
	bigStepHeightBig, err := chalManagerBindings.LAYERZEROBIGSTEPEDGEHEIGHT(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get big step challenge height: %w", err)
	}
	if !bigStepHeightBig.IsUint64() {
		return nil, errors.New("big step challenge height was not a uint64")
	}
	smallStepHeightBig, err := chalManagerBindings.LAYERZEROSMALLSTEPEDGEHEIGHT(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get small step challenge height: %w", err)
	}
	if !smallStepHeightBig.IsUint64() {
		return nil, errors.New("small step challenge height was not a uint64")
	}
	numBigSteps, err := chalManagerBindings.NUMBIGSTEPLEVEL(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get number of big steps: %w", err)
	}
	blockChallengeLeafHeight := l2stateprovider.Height(blockChallengeHeightBig.Uint64())
	bigStepHeight := l2stateprovider.Height(bigStepHeightBig.Uint64())
	smallStepHeight := l2stateprovider.Height(smallStepHeightBig.Uint64())

	apiDBPath := config.APIDBPath
	if apiDBPath != "" {
		apiDBPath = stack.ResolvePath(apiDBPath)
	}
	machineHashesPath := config.StateProviderConfig.MachineLeavesCachePath
	if machineHashesPath != "" {
		machineHashesPath = stack.ResolvePath(machineHashesPath)
	}

	// Sets up the state provider interface that BOLD will use to request data such as
	// execution states for assertions, history commitments for machine execution, and one step proofs.
	stateProvider, err := NewBOLDStateProvider(
		blockValidator,
		statelessBlockValidator,
		// Specify the height constants needed for the state provider.
		// TODO: Fetch these from the smart contract instead.
		blockChallengeLeafHeight,
		&config.StateProviderConfig,
		machineHashesPath,
		inboxTracker,
		inboxStreamer,
		inboxReader,
		proofEnhancer,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create state manager: %w", err)
	}
	providerHeights := []l2stateprovider.Height{blockChallengeLeafHeight}
	for i := uint8(0); i < numBigSteps; i++ {
		providerHeights = append(providerHeights, bigStepHeight)
	}
	providerHeights = append(providerHeights, smallStepHeight)
	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateProvider,
		stateProvider,
		stateProvider,
		providerHeights,
		stateProvider,
		nil, // Nil API database for the history commitment provider, as it will be provided later. TODO: Improve this dependency injection.
	)
	// The interval at which the challenge manager will attempt to post assertions.
	postingInterval := config.AssertionPostingInterval
	// The interval at which the manager will scan for newly created assertions onchain.
	scanningInterval := config.AssertionScanningInterval
	// The interval at which the manager will attempt to confirm assertions.
	confirmingInterval := config.AssertionConfirmingInterval

	stackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithName(config.StateProviderConfig.ValidatorName),
		challengemanager.StackWithMode(BoldModes[strategy]),
		challengemanager.StackWithPollingInterval(scanningInterval),
		challengemanager.StackWithPostingInterval(postingInterval),
		challengemanager.StackWithConfirmationInterval(confirmingInterval),
		challengemanager.StackWithMinimumGapToParentAssertion(config.MinimumGapToParentAssertion),
		challengemanager.StackWithTrackChallengeParentAssertionHashes(config.TrackChallengeParentAssertionHashes),
		challengemanager.StackWithHeaderProvider(l1Reader),
		challengemanager.StackWithAverageBlockCreationTime(config.ParentChainBlockTime),
		challengemanager.StackWithSyncMaxGetLogBlocks(config.MaxGetLogBlocks),
	}
	if config.API {
		apiAddr := fmt.Sprintf("%s:%d", config.APIHost, config.APIPort)
		stackOpts = append(stackOpts, challengemanager.StackWithAPIEnabled(apiAddr, apiDBPath))
	}
	if !config.AutoDeposit {
		stackOpts = append(stackOpts, challengemanager.StackWithoutAutoDeposit())
	}
	if !config.AutoIncreaseAllowance {
		stackOpts = append(stackOpts, challengemanager.StackWithoutAutoAllowanceApproval())
	}
	if config.DelegatedStaking.Enable {
		stackOpts = append(stackOpts, challengemanager.StackWithDelegatedStaking())
	}
	if config.EnableFastConfirmation {
		stackOpts = append(stackOpts, challengemanager.StackWithFastConfirmationEnabled())
	}

	manager, err := challengemanager.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create challenge manager: %w", err)
	}
	return manager, nil
}

// Read the creation info for an assertion by looking up its creation
// event from the rollup contracts.
func ReadBoldAssertionCreationInfo(
	ctx context.Context,
	rollup *rollupgen.RollupUserLogic,
	client bind.ContractBackend,
	rollupAddress common.Address,
	assertionHash common.Hash,
) (*protocol.AssertionCreatedInfo, error) {
	var creationBlock uint64
	var topics [][]common.Hash
	if assertionHash == (common.Hash{}) {
		rollupDeploymentBlock, err := rollup.RollupDeploymentBlock(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		if !rollupDeploymentBlock.IsUint64() {
			return nil, errors.New("rollup deployment block was not a uint64")
		}
		creationBlock = rollupDeploymentBlock.Uint64()
	} else {
		var b [32]byte
		copy(b[:], assertionHash[:])
		assertionCreationBlock, err := rollup.GetAssertionCreationBlockForLogLookup(&bind.CallOpts{Context: ctx}, b)
		if err != nil {
			return nil, err
		}
		if !assertionCreationBlock.IsUint64() {
			return nil, errors.New("assertion creation block was not a uint64")
		}
		creationBlock = assertionCreationBlock.Uint64()
	}
	topics = [][]common.Hash{{assertionCreatedId}, {assertionHash}}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(creationBlock),
		ToBlock:   new(big.Int).SetUint64(creationBlock),
		Addresses: []common.Address{rollupAddress},
		Topics:    topics,
	}
	logs, err := client.FilterLogs(ctx, query)
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
	parsedLog, err := rollup.ParseAssertionCreated(ethLog)
	if err != nil {
		return nil, err
	}
	afterState := parsedLog.Assertion.AfterState
	creationL1Block, err := arbutil.CorrespondingL1BlockNumber(ctx, client, ethLog.BlockNumber)
	if err != nil {
		return nil, err
	}
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
