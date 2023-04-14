// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type RollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	DeployedAt             uint64         `json:"deployed-at"`
}

type RollupAddressesConfig struct {
	Bridge                 string `koanf:"bridge"`
	Inbox                  string `koanf:"inbox"`
	SequencerInbox         string `koanf:"sequencer-inbox"`
	Rollup                 string `koanf:"rollup"`
	ValidatorUtils         string `koanf:"validator-utils"`
	ValidatorWalletCreator string `koanf:"validator-wallet-creator"`
	DeployedAt             uint64 `koanf:"deployed-at"`
}

var RollupAddressesConfigDefault = RollupAddressesConfig{}

func RollupAddressesConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".bridge", "", "the bridge contract address")
	f.String(prefix+".inbox", "", "the inbox contract address")
	f.String(prefix+".sequencer-inbox", "", "the sequencer inbox contract address")
	f.String(prefix+".rollup", "", "the rollup contract address")
	f.String(prefix+".validator-utils", "", "the validator utils contract address")
	f.String(prefix+".validator-wallet-creator", "", "the validator wallet creator contract address")
	f.Uint64(prefix+".deployed-at", 0, "the block number at which the rollup was deployed")
}

func (c *RollupAddressesConfig) ParseAddresses() (RollupAddresses, error) {
	a := RollupAddresses{
		DeployedAt: c.DeployedAt,
	}
	strs := []string{
		c.Bridge,
		c.Inbox,
		c.SequencerInbox,
		c.Rollup,
		c.ValidatorUtils,
		c.ValidatorWalletCreator,
	}
	addrs := []*common.Address{
		&a.Bridge,
		&a.Inbox,
		&a.SequencerInbox,
		&a.Rollup,
		&a.ValidatorUtils,
		&a.ValidatorWalletCreator,
	}
	names := []string{
		"Bridge",
		"Inbox",
		"SequencerInbox",
		"Rollup",
		"ValidatorUtils",
		"ValidatorWalletCreator",
	}
	if len(strs) != len(addrs) {
		return RollupAddresses{}, fmt.Errorf("internal error: attempting to parse %v strings into %v addresses", len(strs), len(addrs))
	}
	complete := true
	for i, s := range strs {
		if !common.IsHexAddress(s) {
			log.Error("invalid address", "name", names[i], "value", s)
			complete = false
		}
		*addrs[i] = common.HexToAddress(s)
	}
	if !complete {
		return RollupAddresses{}, fmt.Errorf("invalid addresses")
	}
	return a, nil
}

func andTxSucceeded(ctx context.Context, l1Reader *headerreader.HeaderReader, tx *types.Transaction, err error) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	_, err = l1Reader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return fmt.Errorf("error executing tx: %w", err)
	}
	return nil
}

func deployBridgeCreator(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts) (common.Address, error) {
	client := l1Reader.Client()
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("sequencer inbox deploy error: %w", err)
	}

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventInbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	bridgeCreatorAddr, tx, bridgeCreator, err := rollupgen.DeployBridgeCreator(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	tx, err = bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator update templates error: %w", err)
	}

	return bridgeCreatorAddr, nil
}

func deployChallengeFactory(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts) (common.Address, common.Address, error) {
	client := l1Reader.Client()
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("osp0 deploy error: %w", err)
	}

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMemory deploy error: %w", err)
	}

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMath deploy error: %w", err)
	}

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospHostIo deploy error: %w", err)
	}

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	challengeManagerAddr, tx, _, err := challengegen.DeployChallengeManager(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	return ospEntryAddr, challengeManagerAddr, nil
}

func deployRollupCreator(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, l1Reader, auth)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, err
	}

	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, l1Reader, auth)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup admin logic deploy error: %w", err)
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup creator deploy error: %w", err)
	}

	validatorUtils, tx, _, err := rollupgen.DeployValidatorUtils(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator utils deploy error: %w", err)
	}

	validatorWalletCreator, tx, _, err := rollupgen.DeployValidatorWalletCreator(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator wallet creator deploy error: %w", err)
	}

	tx, err = rollupCreator.SetTemplates(
		auth,
		bridgeCreator,
		ospEntryAddr,
		challengeManagerAddr,
		rollupAdminLogic,
		rollupUserLogic,
		validatorUtils,
		validatorWalletCreator,
	)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup set template error: %w", err)
	}

	return rollupCreator, rollupCreatorAddress, validatorUtils, validatorWalletCreator, nil
}

