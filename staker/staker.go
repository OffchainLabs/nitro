// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	flag "github.com/spf13/pflag"

	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/staker/txbuilder"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

var (
	stakerBalanceGauge              = metrics.NewRegisteredGaugeFloat64("arb/staker/balance", nil)
	stakerAmountStakedGauge         = metrics.NewRegisteredGauge("arb/staker/amount_staked", nil)
	stakerLatestStakedNodeGauge     = metrics.NewRegisteredGauge("arb/staker/staked_node", nil)
	stakerLatestConfirmedNodeGauge  = metrics.NewRegisteredGauge("arb/staker/confirmed_node", nil)
	stakerLastSuccessfulActionGauge = metrics.NewRegisteredGauge("arb/staker/action/last_success", nil)
	stakerActionSuccessCounter      = metrics.NewRegisteredCounter("arb/staker/action/success", nil)
	stakerActionFailureCounter      = metrics.NewRegisteredCounter("arb/staker/action/failure", nil)
	validatorGasRefunderBalance     = metrics.NewRegisteredGaugeFloat64("arb/validator/gasrefunder/balanceether", nil)
)

type StakerStrategy uint8

const (
	// Watchtower: don't do anything on L1, but log if there's a bad assertion
	WatchtowerStrategy StakerStrategy = iota
	// Defensive: stake if there's a bad assertion
	DefensiveStrategy
	// Stake latest: stay staked on the latest node, challenging bad assertions
	StakeLatestStrategy
	// Resolve nodes: stay staked on the latest node and resolve any unconfirmed nodes, challenging bad assertions
	ResolveNodesStrategy
	// Make nodes: continually create new nodes, challenging bad assertions
	MakeNodesStrategy
)

type L1PostingStrategy struct {
	HighGasThreshold   float64 `koanf:"high-gas-threshold"`
	HighGasDelayBlocks int64   `koanf:"high-gas-delay-blocks"`
}

var DefaultL1PostingStrategy = L1PostingStrategy{
	HighGasThreshold:   0,
	HighGasDelayBlocks: 0,
}

func L1PostingStrategyAddOptions(prefix string, f *flag.FlagSet) {
	f.Float64(prefix+".high-gas-threshold", DefaultL1PostingStrategy.HighGasThreshold, "high gas threshold")
	f.Int64(prefix+".high-gas-delay-blocks", DefaultL1PostingStrategy.HighGasDelayBlocks, "high gas delay blocks")
}

type L1ValidatorConfig struct {
	Enable                    bool                        `koanf:"enable"`
	Bold                      BoldConfig                  `koanf:"bold"`
	Strategy                  string                      `koanf:"strategy"`
	StakerInterval            time.Duration               `koanf:"staker-interval"`
	MakeAssertionInterval     time.Duration               `koanf:"make-assertion-interval"`
	PostingStrategy           L1PostingStrategy           `koanf:"posting-strategy"`
	DisableChallenge          bool                        `koanf:"disable-challenge"`
	ConfirmationBlocks        int64                       `koanf:"confirmation-blocks"`
	UseSmartContractWallet    bool                        `koanf:"use-smart-contract-wallet"`
	OnlyCreateWalletContract  bool                        `koanf:"only-create-wallet-contract"`
	StartValidationFromStaked bool                        `koanf:"start-validation-from-staked"`
	ContractWalletAddress     string                      `koanf:"contract-wallet-address"`
	GasRefunderAddress        string                      `koanf:"gas-refunder-address"`
	DataPoster                dataposter.DataPosterConfig `koanf:"data-poster" reload:"hot"`
	RedisUrl                  string                      `koanf:"redis-url"`
	ExtraGas                  uint64                      `koanf:"extra-gas" reload:"hot"`
	Dangerous                 DangerousConfig             `koanf:"dangerous"`
	ParentChainWallet         genericconf.WalletConfig    `koanf:"parent-chain-wallet"`

	strategy    StakerStrategy
	gasRefunder common.Address
}

func (c *L1ValidatorConfig) ParseStrategy() (StakerStrategy, error) {
	switch strings.ToLower(c.Strategy) {
	case "watchtower":
		return WatchtowerStrategy, nil
	case "defensive":
		return DefensiveStrategy, nil
	case "stakelatest":
		return StakeLatestStrategy, nil
	case "resolvenodes":
		return ResolveNodesStrategy, nil
	case "makenodes":
		return MakeNodesStrategy, nil
	default:
		return WatchtowerStrategy, fmt.Errorf("unknown staker strategy \"%v\"", c.Strategy)
	}
}

func (c *L1ValidatorConfig) ValidatorRequired() bool {
	if !c.Enable {
		return false
	}
	if c.Dangerous.WithoutBlockValidator {
		return false
	}
	if c.strategy == WatchtowerStrategy {
		return false
	}
	return true
}

func (c *L1ValidatorConfig) Validate() error {
	strategy, err := c.ParseStrategy()
	if err != nil {
		return err
	}
	c.strategy = strategy
	if len(c.GasRefunderAddress) > 0 && !common.IsHexAddress(c.GasRefunderAddress) {
		return errors.New("invalid validator gas refunder address")
	}
	c.gasRefunder = common.HexToAddress(c.GasRefunderAddress)
	return nil
}

