// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasserver"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker"
	boldstaker "github.com/offchainlabs/nitro/staker/bold"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
	multiprotocolstaker "github.com/offchainlabs/nitro/staker/multi_protocol"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Config struct {
	Sequencer                bool                           `koanf:"sequencer"`
	ParentChainReader        headerreader.Config            `koanf:"parent-chain-reader" reload:"hot"`
	InboxReader              InboxReaderConfig              `koanf:"inbox-reader" reload:"hot"`
	DelayedSequencer         DelayedSequencerConfig         `koanf:"delayed-sequencer" reload:"hot"`
	BatchPoster              BatchPosterConfig              `koanf:"batch-poster" reload:"hot"`
	MessagePruner            MessagePrunerConfig            `koanf:"message-pruner" reload:"hot"`
	BlockValidator           staker.BlockValidatorConfig    `koanf:"block-validator" reload:"hot"`
	Feed                     broadcastclient.FeedConfig     `koanf:"feed" reload:"hot"`
	Staker                   legacystaker.L1ValidatorConfig `koanf:"staker" reload:"hot"`
	Bold                     boldstaker.BoldConfig          `koanf:"bold"`
	SeqCoordinator           SeqCoordinatorConfig           `koanf:"seq-coordinator"`
	DataAvailability         das.DataAvailabilityConfig     `koanf:"data-availability"`
	DAProvider               daclient.ClientConfig          `koanf:"da-provider" reload:"hot"`
	SyncMonitor              SyncMonitorConfig              `koanf:"sync-monitor"`
	Dangerous                DangerousConfig                `koanf:"dangerous"`
	TransactionStreamer      TransactionStreamerConfig      `koanf:"transaction-streamer" reload:"hot"`
	Maintenance              MaintenanceConfig              `koanf:"maintenance" reload:"hot"`
	ResourceMgmt             resourcemanager.Config         `koanf:"resource-mgmt" reload:"hot"`
	BlockMetadataFetcher     BlockMetadataFetcherConfig     `koanf:"block-metadata-fetcher" reload:"hot"`
	ConsensusExecutionSyncer ConsensusExecutionSyncerConfig `koanf:"consensus-execution-syncer"`
	// SnapSyncConfig is only used for testing purposes, these should not be configured in production.
	SnapSyncTest SnapSyncConfig
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
	if c.TransactionStreamer.TrackBlockMetadataFrom != 0 && !c.BlockMetadataFetcher.Enable {
		log.Warn("track-block-metadata-from is set but blockMetadata fetcher is not enabled")
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
	legacystaker.L1ValidatorConfigAddOptions(prefix+".staker", f)
	boldstaker.BoldConfigAddOptions(prefix+".bold", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	das.DataAvailabilityConfigAddNodeOptions(prefix+".data-availability", f)
	daclient.ClientConfigAddOptions(prefix+".da-provider", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	TransactionStreamerConfigAddOptions(prefix+".transaction-streamer", f)
	MaintenanceConfigAddOptions(prefix+".maintenance", f)
	resourcemanager.ConfigAddOptions(prefix+".resource-mgmt", f)
	BlockMetadataFetcherConfigAddOptions(prefix+".block-metadata-fetcher", f)
	ConsensusExecutionSyncerConfigAddOptions(prefix+".consensus-execution-syncer", f)
}

var ConfigDefault = Config{
	Sequencer:                false,
	ParentChainReader:        headerreader.DefaultConfig,
	InboxReader:              DefaultInboxReaderConfig,
	DelayedSequencer:         DefaultDelayedSequencerConfig,
	BatchPoster:              DefaultBatchPosterConfig,
	MessagePruner:            DefaultMessagePrunerConfig,
	BlockValidator:           staker.DefaultBlockValidatorConfig,
	Feed:                     broadcastclient.FeedConfigDefault,
	Staker:                   legacystaker.DefaultL1ValidatorConfig,
	Bold:                     boldstaker.DefaultBoldConfig,
	SeqCoordinator:           DefaultSeqCoordinatorConfig,
	DataAvailability:         das.DefaultDataAvailabilityConfig,
	DAProvider:               daclient.DefaultClientConfig,
	SyncMonitor:              DefaultSyncMonitorConfig,
	Dangerous:                DefaultDangerousConfig,
	TransactionStreamer:      DefaultTransactionStreamerConfig,
	ResourceMgmt:             resourcemanager.DefaultConfig,
	BlockMetadataFetcher:     DefaultBlockMetadataFetcherConfig,
	Maintenance:              DefaultMaintenanceConfig,
	ConsensusExecutionSyncer: DefaultConsensusExecutionSyncerConfig,
	SnapSyncTest:             DefaultSnapSyncConfig,
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
	config.Dangerous = TestDangerousConfig
	config.ParentChainReader = headerreader.TestConfig
	config.InboxReader = TestInboxReaderConfig
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.SeqCoordinator.Enable = false
	config.BlockValidator = staker.TestBlockValidatorConfig
	config.SyncMonitor = TestSyncMonitorConfig
	config.Staker = legacystaker.TestL1ValidatorConfig
	config.Staker.Enable = false
	config.BlockValidator.ValidationServerConfigs = []rpcclient.ClientConfig{{URL: ""}}
	config.Bold.MinimumGapToParentAssertion = 0

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.Dangerous = TestDangerousConfig
	config.ParentChainReader.Enable = false
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Feed.Input.Verify.Dangerous.AcceptMissing = true
	config.Feed.Output.Signed = false
	config.SeqCoordinator.Signer.ECDSA.AcceptSequencer = false
	config.SeqCoordinator.Signer.ECDSA.Dangerous.AcceptMissing = true
	config.Staker = legacystaker.TestL1ValidatorConfig
	config.SyncMonitor = TestSyncMonitorConfig
	config.Staker.Enable = false
	config.BlockValidator.ValidationServerConfigs = []rpcclient.ClientConfig{{URL: ""}}
	config.TransactionStreamer = DefaultTransactionStreamerConfig
	config.Bold.MinimumGapToParentAssertion = 0

	return &config
}

type DangerousConfig struct {
	NoL1Listener           bool `koanf:"no-l1-listener"`
	NoSequencerCoordinator bool `koanf:"no-sequencer-coordinator"`
	DisableBlobReader      bool `koanf:"disable-blob-reader"`
}

var DefaultDangerousConfig = DangerousConfig{
	NoL1Listener:           false,
	NoSequencerCoordinator: false,
	DisableBlobReader:      false,
}

var TestDangerousConfig = DangerousConfig{
	NoL1Listener:           false,
	NoSequencerCoordinator: false,
	DisableBlobReader:      true,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".no-l1-listener", DefaultDangerousConfig.NoL1Listener, "DANGEROUS! disables listening to L1. To be used in test nodes only")
	f.Bool(prefix+".no-sequencer-coordinator", DefaultDangerousConfig.NoSequencerCoordinator, "DANGEROUS! allows sequencing without sequencer-coordinator")
	f.Bool(prefix+".disable-blob-reader", DefaultDangerousConfig.DisableBlobReader, "DANGEROUS! disables the EIP-4844 blob reader, which is necessary to read batches")
}

type Node struct {
	ArbDB                    ethdb.Database
	Stack                    *node.Node
	ExecutionClient          execution.ExecutionClient
	ExecutionSequencer       execution.ExecutionSequencer
	ExecutionRecorder        execution.ExecutionRecorder
	L1Reader                 *headerreader.HeaderReader
	TxStreamer               *TransactionStreamer
	DeployInfo               *chaininfo.RollupAddresses
	BlobReader               daprovider.BlobReader
	InboxReader              *InboxReader
	InboxTracker             *InboxTracker
	DelayedSequencer         *DelayedSequencer
	BatchPoster              *BatchPoster
	MessagePruner            *MessagePruner
	BlockValidator           *staker.BlockValidator
	StatelessBlockValidator  *staker.StatelessBlockValidator
	Staker                   *multiprotocolstaker.MultiProtocolStaker
	BroadcastServer          *broadcaster.Broadcaster
	BroadcastClients         *broadcastclients.BroadcastClients
	SeqCoordinator           *SeqCoordinator
	MaintenanceRunner        *MaintenanceRunner
	dasServerCloseFn         func()
	DASLifecycleManager      *das.LifecycleManager
	SyncMonitor              *SyncMonitor
	blockMetadataFetcher     *BlockMetadataFetcher
	configFetcher            ConfigFetcher
	ctx                      context.Context
	ConsensusExecutionSyncer *ConsensusExecutionSyncer
}

type SnapSyncConfig struct {
	Enabled                   bool
	PrevBatchMessageCount     uint64
	PrevDelayedRead           uint64
	BatchCount                uint64
	DelayedCount              uint64
	ParentChainAssertionBlock uint64
}

var DefaultSnapSyncConfig = SnapSyncConfig{
	Enabled:                   false,
	PrevBatchMessageCount:     0,
	PrevDelayedRead:           0,
	BatchCount:                0,
	DelayedCount:              0,
	ParentChainAssertionBlock: 0,
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

func DataposterOnlyUsedToCreateValidatorWalletContract(
	ctx context.Context,
	l1Reader *headerreader.HeaderReader,
	transactOpts *bind.TransactOpts,
	cfg *dataposter.DataPosterConfig,
	parentChainID *big.Int,
) (*dataposter.DataPoster, error) {
	cfg.UseNoOpStorage = true
	return dataposter.NewDataPoster(ctx,
		&dataposter.DataPosterOpts{
			HeaderReader: l1Reader,
			Auth:         transactOpts,
			Config: func() *dataposter.DataPosterConfig {
				return cfg
			},
			MetadataRetriever: func(ctx context.Context, blockNum *big.Int) ([]byte, error) {
				return nil, nil
			},
			ParentChainID: parentChainID,
		},
	)
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

func getSyncMonitor(configFetcher ConfigFetcher) *SyncMonitor {
	syncConfigFetcher := func() *SyncMonitorConfig {
		return &configFetcher.Get().SyncMonitor
	}
	return NewSyncMonitor(syncConfigFetcher)
}

func getL1Reader(
	ctx context.Context,
	config *Config,
	configFetcher ConfigFetcher,
	l1client *ethclient.Client,
) (*headerreader.HeaderReader, error) {
	var l1Reader *headerreader.HeaderReader
	if config.ParentChainReader.Enable {
		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1client)
		var err error
		l1Reader, err = headerreader.New(ctx, l1client, func() *headerreader.Config { return &configFetcher.Get().ParentChainReader }, arbSys)
		if err != nil {
			return nil, err
		}
	}
	return l1Reader, nil
}

func getBroadcastServer(
	config *Config,
	configFetcher ConfigFetcher,
	dataSigner signature.DataSignerFunc,
	l2ChainId uint64,
	fatalErrChan chan error,
) (*broadcaster.Broadcaster, error) {
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
	return broadcastServer, nil
}

func getBPVerifier(
	deployInfo *chaininfo.RollupAddresses,
	l1client *ethclient.Client,
) (*contracts.AddressVerifier, error) {
	var bpVerifier *contracts.AddressVerifier
	if deployInfo != nil && l1client != nil {
		sequencerInboxAddr := deployInfo.SequencerInbox

		seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(sequencerInboxAddr, l1client)
		if err != nil {
			return nil, err
		}
		bpVerifier = contracts.NewAddressVerifier(seqInboxCaller)
	}
	return bpVerifier, nil
}

func getMaintenanceRunner(
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	coordinator *SeqCoordinator,
	exec execution.ExecutionClient,
) (*MaintenanceRunner, error) {
	dbs := []ethdb.Database{arbDb}
	maintenanceRunner, err := NewMaintenanceRunner(func() *MaintenanceConfig { return &configFetcher.Get().Maintenance }, coordinator, dbs, exec)
	if err != nil {
		return nil, err
	}
	return maintenanceRunner, nil
}

func getBroadcastClients(
	config *Config,
	configFetcher ConfigFetcher,
	txStreamer *TransactionStreamer,
	l2ChainId uint64,
	bpVerifier *contracts.AddressVerifier,
	fatalErrChan chan error,
) (*broadcastclients.BroadcastClients, error) {
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
	return broadcastClients, nil
}

func getBlockMetadataFetcher(
	ctx context.Context,
	configFetcher ConfigFetcher,
	arbDb ethdb.Database,
	exec execution.ExecutionClient,
	expectedChainId uint64,
) (*BlockMetadataFetcher, error) {
	config := configFetcher.Get()

	var blockMetadataFetcher *BlockMetadataFetcher
	if config.BlockMetadataFetcher.Enable {
		var err error
		blockMetadataFetcher, err = NewBlockMetadataFetcher(ctx, config.BlockMetadataFetcher, arbDb, exec, config.TransactionStreamer.TrackBlockMetadataFrom, expectedChainId)
		if err != nil {
			return nil, err
		}
	}
	return blockMetadataFetcher, nil
}

func getDelayedBridgeAndSequencerInbox(
	deployInfo *chaininfo.RollupAddresses,
	l1client *ethclient.Client,
) (*DelayedBridge, *SequencerInbox, error) {
	if deployInfo == nil {
		return nil, nil, errors.New("deployinfo is nil")
	}
	delayedBridge, err := NewDelayedBridge(l1client, deployInfo.Bridge, deployInfo.DeployedAt)
	if err != nil {
		return nil, nil, err
	}
	// #nosec G115
	sequencerInbox, err := NewSequencerInbox(l1client, deployInfo.SequencerInbox, int64(deployInfo.DeployedAt))
	if err != nil {
		return nil, nil, err
	}
	return delayedBridge, sequencerInbox, nil
}

func getDAS(
	ctx context.Context,
	config *Config,
	l2Config *params.ChainConfig,
	txStreamer *TransactionStreamer,
	blobReader daprovider.BlobReader,
	l1Reader *headerreader.HeaderReader,
	deployInfo *chaininfo.RollupAddresses,
	dataSigner signature.DataSignerFunc,
	l1client *ethclient.Client,
	stack *node.Node,
) (daprovider.Writer, func(), []daprovider.Reader, error) {
	if config.DAProvider.Enable && config.DataAvailability.Enable {
		return nil, nil, nil, errors.New("da-provider and data-availability cannot be enabled together")
	}

	var err error
	var daClient *daclient.Client
	var withDAWriter bool
	var dasServerCloseFn func()
	if config.DAProvider.Enable {
		daClient, err = daclient.NewClient(ctx, func() *rpcclient.ClientConfig { return &config.DAProvider.RPC })
		if err != nil {
			return nil, nil, nil, err
		}
		// Only allow dawriter if batchposter is enabled
		withDAWriter = config.DAProvider.WithWriter && config.BatchPoster.Enable
	} else if config.DataAvailability.Enable {
		jwtPath := path.Join(filepath.Dir(stack.InstanceDir()), "dasserver-jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(jwtPath); err != nil {
			return nil, nil, nil, fmt.Errorf("error writing ephemeral jwtsecret of dasserver to file: %w", err)
		}
		log.Info("Generated ephemeral JWT secret for dasserver", "jwtPath", jwtPath)
		// JWTSecret is no longer needed, cleanup when returning
		defer func() {
			if err := os.Remove(jwtPath); err != nil {
				log.Error("error deleting generated ephemeral JWT secret of dasserver", "jwtPath", jwtPath)
			}
		}()

		serverConfig := dasserver.DefaultServerConfig
		serverConfig.Port = 0 // Initializes server at a random available port
		serverConfig.DataAvailability = config.DataAvailability
		serverConfig.EnableDAWriter = config.BatchPoster.Enable
		serverConfig.JWTSecret = jwtPath
		withDAWriter = config.BatchPoster.Enable
		dasServer, closeFn, err := dasserver.NewServer(ctx, &serverConfig, dataSigner, l1client, l1Reader, deployInfo.SequencerInbox)
		if err != nil {
			return nil, nil, nil, err
		}
		clientConfig := rpcclient.DefaultClientConfig
		clientConfig.URL = dasServer.Addr
		clientConfig.JWTSecret = jwtPath
		daClient, err = daclient.NewClient(ctx, func() *rpcclient.ClientConfig { return &clientConfig })
		if err != nil {
			return nil, nil, nil, err
		}
		dasServerCloseFn = func() {
			_ = dasServer.Shutdown(ctx)
			if closeFn != nil {
				closeFn()
			}
		}
	} else if l2Config.ArbitrumChainParams.DataAvailabilityCommittee {
		return nil, nil, nil, errors.New("a data availability service is required for this chain, but it was not configured")
	}

	// We support a nil txStreamer for the pruning code
	if txStreamer != nil && txStreamer.chainConfig.ArbitrumChainParams.DataAvailabilityCommittee && daClient == nil {
		return nil, nil, nil, errors.New("data availability service required but unconfigured")
	}
	var dapReaders []daprovider.Reader
	if daClient != nil {
		dapReaders = append(dapReaders, daClient)
	}
	if blobReader != nil {
		dapReaders = append(dapReaders, daprovider.NewReaderForBlobReader(blobReader))
	}
	if withDAWriter {
		return daClient, dasServerCloseFn, dapReaders, nil
	}
	return nil, dasServerCloseFn, dapReaders, nil
}

func getInboxTrackerAndReader(
	ctx context.Context,
	arbDb ethdb.Database,
	txStreamer *TransactionStreamer,
	dapReaders []daprovider.Reader,
	config *Config,
	configFetcher ConfigFetcher,
	l1client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	deployInfo *chaininfo.RollupAddresses,
	delayedBridge *DelayedBridge,
	sequencerInbox *SequencerInbox,
	exec execution.ExecutionSequencer,
) (*InboxTracker, *InboxReader, error) {
	inboxTracker, err := NewInboxTracker(arbDb, txStreamer, dapReaders, config.SnapSyncTest)
	if err != nil {
		return nil, nil, err
	}
	firstMessageBlock := new(big.Int).SetUint64(deployInfo.DeployedAt)
	if config.SnapSyncTest.Enabled {
		if exec == nil {
			return nil, nil, errors.New("snap sync test requires an execution sequencer")
		}

		batchCount := config.SnapSyncTest.BatchCount
		delayedMessageNumber, err := exec.NextDelayedMessageNumber()
		if err != nil {
			return nil, nil, err
		}
		if batchCount > delayedMessageNumber {
			batchCount = delayedMessageNumber
		}
		// Find the first block containing the batch count.
		// Subtract 1 to get the block before the needed batch count,
		// this is done to fetch previous batch metadata needed for snap sync.
		if batchCount > 0 {
			batchCount--
		}
		block, err := FindBlockContainingBatchCount(ctx, deployInfo.Bridge, l1client, config.SnapSyncTest.ParentChainAssertionBlock, batchCount)
		if err != nil {
			return nil, nil, err
		}
		firstMessageBlock.SetUint64(block)
	}
	inboxReader, err := NewInboxReader(inboxTracker, l1client, l1Reader, firstMessageBlock, delayedBridge, sequencerInbox, func() *InboxReaderConfig { return &configFetcher.Get().InboxReader })
	if err != nil {
		return nil, nil, err
	}
	txStreamer.SetInboxReaders(inboxReader, delayedBridge)

	return inboxTracker, inboxReader, nil
}

func getBlockValidator(
	config *Config,
	configFetcher ConfigFetcher,
	statelessBlockValidator *staker.StatelessBlockValidator,
	inboxTracker *InboxTracker,
	txStreamer *TransactionStreamer,
	fatalErrChan chan error,
) (*staker.BlockValidator, error) {
	var err error
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
	return blockValidator, err
}

func getStaker(
	ctx context.Context,
	config *Config,
	configFetcher ConfigFetcher,
	arbDb ethdb.Database,
	l1Reader *headerreader.HeaderReader,
	txOptsValidator *bind.TransactOpts,
	syncMonitor *SyncMonitor,
	parentChainID *big.Int,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txStreamer *TransactionStreamer,
	inboxTracker *InboxTracker,
	stack *node.Node,
	fatalErrChan chan error,
	statelessBlockValidator *staker.StatelessBlockValidator,
	blockValidator *staker.BlockValidator,
) (*multiprotocolstaker.MultiProtocolStaker, *MessagePruner, common.Address, error) {
	var stakerObj *multiprotocolstaker.MultiProtocolStaker
	var messagePruner *MessagePruner
	var stakerAddr common.Address

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
			return nil, nil, common.Address{}, err
		}
		getExtraGas := func() uint64 { return configFetcher.Get().Staker.ExtraGas }
		// TODO: factor this out into separate helper, and split rest of node
		// creation into multiple helpers.
		var wallet legacystaker.ValidatorWalletInterface = validatorwallet.NewNoOp(l1client, deployInfo.Rollup)
		if !strings.EqualFold(config.Staker.Strategy, "watchtower") {
			if config.Staker.UseSmartContractWallet || (txOptsValidator == nil && config.Staker.DataPoster.ExternalSigner.URL == "") {
				var existingWalletAddress *common.Address
				if len(config.Staker.ContractWalletAddress) > 0 {
					if !common.IsHexAddress(config.Staker.ContractWalletAddress) {
						log.Error("invalid validator smart contract wallet", "addr", config.Staker.ContractWalletAddress)
						return nil, nil, common.Address{}, errors.New("invalid validator smart contract wallet address")
					}
					tmpAddress := common.HexToAddress(config.Staker.ContractWalletAddress)
					existingWalletAddress = &tmpAddress
				}
				// #nosec G115
				wallet, err = validatorwallet.NewContract(dp, existingWalletAddress, deployInfo.ValidatorWalletCreator, deployInfo.Rollup, l1Reader, txOptsValidator, int64(deployInfo.DeployedAt), func(common.Address) {}, getExtraGas)
				if err != nil {
					return nil, nil, common.Address{}, err
				}
			} else {
				if len(config.Staker.ContractWalletAddress) > 0 {
					return nil, nil, common.Address{}, errors.New("validator contract wallet specified but flag to use a smart contract wallet was not specified")
				}
				wallet, err = validatorwallet.NewEOA(dp, deployInfo.Rollup, l1client, getExtraGas)
				if err != nil {
					return nil, nil, common.Address{}, err
				}
			}
		}

		var confirmedNotifiers []legacystaker.LatestConfirmedNotifier
		if config.MessagePruner.Enable {
			messagePruner = NewMessagePruner(txStreamer, inboxTracker, func() *MessagePrunerConfig { return &configFetcher.Get().MessagePruner })
			confirmedNotifiers = append(confirmedNotifiers, messagePruner)
		}

		stakerObj, err = multiprotocolstaker.NewMultiProtocolStaker(stack, l1Reader, wallet, bind.CallOpts{}, func() *legacystaker.L1ValidatorConfig { return &configFetcher.Get().Staker }, &configFetcher.Get().Bold, blockValidator, statelessBlockValidator, nil, deployInfo.StakeToken, confirmedNotifiers, deployInfo.ValidatorUtils, deployInfo.Bridge, fatalErrChan)
		if err != nil {
			return nil, nil, common.Address{}, err
		}
		if err := wallet.Initialize(ctx); err != nil {
			return nil, nil, common.Address{}, err
		}
		if dp != nil {
			stakerAddr = dp.Sender()
		}
	}

	return stakerObj, messagePruner, stakerAddr, nil
}

func getTransactionStreamer(
	ctx context.Context,
	arbDb ethdb.Database,
	l2Config *params.ChainConfig,
	exec execution.ExecutionClient,
	broadcastServer *broadcaster.Broadcaster,
	configFetcher ConfigFetcher,
	fatalErrChan chan error,
) (*TransactionStreamer, error) {
	transactionStreamerConfigFetcher := func() *TransactionStreamerConfig { return &configFetcher.Get().TransactionStreamer }
	txStreamer, err := NewTransactionStreamer(ctx, arbDb, l2Config, exec, broadcastServer, fatalErrChan, transactionStreamerConfigFetcher, &configFetcher.Get().SnapSyncTest)
	if err != nil {
		return nil, err
	}
	return txStreamer, nil
}

func getSeqCoordinator(
	config *Config,
	dataSigner signature.DataSignerFunc,
	bpVerifier *contracts.AddressVerifier,
	txStreamer *TransactionStreamer,
	syncMonitor *SyncMonitor,
	exec execution.ExecutionSequencer,
) (*SeqCoordinator, error) {
	var coordinator *SeqCoordinator
	if config.SeqCoordinator.Enable {
		if exec == nil {
			return nil, errors.New("sequencer coordinator requires an execution sequencer")
		}

		var err error
		coordinator, err = NewSeqCoordinator(dataSigner, bpVerifier, txStreamer, exec, syncMonitor, config.SeqCoordinator)
		if err != nil {
			return nil, err
		}
	} else if config.Sequencer && !config.Dangerous.NoSequencerCoordinator {
		return nil, errors.New("sequencer must be enabled with coordinator, unless dangerous.no-sequencer-coordinator set")
	}
	return coordinator, nil
}

func getStatelessBlockValidator(
	config *Config,
	configFetcher ConfigFetcher,
	inboxReader *InboxReader,
	inboxTracker *InboxTracker,
	txStreamer *TransactionStreamer,
	exec execution.ExecutionRecorder,
	arbDb ethdb.Database,
	dapReaders []daprovider.Reader,
	stack *node.Node,
	wasmRootPath string,
) (*staker.StatelessBlockValidator, error) {
	var err error
	var statelessBlockValidator *staker.StatelessBlockValidator
	if config.BlockValidator.RedisValidationClientConfig.Enabled() || config.BlockValidator.ValidationServerConfigs[0].URL != "" {
		if exec == nil {
			return nil, errors.New("stateless block validator requires an execution recorder")
		}

		statelessBlockValidator, err = staker.NewStatelessBlockValidator(
			inboxReader,
			inboxTracker,
			txStreamer,
			exec,
			rawdb.NewTable(arbDb, storage.BlockValidatorPrefix),
			dapReaders,
			func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
			stack,
			wasmRootPath,
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

	return statelessBlockValidator, nil
}

func getBatchPoster(
	ctx context.Context,
	config *Config,
	configFetcher ConfigFetcher,
	txOptsBatchPoster *bind.TransactOpts,
	dapWriter daprovider.Writer,
	l1Reader *headerreader.HeaderReader,
	inboxTracker *InboxTracker,
	txStreamer *TransactionStreamer,
	exec execution.ExecutionBatchPoster,
	arbDb ethdb.Database,
	syncMonitor *SyncMonitor,
	deployInfo *chaininfo.RollupAddresses,
	parentChainID *big.Int,
	dapReaders []daprovider.Reader,
	stakerAddr common.Address,
) (*BatchPoster, error) {
	var batchPoster *BatchPoster
	if config.BatchPoster.Enable {
		if exec == nil {
			return nil, errors.New("batch poster requires an execution batch poster")
		}

		if txOptsBatchPoster == nil && config.BatchPoster.DataPoster.ExternalSigner.URL == "" {
			return nil, errors.New("batchposter, but no TxOpts")
		}
		if dapWriter != nil && !config.BatchPoster.CheckBatchCorrectness {
			return nil, errors.New("when da-provider is used by batch-poster for posting, check-batch-correctness needs to be enabled")
		}
		var err error
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
			DAPWriter:     dapWriter,
			ParentChainID: parentChainID,
			DAPReaders:    dapReaders,
		})
		if err != nil {
			return nil, err
		}

		// Check if staker and batch poster are using the same address
		if stakerAddr != (common.Address{}) && !strings.EqualFold(config.Staker.Strategy, "watchtower") && stakerAddr == batchPoster.dataPoster.Sender() {
			return nil, fmt.Errorf("staker and batch poster are using the same address which is not allowed: %v", stakerAddr)
		}
	}

	return batchPoster, nil
}

func getDelayedSequencer(
	l1Reader *headerreader.HeaderReader,
	inboxReader *InboxReader,
	exec execution.ExecutionSequencer,
	configFetcher ConfigFetcher,
	coordinator *SeqCoordinator,
) (*DelayedSequencer, error) {
	if exec == nil {
		return nil, nil
	}

	// always create DelayedSequencer if exec is non nil, it won't do anything if it is disabled
	delayedSequencer, err := NewDelayedSequencer(l1Reader, inboxReader, exec, coordinator, func() *DelayedSequencerConfig { return &configFetcher.Get().DelayedSequencer })
	if err != nil {
		return nil, err
	}
	return delayedSequencer, nil
}

func getNodeParentChainReaderDisabled(
	ctx context.Context,
	arbDb ethdb.Database,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	executionSequencer execution.ExecutionSequencer,
	executionRecorder execution.ExecutionRecorder,
	txStreamer *TransactionStreamer,
	blobReader daprovider.BlobReader,
	broadcastServer *broadcaster.Broadcaster,
	broadcastClients *broadcastclients.BroadcastClients,
	coordinator *SeqCoordinator,
	maintenanceRunner *MaintenanceRunner,
	syncMonitor *SyncMonitor,
	configFetcher ConfigFetcher,
	blockMetadataFetcher *BlockMetadataFetcher,
) *Node {
	return &Node{
		ArbDB:                   arbDb,
		Stack:                   stack,
		ExecutionClient:         executionClient,
		ExecutionSequencer:      executionSequencer,
		ExecutionRecorder:       executionRecorder,
		L1Reader:                nil,
		TxStreamer:              txStreamer,
		DeployInfo:              nil,
		BlobReader:              blobReader,
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
		SyncMonitor:             syncMonitor,
		configFetcher:           configFetcher,
		ctx:                     ctx,
		blockMetadataFetcher:    blockMetadataFetcher,
	}
}

func createNodeImpl(
	ctx context.Context,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	executionSequencer execution.ExecutionSequencer,
	executionRecorder execution.ExecutionRecorder,
	executionBatchPoster execution.ExecutionBatchPoster,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	parentChainID *big.Int,
	blobReader daprovider.BlobReader,
	wasmRootPath string,
) (*Node, error) {
	config := configFetcher.Get()

	err := checkArbDbSchemaVersion(arbDb)
	if err != nil {
		return nil, err
	}

	syncMonitor := getSyncMonitor(configFetcher)

	l1Reader, err := getL1Reader(ctx, config, configFetcher, l1client)
	if err != nil {
		return nil, err
	}

	broadcastServer, err := getBroadcastServer(config, configFetcher, dataSigner, l2Config.ChainID.Uint64(), fatalErrChan)
	if err != nil {
		return nil, err
	}

	txStreamer, err := getTransactionStreamer(ctx, arbDb, l2Config, executionClient, broadcastServer, configFetcher, fatalErrChan)
	if err != nil {
		return nil, err
	}

	bpVerifier, err := getBPVerifier(deployInfo, l1client)
	if err != nil {
		return nil, err
	}

	coordinator, err := getSeqCoordinator(config, dataSigner, bpVerifier, txStreamer, syncMonitor, executionSequencer)
	if err != nil {
		return nil, err
	}

	maintenanceRunner, err := getMaintenanceRunner(arbDb, configFetcher, coordinator, executionClient)
	if err != nil {
		return nil, err
	}

	broadcastClients, err := getBroadcastClients(config, configFetcher, txStreamer, l2Config.ChainID.Uint64(), bpVerifier, fatalErrChan)
	if err != nil {
		return nil, err
	}

	blockMetadataFetcher, err := getBlockMetadataFetcher(ctx, configFetcher, arbDb, executionClient, l2Config.ChainID.Uint64())
	if err != nil {
		return nil, err
	}

	if !config.ParentChainReader.Enable {
		return getNodeParentChainReaderDisabled(ctx, arbDb, stack, executionClient, executionSequencer, executionRecorder, txStreamer, blobReader, broadcastServer, broadcastClients, coordinator, maintenanceRunner, syncMonitor, configFetcher, blockMetadataFetcher), nil
	}

	delayedBridge, sequencerInbox, err := getDelayedBridgeAndSequencerInbox(deployInfo, l1client)
	if err != nil {
		return nil, err
	}

	dapWriter, dasServerCloseFn, dapReaders, err := getDAS(ctx, config, l2Config, txStreamer, blobReader, l1Reader, deployInfo, dataSigner, l1client, stack)
	if err != nil {
		return nil, err
	}

	inboxTracker, inboxReader, err := getInboxTrackerAndReader(ctx, arbDb, txStreamer, dapReaders, config, configFetcher, l1client, l1Reader, deployInfo, delayedBridge, sequencerInbox, executionSequencer)
	if err != nil {
		return nil, err
	}

	statelessBlockValidator, err := getStatelessBlockValidator(config, configFetcher, inboxReader, inboxTracker, txStreamer, executionRecorder, arbDb, dapReaders, stack, wasmRootPath)
	if err != nil {
		return nil, err
	}

	blockValidator, err := getBlockValidator(config, configFetcher, statelessBlockValidator, inboxTracker, txStreamer, fatalErrChan)
	if err != nil {
		return nil, err
	}

	stakerObj, messagePruner, stakerAddr, err := getStaker(ctx, config, configFetcher, arbDb, l1Reader, txOptsValidator, syncMonitor, parentChainID, l1client, deployInfo, txStreamer, inboxTracker, stack, fatalErrChan, statelessBlockValidator, blockValidator)
	if err != nil {
		return nil, err
	}

	batchPoster, err := getBatchPoster(ctx, config, configFetcher, txOptsBatchPoster, dapWriter, l1Reader, inboxTracker, txStreamer, executionBatchPoster, arbDb, syncMonitor, deployInfo, parentChainID, dapReaders, stakerAddr)
	if err != nil {
		return nil, err
	}

	delayedSequencer, err := getDelayedSequencer(l1Reader, inboxReader, executionSequencer, configFetcher, coordinator)
	if err != nil {
		return nil, err
	}

	consensusExecutionSyncerConfigFetcher := func() *ConsensusExecutionSyncerConfig {
		return &configFetcher.Get().ConsensusExecutionSyncer
	}
	consensusExecutionSyncer := NewConsensusExecutionSyncer(consensusExecutionSyncerConfigFetcher, inboxReader, executionClient, blockValidator, txStreamer)

	return &Node{
		ArbDB:                    arbDb,
		Stack:                    stack,
		ExecutionClient:          executionClient,
		ExecutionSequencer:       executionSequencer,
		ExecutionRecorder:        executionRecorder,
		L1Reader:                 l1Reader,
		TxStreamer:               txStreamer,
		DeployInfo:               deployInfo,
		BlobReader:               blobReader,
		InboxReader:              inboxReader,
		InboxTracker:             inboxTracker,
		DelayedSequencer:         delayedSequencer,
		BatchPoster:              batchPoster,
		MessagePruner:            messagePruner,
		BlockValidator:           blockValidator,
		StatelessBlockValidator:  statelessBlockValidator,
		Staker:                   stakerObj,
		BroadcastServer:          broadcastServer,
		BroadcastClients:         broadcastClients,
		SeqCoordinator:           coordinator,
		MaintenanceRunner:        maintenanceRunner,
		dasServerCloseFn:         dasServerCloseFn,
		SyncMonitor:              syncMonitor,
		blockMetadataFetcher:     blockMetadataFetcher,
		configFetcher:            configFetcher,
		ctx:                      ctx,
		ConsensusExecutionSyncer: consensusExecutionSyncer,
	}, nil
}

func FindBlockContainingBatchCount(ctx context.Context, bridgeAddress common.Address, l1Client *ethclient.Client, parentChainAssertionBlock uint64, batchCount uint64) (uint64, error) {
	bridge, err := bridgegen.NewIBridge(bridgeAddress, l1Client)
	if err != nil {
		return 0, err
	}
	high := parentChainAssertionBlock
	low := uint64(0)
	reduceBy := uint64(100)
	if high > reduceBy {
		low = high - reduceBy
	}
	// Reduce high and low by 100 until lowNode.InboxMaxCount < batchCount
	// This will give us a range (low to high) of blocks that contain the batch count.
	for low > 0 {
		lowCount, err := bridge.SequencerMessageCount(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(low)})
		if err != nil {
			return 0, err
		}
		if lowCount.Uint64() > batchCount {
			high = low
			reduceBy = reduceBy * 2
			if low > reduceBy {
				low = low - reduceBy
			} else {
				low = 0
			}
		} else {
			break
		}
	}
	// Then binary search between low and high to find the block containing the batch count.
	for low < high {
		mid := low + (high-low)/2

		midCount, err := bridge.SequencerMessageCount(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(mid)})
		if err != nil {
			return 0, err
		}
		if midCount.Uint64() < batchCount {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, nil
}

func (n *Node) OnConfigReload(_ *Config, _ *Config) error {
	// TODO
	return nil
}

func registerAPIs(currentNode *Node, stack *node.Node) {
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
			Namespace: "arbdebug",
			Version:   "1.0",
			Service: &BlockValidatorDebugAPI{
				val: currentNode.StatelessBlockValidator,
			},
			Public: false,
		})
	}
	if currentNode.MaintenanceRunner != nil {
		apis = append(apis, rpc.API{
			Namespace: "maintenance",
			Version:   "1.0",
			Service: &MaintenanceAPI{
				runner: currentNode.MaintenanceRunner,
			},
			Public: false,
		})
	}
	stack.RegisterAPIs(apis)
}