func GenerateRollupConfig(prod bool, wasmModuleRoot common.Hash, rollupOwner common.Address, chainId *big.Int, loserStakeEscrow common.Address) rollupgen.Config {
	var confirmPeriod uint64
	if prod {
		confirmPeriod = 45818
	} else {
		confirmPeriod = 20
	}
	return rollupgen.Config{
		ConfirmPeriodBlocks:      confirmPeriod,
		ExtraChallengeTimeBlocks: 200,
		StakeToken:               common.Address{},
		BaseStake:                big.NewInt(params.Ether),
		WasmModuleRoot:           wasmModuleRoot,
		Owner:                    rollupOwner,
		LoserStakeEscrow:         loserStakeEscrow,
		ChainId:                  chainId,
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
	}
}

func DeployOnL1(ctx context.Context, l1client arbutil.L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address, authorizeValidators uint64, readerConfig headerreader.ConfigFetcher, config rollupgen.Config) (*RollupAddresses, error) {
	l1Reader := headerreader.New(l1client, readerConfig)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	if config.WasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("no machine specified")
	}

	rollupCreator, rollupCreatorAddress, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, l1Reader, deployAuth)
	if err != nil {
		return nil, fmt.Errorf("error deploying rollup creator: %w", err)
	}

	nonce, err := l1client.PendingNonceAt(ctx, rollupCreatorAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting pending nonce: %w", err)
	}
	expectedRollupAddr := crypto.CreateAddress(rollupCreatorAddress, nonce+2)
	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		config,
		expectedRollupAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("error submitting create rollup tx: %w", err)
	}
	receipt, err := l1Reader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("error executing create rollup tx: %w", err)
	}
	info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, fmt.Errorf("error parsing rollup created log: %w", err)
	}

	sequencerInbox, err := bridgegen.NewSequencerInbox(info.SequencerInbox, l1client)
	if err != nil {
		return nil, fmt.Errorf("error getting sequencer inbox: %w", err)
	}

	// if a zero sequencer address is specified, don't authorize any sequencers
	if sequencer != (common.Address{}) {
		tx, err = sequencerInbox.SetIsBatchPoster(deployAuth, sequencer, true)
		err = andTxSucceeded(ctx, l1Reader, tx, err)
		if err != nil {
			return nil, fmt.Errorf("error setting is batch poster: %w", err)
		}
	}

	var allowValidators []bool
	var validatorAddrs []common.Address
	for i := uint64(1); i <= authorizeValidators; i++ {
		validatorAddrs = append(validatorAddrs, crypto.CreateAddress(validatorWalletCreator, i))
		allowValidators = append(allowValidators, true)
	}
	if len(validatorAddrs) > 0 {
		rollup, err := rollupgen.NewRollupAdminLogic(info.RollupAddress, l1client)
		if err != nil {
			return nil, fmt.Errorf("error getting rollup admin: %w", err)
		}
		tx, err = rollup.SetValidator(deployAuth, validatorAddrs, allowValidators)
		err = andTxSucceeded(ctx, l1Reader, tx, err)
		if err != nil {
			return nil, fmt.Errorf("error setting validator: %w", err)
		}
	}

	return &RollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             receipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
	}, nil
}

type Config struct {
	RPC                    arbitrum.Config             `koanf:"rpc"`
	Sequencer              execution.SequencerConfig   `koanf:"sequencer" reload:"hot"`
	L1Reader               headerreader.Config         `koanf:"l1-reader" reload:"hot"`
	InboxReader            InboxReaderConfig           `koanf:"inbox-reader" reload:"hot"`
	DelayedSequencer       DelayedSequencerConfig      `koanf:"delayed-sequencer" reload:"hot"`
	BatchPoster            BatchPosterConfig           `koanf:"batch-poster" reload:"hot"`
	ForwardingTargetImpl   string                      `koanf:"forwarding-target"`
	Forwarder              execution.ForwarderConfig   `koanf:"forwarder"`
	TxPreCheckerStrictness uint                        `koanf:"tx-pre-checker-strictness" reload:"hot"`
	BlockValidator         staker.BlockValidatorConfig `koanf:"block-validator" reload:"hot"`
	Feed                   broadcastclient.FeedConfig  `koanf:"feed" reload:"hot"`
	Staker                 staker.L1ValidatorConfig    `koanf:"staker"`
	SeqCoordinator         SeqCoordinatorConfig        `koanf:"seq-coordinator"`
	DataAvailability       das.DataAvailabilityConfig  `koanf:"data-availability"`
	SyncMonitor            SyncMonitorConfig           `koanf:"sync-monitor"`
	Dangerous              DangerousConfig             `koanf:"dangerous"`
	Caching                execution.CachingConfig     `koanf:"caching"`
	Archive                bool                        `koanf:"archive"`
	TxLookupLimit          uint64                      `koanf:"tx-lookup-limit"`
	TransactionStreamer    TransactionStreamerConfig   `koanf:"transaction-streamer" reload:"hot"`
	Maintenance            MaintenanceConfig           `koanf:"maintenance" reload:"hot"`
}

