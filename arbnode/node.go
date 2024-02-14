// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func GenerateRollupConfig(prod bool, wasmModuleRoot common.Hash, rollupOwner common.Address, chainConfig *params.ChainConfig, serializedChainConfig []byte, loserStakeEscrow common.Address) rollupgen.Config {
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
		ChainId:                  chainConfig.ChainID,
		// TODO could the ChainConfig be just []byte?
		ChainConfig: string(serializedChainConfig),
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   60 * 60 * 24 / 15,
			FutureBlocks:  12,
			DelaySeconds:  60 * 60 * 24,
			FutureSeconds: 60 * 60,
		},
	}
}

type Config struct {
	Sequencer           bool                        `koanf:"sequencer"`
	ParentChainReader   headerreader.Config         `koanf:"parent-chain-reader" reload:"hot"`
	InboxReader         InboxReaderConfig           `koanf:"inbox-reader" reload:"hot"`
	DelayedSequencer    DelayedSequencerConfig      `koanf:"delayed-sequencer" reload:"hot"`
	BatchPoster         BatchPosterConfig           `koanf:"batch-poster" reload:"hot"`
	MessagePruner       MessagePrunerConfig         `koanf:"message-pruner" reload:"hot"`
	BlockValidator      staker.BlockValidatorConfig `koanf:"block-validator" reload:"hot"`
	Feed                broadcastclient.FeedConfig  `koanf:"feed" reload:"hot"`
	Staker              staker.L1ValidatorConfig    `koanf:"staker" reload:"hot"`
	SeqCoordinator      SeqCoordinatorConfig        `koanf:"seq-coordinator"`
	DataAvailability    das.DataAvailabilityConfig  `koanf:"data-availability"`
	BlobClient          BlobClientConfig            `koanf:"blob-client"`
	SyncMonitor         SyncMonitorConfig           `koanf:"sync-monitor"`
	Dangerous           DangerousConfig             `koanf:"dangerous"`
	TransactionStreamer TransactionStreamerConfig   `koanf:"transaction-streamer" reload:"hot"`
	Maintenance         MaintenanceConfig           `koanf:"maintenance" reload:"hot"`
	ResourceMgmt        resourcemanager.Config      `koanf:"resource-mgmt" reload:"hot"`
}