func CreateNodeExecutionClient(
	ctx context.Context,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	parentChainID *big.Int,
	blobReader daprovider.BlobReader,
	wasmRootPath string,
) (*Node, error) {
	if executionClient == nil {
		return nil, errors.New("execution client must be non-nil")
	}
	currentNode, err := createNodeImpl(ctx, stack, executionClient, nil, nil, nil, arbDb, configFetcher, l2Config, l1client, deployInfo, txOptsValidator, txOptsBatchPoster, dataSigner, fatalErrChan, parentChainID, blobReader, wasmRootPath)
	if err != nil {
		return nil, err
	}
	registerAPIs(currentNode, stack)
	return currentNode, nil
}

func CreateNodeFullExecutionClient(
	ctx context.Context,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	executionSequencer execution.ExecutionSequencer,
	executionRecorder execution.ExecutionRecorder,
	executionBatchPoster execution.ExecutionBatchPoster,
	arbDb ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	parentChainID *big.Int,
	blobReader daprovider.BlobReader,
	wasmRootPath string,
) (*Node, error) {
	if (executionClient == nil) || (executionSequencer == nil) || (executionRecorder == nil) || (executionBatchPoster == nil) {
		return nil, errors.New("execution client, sequencer, recorder, and batch poster must be non-nil")
	}
	currentNode, err := createNodeImpl(ctx, stack, executionClient, executionSequencer, executionRecorder, executionBatchPoster, arbDb, configFetcher, l2Config, l1client, deployInfo, txOptsValidator, txOptsBatchPoster, dataSigner, fatalErrChan, parentChainID, blobReader, wasmRootPath)
	if err != nil {
		return nil, err
	}
	registerAPIs(currentNode, stack)
	return currentNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	execClient, ok := n.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		execClient = nil
	}
	if execClient != nil {
		err := execClient.Initialize(ctx)
		if err != nil {
			return fmt.Errorf("error initializing exec client: %w", err)
		}
	}
	err := n.Stack.Start()
	if err != nil {
		return fmt.Errorf("error starting geth stack: %w", err)
	}
	if execClient != nil {
		execClient.SetConsensusClient(n)
	}
	err = n.ExecutionClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting exec client: %w", err)
	}
	if n.BlobReader != nil {
		err = n.BlobReader.Initialize(ctx)
		if err != nil {
			return fmt.Errorf("error initializing blob reader: %w", err)
		}
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
	} else if n.ExecutionSequencer != nil {
		n.ExecutionSequencer.Activate()
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
	if n.blockMetadataFetcher != nil {
		n.blockMetadataFetcher.Start(ctx)
	}
	if n.configFetcher != nil {
		n.configFetcher.Start(ctx)
	}
	// Also make sure to call initialize on the sync monitor after the inbox reader, tx streamer, and block validator are started.
	// Else sync might call inbox reader or tx streamer before they are started, and it will lead to panic.
	n.SyncMonitor.Initialize(n.InboxReader, n.TxStreamer, n.SeqCoordinator)
	n.SyncMonitor.Start(ctx)
	if n.ConsensusExecutionSyncer != nil {
		n.ConsensusExecutionSyncer.Start(ctx)
	}
	return nil
}