func (c *Config) Validate() error {
	if c.L1Reader.Enable && c.Sequencer.Enable && !c.DelayedSequencer.Enable {
		log.Warn("delayed sequencer is not enabled, despite sequencer and l1 reader being enabled")
	}
	if c.DelayedSequencer.Enable && !c.Sequencer.Enable {
		return errors.New("cannot enable delayed sequencer without enabling sequencer")
	}
	if err := c.Sequencer.Validate(); err != nil {
		return err
	}
	if err := c.Maintenance.Validate(); err != nil {
		return err
	}
	if err := c.InboxReader.Validate(); err != nil {
		return err
	}
	if err := c.BatchPoster.Validate(); err != nil {
		return err
	}
	if err := c.Feed.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) Get() *Config {
	return c
}

func (c *Config) Start(context.Context) {}

func (c *Config) StopAndWait() {}

func (c *Config) Started() bool {
	return true
}

func (c *Config) ForwardingTarget() string {
	if c.ForwardingTargetImpl == "null" {
		return ""
	}

	return c.ForwardingTargetImpl
}

func (c *Config) ValidatorRequired() bool {
	if c.BlockValidator.Enable {
		return true
	}
	if c.Staker.Enable {
		return !c.Staker.Dangerous.WithoutBlockValidator
	}
	return false
}

func ConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	execution.SequencerConfigAddOptions(prefix+".sequencer", f)
	headerreader.AddOptions(prefix+".l1-reader", f)
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	BatchPosterConfigAddOptions(prefix+".batch-poster", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTargetImpl, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	execution.AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	txPreCheckerDescription := "how strict to be when checking txs before forwarding them. 0 = accept anything, " +
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, " +
		"30 = full validation which may reject txs that would succeed"
	f.Uint(prefix+".tx-pre-checker-strictness", ConfigDefault.TxPreCheckerStrictness, txPreCheckerDescription)
	staker.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, feedInputEnable, feedOutputEnable)
	staker.L1ValidatorConfigAddOptions(prefix+".staker", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	das.DataAvailabilityConfigAddNodeOptions(prefix+".data-availability", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	execution.CachingConfigAddOptions(prefix+".caching", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")
	TransactionStreamerConfigAddOptions(prefix+".transaction-streamer", f)
	MaintenanceConfigAddOptions(prefix+".maintenance", f)

	archiveMsg := fmt.Sprintf("retain past block state (deprecated, please use %v.caching.archive)", prefix)
	f.Bool(prefix+".archive", ConfigDefault.Archive, archiveMsg)
}

var ConfigDefault = Config{
	RPC:                    arbitrum.DefaultConfig,
	Sequencer:              execution.DefaultSequencerConfig,
	L1Reader:               headerreader.DefaultConfig,
	InboxReader:            DefaultInboxReaderConfig,
	DelayedSequencer:       DefaultDelayedSequencerConfig,
	BatchPoster:            DefaultBatchPosterConfig,
	ForwardingTargetImpl:   "",
	TxPreCheckerStrictness: execution.TxPreCheckerStrictnessNone,
	BlockValidator:         staker.DefaultBlockValidatorConfig,
	Feed:                   broadcastclient.FeedConfigDefault,
	Staker:                 staker.DefaultL1ValidatorConfig,
	SeqCoordinator:         DefaultSeqCoordinatorConfig,
	DataAvailability:       das.DefaultDataAvailabilityConfig,
	SyncMonitor:            DefaultSyncMonitorConfig,
	Dangerous:              DefaultDangerousConfig,
	Archive:                false,
	TxLookupLimit:          126_230_400, // 1 year at 4 blocks per second
	Caching:                execution.DefaultCachingConfig,
	TransactionStreamer:    DefaultTransactionStreamerConfig,
}

func ConfigDefaultL1Test() *Config {
	config := ConfigDefaultL1NonSequencerTest()
	config.Sequencer = execution.TestSequencerConfig
	config.DelayedSequencer = TestDelayedSequencerConfig
	config.BatchPoster = TestBatchPosterConfig
	config.SeqCoordinator = TestSeqCoordinatorConfig

	return config
}

func ConfigDefaultL1NonSequencerTest() *Config {
	config := ConfigDefault
	config.L1Reader = headerreader.TestConfig
	config.InboxReader = TestInboxReaderConfig
	config.Sequencer.Enable = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.SeqCoordinator.Enable = false
	config.BlockValidator = staker.TestBlockValidatorConfig
	config.Forwarder = execution.DefaultTestForwarderConfig
	config.TransactionStreamer = DefaultTransactionStreamerConfig

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.Sequencer = execution.TestSequencerConfig
	config.L1Reader.Enable = false
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Feed.Input.Verifier.Dangerous.AcceptMissing = true
	config.Feed.Output.Signed = false
	config.SeqCoordinator.Signing.ECDSA.AcceptSequencer = false
	config.SeqCoordinator.Signing.ECDSA.Dangerous.AcceptMissing = true
	config.TransactionStreamer = DefaultTransactionStreamerConfig

	return &config
}

type DangerousConfig struct {
	NoL1Listener bool  `koanf:"no-l1-listener"`
	ReorgToBlock int64 `koanf:"reorg-to-block"`
}

var DefaultDangerousConfig = DangerousConfig{
	NoL1Listener: false,
	ReorgToBlock: -1,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".no-l1-listener", DefaultDangerousConfig.NoL1Listener, "DANGEROUS! disables listening to L1. To be used in test nodes only")
	f.Int64(prefix+".reorg-to-block", DefaultDangerousConfig.ReorgToBlock, "DANGEROUS! forces a reorg to an old block height. To be used for testing only. -1 to disable")
}

type Node struct {
	ArbDB                   ethdb.Database
	Stack                   *node.Node
	Execution               *execution.ExecutionNode
	L1Reader                *headerreader.HeaderReader
	TxStreamer              *TransactionStreamer
	DeployInfo              *RollupAddresses
	InboxReader             *InboxReader
	InboxTracker            *InboxTracker
	DelayedSequencer        *DelayedSequencer
	BatchPoster             *BatchPoster
	BlockValidator          *staker.BlockValidator
	StatelessBlockValidator *staker.StatelessBlockValidator
	Staker                  *staker.Staker
	BroadcastServer         *broadcaster.Broadcaster
	BroadcastClients        *broadcastclients.BroadcastClients
	SeqCoordinator          *SeqCoordinator
	MaintenanceRunner       *MaintenanceRunner
	DASLifecycleManager     *das.LifecycleManager
	ClassicOutboxRetriever  *ClassicOutboxRetriever
	SyncMonitor             *SyncMonitor
	configFetcher           ConfigFetcher
	ctx                     context.Context
}

type ConfigFetcher interface {
	Get() *Config
	Start(context.Context)
	StopAndWait()
	Started() bool
}

func checkArbDbSchemaVersion(arbDb ethdb.Database) error {
	var version uint64
	hasVersion, err := arbDb.Has(dbSchemaVersion)
	if err != nil {
		return err
	}
	if hasVersion {
		versionBytes, err := arbDb.Get(dbSchemaVersion)
		if err != nil {
			return err
		}
		version = binary.BigEndian.Uint64(versionBytes)
	}
	for version != currentDbSchemaVersion {
		batch := arbDb.NewBatch()
		switch version {
		case 0:
			// No database updates are necessary for database format version 0->1.
			// This version adds a new format for delayed messages in the inbox tracker,
			// but it can still read the old format for old messages.
		default:
			return fmt.Errorf("unsupported database format version %v", version)
		}

		// Increment version and flush the batch
		version++
		versionBytes := make([]uint8, 8)
		binary.BigEndian.PutUint64(versionBytes, version)
		err = batch.Put(dbSchemaVersion, versionBytes)
		if err != nil {
			return err
		}
		err = batch.Write()
		if err != nil {
			return err
		}
	}
	return nil
}

func createNodeImpl(
	ctx context.Context,
	stack *node.Node,
	chainDb ethdb.Database,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2BlockChain *core.BlockChain,
	l1client arbutil.L1Interface,
	deployInfo *RollupAddresses,
	txOpts *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
) (*Node, error) {
	config := configFetcher.Get()

	err := checkArbDbSchemaVersion(arbDb)
	if err != nil {
		return nil, err
	}

	l2Config := l2BlockChain.Config()
	l2ChainId := l2Config.ChainID.Uint64()

	var reorgingToBlock *types.Block
	if config.Dangerous.ReorgToBlock >= 0 {
		reorgingToBlock, err = execution.ReorgToBlock(l2BlockChain, uint64(config.Dangerous.ReorgToBlock))
		if err != nil {
			return nil, err
		}
	}

	syncMonitor := NewSyncMonitor(&config.SyncMonitor)
	var classicOutbox *ClassicOutboxRetriever
	classicMsgDb, err := stack.OpenDatabase("classic-msg", 0, 0, "", true)
	if err != nil {
		if l2Config.ArbitrumChainParams.GenesisBlockNum > 0 {
			log.Warn("Classic Msg Database not found", "err", err)
		}
		classicOutbox = nil
	} else {
		classicOutbox = NewClassicOutboxRetriever(classicMsgDb)
	}

	var l1Reader *headerreader.HeaderReader
	if config.L1Reader.Enable {
		l1Reader = headerreader.New(l1client, func() *headerreader.Config { return &configFetcher.Get().L1Reader })
	}

	sequencerConfigFetcher := func() *execution.SequencerConfig { return &configFetcher.Get().Sequencer }
	txprecheckStrictFetcher := func() uint { return configFetcher.Get().TxPreCheckerStrictness }
	exec, err := execution.CreateExecutionNode(stack, chainDb, l2BlockChain, l1Reader, syncMonitor,
		config.ForwardingTarget(), &config.Forwarder, config.RPC,
		sequencerConfigFetcher, txprecheckStrictFetcher)
	if err != nil {
		return nil, err
	}

	var broadcastServer *broadcaster.Broadcaster
	if config.Feed.Output.Enable {
		var maybeDataSigner signature.DataSignerFunc
		if config.Feed.Output.Signed {
			if dataSigner == nil {
				return nil, errors.New("cannot sign outgoing feed")
			}
			maybeDataSigner = dataSigner
		}
		broadcastServer = broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &configFetcher.Get().Feed.Output }, l2ChainId, fatalErrChan, maybeDataSigner)
	}

	transactionStreamerConfigFetcher := func() *TransactionStreamerConfig { return &configFetcher.Get().TransactionStreamer }
	txStreamer, err := NewTransactionStreamer(arbDb, l2Config, exec.ExecEngine, broadcastServer, fatalErrChan, transactionStreamerConfigFetcher)
	if err != nil {
		return nil, err
	}
	var coordinator *SeqCoordinator
	var bpVerifier *contracts.BatchPosterVerifier
	if deployInfo != nil && l1client != nil {
		sequencerInboxAddr := deployInfo.SequencerInbox

		seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(sequencerInboxAddr, l1client)
		if err != nil {
			return nil, err
		}
		bpVerifier = contracts.NewBatchPosterVerifier(seqInboxCaller)
	}

	if config.SeqCoordinator.Enable {
		coordinator, err = NewSeqCoordinator(dataSigner, bpVerifier, txStreamer, exec.Sequencer, syncMonitor, config.SeqCoordinator)
		if err != nil {
			return nil, err
		}
	} else if config.Sequencer.Enable && !config.Sequencer.Dangerous.NoCoordinator {
		return nil, errors.New("sequencer must be enabled with coordinator, unless dangerous.no-coordinator set")
	}
	dbs := []ethdb.Database{chainDb, arbDb}
	maintenanceRunner, err := NewMaintenanceRunner(func() *MaintenanceConfig { return &configFetcher.Get().Maintenance }, coordinator, dbs)
	if err != nil {
		return nil, err
	}

	var broadcastClients *broadcastclients.BroadcastClients
	if config.Feed.Input.Enable() {
		currentMessageCount, err := txStreamer.GetMessageCount()
		if err != nil {
			return nil, err
		}

		broadcastClients, err = broadcastclients.NewBroadcastClients(
			func() *broadcastclient.Config { return &configFetcher.Get().Feed.Input },
			l2ChainId,
			currentMessageCount,
			txStreamer,
			nil,
			fatalErrChan,
			bpVerifier,
		)
		if err != nil {
			return nil, err
		}
	}

	if !config.L1Reader.Enable {
		return &Node{
			arbDb,
			stack,
			exec,
			nil,
			txStreamer,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			broadcastServer,
			broadcastClients,
			coordinator,
			maintenanceRunner,
			nil,
			classicOutbox,
			syncMonitor,
			configFetcher,
			ctx,
		}, nil
	}

	if deployInfo == nil {
		return nil, errors.New("deployinfo is nil")
	}
	delayedBridge, err := NewDelayedBridge(l1client, deployInfo.Bridge, deployInfo.DeployedAt)
	if err != nil {
		return nil, err
	}
	sequencerInbox, err := NewSequencerInbox(l1client, deployInfo.SequencerInbox, int64(deployInfo.DeployedAt))
	if err != nil {
		return nil, err
	}

	var daWriter das.DataAvailabilityServiceWriter
	var daReader das.DataAvailabilityServiceReader
	var dasLifecycleManager *das.LifecycleManager
	if config.DataAvailability.Enable {
		if config.BatchPoster.Enable {
			daWriter, daReader, dasLifecycleManager, err = das.CreateBatchPosterDAS(ctx, &config.DataAvailability, dataSigner, l1client, deployInfo.SequencerInbox)
			if err != nil {
				return nil, err
			}
		} else {
			daReader, dasLifecycleManager, err = das.CreateDAReaderForNode(ctx, &config.DataAvailability, l1Reader, &deployInfo.SequencerInbox)
			if err != nil {
				return nil, err
			}
		}

		daReader = das.NewReaderTimeoutWrapper(daReader, config.DataAvailability.RequestTimeout)

		if config.DataAvailability.PanicOnError {
			if daWriter != nil {
				daWriter = das.NewWriterPanicWrapper(daWriter)
			}
			daReader = das.NewReaderPanicWrapper(daReader)
		}
	} else if l2BlockChain.Config().ArbitrumChainParams.DataAvailabilityCommittee {
		return nil, errors.New("a data availability service is required for this chain, but it was not configured")
	}

	inboxTracker, err := NewInboxTracker(arbDb, txStreamer, daReader)
	if err != nil {
		return nil, err
	}
	inboxReader, err := NewInboxReader(inboxTracker, l1client, l1Reader, new(big.Int).SetUint64(deployInfo.DeployedAt), delayedBridge, sequencerInbox, func() *InboxReaderConfig { return &configFetcher.Get().InboxReader })
	if err != nil {
		return nil, err
	}
	txStreamer.SetInboxReaders(inboxReader, delayedBridge)

	var statelessBlockValidator *staker.StatelessBlockValidator
	if config.BlockValidator.URL != "" {
		statelessBlockValidator, err = staker.NewStatelessBlockValidator(
			inboxReader,
			inboxTracker,
			txStreamer,
			l2BlockChain,
			chainDb,
			rawdb.NewTable(arbDb, blockValidatorPrefix),
			daReader,
			&configFetcher.Get().BlockValidator,
		)
	} else {
		err = errors.New("no validator url specified")
	}
	if err != nil {
		if config.ValidatorRequired() {
			return nil, fmt.Errorf("%w: failed to init block validator", err)
		} else {
			log.Warn("validation not supported", "err", err)
		}
		statelessBlockValidator = nil
	}

	var blockValidator *staker.BlockValidator
	if config.BlockValidator.Enable {
		blockValidator, err = staker.NewBlockValidator(
			statelessBlockValidator,
			inboxTracker,
			txStreamer,
			reorgingToBlock,
			func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
			fatalErrChan,
		)
		if err != nil {
			return nil, err
		}
	}

	var stakerObj *staker.Staker
	if config.Staker.Enable {
		var wallet staker.ValidatorWalletInterface
		if config.Staker.UseSmartContractWallet || txOpts == nil {
			var existingWalletAddress *common.Address
			if len(config.Staker.ContractWalletAddress) > 0 {
				if !common.IsHexAddress(config.Staker.ContractWalletAddress) {
					log.Error("invalid validator smart contract wallet", "addr", config.Staker.ContractWalletAddress)
					return nil, errors.New("invalid validator smart contract wallet address")
				}
				tmpAddress := common.HexToAddress(config.Staker.ContractWalletAddress)
				existingWalletAddress = &tmpAddress
			}
			wallet, err = staker.NewContractValidatorWallet(existingWalletAddress, deployInfo.ValidatorWalletCreator, deployInfo.Rollup, l1Reader, txOpts, int64(deployInfo.DeployedAt), func(common.Address) {})
			if err != nil {
				return nil, err
			}
		} else {
			if len(config.Staker.ContractWalletAddress) > 0 {
				return nil, errors.New("validator contract wallet specified but flag to use a smart contract wallet was not specified")
			}
			wallet, err = staker.NewEoaValidatorWallet(deployInfo.Rollup, l1client, txOpts)
			if err != nil {
				return nil, err
			}
		}
		stakerObj, err = staker.NewStaker(l1Reader, wallet, bind.CallOpts{}, config.Staker, blockValidator, statelessBlockValidator, deployInfo.ValidatorUtils)
		if err != nil {
			return nil, err
		}
		if stakerObj.Strategy() != staker.WatchtowerStrategy {
			err := wallet.Initialize(ctx)
			if err != nil {
				return nil, err
			}
		}
		var txSenderPtr *common.Address
		if txOpts != nil {
			txSenderPtr = &txOpts.From
		}
		whitelisted, err := stakerObj.IsWhitelisted(ctx)
		if err != nil {
			return nil, err
		}
		log.Info("running as validator", "txSender", txSenderPtr, "actingAsWallet", wallet.Address(), "whitelisted", whitelisted, "strategy", config.Staker.Strategy)
	}

	var batchPoster *BatchPoster
	var delayedSequencer *DelayedSequencer
	if config.BatchPoster.Enable {
		if txOpts == nil {
			return nil, errors.New("batchposter, but no TxOpts")
		}
		batchPoster, err = NewBatchPoster(l1Reader, inboxTracker, txStreamer, syncMonitor, func() *BatchPosterConfig { return &configFetcher.Get().BatchPoster }, deployInfo, txOpts, daWriter)
		if err != nil {
			return nil, err
		}
	}
	// always create DelayedSequencer, it won't do anything if it is disabled
	delayedSequencer, err = NewDelayedSequencer(l1Reader, inboxReader, exec.ExecEngine, coordinator, func() *DelayedSequencerConfig { return &configFetcher.Get().DelayedSequencer })
	if err != nil {
		return nil, err
	}

	return &Node{
		arbDb,
		stack,
		exec,
		l1Reader,
		txStreamer,
		deployInfo,
		inboxReader,
		inboxTracker,
		delayedSequencer,
		batchPoster,
		blockValidator,
		statelessBlockValidator,
		stakerObj,
		broadcastServer,
		broadcastClients,
		coordinator,
		maintenanceRunner,
		dasLifecycleManager,
		classicOutbox,
		syncMonitor,
		configFetcher,
		ctx,
	}, nil
}