var DefaultL1ValidatorConfig = L1ValidatorConfig{
	Enable:                    true,
	Bold:                      DefaultBoldConfig,
	Strategy:                  "Watchtower",
	StakerInterval:            time.Minute,
	MakeAssertionInterval:     time.Hour,
	PostingStrategy:           L1PostingStrategy{},
	DisableChallenge:          false,
	ConfirmationBlocks:        12,
	UseSmartContractWallet:    false,
	OnlyCreateWalletContract:  false,
	StartValidationFromStaked: true,
	ContractWalletAddress:     "",
	GasRefunderAddress:        "",
	DataPoster:                dataposter.DefaultDataPosterConfigForValidator,
	RedisUrl:                  "",
	ExtraGas:                  50000,
	Dangerous:                 DefaultDangerousConfig,
	ParentChainWallet:         DefaultValidatorL1WalletConfig,
}

var TestL1ValidatorConfig = L1ValidatorConfig{
	Enable:                    true,
	Strategy:                  "Watchtower",
	StakerInterval:            time.Millisecond * 10,
	MakeAssertionInterval:     -time.Hour * 1000,
	PostingStrategy:           L1PostingStrategy{},
	DisableChallenge:          false,
	ConfirmationBlocks:        0,
	UseSmartContractWallet:    false,
	OnlyCreateWalletContract:  false,
	StartValidationFromStaked: true,
	ContractWalletAddress:     "",
	GasRefunderAddress:        "",
	DataPoster:                dataposter.TestDataPosterConfigForValidator,
	RedisUrl:                  "",
	ExtraGas:                  50000,
	Dangerous:                 DefaultDangerousConfig,
	ParentChainWallet:         DefaultValidatorL1WalletConfig,
}

var DefaultValidatorL1WalletConfig = genericconf.WalletConfig{
	Pathname:      "validator-wallet",
	Password:      genericconf.WalletConfigDefault.Password,
	PrivateKey:    genericconf.WalletConfigDefault.PrivateKey,
	Account:       genericconf.WalletConfigDefault.Account,
	OnlyCreateKey: genericconf.WalletConfigDefault.OnlyCreateKey,
}

func L1ValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultL1ValidatorConfig.Enable, "enable validator")
	f.String(prefix+".strategy", DefaultL1ValidatorConfig.Strategy, "L1 validator strategy, either watchtower, defensive, stakeLatest, or makeNodes")
	f.Duration(prefix+".staker-interval", DefaultL1ValidatorConfig.StakerInterval, "how often the L1 validator should check the status of the L1 rollup and maybe take action with its stake")
	f.Duration(prefix+".make-assertion-interval", DefaultL1ValidatorConfig.MakeAssertionInterval, "if configured with the makeNodes strategy, how often to create new assertions (bypassed in case of a dispute)")
	L1PostingStrategyAddOptions(prefix+".posting-strategy", f)
	BoldConfigAddOptions(prefix+".bold", f)
	f.Bool(prefix+".disable-challenge", DefaultL1ValidatorConfig.DisableChallenge, "disable validator challenge")
	f.Int64(prefix+".confirmation-blocks", DefaultL1ValidatorConfig.ConfirmationBlocks, "confirmation blocks")
	f.Bool(prefix+".use-smart-contract-wallet", DefaultL1ValidatorConfig.UseSmartContractWallet, "use a smart contract wallet instead of an EOA address")
	f.Bool(prefix+".only-create-wallet-contract", DefaultL1ValidatorConfig.OnlyCreateWalletContract, "only create smart wallet contract and exit")
	f.Bool(prefix+".start-validation-from-staked", DefaultL1ValidatorConfig.StartValidationFromStaked, "assume staked nodes are valid")
	f.String(prefix+".contract-wallet-address", DefaultL1ValidatorConfig.ContractWalletAddress, "validator smart contract wallet public address")
	f.String(prefix+".gas-refunder-address", DefaultL1ValidatorConfig.GasRefunderAddress, "The gas refunder contract address (optional)")
	f.String(prefix+".redis-url", DefaultL1ValidatorConfig.RedisUrl, "redis url for L1 validator")
	f.Uint64(prefix+".extra-gas", DefaultL1ValidatorConfig.ExtraGas, "use this much more gas than estimation says is necessary to post transactions")
	dataposter.DataPosterConfigAddOptions(prefix+".data-poster", f, dataposter.DefaultDataPosterConfigForValidator)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	genericconf.WalletConfigAddOptions(prefix+".parent-chain-wallet", f, DefaultL1ValidatorConfig.ParentChainWallet.Pathname)
}

type DangerousConfig struct {
	IgnoreRollupWasmModuleRoot bool `koanf:"ignore-rollup-wasm-module-root"`
	WithoutBlockValidator      bool `koanf:"without-block-validator"`
}

var DefaultDangerousConfig = DangerousConfig{
	IgnoreRollupWasmModuleRoot: false,
	WithoutBlockValidator:      false,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".ignore-rollup-wasm-module-root", DefaultL1ValidatorConfig.Dangerous.IgnoreRollupWasmModuleRoot, "DANGEROUS! make assertions even when the wasm module root is wrong")
	f.Bool(prefix+".without-block-validator", DefaultL1ValidatorConfig.Dangerous.WithoutBlockValidator, "DANGEROUS! allows running an L1 validator without a block validator")
}

type nodeAndHash struct {
	id   uint64
	hash common.Hash
}

type LatestStakedNotifier interface {
	UpdateLatestStaked(count arbutil.MessageIndex, globalState validator.GoGlobalState)
}