func (c *Config) Validate() error {
	if c.ParentChainReader.Enable && c.Sequencer && !c.DelayedSequencer.Enable {
		log.Warn("delayed sequencer is not enabled, despite sequencer and l1 reader being enabled")
	}
	if c.DelayedSequencer.Enable && !c.Sequencer {
		return errors.New("cannot enable delayed sequencer without enabling sequencer")
	}
	if c.InboxReader.ReadMode != "latest" {
		if c.Sequencer {
			return errors.New("cannot enable inboxreader in safe or finalized mode along with sequencer")
		}
		c.Feed.Output.Enable = false
		c.Feed.Input.URL = []string{}
	}
	if err := c.BlockValidator.Validate(); err != nil {
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
	if err := c.Staker.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) ValidatorRequired() bool {
	if c.BlockValidator.Enable {
		return true
	}
	if c.Staker.Enable {
		return c.Staker.ValidatorRequired()
	}
	return false
}

func ConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	f.Bool(prefix+".sequencer", ConfigDefault.Sequencer, "enable sequencer")
	headerreader.AddOptions(prefix+".parent-chain-reader", f)
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	BatchPosterConfigAddOptions(prefix+".batch-poster", f)
	MessagePrunerConfigAddOptions(prefix+".message-pruner", f)
	staker.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, feedInputEnable, feedOutputEnable)
	staker.L1ValidatorConfigAddOptions(prefix+".staker", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	das.DataAvailabilityConfigAddNodeOptions(prefix+".data-availability", f)
	BlobClientAddOptions(prefix+".blob-client", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	TransactionStreamerConfigAddOptions(prefix+".transaction-streamer", f)
	MaintenanceConfigAddOptions(prefix+".maintenance", f)
}

var ConfigDefault = Config{
	Sequencer:           false,
	ParentChainReader:   headerreader.DefaultConfig,
	InboxReader:         DefaultInboxReaderConfig,
	DelayedSequencer:    DefaultDelayedSequencerConfig,
	BatchPoster:         DefaultBatchPosterConfig,
	MessagePruner:       DefaultMessagePrunerConfig,
	BlockValidator:      staker.DefaultBlockValidatorConfig,
	Feed:                broadcastclient.FeedConfigDefault,
	Staker:              staker.DefaultL1ValidatorConfig,
	SeqCoordinator:      DefaultSeqCoordinatorConfig,
	DataAvailability:    das.DefaultDataAvailabilityConfig,
	SyncMonitor:         DefaultSyncMonitorConfig,
	Dangerous:           DefaultDangerousConfig,
	TransactionStreamer: DefaultTransactionStreamerConfig,
	ResourceMgmt:        resourcemanager.DefaultConfig,
	Maintenance:         DefaultMaintenanceConfig,
}

func ConfigDefaultL1Test() *Config {
	config := ConfigDefaultL1NonSequencerTest()
	config.DelayedSequencer = TestDelayedSequencerConfig
	config.BatchPoster = TestBatchPosterConfig
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Sequencer = true
	config.Dangerous.NoSequencerCoordinator = true

	return config
}

func ConfigDefaultL1NonSequencerTest() *Config {
	config := ConfigDefault
	config.ParentChainReader = headerreader.TestConfig
	config.InboxReader = TestInboxReaderConfig
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.SeqCoordinator.Enable = false
	config.BlockValidator = staker.TestBlockValidatorConfig
	config.Staker = staker.TestL1ValidatorConfig
	config.Staker.Enable = false
	config.BlockValidator.ValidationServerConfigs = []rpcclient.ClientConfig{{URL: ""}}

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.ParentChainReader.Enable = false
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Feed.Input.Verify.Dangerous.AcceptMissing = true
	config.Feed.Output.Signed = false
	config.SeqCoordinator.Signer.ECDSA.AcceptSequencer = false
	config.SeqCoordinator.Signer.ECDSA.Dangerous.AcceptMissing = true
	config.Staker = staker.TestL1ValidatorConfig
	config.Staker.Enable = false
	config.BlockValidator.ValidationServerConfigs = []rpcclient.ClientConfig{{URL: ""}}
	config.TransactionStreamer = DefaultTransactionStreamerConfig

	return &config
}

type DangerousConfig struct {
	NoL1Listener           bool `koanf:"no-l1-listener"`
	NoSequencerCoordinator bool `koanf:"no-sequencer-coordinator"`
}

var DefaultDangerousConfig = DangerousConfig{
	NoL1Listener:           false,
	NoSequencerCoordinator: false,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".no-l1-listener", DefaultDangerousConfig.NoL1Listener, "DANGEROUS! disables listening to L1. To be used in test nodes only")
	f.Bool(prefix+".no-sequencer-coordinator", DefaultDangerousConfig.NoSequencerCoordinator, "DANGEROUS! allows sequencing without sequencer-coordinator")
}