func (n *Node) OnConfigReload(_ *Config, _ *Config) error {
	// TODO
	return nil
}

func CreateNode(
	ctx context.Context,
	stack *node.Node,
	chainDb ethdb.Database,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2BlockChain *core.BlockChain,
	l1client arbutil.L1Interface,
	deployInfo *RollupAddresses,
	txOpts *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
) (*Node, error) {
	currentNode, err := createNodeImpl(ctx, stack, chainDb, arbDb, configFetcher, l2BlockChain, l1client, deployInfo, txOpts, dataSigner, fatalErrChan)
	if err != nil {
		return nil, err
	}
	var apis []rpc.API
	if currentNode.BlockValidator != nil {
		apis = append(apis, rpc.API{
			Namespace: "arb",
			Version:   "1.0",
			Service:   &BlockValidatorAPI{val: currentNode.BlockValidator},
			Public:    false,
		})
	}
	if currentNode.StatelessBlockValidator != nil {
		apis = append(apis, rpc.API{
			Namespace: "arbvalidator",
			Version:   "1.0",
			Service: &BlockValidatorDebugAPI{
				val:        currentNode.StatelessBlockValidator,
				blockchain: l2BlockChain,
			},
			Public: false,
		})
	}

	apis = append(apis, rpc.API{
		Namespace: "arb",
		Version:   "1.0",
		Service:   execution.NewArbAPI(currentNode.Execution.TxPublisher),
		Public:    false,
	})
	config := configFetcher.Get()
	apis = append(apis, rpc.API{
		Namespace: "arbdebug",
		Version:   "1.0",
		Service: execution.NewArbDebugAPI(
			l2BlockChain,
			config.RPC.ArbDebug.BlockRangeBound,
			config.RPC.ArbDebug.TimeoutQueueBound,
		),
		Public: false,
	})
	apis = append(apis, rpc.API{
		Namespace: "arbtrace",
		Version:   "1.0",
		Service: execution.NewArbTraceForwarderAPI(
			config.RPC.ClassicRedirect,
			config.RPC.ClassicRedirectTimeout,
		),
		Public: false,
	})
	apis = append(apis, rpc.API{
		Namespace: "debug",
		Service:   eth.NewDebugAPI(eth.NewArbEthereum(l2BlockChain, chainDb)),
		Public:    false,
	})
	stack.RegisterAPIs(apis)

	return currentNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	n.SyncMonitor.Initialize(n.InboxReader, n.TxStreamer, n.SeqCoordinator)
	n.Execution.ArbInterface.Initialize(n)
	err := n.Stack.Start()
	if err != nil {
		return fmt.Errorf("error starting geth stack: %w", err)
	}
	err = n.Execution.Backend.Start()
	if err != nil {
		return fmt.Errorf("error starting geth backend: %w", err)
	}
	err = n.Execution.TxPublisher.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("error initializing transaction publisher: %w", err)
	}
	if n.InboxTracker != nil {
		err = n.InboxTracker.Initialize()
		if err != nil {
			return fmt.Errorf("error initializing inbox tracker: %w", err)
		}
	}
	if n.BroadcastServer != nil {
		err = n.BroadcastServer.Initialize()
		if err != nil {
			return fmt.Errorf("error initializing feed broadcast server: %w", err)
		}
	}
	err = n.TxStreamer.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction streamer: %w", err)
	}
	n.Execution.ExecEngine.Start(ctx)
	if n.InboxReader != nil {
		err = n.InboxReader.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting inbox reader: %w", err)
		}
	}
	err = n.Execution.TxPublisher.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction puiblisher: %w", err)
	}
	if n.SeqCoordinator != nil {
		n.SeqCoordinator.Start(ctx)
	}
	if n.MaintenanceRunner != nil {
		n.MaintenanceRunner.Start(ctx)
	}
	if n.DelayedSequencer != nil {
		n.DelayedSequencer.Start(ctx)
	}
	if n.BatchPoster != nil {
		n.BatchPoster.Start(ctx)
	}
	if n.Staker != nil {
		err = n.Staker.Initialize(ctx)
		if err != nil {
			return fmt.Errorf("error initializing staker: %w", err)
		}
	}
	if n.StatelessBlockValidator != nil {
		err = n.StatelessBlockValidator.Start(ctx)
		if err != nil {
			if n.configFetcher.Get().ValidatorRequired() {
				return fmt.Errorf("error initializing stateless block validator: %w", err)
			} else {
				log.Info("validation not set up", "err", err)
			}
			n.StatelessBlockValidator = nil
			n.BlockValidator = nil
		}
	}
	if n.BlockValidator != nil {
		err = n.BlockValidator.Initialize()
		if err != nil {
			return fmt.Errorf("error initializing block validator: %w", err)
		}
		err = n.BlockValidator.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting block validator: %w", err)
		}
	}
	if n.Staker != nil {
		n.Staker.Start(ctx)
	}
	if n.L1Reader != nil {
		n.L1Reader.Start(ctx)
	}
	if n.BroadcastServer != nil {
		err = n.BroadcastServer.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting feed broadcast server: %w", err)
		}
	}
	if n.BroadcastClients != nil {
		go func() {
			if n.InboxReader != nil {
				select {
				case <-n.InboxReader.CaughtUp():
				case <-ctx.Done():
					return
				}
			}
			n.BroadcastClients.Start(ctx)
		}()
	}
	if n.configFetcher != nil {
		n.configFetcher.Start(ctx)
	}
	return nil
}

