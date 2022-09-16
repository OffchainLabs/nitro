// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator"
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

func DeployOnL1(ctx context.Context, l1client arbutil.L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address, authorizeValidators uint64, readerConfig headerreader.ConfigFetcher, machineConfig validator.NitroMachineConfig, config rollupgen.Config) (*RollupAddresses, error) {
	l1Reader := headerreader.New(l1client, readerConfig)
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	if config.WasmModuleRoot == (common.Hash{}) {
		var err error
		config.WasmModuleRoot, err = machineConfig.ReadLatestWasmModuleRoot()
		if err != nil {
			return nil, err
		}
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
	RPC                    arbitrum.Config                `koanf:"rpc"`
	Sequencer              SequencerConfig                `koanf:"sequencer" reload:"hot"`
	L1Reader               headerreader.Config            `koanf:"l1-reader" reload:"hot"`
	InboxReader            InboxReaderConfig              `koanf:"inbox-reader" reload:"hot"`
	DelayedSequencer       DelayedSequencerConfig         `koanf:"delayed-sequencer" reload:"hot"`
	BatchPoster            BatchPosterConfig              `koanf:"batch-poster" reload:"hot"`
	ForwardingTargetImpl   string                         `koanf:"forwarding-target"`
	Forwarder              ForwarderConfig                `koanf:"forwarder"`
	TxPreCheckerStrictness uint                           `koanf:"tx-pre-checker-strictness" reload:"hot"`
	BlockValidator         validator.BlockValidatorConfig `koanf:"block-validator" reload:"hot"`
	Feed                   broadcastclient.FeedConfig     `koanf:"feed" reload:"hot"`
	Validator              validator.L1ValidatorConfig    `koanf:"validator"`
	SeqCoordinator         SeqCoordinatorConfig           `koanf:"seq-coordinator"`
	DataAvailability       das.DataAvailabilityConfig     `koanf:"data-availability"`
	Wasm                   WasmConfig                     `koanf:"wasm"`
	SyncMonitor            SyncMonitorConfig              `koanf:"sync-monitor"`
	Dangerous              DangerousConfig                `koanf:"dangerous"`
	Caching                CachingConfig                  `koanf:"caching"`
	Archive                bool                           `koanf:"archive"`
	TxLookupLimit          uint64                         `koanf:"tx-lookup-limit"`
}

func (c *Config) Validate() error {
	if c.L1Reader.Enable && c.Sequencer.Enable && !c.DelayedSequencer.Enable {
		log.Warn("delayed sequencer is not enabled, despite sequencer and l1 reader being enabled")
	}
	if err := c.Sequencer.Validate(); err != nil {
		return err
	}
	if err := c.InboxReader.Validate(); err != nil {
		return err
	}
	if err := c.BatchPoster.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) Get() *Config {
	return c
}

func (c *Config) Start(context.Context) {}

func (c *Config) StopAndWait() {}

func (c *Config) ForwardingTarget() string {
	if c.ForwardingTargetImpl == "null" {
		return ""
	}

	return c.ForwardingTargetImpl
}

func ConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions(prefix+".sequencer", f)
	headerreader.AddOptions(prefix+".l1-reader", f)
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	BatchPosterConfigAddOptions(prefix+".batch-poster", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTargetImpl, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	txPreCheckerDescription := "how strict to be when checking txs before forwarding them. 0 = accept anything, " +
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, " +
		"30 = full validation which may reject txs that would succeed"
	f.Uint(prefix+".tx-pre-checker-strictness", ConfigDefault.TxPreCheckerStrictness, txPreCheckerDescription)
	validator.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, feedInputEnable, feedOutputEnable)
	validator.L1ValidatorConfigAddOptions(prefix+".validator", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	das.DataAvailabilityConfigAddOptions(prefix+".data-availability", f)
	WasmConfigAddOptions(prefix+".wasm", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	CachingConfigAddOptions(prefix+".caching", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")

	archiveMsg := fmt.Sprintf("retain past block state (deprecated, please use %v.caching.archive)", prefix)
	f.Bool(prefix+".archive", ConfigDefault.Archive, archiveMsg)
}

var ConfigDefault = Config{
	RPC:                    arbitrum.DefaultConfig,
	Sequencer:              DefaultSequencerConfig,
	L1Reader:               headerreader.DefaultConfig,
	InboxReader:            DefaultInboxReaderConfig,
	DelayedSequencer:       DefaultDelayedSequencerConfig,
	BatchPoster:            DefaultBatchPosterConfig,
	ForwardingTargetImpl:   "",
	TxPreCheckerStrictness: TxPreCheckerStrictnessNone,
	BlockValidator:         validator.DefaultBlockValidatorConfig,
	Feed:                   broadcastclient.FeedConfigDefault,
	Validator:              validator.DefaultL1ValidatorConfig,
	SeqCoordinator:         DefaultSeqCoordinatorConfig,
	DataAvailability:       das.DefaultDataAvailabilityConfig,
	Wasm:                   DefaultWasmConfig,
	SyncMonitor:            DefaultSyncMonitorConfig,
	Dangerous:              DefaultDangerousConfig,
	Archive:                false,
	TxLookupLimit:          40_000_000,
	Caching:                DefaultCachingConfig,
}

func ConfigDefaultL1Test() *Config {
	config := ConfigDefaultL1NonSequencerTest()
	config.Sequencer = TestSequencerConfig
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
	config.Wasm.RootPath = validator.DefaultNitroMachineConfig.RootPath
	config.BlockValidator = validator.TestBlockValidatorConfig

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.Sequencer = TestSequencerConfig
	config.L1Reader.Enable = false
	config.SeqCoordinator = TestSeqCoordinatorConfig

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

type DangerousSequencerConfig struct {
	NoCoordinator bool `koanf:"no-coordinator"`
}

var DefaultDangerousSequencerConfig = DangerousSequencerConfig{
	NoCoordinator: false,
}

var TestDangerousSequencerConfig = DangerousSequencerConfig{
	NoCoordinator: true,
}

func DangerousSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".no-coordinator", DefaultDangerousSequencerConfig.NoCoordinator, "DANGEROUS! allows sequencer without coordinator.")
}

type WasmConfig struct {
	RootPath string `koanf:"root-path"`
}

func WasmConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".root-path", DefaultWasmConfig.RootPath, "path to machine folders, each containing wasm files (replay.wasm, wasi_stub.wasm, soft-float.wasm, go_stub.wasm, host_io.wasm, brotli.wasm")
}