type LatestConfirmedNotifier interface {
	UpdateLatestConfirmed(count arbutil.MessageIndex, globalState validator.GoGlobalState)
}

type Staker struct {
	*L1Validator
	stopwaiter.StopWaiter
	l1Reader                *headerreader.HeaderReader
	stakedNotifiers         []LatestStakedNotifier
	confirmedNotifiers      []LatestConfirmedNotifier
	activeChallenge         *ChallengeManager
	baseCallOpts            bind.CallOpts
	config                  L1ValidatorConfig
	highGasBlocksBuffer     *big.Int
	lastActCalledBlock      *big.Int
	inactiveLastCheckedNode *nodeAndHash
	bringActiveUntilNode    uint64
	inboxReader             InboxReaderInterface
	statelessBlockValidator *StatelessBlockValidator
	bridge                  *bridgegen.IBridge
	fatalErr                chan<- error
}

type ValidatorWalletInterface interface {
	Initialize(context.Context) error
	// Address must be able to be called concurrently with other functions
	Address() *common.Address
	// Address must be able to be called concurrently with other functions
	AddressOrZero() common.Address
	TxSenderAddress() *common.Address
	RollupAddress() common.Address
	ChallengeManagerAddress() common.Address
	L1Client() arbutil.L1Interface
	TestTransactions(context.Context, []*types.Transaction) error
	ExecuteTransactions(context.Context, *txbuilder.Builder, common.Address) (*types.Transaction, error)
	TimeoutChallenges(context.Context, []uint64) (*types.Transaction, error)
	CanBatchTxs() bool
	AuthIfEoa() *bind.TransactOpts
	Start(context.Context)
	StopAndWait()
	// May be nil
	DataPoster() *dataposter.DataPoster
}

func NewStaker(
	l1Reader *headerreader.HeaderReader,
	wallet ValidatorWalletInterface,
	callOpts bind.CallOpts,
	config L1ValidatorConfig,
	blockValidator *BlockValidator,
	statelessBlockValidator *StatelessBlockValidator,
	stakedNotifiers []LatestStakedNotifier,
	confirmedNotifiers []LatestConfirmedNotifier,
	validatorUtilsAddress common.Address,
	bridgeAddress common.Address,
	fatalErr chan<- error,
) (*Staker, error) {

	if err := config.Validate(); err != nil {
		return nil, err
	}
	client := l1Reader.Client()
	val, err := NewL1Validator(client, wallet, validatorUtilsAddress, callOpts,
		statelessBlockValidator.inboxTracker, statelessBlockValidator.streamer, blockValidator)
	if err != nil {
		return nil, err
	}
	stakerLastSuccessfulActionGauge.Update(time.Now().Unix())
	if config.StartValidationFromStaked && blockValidator != nil {
		stakedNotifiers = append(stakedNotifiers, blockValidator)
	}
	var bridge *bridgegen.IBridge
	if config.Bold.Enable {
		bridge, err = bridgegen.NewIBridge(bridgeAddress, client)
		if err != nil {
			return nil, err
		}
	}
	return &Staker{
		L1Validator:             val,
		l1Reader:                l1Reader,
		stakedNotifiers:         stakedNotifiers,
		confirmedNotifiers:      confirmedNotifiers,
		baseCallOpts:            callOpts,
		config:                  config,
		highGasBlocksBuffer:     big.NewInt(config.PostingStrategy.HighGasDelayBlocks),
		lastActCalledBlock:      nil,
		inboxReader:             statelessBlockValidator.inboxReader,
		statelessBlockValidator: statelessBlockValidator,
		bridge:                  bridge,
		fatalErr:                fatalErr,
	}, nil
}

func (s *Staker) Initialize(ctx context.Context) error {
	walletAddressOrZero := s.wallet.AddressOrZero()
	if walletAddressOrZero != (common.Address{}) {
		s.updateStakerBalanceMetric(ctx)
	}
	err := s.L1Validator.Initialize(ctx)
	if err != nil {
		return err
	}
	if s.blockValidator != nil && s.config.StartValidationFromStaked {
		latestStaked, _, err := s.validatorUtils.LatestStaked(&s.baseCallOpts, s.rollupAddress, *s.wallet.Address())
		if err != nil {
			return err
		}
		stakerLatestStakedNodeGauge.Update(int64(latestStaked))
		if latestStaked == 0 {
			return nil
		}
		stakedInfo, err := s.rollup.LookupNode(ctx, latestStaked)
		if err != nil {
			return err
		}
		return s.blockValidator.InitAssumeValid(stakedInfo.AfterState().GlobalState)
	}
	return nil
}