func (n *Node) StopAndWait() {
	if n.ConsensusExecutionSyncer != nil {
		n.ConsensusExecutionSyncer.StopAndWait()
	}
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
	// n.BroadcastServer is stopped after txStreamer and inboxReader because if done before it would lead to a deadlock, as the threads from these two components
	// attempt to Broadcast i.e send feedMessage to clientManager's broadcastChan when there wont be any reader to read it as n.BroadcastServer would've been stopped
	if n.BroadcastServer != nil && n.BroadcastServer.Started() {
		n.BroadcastServer.StopAndWait()
	}
	if n.SeqCoordinator != nil && n.SeqCoordinator.Started() {
		// Just stops the redis client (most other stuff was stopped earlier)
		n.SeqCoordinator.StopAndWait()
	}
	n.SyncMonitor.StopAndWait()
	if n.dasServerCloseFn != nil {
		n.dasServerCloseFn()
	}
	if n.ExecutionClient != nil {
		n.ExecutionClient.StopAndWait()
	}
	if err := n.Stack.Close(); err != nil {
		log.Error("error on stack close", "err", err)
	}
}

func (n *Node) FindInboxBatchContainingMessage(message arbutil.MessageIndex) containers.PromiseInterface[execution.InboxBatch] {
	batchNum, found, err := n.InboxTracker.FindInboxBatchContainingMessage(message)
	inboxBatch := execution.InboxBatch{
		BatchNum: batchNum,
		Found:    found,
	}
	return containers.NewReadyPromise(inboxBatch, err)
}

func (n *Node) GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise(n.InboxTracker.GetBatchParentChainBlock(seqNum))
}

func (n *Node) FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return containers.NewReadyPromise(n.SyncMonitor.FullSyncProgressMap(), nil)
}

func (n *Node) Synced() containers.PromiseInterface[bool] {
	return containers.NewReadyPromise(n.SyncMonitor.Synced(), nil)
}

func (n *Node) SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise(n.SyncMonitor.SyncTargetMessageCount(), nil)
}

func (n *Node) WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult execution.MessageResult, blockMetadata common.BlockMetadata) containers.PromiseInterface[struct{}] {
	err := n.TxStreamer.WriteMessageFromSequencer(pos, msgWithMeta, msgResult, blockMetadata)
	return containers.NewReadyPromise(struct{}{}, err)
}

func (n *Node) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	err := n.TxStreamer.ExpectChosenSequencer()
	return containers.NewReadyPromise(struct{}{}, err)
}

func (n *Node) BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata] {
	return containers.NewReadyPromise(n.TxStreamer.BlockMetadataAtMessageIndex(msgIdx))
}