var DefaultWasmConfig = WasmConfig{
	RootPath: "",
}

func (w *WasmConfig) FindMachineDir() (string, bool) {
	places := []string{}

	if w.RootPath != "" {
		places = append(places, w.RootPath)
	} else {
		// Check the project dir: <project>/arbnode/node.go => ../../target/machines
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("failed to find root path")
		}
		projectDir := filepath.Dir(filepath.Dir(thisFile))
		projectPath := filepath.Join(filepath.Join(projectDir, "target"), "machines")
		places = append(places, projectPath)

		// Check the working directory: ./machines and ./target/machines
		workDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		workPath1 := filepath.Join(workDir, "machines")
		workPath2 := filepath.Join(filepath.Join(workDir, "target"), "machines")
		places = append(places, workPath1)
		places = append(places, workPath2)

		// Check above the executable: <binary> => ../../machines
		execfile, err := os.Executable()
		if err != nil {
			panic(err)
		}
		execPath := filepath.Join(filepath.Dir(filepath.Dir(execfile)), "machines")
		places = append(places, execPath)

		// Check the default
		places = append(places, validator.DefaultNitroMachineConfig.RootPath)
	}

	for _, place := range places {
		if _, err := os.Stat(place); err == nil {
			return place, true
		}
	}
	return "", false
}

type CachingConfig struct {
	Archive       bool          `koanf:"archive"`
	BlockCount    uint64        `koanf:"block-count"`
	BlockAge      time.Duration `koanf:"block-age"`
	TrieTimeLimit time.Duration `koanf:"trie-time-limit"`
}

func CachingConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".archive", DefaultCachingConfig.Archive, "retain past block state")
	f.Uint64(prefix+".block-count", DefaultCachingConfig.BlockCount, "minimum number of recent blocks to keep in memory")
	f.Duration(prefix+".block-age", DefaultCachingConfig.BlockAge, "minimum age a block must be to be pruned")
	f.Duration(prefix+".trie-time-limit", DefaultCachingConfig.TrieTimeLimit, "maximum block processing time before trie is written to hard-disk")
}

var DefaultCachingConfig = CachingConfig{
	Archive:       false,
	BlockCount:    128,
	BlockAge:      30 * time.Minute,
	TrieTimeLimit: time.Hour,
}