func (s *Staker) getLatestStakedState(ctx context.Context, staker common.Address) (uint64, arbutil.MessageIndex, *validator.GoGlobalState, error) {
	callOpts := s.getCallOpts(ctx)
	if s.l1Reader.UseFinalityData() {
		callOpts.BlockNumber = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
	latestStaked, _, err := s.validatorUtils.LatestStaked(s.getCallOpts(ctx), s.rollupAddress, staker)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("couldn't get LatestStaked(%v): %w", staker, err)
	}
	if latestStaked == 0 {
		return latestStaked, 0, nil, nil
	}

	stakedInfo, err := s.rollup.LookupNode(ctx, latestStaked)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("couldn't look up latest assertion of %v (%v): %w", staker, latestStaked, err)
	}

	globalState := stakedInfo.AfterState().GlobalState
	caughtUp, count, err := GlobalStateToMsgCount(s.inboxTracker, s.txStreamer, globalState)
	if err != nil {
		if errors.Is(err, ErrGlobalStateNotInChain) && s.fatalErr != nil {
			fatal := fmt.Errorf("latest assertion of %v (%v) not in chain: %w", staker, latestStaked, err)
			s.fatalErr <- fatal
		}
		return 0, 0, nil, fmt.Errorf("latest assertion of %v (%v): %w", staker, latestStaked, err)
	}

	if !caughtUp {
		log.Info("latest assertion not yet in our node", "staker", staker, "assertion", latestStaked, "state", globalState)
		return latestStaked, 0, nil, nil
	}

	processedCount, err := s.txStreamer.GetProcessedMessageCount()
	if err != nil {
		return 0, 0, nil, err
	}

	if processedCount < count {
		log.Info("execution catching up to rollup", "staker", staker, "rollupCount", count, "processedCount", processedCount)
		return latestStaked, 0, nil, nil
	}

	return latestStaked, count, &globalState, nil
}

func (s *Staker) StopAndWait() {
	s.StopWaiter.StopAndWait()
	if s.Strategy() != WatchtowerStrategy {
		s.wallet.StopAndWait()
	}
}

func (s *Staker) Start(ctxIn context.Context) {
	if s.Strategy() != WatchtowerStrategy {
		s.wallet.Start(ctxIn)
	}
	s.StopWaiter.Start(ctxIn, s)
	backoff := time.Second
	ephemeralErrorHandler := util.NewEphemeralErrorHandler(10*time.Minute, "is ahead of on-chain nonce", 0)

	s.CallIteratively(func(ctx context.Context) (returningWait time.Duration) {
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				log.Error("staker Act call panicked", "panic", panicErr, "backtrace", string(debug.Stack()))
				s.builder.ClearTransactions()
				returningWait = time.Minute
			}
		}()
		var err error
		if common.HexToAddress(s.config.GasRefunderAddress) != (common.Address{}) {
			gasRefunderBalance, err := s.client.BalanceAt(ctx, common.HexToAddress(s.config.GasRefunderAddress), nil)
			if err != nil {
				log.Warn("error fetching validator gas refunder balance", "err", err)
			} else {
				validatorGasRefunderBalance.Update(arbmath.BalancePerEther(gasRefunderBalance))
			}
		}
		err = s.updateBlockValidatorModuleRoot(ctx)
		if err != nil {
			log.Warn("error updating latest wasm module root", "err", err)
		}
		arbTx, err := s.Act(ctx)
		if err == nil && arbTx != nil {
			_, err = s.l1Reader.WaitForTxApproval(ctx, arbTx)
			if err == nil {
				log.Info("successfully executed staker transaction", "hash", arbTx.Hash())
			} else {
				err = fmt.Errorf("error waiting for tx receipt: %w", err)
			}
		}
		if err == nil {
			ephemeralErrorHandler.Reset()
			backoff = time.Second
			stakerLastSuccessfulActionGauge.Update(time.Now().Unix())
			stakerActionSuccessCounter.Inc(1)
			if arbTx != nil && !s.wallet.CanBatchTxs() {
				// Try to create another tx
				return 0
			}
			return s.config.StakerInterval
		}
		stakerActionFailureCounter.Inc(1)
		backoff *= 2
		logLevel := log.Error
		if backoff > time.Minute {
			backoff = time.Minute
		} else {
			logLevel = log.Warn
		}
		logLevel = ephemeralErrorHandler.LogLevel(err, logLevel)
		logLevel("error acting as staker", "err", err)
		return backoff
	})
	s.CallIteratively(func(ctx context.Context) time.Duration {
		wallet := s.wallet.AddressOrZero()
		staked, stakedMsgCount, stakedGlobalState, err := s.getLatestStakedState(ctx, wallet)
		if err != nil && ctx.Err() == nil {
			log.Error("staker: error checking latest staked", "err", err)
		}
		stakerLatestStakedNodeGauge.Update(int64(staked))
		if stakedGlobalState != nil {
			for _, notifier := range s.stakedNotifiers {
				notifier.UpdateLatestStaked(stakedMsgCount, *stakedGlobalState)
			}
		}
		confirmed := staked
		confirmedMsgCount := stakedMsgCount
		confirmedGlobalState := stakedGlobalState
		if wallet != (common.Address{}) {
			confirmed, confirmedMsgCount, confirmedGlobalState, err = s.getLatestStakedState(ctx, common.Address{})
			if err != nil && ctx.Err() == nil {
				log.Error("staker: error checking latest confirmed", "err", err)
			}
		}
		stakerLatestConfirmedNodeGauge.Update(int64(confirmed))
		if confirmedGlobalState != nil {
			for _, notifier := range s.confirmedNotifiers {
				notifier.UpdateLatestConfirmed(confirmedMsgCount, *confirmedGlobalState)
			}
		}
		return s.config.StakerInterval
	})
}

func (s *Staker) IsWhitelisted(ctx context.Context) (bool, error) {
	callOpts := s.getCallOpts(ctx)
	whitelistDisabled, err := s.rollup.ValidatorWhitelistDisabled(callOpts)
	if err != nil {
		return false, err
	}
	if whitelistDisabled {
		return true, nil
	}
	addr := s.wallet.Address()
	if addr != nil {
		return s.rollup.IsValidator(callOpts, *addr)
	}
	return false, nil
}