func (n *Node) StopAndWait() {
	if n.MaintenanceRunner != nil && n.MaintenanceRunner.Started() {
		n.MaintenanceRunner.StopAndWait()
	}
	if n.configFetcher != nil && n.configFetcher.Started() {
		n.configFetcher.StopAndWait()
	}
	if n.SeqCoordinator != nil && n.SeqCoordinator.Started() {
		// Releases the chosen sequencer lockout,
		// and stops the background thread but not the redis client.
		n.SeqCoordinator.PrepareForShutdown()
	}
	n.Stack.StopRPC() // does nothing if not running
	if n.Execution.TxPublisher.Started() {
		n.Execution.TxPublisher.StopAndWait()
	}
	if n.DelayedSequencer != nil && n.DelayedSequencer.Started() {
		n.DelayedSequencer.StopAndWait()
	}
	if n.BatchPoster != nil && n.BatchPoster.Started() {
		n.BatchPoster.StopAndWait()
	}
	if n.BroadcastServer != nil && n.BroadcastServer.Started() {
		n.BroadcastServer.StopAndWait()
	}
	if n.BroadcastClients != nil {
		n.BroadcastClients.StopAndWait()
	}
	if n.BlockValidator != nil && n.BlockValidator.Started() {
		n.BlockValidator.StopAndWait()
	}
	if n.Staker != nil {
		n.Staker.StopAndWait()
	}
	if n.StatelessBlockValidator != nil {
		n.StatelessBlockValidator.Stop()
	}
	if n.InboxReader != nil && n.InboxReader.Started() {
		n.InboxReader.StopAndWait()
	}
	if n.L1Reader != nil && n.L1Reader.Started() {
		n.L1Reader.StopAndWait()
	}
	if n.TxStreamer.Started() {
		n.TxStreamer.StopAndWait()
	}
	if n.Execution.ExecEngine.Started() {
		n.Execution.ExecEngine.StopAndWait()
	}
	if n.SeqCoordinator != nil && n.SeqCoordinator.Started() {
		// Just stops the redis client (most other stuff was stopped earlier)
		n.SeqCoordinator.StopAndWait()
	}
	n.Execution.ArbInterface.BlockChain().Stop() // does nothing if not running
	if err := n.Execution.Backend.Stop(); err != nil {
		log.Error("backend stop", "err", err)
	}
	if n.DASLifecycleManager != nil {
		n.DASLifecycleManager.StopAndWaitUntil(2 * time.Second)
	}
	if err := n.Stack.Close(); err != nil {
		log.Error("error on stak close", "err", err)
	}
}

func CreateDefaultStackForTest(dataDir string) (*node.Node, error) {
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir = dataDir
	stackConf.HTTPHost = ""
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	if err != nil {
		return nil, fmt.Errorf("error creating protocol stack: %w", err)
	}
	return stack, nil
}