type Node struct {
	Backend                 *arbitrum.Backend
	ArbInterface            *ArbInterface
	L1Reader                *headerreader.HeaderReader
	TxStreamer              *TransactionStreamer
	TxPublisher             TransactionPublisher
	DeployInfo              *RollupAddresses
	InboxReader             *InboxReader
	InboxTracker            *InboxTracker
	DelayedSequencer        *DelayedSequencer
	BatchPoster             *BatchPoster
	BlockValidator          *validator.BlockValidator
	StatelessBlockValidator *validator.StatelessBlockValidator
	Staker                  *validator.Staker
	BroadcastServer         *broadcaster.Broadcaster
	BroadcastClients        []*broadcastclient.BroadcastClient
	SeqCoordinator          *SeqCoordinator
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
	var reorgingToBlock *types.Block

	l2Config := l2BlockChain.Config()
	l2ChainId := l2Config.ChainID.Uint64()

	if config.Dangerous.ReorgToBlock >= 0 {
		blockNum := uint64(config.Dangerous.ReorgToBlock)
		if blockNum < l2Config.ArbitrumChainParams.GenesisBlockNum {
			return nil, fmt.Errorf("cannot reorg to block %v past nitro genesis of %v", blockNum, l2Config.ArbitrumChainParams.GenesisBlockNum)
		}
		reorgingToBlock = l2BlockChain.GetBlockByNumber(blockNum)
		if reorgingToBlock == nil {
			return nil, fmt.Errorf("didn't find reorg target block number %v", blockNum)
		}
		err := l2BlockChain.ReorgToOldBlock(reorgingToBlock)
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

	var broadcastServer *broadcaster.Broadcaster
	if config.Feed.Output.Enable {
		broadcastServer = broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config.Get().Feed.Output }, l2ChainId, fatalErrChan, dataSigner)
	}

	var l1Reader *headerreader.HeaderReader
	if config.L1Reader.Enable {
		l1Reader = headerreader.New(l1client, func() *headerreader.Config { return &config.Get().L1Reader })
	}

	var sequencerInboxAddr common.Address
	if deployInfo != nil {
		sequencerInboxAddr = deployInfo.SequencerInbox
	}
	txStreamer, err := NewTransactionStreamer(arbDb, l2BlockChain, broadcastServer)
	if err != nil {
		return nil, err
	}
	var txPublisher TransactionPublisher
	var coordinator *SeqCoordinator
	var sequencer *Sequencer
	if config.Sequencer.Enable {
		if config.ForwardingTarget() != "" {
			return nil, errors.New("sequencer and forwarding target both set")
		}
		if !(config.SeqCoordinator.Enable || config.Sequencer.Dangerous.NoCoordinator) {
			return nil, errors.New("sequencer must be enabled with coordinator, unless dangerous.no-coordinator set")
		}
		sequencerConfigFetcher := func() *SequencerConfig { return &configFetcher.Get().Sequencer }
		if config.L1Reader.Enable {
			if l1client == nil {
				return nil, errors.New("l1client is nil")
			}
			sequencer, err = NewSequencer(txStreamer, l1Reader, sequencerConfigFetcher)
		} else {
			sequencer, err = NewSequencer(txStreamer, nil, sequencerConfigFetcher)
		}
		if err != nil {
			return nil, err
		}
		txPublisher = sequencer
	} else {
		if config.DelayedSequencer.Enable {
			return nil, errors.New("cannot have delayedsequencer without sequencer")
		}
		if config.ForwardingTarget() == "" {
			txPublisher = NewTxDropper()
		} else {
			txPublisher = NewForwarder(config.ForwardingTarget(), &config.Forwarder)
		}
	}
	if config.SeqCoordinator.Enable {
		coordinator, err = NewSeqCoordinator(txStreamer, sequencer, syncMonitor, config.SeqCoordinator)
		if err != nil {
			return nil, err
		}
	}
	txPublisher = NewTxPreChecker(txPublisher, l2BlockChain, func() uint { return configFetcher.Get().TxPreCheckerStrictness })
	arbInterface, err := NewArbInterface(txStreamer, txPublisher)
	if err != nil {
		return nil, err
	}
	backend, err := arbitrum.NewBackend(stack, &config.RPC, chainDb, arbInterface, syncMonitor)
	if err != nil {
		return nil, err
	}

	var bpVerifier *contracts.BatchPosterVerifier
	if l1client != nil {
		seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(sequencerInboxAddr, l1client)
		if err != nil {
			return nil, err
		}
		bpVerifier = contracts.NewBatchPosterVerifier(seqInboxCaller)
	}
	sigVerifier := signature.NewVerifier(config.Feed.Input.RequireSignature, nil, bpVerifier)
	currentMessageCount, err := txStreamer.GetMessageCount()
	if err != nil {
		return nil, err
	}
	var broadcastClients []*broadcastclient.BroadcastClient
	if config.Feed.Input.Enable() {
		for _, address := range config.Feed.Input.URLs {
			client := broadcastclient.NewBroadcastClient(
				config.Feed.Input,
				address,
				l2ChainId,
				currentMessageCount,
				txStreamer,
				fatalErrChan,
				sigVerifier,
			)
			broadcastClients = append(broadcastClients, client)
		}
	}
	if !config.L1Reader.Enable {
		return &Node{
			backend,
			arbInterface,
			nil,
			txStreamer,
			txPublisher,
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
			daReader, dasLifecycleManager, err = SetUpDataAvailability(ctx, &config.DataAvailability, l1Reader, deployInfo)
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
	inboxReader, err := NewInboxReader(inboxTracker, l1client, l1Reader, new(big.Int).SetUint64(deployInfo.DeployedAt), delayedBridge, sequencerInbox, func() *InboxReaderConfig { return &config.Get().InboxReader })
	if err != nil {
		return nil, err
	}
	txStreamer.SetInboxReader(inboxReader)

	blockValidatorConf := &config.BlockValidator
	if blockValidatorConf.Enable && !(blockValidatorConf.ArbitratorValidator || blockValidatorConf.JitValidator) {
		log.Warn("No block-by-block validator configured. Enabling the JIT block validator")
		blockValidatorConf.JitValidator = true
	}

	nitroMachineConfig := validator.DefaultNitroMachineConfig
	machinesPath, foundMachines := config.Wasm.FindMachineDir()
	nitroMachineConfig.RootPath = machinesPath
	nitroMachineConfig.JitCranelift = blockValidatorConf.JitValidatorCranelift
	nitroMachineLoader := validator.NewNitroMachineLoader(nitroMachineConfig, fatalErrChan)

	var blockValidator *validator.BlockValidator
	var statelessBlockValidator *validator.StatelessBlockValidator

	if !foundMachines && blockValidatorConf.Enable {
		return nil, fmt.Errorf("Failed to find machines %v", machinesPath)
	} else if !foundMachines {
		log.Warn("Failed to find machines", "path", machinesPath)
	} else {
		statelessBlockValidator, err = validator.NewStatelessBlockValidator(
			nitroMachineLoader,
			inboxReader,
			inboxTracker,
			txStreamer,
			l2BlockChain,
			rawdb.NewTable(arbDb, blockValidatorPrefix),
			daReader,
			&config.Get().BlockValidator,
			fatalErrChan,
		)
		if err != nil {
			return nil, err
		}

		if blockValidatorConf.Enable {
			blockValidator, err = validator.NewBlockValidator(
				statelessBlockValidator,
				inboxTracker,
				txStreamer,
				nitroMachineLoader,
				reorgingToBlock,
				func() *validator.BlockValidatorConfig { return &config.Get().BlockValidator },
			)
			if err != nil {
				return nil, err
			}
		}
	}

	var staker *validator.Staker
	if config.Validator.Enable {
		// TODO: remember validator wallet in JSON instead of querying it from L1 every time
		wallet, err := validator.NewValidatorWallet(nil, deployInfo.ValidatorWalletCreator, deployInfo.Rollup, l1Reader, txOpts, int64(deployInfo.DeployedAt), func(common.Address) {})
		if err != nil {
			return nil, err
		}
		staker, err = validator.NewStaker(l1Reader, wallet, bind.CallOpts{}, config.Validator, l2BlockChain, daReader, inboxReader, inboxTracker, txStreamer, blockValidator, nitroMachineLoader, deployInfo.ValidatorUtils)
		if err != nil {
			return nil, err
		}
		log.Info("running as validator", "smartContractWallet", wallet.Address(), "strategy", config.Validator.Strategy)
	}

	var batchPoster *BatchPoster
	var delayedSequencer *DelayedSequencer
	if config.BatchPoster.Enable {
		if txOpts == nil {
			return nil, errors.New("batchposter, but no TxOpts")
		}
		batchPoster, err = NewBatchPoster(l1Reader, inboxTracker, txStreamer, syncMonitor, func() *BatchPosterConfig { return &configFetcher.Get().BatchPoster }, deployInfo.SequencerInbox, txOpts, daWriter)
		if err != nil {
			return nil, err
		}
	}
	// always create DelayedSequencer, it won't do anything if it is disabled
	delayedSequencer, err = NewDelayedSequencer(l1Reader, inboxReader, txStreamer, coordinator, func() *DelayedSequencerConfig { return &configFetcher.Get().DelayedSequencer })
	if err != nil {
		return nil, err
	}

	return &Node{
		backend,
		arbInterface,
		l1Reader,
		txStreamer,
		txPublisher,
		deployInfo,
		inboxReader,
		inboxTracker,
		delayedSequencer,
		batchPoster,
		blockValidator,
		statelessBlockValidator,
		staker,
		broadcastServer,
		broadcastClients,
		coordinator,
		dasLifecycleManager,
		classicOutbox,
		syncMonitor,
		configFetcher,
		ctx,
	}, nil
}

func (n *Node) OnConfigReload(old *Config, new *Config) error {
	// TODO
	return nil
}

type L1ReaderCloser struct {
	l1Reader *headerreader.HeaderReader
}

func (c *L1ReaderCloser) Close(_ context.Context) error {
	c.l1Reader.StopOnly()
	return nil
}

func (c *L1ReaderCloser) String() string {
	return "l1 reader closer"
}

// SetUpDataAvailabilityWithoutNode sets up a das.DataAvailabilityService stack
// without relying on any objects already created for setting up the Node.
func SetUpDataAvailabilityWithoutNode(
	ctx context.Context,
	config *das.DataAvailabilityConfig,
) (das.DataAvailabilityService, *das.LifecycleManager, error) {
	var l1Reader *headerreader.HeaderReader
	if config.L1NodeURL != "" && config.L1NodeURL != "none" {
		l1Client, err := das.GetL1Client(ctx, config.L1ConnectionAttempts, config.L1NodeURL)
		if err != nil {
			return nil, nil, err
		}
		l1Reader = headerreader.New(l1Client, func() *headerreader.Config { return &headerreader.DefaultConfig }) // TODO: config
	}
	newDas, lifeCycle, err := SetUpDataAvailability(ctx, config, l1Reader, nil)
	if err != nil {
		return nil, nil, err
	}
	if l1Reader != nil {
		l1Reader.Start(ctx)
		lifeCycle.Register(&L1ReaderCloser{l1Reader})
	}
	return newDas, lifeCycle, err
}

// SetUpDataAvailability sets up a das.DataAvailabilityService stack allowing
// some dependencies that were created for the Node to be injected.
func SetUpDataAvailability(
	ctx context.Context,
	config *das.DataAvailabilityConfig,
	l1Reader *headerreader.HeaderReader,
	deployInfo *RollupAddresses,
) (das.DataAvailabilityService, *das.LifecycleManager, error) {
	if !config.Enable {
		return nil, nil, nil
	}

	var seqInbox *bridgegen.SequencerInbox
	var err error
	var seqInboxCaller *bridgegen.SequencerInboxCaller
	var seqInboxAddress *common.Address

	if l1Reader != nil && deployInfo != nil {
		seqInboxAddress = &deployInfo.SequencerInbox
		seqInbox, err = bridgegen.NewSequencerInbox(deployInfo.SequencerInbox, l1Reader.Client())
		if err != nil {
			return nil, nil, err
		}
		seqInboxCaller = &seqInbox.SequencerInboxCaller
	} else if config.L1NodeURL == "none" && config.SequencerInboxAddress == "none" {
		l1Reader = nil
		seqInboxAddress = nil
	} else if l1Reader != nil && len(config.SequencerInboxAddress) > 0 {
		seqInboxAddress, err = das.OptionalAddressFromString(config.SequencerInboxAddress)
		if err != nil {
			return nil, nil, err
		}
		if seqInboxAddress == nil {
			return nil, nil, errors.New("must provide data-availability.sequencer-inbox-address set to a valid contract address or 'none'")
		}
		seqInbox, err = bridgegen.NewSequencerInbox(*seqInboxAddress, l1Reader.Client())
		if err != nil {
			return nil, nil, err
		}
		seqInboxCaller = &seqInbox.SequencerInboxCaller
	} else {
		return nil, nil, errors.New("data-availabilty.l1-node-url and sequencer-inbox-address must be set to a valid L1 URL and contract address or 'none' if running daserver executable")
	}

	// This function builds up the DataAvailabilityService with the following topology, starting from the leaves.
	/*
			      ChainFetchDAS → Bigcache → Redis →
				       SignAfterStoreDAS →
				              FallbackDAS (if the REST client aggregator was specified)
				              (primary) → RedundantStorage (if multiple persistent backing stores were specified)
				                            → S3
				                            → DiskStorage
				                            → Database
				         (fallback only)→ RESTful client aggregator

		          → : X--delegates to-->Y
	*/
	topLevelStorageService, dasLifecycleManager, err := das.CreatePersistentStorageService(ctx, config)
	if err != nil {
		return nil, nil, err
	}
	hasPersistentStorage := topLevelStorageService != nil

	// Create the REST aggregator if one was requested. If other storage types were enabled above, then
	// the REST aggregator is used as the fallback to them.
	if config.RestfulClientAggregatorConfig.Enable {
		restAgg, err := das.NewRestfulClientAggregator(ctx, &config.RestfulClientAggregatorConfig)
		if err != nil {
			return nil, nil, err
		}
		restAgg.Start(ctx)
		dasLifecycleManager.Register(restAgg)

		// Wrap the primary storage service with the fallback to the restful aggregator
		if hasPersistentStorage {
			syncConf := &config.RestfulClientAggregatorConfig.SyncToStorageConfig
			var retentionPeriodSeconds uint64
			if uint64(syncConf.RetentionPeriod) == math.MaxUint64 {
				retentionPeriodSeconds = math.MaxUint64
			} else {
				retentionPeriodSeconds = uint64(syncConf.RetentionPeriod.Seconds())
			}
			if syncConf.Eager {
				if l1Reader == nil || seqInboxAddress == nil {
					return nil, nil, errors.New("l1-node-url and sequencer-inbox-address must be specified along with sync-to-storage.eager")
				}
				topLevelStorageService, err = das.NewSyncingFallbackStorageService(
					ctx,
					topLevelStorageService,
					restAgg,
					l1Reader,
					*seqInboxAddress,
					syncConf)
				if err != nil {
					return nil, nil, err
				}
			} else {
				topLevelStorageService = das.NewFallbackStorageService(topLevelStorageService, restAgg,
					retentionPeriodSeconds, syncConf.IgnoreWriteErrors, true)
			}
		} else {
			topLevelStorageService = das.NewReadLimitedStorageService(restAgg)
		}
		dasLifecycleManager.Register(topLevelStorageService)
	}

	var topLevelDas das.DataAvailabilityService
	if config.AggregatorConfig.Enable {
		panic("Tried to make an aggregator using wrong factory method")
	}
	if hasPersistentStorage && (config.KeyConfig.KeyDir != "" || config.KeyConfig.PrivKey != "") {
		_seqInboxCaller := seqInboxCaller
		if config.DisableSignatureChecking {
			_seqInboxCaller = nil
		}

		privKey, err := config.KeyConfig.BLSPrivKey()
		if err != nil {
			return nil, nil, err
		}

		// TODO rename StorageServiceDASAdapter
		topLevelDas, err = das.NewSignAfterStoreDASWithSeqInboxCaller(
			privKey,
			_seqInboxCaller,
			topLevelStorageService,
			config.ExtraSignatureCheckingPublicKey,
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		topLevelDas = das.NewReadLimitedDataAvailabilityService(topLevelStorageService)
	}

	// Enable caches, Redis and (local) BigCache. Local is the outermost so it will be tried first.
	if config.RedisCacheConfig.Enable {
		cache, err := das.NewRedisStorageService(config.RedisCacheConfig, das.NewEmptyStorageService())
		dasLifecycleManager.Register(cache)
		if err != nil {
			return nil, nil, err
		}
		topLevelDas = das.NewCacheStorageToDASAdapter(topLevelDas, cache)
	}
	if config.LocalCacheConfig.Enable {
		cache, err := das.NewBigCacheStorageService(config.LocalCacheConfig, das.NewEmptyStorageService())
		dasLifecycleManager.Register(cache)
		if err != nil {
			return nil, nil, err
		}
		topLevelDas = das.NewCacheStorageToDASAdapter(topLevelDas, cache)
	}

	if topLevelDas != nil && seqInbox != nil {
		topLevelDas, err = das.NewChainFetchDASWithSeqInbox(topLevelDas, seqInbox)
		if err != nil {
			return nil, nil, err
		}
	}

	if topLevelDas == nil {
		return nil, nil, errors.New("data-availability.enable was specified but no Data Availability server types were enabled")
	}

	return topLevelDas, dasLifecycleManager, nil
}

type arbNodeLifecycle struct {
	node *Node
}

func (l arbNodeLifecycle) Start() error {
	err := l.node.Start(context.Background())
	if err != nil {
		log.Error("failed to start node", "err", err)
	}
	return err
}

func (l arbNodeLifecycle) Stop() error {
	l.node.StopAndWait()
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
		Service:   &ArbAPI{currentNode.TxPublisher},
		Public:    false,
	})
	config := configFetcher.Get()
	apis = append(apis, rpc.API{
		Namespace: "arbdebug",
		Version:   "1.0",
		Service: &ArbDebugAPI{
			blockchain:        l2BlockChain,
			blockRangeBound:   config.RPC.ArbDebug.BlockRangeBound,
			timeoutQueueBound: config.RPC.ArbDebug.TimeoutQueueBound,
		},
		Public: false,
	})
	stack.RegisterAPIs(apis)

	stack.RegisterLifecycle(arbNodeLifecycle{currentNode})
	return currentNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	n.SyncMonitor.Initialize(n.InboxReader, n.TxStreamer, n.SeqCoordinator)
	n.ArbInterface.Initialize(n)
	err := n.Backend.Start()
	if err != nil {
		return err
	}
	err = n.TxPublisher.Initialize(ctx)
	if err != nil {
		return err
	}
	if n.InboxTracker != nil {
		err = n.InboxTracker.Initialize()
		if err != nil {
			return err
		}
	}
	if n.BroadcastServer != nil {
		err = n.BroadcastServer.Initialize()
		if err != nil {
			return err
		}
	}
	n.TxStreamer.Start(ctx)
	if n.InboxReader != nil {
		err = n.InboxReader.Start(ctx)
		if err != nil {
			return err
		}
	}
	err = n.TxPublisher.Start(ctx)
	if err != nil {
		return err
	}
	if n.SeqCoordinator != nil {
		n.SeqCoordinator.Start(ctx)
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
			return err
		}
	}
	if n.BlockValidator != nil {
		err = n.BlockValidator.Initialize()
		if err != nil {
			return err
		}
		err = n.BlockValidator.Start(ctx)
		if err != nil {
			return err
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
			return err
		}
	}
	for _, client := range n.BroadcastClients {
		client.Start(ctx)
	}
	if n.configFetcher != nil {
		n.configFetcher.Start(ctx)
	}
	return nil
}