func (s *Staker) shouldAct(ctx context.Context) bool {
	var gasPriceHigh = false
	var gasPriceFloat float64
	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		log.Warn("error getting gas price", "err", err)
	} else {
		gasPriceFloat = float64(gasPrice.Int64()) / 1e9
		if gasPriceFloat >= s.config.PostingStrategy.HighGasThreshold {
			gasPriceHigh = true
		}
	}
	latestBlockInfo, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Warn("error getting latest block", "err", err)
		return true
	}
	latestBlockNum := latestBlockInfo.Number
	if s.lastActCalledBlock == nil {
		s.lastActCalledBlock = latestBlockNum
	}
	blocksSinceActCalled := new(big.Int).Sub(latestBlockNum, s.lastActCalledBlock)
	s.lastActCalledBlock = latestBlockNum
	if gasPriceHigh {
		// We're eating into the high gas buffer to delay our tx
		s.highGasBlocksBuffer.Sub(s.highGasBlocksBuffer, blocksSinceActCalled)
	} else {
		// We'll try to make a tx if necessary, so we can add to the buffer for future high gas
		s.highGasBlocksBuffer.Add(s.highGasBlocksBuffer, blocksSinceActCalled)
	}
	// Clamp `s.highGasBlocksBuffer` to between 0 and HighGasDelayBlocks
	if s.highGasBlocksBuffer.Sign() < 0 {
		s.highGasBlocksBuffer.SetInt64(0)
	} else if s.highGasBlocksBuffer.Cmp(big.NewInt(s.config.PostingStrategy.HighGasDelayBlocks)) > 0 {
		s.highGasBlocksBuffer.SetInt64(s.config.PostingStrategy.HighGasDelayBlocks)
	}
	if gasPriceHigh && s.highGasBlocksBuffer.Sign() > 0 {
		log.Warn(
			"not acting yet as gas price is high",
			"gasPrice", gasPriceFloat,
			"highGasPriceConfig", s.config.PostingStrategy.HighGasThreshold,
			"highGasBuffer", s.highGasBlocksBuffer,
		)
		return false
	}
	return true
}

func (s *Staker) confirmDataPosterIsReady(ctx context.Context) error {
	dp := s.wallet.DataPoster()
	if dp == nil {
		return nil
	}
	dataPosterNonce, _, err := dp.GetNextNonceAndMeta(ctx)
	if err != nil {
		return err
	}
	latestNonce, err := s.l1Reader.Client().NonceAt(ctx, dp.Sender(), nil)
	if err != nil {
		return err
	}
	if dataPosterNonce > latestNonce {
		return fmt.Errorf("data poster nonce %v is ahead of on-chain nonce %v -- probably waiting for a pending transaction to be included in a block", dataPosterNonce, latestNonce)
	}
	if dataPosterNonce < latestNonce {
		return fmt.Errorf("data poster nonce %v is behind on-chain nonce %v -- is something else making transactions on this address?", dataPosterNonce, latestNonce)
	}
	return nil
}