type Node struct {
	ArbDB                   ethdb.Database
	Stack                   *node.Node
	Execution               execution.FullExecutionClient
	L1Reader                *headerreader.HeaderReader
	TxStreamer              *TransactionStreamer
	DeployInfo              *chaininfo.RollupAddresses
	InboxReader             *InboxReader
	InboxTracker            *InboxTracker
	DelayedSequencer        *DelayedSequencer
	BatchPoster             *BatchPoster
	MessagePruner           *MessagePruner
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

func StakerDataposter(
	ctx context.Context, db ethdb.Database, l1Reader *headerreader.HeaderReader,
	transactOpts *bind.TransactOpts, cfgFetcher ConfigFetcher, syncMonitor *SyncMonitor,
	parentChainID *big.Int,
) (*dataposter.DataPoster, error) {
	cfg := cfgFetcher.Get()
	if transactOpts == nil && cfg.Staker.DataPoster.ExternalSigner.URL == "" {
		return nil, nil
	}
	mdRetriever := func(ctx context.Context, blockNum *big.Int) ([]byte, error) {
		return nil, nil
	}
	redisC, err := redisutil.RedisClientFromURL(cfg.Staker.RedisUrl)
	if err != nil {
		return nil, fmt.Errorf("creating redis client from url: %w", err)
	}
	dpCfg := func() *dataposter.DataPosterConfig {
		return &cfg.Staker.DataPoster
	}
	var sender string
	if transactOpts != nil {
		sender = transactOpts.From.String()
	} else {
		sender = cfg.Staker.DataPoster.ExternalSigner.Address
	}
	return dataposter.NewDataPoster(ctx,
		&dataposter.DataPosterOpts{
			Database:          db,
			HeaderReader:      l1Reader,
			Auth:              transactOpts,
			RedisClient:       redisC,
			Config:            dpCfg,
			MetadataRetriever: mdRetriever,
			RedisKey:          sender + ".staker-data-poster.queue",
			ParentChainID:     parentChainID,
		})
}

func createNodeImpl(
	ctx context.Context,
	stack *node.Node,
	exec execution.FullExecutionClient,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client arbutil.L1Interface,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	parentChainID *big.Int,
) (*Node, error) {
	config := configFetcher.Get()

	err := checkArbDbSchemaVersion(arbDb)
	if err != nil {
		return nil, err
	}

	l2ChainId := l2Config.ChainID.Uint64()

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
	if config.ParentChainReader.Enable {
		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1client)
		l1Reader, err = headerreader.New(ctx, l1client, func() *headerreader.Config { return &configFetcher.Get().ParentChainReader }, arbSys)
		if err != nil {
			return nil, err
		}
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
	txStreamer, err := NewTransactionStreamer(arbDb, l2Config, exec, broadcastServer, fatalErrChan, transactionStreamerConfigFetcher)
	if err != nil {
		return nil, err
	}
	var coordinator *SeqCoordinator
	var bpVerifier *contracts.AddressVerifier
	if deployInfo != nil && l1client != nil {
		sequencerInboxAddr := deployInfo.SequencerInbox

		seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(sequencerInboxAddr, l1client)
		if err != nil {
			return nil, err
		}
		bpVerifier = contracts.NewAddressVerifier(seqInboxCaller)
	}

	if config.SeqCoordinator.Enable {
		coordinator, err = NewSeqCoordinator(dataSigner, bpVerifier, txStreamer, exec, syncMonitor, config.SeqCoordinator)
		if err != nil {
			return nil, err
		}
	} else if config.Sequencer && !config.Dangerous.NoSequencerCoordinator {
		return nil, errors.New("sequencer must be enabled with coordinator, unless dangerous.no-sequencer-coordinator set")
	}
	dbs := []ethdb.Database{arbDb}
	maintenanceRunner, err := NewMaintenanceRunner(func() *MaintenanceConfig { return &configFetcher.Get().Maintenance }, coordinator, dbs, exec)
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

	if !config.ParentChainReader.Enable {
		return &Node{
			ArbDB:                   arbDb,
			Stack:                   stack,
			Execution:               exec,
			L1Reader:                nil,
			TxStreamer:              txStreamer,
			DeployInfo:              nil,
			InboxReader:             nil,
			InboxTracker:            nil,
			DelayedSequencer:        nil,
			BatchPoster:             nil,
			MessagePruner:           nil,
			BlockValidator:          nil,
			StatelessBlockValidator: nil,
			Staker:                  nil,
			BroadcastServer:         broadcastServer,
			BroadcastClients:        broadcastClients,
			SeqCoordinator:          coordinator,
			MaintenanceRunner:       maintenanceRunner,
			DASLifecycleManager:     nil,
			ClassicOutboxRetriever:  classicOutbox,
			SyncMonitor:             syncMonitor,
			configFetcher:           configFetcher,
			ctx:                     ctx,
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
	} else if l2Config.ArbitrumChainParams.DataAvailabilityCommittee {
		return nil, errors.New("a data availability service is required for this chain, but it was not configured")
	}

	var blobReader arbstate.BlobReader
	if config.BlobClient.BeaconChainUrl != "" {
		blobReader, err = NewBlobClient(config.BlobClient, l1client)
		if err != nil {
			return nil, err
		}
	}

	inboxTracker, err := NewInboxTracker(arbDb, txStreamer, daReader, blobReader)
	if err != nil {
		return nil, err
	}
	inboxReader, err := NewInboxReader(inboxTracker, l1client, l1Reader, new(big.Int).SetUint64(deployInfo.DeployedAt), delayedBridge, sequencerInbox, func() *InboxReaderConfig { return &configFetcher.Get().InboxReader })
	if err != nil {
		return nil, err
	}
	txStreamer.SetInboxReaders(inboxReader, delayedBridge)

	var statelessBlockValidator *staker.StatelessBlockValidator
	if config.BlockValidator.ValidationServerConfigs[0].URL != "" {
		statelessBlockValidator, err = staker.NewStatelessBlockValidator(
			inboxReader,
			inboxTracker,
			txStreamer,
			exec,
			rawdb.NewTable(arbDb, storage.BlockValidatorPrefix),
			daReader,
			blobReader,
			func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
			stack,
		)
	} else {
		err = errors.New("no validator url specified")
	}
	if err != nil {
		if config.ValidatorRequired() || config.Staker.Enable {
			return nil, fmt.Errorf("%w: failed to init block validator", err)
		}
		log.Warn("validation not supported", "err", err)
		statelessBlockValidator = nil
	}

	var blockValidator *staker.BlockValidator
	if config.ValidatorRequired() {
		blockValidator, err = staker.NewBlockValidator(
			statelessBlockValidator,
			inboxTracker,
			txStreamer,
			func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
			fatalErrChan,
		)
		if err != nil {
			return nil, err
		}
	}

	var stakerObj *staker.Staker
	var messagePruner *MessagePruner

	if config.Staker.Enable {
		dp, err := StakerDataposter(
			ctx,
			rawdb.NewTable(arbDb, storage.StakerPrefix),
			l1Reader,
			txOptsValidator,
			configFetcher,
			syncMonitor,
			parentChainID,
		)
		if err != nil {
			return nil, err
		}
		getExtraGas := func() uint64 { return configFetcher.Get().Staker.ExtraGas }
		// TODO: factor this out into separate helper, and split rest of node
		// creation into multiple helpers.
		var wallet staker.ValidatorWalletInterface = validatorwallet.NewNoOp(l1client, deployInfo.Rollup)
		if !strings.EqualFold(config.Staker.Strategy, "watchtower") {
			if config.Staker.UseSmartContractWallet || (txOptsValidator == nil && config.Staker.DataPoster.ExternalSigner.URL == "") {
				var existingWalletAddress *common.Address
				if len(config.Staker.ContractWalletAddress) > 0 {
					if !common.IsHexAddress(config.Staker.ContractWalletAddress) {
						log.Error("invalid validator smart contract wallet", "addr", config.Staker.ContractWalletAddress)
						return nil, errors.New("invalid validator smart contract wallet address")
					}
					tmpAddress := common.HexToAddress(config.Staker.ContractWalletAddress)
					existingWalletAddress = &tmpAddress
				}
				wallet, err = validatorwallet.NewContract(dp, existingWalletAddress, deployInfo.ValidatorWalletCreator, deployInfo.Rollup, l1Reader, txOptsValidator, int64(deployInfo.DeployedAt), func(common.Address) {}, getExtraGas)
				if err != nil {
					return nil, err
				}
			} else {
				if len(config.Staker.ContractWalletAddress) > 0 {
					return nil, errors.New("validator contract wallet specified but flag to use a smart contract wallet was not specified")
				}
				wallet, err = validatorwallet.NewEOA(dp, deployInfo.Rollup, l1client, getExtraGas)
				if err != nil {
					return nil, err
				}
			}
		}

		var confirmedNotifiers []staker.LatestConfirmedNotifier
		if config.MessagePruner.Enable {
			messagePruner = NewMessagePruner(txStreamer, inboxTracker, func() *MessagePrunerConfig { return &configFetcher.Get().MessagePruner })
			confirmedNotifiers = append(confirmedNotifiers, messagePruner)
		}

		stakerObj, err = staker.NewStaker(l1Reader, wallet, bind.CallOpts{}, config.Staker, blockValidator, statelessBlockValidator, nil, confirmedNotifiers, deployInfo.ValidatorUtils, fatalErrChan)
		if err != nil {
			return nil, err
		}
		if err := wallet.Initialize(ctx); err != nil {
			return nil, err
		}
		var validatorAddr string
		if txOptsValidator != nil {
			validatorAddr = txOptsValidator.From.String()
		} else {
			validatorAddr = config.Staker.DataPoster.ExternalSigner.Address
		}
		whitelisted, err := stakerObj.IsWhitelisted(ctx)
		if err != nil {
			return nil, err
		}
		log.Info("running as validator", "txSender", validatorAddr, "actingAsWallet", wallet.Address(), "whitelisted", whitelisted, "strategy", config.Staker.Strategy)
	}

	var batchPoster *BatchPoster
	var delayedSequencer *DelayedSequencer
	if config.BatchPoster.Enable {
		if txOptsBatchPoster == nil && config.BatchPoster.DataPoster.ExternalSigner.URL == "" {
			return nil, errors.New("batchposter, but no TxOpts")
		}
		batchPoster, err = NewBatchPoster(ctx, &BatchPosterOpts{
			DataPosterDB:  rawdb.NewTable(arbDb, storage.BatchPosterPrefix),
			L1Reader:      l1Reader,
			Inbox:         inboxTracker,
			Streamer:      txStreamer,
			VersionGetter: exec,
			SyncMonitor:   syncMonitor,
			Config:        func() *BatchPosterConfig { return &configFetcher.Get().BatchPoster },
			DeployInfo:    deployInfo,
			TransactOpts:  txOptsBatchPoster,
			DAWriter:      daWriter,
			ParentChainID: parentChainID,
		})
		if err != nil {
			return nil, err
		}
	}

	// always create DelayedSequencer, it won't do anything if it is disabled
	delayedSequencer, err = NewDelayedSequencer(l1Reader, inboxReader, exec, coordinator, func() *DelayedSequencerConfig { return &configFetcher.Get().DelayedSequencer })
	if err != nil {
		return nil, err
	}

	return &Node{
		ArbDB:                   arbDb,
		Stack:                   stack,
		Execution:               exec,
		L1Reader:                l1Reader,
		TxStreamer:              txStreamer,
		DeployInfo:              deployInfo,
		InboxReader:             inboxReader,
		InboxTracker:            inboxTracker,
		DelayedSequencer:        delayedSequencer,
		BatchPoster:             batchPoster,
		MessagePruner:           messagePruner,
		BlockValidator:          blockValidator,
		StatelessBlockValidator: statelessBlockValidator,
		Staker:                  stakerObj,
		BroadcastServer:         broadcastServer,
		BroadcastClients:        broadcastClients,
		SeqCoordinator:          coordinator,
		MaintenanceRunner:       maintenanceRunner,
		DASLifecycleManager:     dasLifecycleManager,
		ClassicOutboxRetriever:  classicOutbox,
		SyncMonitor:             syncMonitor,
		configFetcher:           configFetcher,
		ctx:                     ctx,
	}, nil
}

func (n *Node) OnConfigReload(_ *Config, _ *Config) error {
	// TODO
	return nil
}

func CreateNode(
	ctx context.Context,
	stack *node.Node,
	exec execution.FullExecutionClient,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client arbutil.L1Interface,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	parentChainID *big.Int,
) (*Node, error) {
	currentNode, err := createNodeImpl(ctx, stack, exec, arbDb, configFetcher, l2Config, l1client, deployInfo, txOptsValidator, txOptsBatchPoster, dataSigner, fatalErrChan, parentChainID)
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
				val: currentNode.StatelessBlockValidator,
			},
			Public: false,
		})
	}

	stack.RegisterAPIs(apis)

	return currentNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	execClient, ok := n.Execution.(*gethexec.ExecutionNode)
	if !ok {
		execClient = nil
	}
	if execClient != nil {
		err := execClient.Initialize(ctx, n, n.SyncMonitor)
		if err != nil {
			return fmt.Errorf("error initializing exec client: %w", err)
		}
	}
	n.SyncMonitor.Initialize(n.InboxReader, n.TxStreamer, n.SeqCoordinator, n.Execution)
	err := n.Stack.Start()
	if err != nil {
		return fmt.Errorf("error starting geth stack: %w", err)
	}
	err = n.Execution.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting exec client: %w", err)
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
	if n.InboxTracker != nil && n.BroadcastServer != nil {
		// Even if the sequencer coordinator will populate this backlog,
		// we want to make sure it's populated before any clients connect.
		err = n.InboxTracker.PopulateFeedBacklog(n.BroadcastServer)
		if err != nil {
			return fmt.Errorf("error populating feed backlog on startup: %w", err)
		}
	}
	err = n.TxStreamer.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction streamer: %w", err)
	}
	if n.InboxReader != nil {
		err = n.InboxReader.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting inbox reader: %w", err)
		}
	}
	// must init broadcast server before trying to sequence anything
	if n.BroadcastServer != nil {
		err = n.BroadcastServer.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting feed broadcast server: %w", err)
		}
	}
	if n.SeqCoordinator != nil {
		n.SeqCoordinator.Start(ctx)
	} else {
		if n.DelayedSequencer != nil {
			err := n.DelayedSequencer.ForceSequenceDelayed(ctx)
			if err != nil {
				return fmt.Errorf("error initially sequencing delayed instructions: %w", err)
			}
		}
		n.Execution.Activate()
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
	if n.MessagePruner != nil {
		n.MessagePruner.Start(ctx)
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
			}
			log.Info("validation not set up", "err", err)
			n.StatelessBlockValidator = nil
			n.BlockValidator = nil
		}
	}
	if n.BlockValidator != nil {
		err = n.BlockValidator.Initialize(ctx)
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
	if n.DelayedSequencer != nil && n.DelayedSequencer.Started() {
		n.DelayedSequencer.StopAndWait()
	}
	if n.BatchPoster != nil && n.BatchPoster.Started() {
		n.BatchPoster.StopAndWait()
	}
	if n.MessagePruner != nil && n.MessagePruner.Started() {
		n.MessagePruner.StopAndWait()
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
	if n.SeqCoordinator != nil && n.SeqCoordinator.Started() {
		// Just stops the redis client (most other stuff was stopped earlier)
		n.SeqCoordinator.StopAndWait()
	}
	if n.DASLifecycleManager != nil {
		n.DASLifecycleManager.StopAndWaitUntil(2 * time.Second)
	}
	if n.Execution != nil {
		n.Execution.StopAndWait()
	}
	if err := n.Stack.Close(); err != nil {
		log.Error("error on stack close", "err", err)
	}
}