func (n *Node) StopAndWait() {
	if n.configFetcher != nil {
		n.configFetcher.StopAndWait()
	}
	for _, client := range n.BroadcastClients {
		client.StopAndWait()
	}
	if n.BroadcastServer != nil {
		n.BroadcastServer.StopAndWait()
	}
	if n.L1Reader != nil {
		n.L1Reader.StopAndWait()
	}
	if n.BlockValidator != nil {
		n.BlockValidator.StopAndWait()
	}
	if n.BatchPoster != nil {
		n.BatchPoster.StopAndWait()
	}
	if n.DelayedSequencer != nil {
		n.DelayedSequencer.StopAndWait()
	}
	if n.InboxReader != nil {
		n.InboxReader.StopAndWait()
	}
	n.TxPublisher.StopAndWait()
	if n.SeqCoordinator != nil {
		n.SeqCoordinator.StopAndWait()
	}
	n.TxStreamer.StopAndWait()
	n.ArbInterface.BlockChain().Stop()
	if err := n.Backend.Stop(); err != nil {
		log.Error("backend stop", "err", err)
	}
	if n.DASLifecycleManager != nil {
		n.DASLifecycleManager.StopAndWaitUntil(2 * time.Second)
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

func DefaultCacheConfigFor(stack *node.Node, cachingConfig *CachingConfig) *core.CacheConfig {
	baseConf := ethconfig.Defaults
	if cachingConfig.Archive {
		baseConf = ethconfig.ArchiveDefaults
	}

	return &core.CacheConfig{
		TrieCleanLimit:      baseConf.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(baseConf.TrieCleanCacheJournal),
		TrieCleanRejournal:  baseConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: baseConf.NoPrefetch,
		TrieDirtyLimit:      baseConf.TrieDirtyCache,
		TrieDirtyDisabled:   cachingConfig.Archive,
		TrieTimeLimit:       cachingConfig.TrieTimeLimit,
		TriesInMemory:       cachingConfig.BlockCount,
		TrieRetention:       cachingConfig.BlockAge,
		SnapshotLimit:       baseConf.SnapshotCache,
		Preimages:           baseConf.Preimages,
	}
}

func WriteOrTestGenblock(chainDb ethdb.Database, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, accountsPerSync uint) error {
	arbstate.RequireHookedGeth()

	EmptyHash := common.Hash{}
	prevHash := EmptyHash
	prevDifficulty := big.NewInt(0)
	blockNumber, err := initData.GetNextBlockNumber()
	if err != nil {
		return err
	}
	storedGenHash := rawdb.ReadCanonicalHash(chainDb, blockNumber)
	timestamp := uint64(0)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(chainDb, blockNumber-1)
		if prevHash == EmptyHash {
			return fmt.Errorf("block number %d not found in database", chainDb)
		}
		prevHeader := rawdb.ReadHeader(chainDb, prevHash, blockNumber-1)
		if prevHeader == nil {
			return fmt.Errorf("block header for block %d not found in database", chainDb)
		}
		timestamp = prevHeader.Time
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, initData, chainConfig, timestamp, accountsPerSync)
	if err != nil {
		return err
	}

	genBlock := arbosState.MakeGenesisBlock(prevHash, blockNumber, timestamp, stateRoot, chainConfig)
	blockHash := genBlock.Hash()

	if storedGenHash == EmptyHash {
		// chainDb did not have genesis block. Initialize it.
		core.WriteHeadBlock(chainDb, genBlock, prevDifficulty)
		log.Info("wrote genesis block", "number", blockNumber, "hash", blockHash)
	} else if storedGenHash != blockHash {
		return fmt.Errorf("database contains data inconsistent with initialization: database has genesis hash %v but we built genesis hash %v", storedGenHash, blockHash)
	} else {
		log.Info("recreated existing genesis block", "number", blockNumber, "hash", blockHash)
	}

	return nil
}

func TryReadStoredChainConfig(chainDb ethdb.Database) *params.ChainConfig {
	EmptyHash := common.Hash{}

	block0Hash := rawdb.ReadCanonicalHash(chainDb, 0)
	if block0Hash == EmptyHash {
		return nil
	}
	return rawdb.ReadChainConfig(chainDb, block0Hash)
}

func WriteOrTestChainConfig(chainDb ethdb.Database, config *params.ChainConfig) error {
	EmptyHash := common.Hash{}

	block0Hash := rawdb.ReadCanonicalHash(chainDb, 0)
	if block0Hash == EmptyHash {
		return errors.New("block 0 not found")
	}
	storedConfig := rawdb.ReadChainConfig(chainDb, block0Hash)
	if storedConfig == nil {
		rawdb.WriteChainConfig(chainDb, block0Hash, config)
		return nil
	}
	height := rawdb.ReadHeaderNumber(chainDb, rawdb.ReadHeadHeaderHash(chainDb))
	if height == nil {
		return errors.New("non empty chain config but empty chain")
	}
	err := storedConfig.CheckCompatible(config, *height)
	if err != nil {
		return err
	}
	rawdb.WriteChainConfig(chainDb, block0Hash, config)
	return nil
}

func GetBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, chainConfig *params.ChainConfig, nodeConfig *Config) (*core.BlockChain, error) {
	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: false,
	}

	return core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &nodeConfig.TxLookupLimit)
}

func WriteOrTestBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, nodeConfig *Config, accountsPerSync uint) (*core.BlockChain, error) {
	err := WriteOrTestGenblock(chainDb, initData, chainConfig, accountsPerSync)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, chainConfig)
	if err != nil {
		return nil, err
	}
	return GetBlockChain(chainDb, cacheConfig, chainConfig, nodeConfig)
}

// Don't preserve reorg'd out blocks
func shouldPreserveFalse(_ *types.Header) bool {
	return false
}