func (s *Staker) Act(ctx context.Context) (*types.Transaction, error) {
	if s.config.strategy != WatchtowerStrategy {
		err := s.confirmDataPosterIsReady(ctx)
		if err != nil {
			return nil, err
		}
		whitelisted, err := s.IsWhitelisted(ctx)
		if err != nil {
			return nil, fmt.Errorf("error checking if whitelisted: %w", err)
		}
		if !whitelisted {
			log.Warn("validator address isn't whitelisted", "address", s.wallet.Address(), "txSender", s.wallet.TxSenderAddress())
		}
	}
	if !s.shouldAct(ctx) {
		// The fact that we're delaying acting is already logged in `shouldAct`
		return nil, nil
	}
	callOpts := s.getCallOpts(ctx)
	s.builder.ClearTransactions()
	var rawInfo *StakerInfo
	walletAddressOrZero := s.wallet.AddressOrZero()
	if walletAddressOrZero != (common.Address{}) {
		var err error
		rawInfo, err = s.rollup.StakerInfo(ctx, walletAddressOrZero)
		if err != nil {
			return nil, fmt.Errorf("error getting own staker (%v) info: %w", walletAddressOrZero, err)
		}
		if rawInfo != nil {
			stakerAmountStakedGauge.Update(rawInfo.AmountStaked.Int64())
		} else {
			stakerAmountStakedGauge.Update(0)
		}
		s.updateStakerBalanceMetric(ctx)
	}
	// If the wallet address is zero, or the wallet address isn't staked,
	// this will return the latest node and its hash (atomically).
	latestStakedNodeNum, latestStakedNodeInfo, err := s.validatorUtils.LatestStaked(
		callOpts, s.rollupAddress, walletAddressOrZero,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting latest staked node of own wallet %v: %w", walletAddressOrZero, err)
	}
	stakerLatestStakedNodeGauge.Update(int64(latestStakedNodeNum))
	if rawInfo != nil {
		rawInfo.LatestStakedNode = latestStakedNodeNum
	}
	info := OurStakerInfo{
		CanProgress:          true,
		LatestStakedNode:     latestStakedNodeNum,
		LatestStakedNodeHash: latestStakedNodeInfo.NodeHash,
		StakerInfo:           rawInfo,
		StakeExists:          rawInfo != nil,
	}

	effectiveStrategy := s.config.strategy
	nodesLinear, err := s.validatorUtils.AreUnresolvedNodesLinear(callOpts, s.rollupAddress)
	if err != nil {
		return nil, fmt.Errorf("error checking for rollup assertion fork: %w", err)
	}
	if !nodesLinear {
		log.Warn("rollup assertion fork detected")
		if effectiveStrategy == DefensiveStrategy {
			effectiveStrategy = StakeLatestStrategy
		}
		s.inactiveLastCheckedNode = nil
	}
	if s.bringActiveUntilNode != 0 {
		if info.LatestStakedNode < s.bringActiveUntilNode {
			if effectiveStrategy == DefensiveStrategy {
				effectiveStrategy = StakeLatestStrategy
			}
		} else {
			log.Info("defensive validator staked past incorrect node; waiting here")
			s.bringActiveUntilNode = 0
		}
		s.inactiveLastCheckedNode = nil
	}
	if effectiveStrategy <= DefensiveStrategy && s.inactiveLastCheckedNode != nil {
		info.LatestStakedNode = s.inactiveLastCheckedNode.id
		info.LatestStakedNodeHash = s.inactiveLastCheckedNode.hash
	}

	latestConfirmedNode, err := s.rollup.LatestConfirmed(callOpts)
	if err != nil {
		return nil, fmt.Errorf("error getting latest confirmed node: %w", err)
	}

	requiredStakeElevated, err := s.isRequiredStakeElevated(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking if required stake is elevated: %w", err)
	}
	// Resolve nodes if either we're on the make nodes strategy,
	// or we're on the stake latest strategy but don't have a stake
	// (attempt to reduce the current required stake).
	shouldResolveNodes := effectiveStrategy >= ResolveNodesStrategy ||
		(effectiveStrategy >= StakeLatestStrategy && rawInfo == nil && requiredStakeElevated)
	resolvingNode := false
	if shouldResolveNodes {
		arbTx, err := s.resolveTimedOutChallenges(ctx)
		if err != nil {
			return nil, fmt.Errorf("error resolving timed out challenges: %w", err)
		}
		if arbTx != nil {
			return arbTx, nil
		}
		resolvingNode, err = s.resolveNextNode(ctx, rawInfo, &latestConfirmedNode)
		if err != nil {
			return nil, fmt.Errorf("error resolving node %v: %w", latestConfirmedNode+1, err)
		}
		if resolvingNode && rawInfo == nil && latestConfirmedNode > info.LatestStakedNode {
			// If we hit this condition, we've resolved what was previously the latest confirmed node,
			// and we don't have a stake yet. That means we were planning to enter the rollup on
			// the latest confirmed node, which has now changed. We fix this by updating our staker info
			// to indicate that we're now entering the rollup on the newly confirmed node.
			nodeInfo, err := s.rollup.GetNode(callOpts, latestConfirmedNode)
			if err != nil {
				return nil, fmt.Errorf("error getting latest confirmed node %v info: %w", latestConfirmedNode, err)
			}
			info.LatestStakedNode = latestConfirmedNode
			info.LatestStakedNodeHash = nodeInfo.NodeHash
		}
	}

	canActFurther := func() bool {
		return s.wallet.CanBatchTxs() || s.builder.BuildingTransactionCount() == 0
	}

	// If we have an old stake, remove it
	if rawInfo != nil && rawInfo.LatestStakedNode <= latestConfirmedNode && canActFurther() {
		stakeIsTooOutdated := rawInfo.LatestStakedNode < latestConfirmedNode
		// We're not trying to stake anyways
		stakeIsUnwanted := effectiveStrategy < StakeLatestStrategy
		if stakeIsTooOutdated || stakeIsUnwanted {
			// Note: we must have an address if rawInfo != nil
			auth, err := s.builder.Auth(ctx)
			if err != nil {
				return nil, err
			}
			_, err = s.rollup.ReturnOldDeposit(auth, walletAddressOrZero)
			if err != nil {
				return nil, fmt.Errorf("error returning old deposit (from our staker %v): %w", walletAddressOrZero, err)
			}
			auth, err = s.builder.Auth(ctx)
			if err != nil {
				return nil, err
			}
			_, err = s.rollup.WithdrawStakerFunds(auth)
			if err != nil {
				return nil, fmt.Errorf("error withdrawing staker funds from our staker %v: %w", walletAddressOrZero, err)
			}
			log.Info("removing old stake and withdrawing funds")
			return s.wallet.ExecuteTransactions(ctx, s.builder, s.config.gasRefunder)
		}
	}

	if walletAddressOrZero != (common.Address{}) && canActFurther() {
		withdrawable, err := s.rollup.WithdrawableFunds(callOpts, walletAddressOrZero)
		if err != nil {
			return nil, fmt.Errorf("error checking withdrawable funds of our staker %v: %w", walletAddressOrZero, err)
		}
		if withdrawable.Sign() > 0 {
			auth, err := s.builder.Auth(ctx)
			if err != nil {
				return nil, err
			}
			_, err = s.rollup.WithdrawStakerFunds(auth)
			if err != nil {
				return nil, fmt.Errorf("error withdrawing our staker %v funds: %w", walletAddressOrZero, err)
			}
		}
	}

	if rawInfo != nil && canActFurther() {
		if err = s.handleConflict(ctx, rawInfo); err != nil {
			return nil, fmt.Errorf("error handling conflict: %w", err)
		}
	}

	// Don't attempt to create a new stake if we're resolving a node and the stake is elevated,
	// as that might affect the current required stake.
	if (rawInfo != nil || !resolvingNode || !requiredStakeElevated) && canActFurther() {
		// Advance stake up to 20 times in one transaction
		for i := 0; info.CanProgress && i < 20; i++ {
			if err := s.advanceStake(ctx, &info, effectiveStrategy); err != nil {
				return nil, fmt.Errorf("error advancing stake from node %v (hash %v): %w", info.LatestStakedNode, info.LatestStakedNodeHash, err)
			}
			if !s.wallet.CanBatchTxs() && effectiveStrategy >= StakeLatestStrategy {
				info.CanProgress = false
			}
		}
	}

	if rawInfo != nil && s.builder.BuildingTransactionCount() == 0 && canActFurther() {
		if err := s.createConflict(ctx, rawInfo); err != nil {
			return nil, fmt.Errorf("error creating conflict: %w", err)
		}
	}

	if s.builder.BuildingTransactionCount() == 0 {
		return nil, nil
	}

	if info.StakerInfo == nil && info.StakeExists {
		log.Info("staking to execute transactions")
	}
	return s.wallet.ExecuteTransactions(ctx, s.builder, s.config.gasRefunder)
}

func (s *Staker) handleConflict(ctx context.Context, info *StakerInfo) error {
	if info.CurrentChallenge == nil {
		s.activeChallenge = nil
		return nil
	}

	if s.activeChallenge == nil || s.activeChallenge.ChallengeIndex() != *info.CurrentChallenge {
		log.Error("entered challenge", "challenge", *info.CurrentChallenge)

		latestConfirmedCreated, err := s.rollup.LatestConfirmedCreationBlock(ctx)
		if err != nil {
			return fmt.Errorf("error getting latest confirmed creation block: %w", err)
		}

		newChallengeManager, err := NewChallengeManager(
			ctx,
			s.builder,
			s.builder.BuilderAuth(),
			*s.builder.WalletAddress(),
			s.wallet.ChallengeManagerAddress(),
			*info.CurrentChallenge,
			s.statelessBlockValidator,
			latestConfirmedCreated,
			s.config.ConfirmationBlocks,
		)
		if err != nil {
			return fmt.Errorf("error creating challenge manager: %w", err)
		}

		s.activeChallenge = newChallengeManager
	}

	_, err := s.activeChallenge.Act(ctx)
	return err
}

func (s *Staker) advanceStake(ctx context.Context, info *OurStakerInfo, effectiveStrategy StakerStrategy) error {
	active := effectiveStrategy >= StakeLatestStrategy
	action, wrongNodesExist, err := s.generateNodeAction(ctx, info, effectiveStrategy, &s.config)
	if err != nil {
		return fmt.Errorf("error generating node action: %w", err)
	}
	if wrongNodesExist && effectiveStrategy == WatchtowerStrategy {
		log.Error("found incorrect assertion in watchtower mode")
	}
	if action == nil {
		info.CanProgress = false
		return nil
	}

	switch action := action.(type) {
	case createNodeAction:
		if wrongNodesExist && s.config.DisableChallenge {
			log.Error("refusing to challenge assertion as config disables challenges")
			info.CanProgress = false
			return nil
		}
		if !active {
			if wrongNodesExist && effectiveStrategy >= DefensiveStrategy {
				log.Error("bringing defensive validator online because of incorrect assertion")
				s.bringActiveUntilNode = info.LatestStakedNode + 1
			}
			info.CanProgress = false
			return nil
		}

		// Details are already logged with more details in generateNodeAction
		info.CanProgress = false
		info.LatestStakedNode = 0
		info.LatestStakedNodeHash = action.hash

		// We'll return early if we already have a stake
		if info.StakeExists {
			auth, err := s.builder.Auth(ctx)
			if err != nil {
				return err
			}
			_, err = s.rollup.StakeOnNewNode(auth, action.assertion.AsSolidityStruct(), action.hash, action.prevInboxMaxCount)
			if err != nil {
				return fmt.Errorf("error staking on new node: %w", err)
			}
			return nil
		}

		// If we have no stake yet, we'll put one down
		stakeAmount, err := s.rollup.CurrentRequiredStake(s.getCallOpts(ctx))
		if err != nil {
			return fmt.Errorf("error getting current required stake: %w", err)
		}
		auth, err := s.builder.AuthWithAmount(ctx, stakeAmount)
		if err != nil {
			return err
		}
		_, err = s.rollup.NewStakeOnNewNode(
			auth,
			action.assertion.AsSolidityStruct(),
			action.hash,
			action.prevInboxMaxCount,
		)
		if err != nil {
			return fmt.Errorf("error placing new stake on new node: %w", err)
		}
		info.StakeExists = true
		return nil
	case existingNodeAction:
		info.LatestStakedNode = action.number
		info.LatestStakedNodeHash = action.hash
		if !active {
			if wrongNodesExist && effectiveStrategy >= DefensiveStrategy {
				log.Error("bringing defensive validator online because of incorrect assertion")
				s.bringActiveUntilNode = action.number
				info.CanProgress = false
			} else {
				s.inactiveLastCheckedNode = &nodeAndHash{
					id:   action.number,
					hash: action.hash,
				}
			}
			return nil
		}
		log.Info("staking on existing node", "node", action.number)
		// We'll return early if we already havea stake
		if info.StakeExists {
			auth, err := s.builder.Auth(ctx)
			if err != nil {
				return err
			}
			_, err = s.rollup.StakeOnExistingNode(auth, action.number, action.hash)
			if err != nil {
				return fmt.Errorf("error staking on existing node: %w", err)
			}
			return nil
		}

		// If we have no stake yet, we'll put one down
		stakeAmount, err := s.rollup.CurrentRequiredStake(s.getCallOpts(ctx))
		if err != nil {
			return fmt.Errorf("error getting current required stake: %w", err)
		}
		auth, err := s.builder.AuthWithAmount(ctx, stakeAmount)
		if err != nil {
			return err
		}
		_, err = s.rollup.NewStakeOnExistingNode(
			auth,
			action.number,
			action.hash,
		)
		if err != nil {
			return fmt.Errorf("error placing new stake on existing node: %w", err)
		}
		info.StakeExists = true
		return nil
	default:
		panic("invalid action type")
	}
}

func (s *Staker) createConflict(ctx context.Context, info *StakerInfo) error {
	if info.CurrentChallenge != nil {
		return nil
	}

	callOpts := s.getCallOpts(ctx)
	stakers, moreStakers, err := s.validatorUtils.GetStakers(callOpts, s.rollupAddress, 0, 1024)
	if err != nil {
		return fmt.Errorf("error getting stakers list: %w", err)
	}
	for moreStakers {
		var newStakers []common.Address
		newStakers, moreStakers, err = s.validatorUtils.GetStakers(callOpts, s.rollupAddress, uint64(len(stakers)), 1024)
		if err != nil {
			return fmt.Errorf("error getting more stakers: %w", err)
		}
		stakers = append(stakers, newStakers...)
	}
	latestNode, err := s.rollup.LatestConfirmed(callOpts)
	if err != nil {
		return err
	}
	// Safe to dereference as createConflict is only called when we have a wallet address
	walletAddr := *s.wallet.Address()
	for _, staker := range stakers {
		stakerInfo, err := s.rollup.StakerInfo(ctx, staker)
		if err != nil {
			return fmt.Errorf("error getting staker %v info: %w", staker, err)
		}
		if stakerInfo == nil {
			return fmt.Errorf("staker %v (returned from ValidatorUtils's GetStakers function) not found in rollup", staker)
		}
		if stakerInfo.CurrentChallenge != nil {
			continue
		}
		conflictInfo, err := s.validatorUtils.FindStakerConflict(callOpts, s.rollupAddress, walletAddr, staker, big.NewInt(1024))
		if err != nil {
			return fmt.Errorf("error finding conflict with staker %v: %w", staker, err)
		}
		if ConflictType(conflictInfo.Ty) != CONFLICT_TYPE_FOUND {
			continue
		}
		staker1 := walletAddr
		staker2 := staker
		if conflictInfo.Node2 < conflictInfo.Node1 {
			staker1, staker2 = staker2, staker1
			conflictInfo.Node1, conflictInfo.Node2 = conflictInfo.Node2, conflictInfo.Node1
		}
		if conflictInfo.Node1 <= latestNode {
			// Immaterial as this is past the confirmation point; this must be a zombie
			continue
		}

		node1Info, err := s.rollup.LookupNode(ctx, conflictInfo.Node1)
		if err != nil {
			return fmt.Errorf("error looking up node %v: %w", conflictInfo.Node1, err)
		}
		node2Info, err := s.rollup.LookupNode(ctx, conflictInfo.Node2)
		if err != nil {
			return fmt.Errorf("error looking up node %v: %w", conflictInfo.Node2, err)
		}
		log.Warn("creating challenge", "node1", conflictInfo.Node1, "node2", conflictInfo.Node2, "otherStaker", staker)
		auth, err := s.builder.Auth(ctx)
		if err != nil {
			return err
		}
		_, err = s.rollup.CreateChallenge(
			auth,
			[2]common.Address{staker1, staker2},
			[2]uint64{conflictInfo.Node1, conflictInfo.Node2},
			node1Info.MachineStatuses(),
			node1Info.GlobalStates(),
			node1Info.Assertion.NumBlocks,
			node2Info.Assertion.ExecutionHash(),
			[2]*big.Int{new(big.Int).SetUint64(node1Info.L1BlockProposed), new(big.Int).SetUint64(node2Info.L1BlockProposed)},
			[2][32]byte{node1Info.WasmModuleRoot, node2Info.WasmModuleRoot},
		)
		if err != nil {
			return fmt.Errorf("error creating challenge: %w", err)
		}
	}
	// No conflicts exist
	return nil
}

func (s *Staker) Strategy() StakerStrategy {
	return s.config.strategy
}

func (s *Staker) Rollup() *RollupWatcher {
	return s.rollup
}

func (s *Staker) updateStakerBalanceMetric(ctx context.Context) {
	txSenderAddress := s.wallet.TxSenderAddress()
	if txSenderAddress == nil {
		stakerBalanceGauge.Update(0)
		return
	}
	balance, err := s.client.BalanceAt(ctx, *txSenderAddress, nil)
	if err != nil {
		log.Error("error getting staker balance", "txSenderAddress", *txSenderAddress, "err", err)
		return
	}
	stakerBalanceGauge.Update(arbmath.BalancePerEther(balance))
}
