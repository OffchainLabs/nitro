// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/spf13/pflag"

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
	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	nitroversionalerter "github.com/offchainlabs/nitro/arbnode/nitro-version-alerter"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/consensus/consensusrpcserver"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/anytrust"
	daconfig "github.com/offchainlabs/nitro/daprovider/config"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/execution"
	executionrpcclient "github.com/offchainlabs/nitro/execution/rpcclient"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
	multiprotocolstaker "github.com/offchainlabs/nitro/staker/multi_protocol"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/rpcserver"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

var FailedToUseArbGetL1ConfirmationsRPCFromParentChainLogMsg = "Failed to get L1 confirmations from parent chain via arb_getL1Confirmations"

type Config struct {
	Sequencer         bool                              `koanf:"sequencer"`
	ParentChainReader headerreader.Config               `koanf:"parent-chain-reader" reload:"hot"`
	InboxReader       InboxReaderConfig                 `koanf:"inbox-reader" reload:"hot"`
	DelayedSequencer  DelayedSequencerConfig            `koanf:"delayed-sequencer" reload:"hot"`
	BatchPoster       BatchPosterConfig                 `koanf:"batch-poster" reload:"hot"`
	MessagePruner     MessagePrunerConfig               `koanf:"message-pruner" reload:"hot"`
	MessageExtraction melrunner.MessageExtractionConfig `koanf:"message-extraction" reload:"hot"`
	BlockValidator    staker.BlockValidatorConfig       `koanf:"block-validator" reload:"hot"`
	Feed              broadcastclient.FeedConfig        `koanf:"feed" reload:"hot"`
	Staker            legacystaker.L1ValidatorConfig    `koanf:"staker" reload:"hot"`
	Bold              bold.BoldConfig                   `koanf:"bold"`
	SeqCoordinator    SeqCoordinatorConfig              `koanf:"seq-coordinator"`
	// Deprecated: Use DA.AnyTrust instead. Will be removed in a future release.
	DataAvailability         anytrust.Config                  `koanf:"data-availability"`
	DA                       daconfig.DAConfig                `koanf:"da" reload:"hot"`
	SyncMonitor              SyncMonitorConfig                `koanf:"sync-monitor"`
	Dangerous                DangerousConfig                  `koanf:"dangerous"`
	TransactionStreamer      TransactionStreamerConfig        `koanf:"transaction-streamer" reload:"hot"`
	Maintenance              MaintenanceConfig                `koanf:"maintenance" reload:"hot"`
	ResourceMgmt             resourcemanager.Config           `koanf:"resource-mgmt" reload:"hot"`
	BlockMetadataFetcher     BlockMetadataFetcherConfig       `koanf:"block-metadata-fetcher" reload:"hot"`
	ConsensusExecutionSyncer ConsensusExecutionSyncerConfig   `koanf:"consensus-execution-syncer"`
	RPCServer                rpcserver.Config                 `koanf:"rpc-server"`
	ExecutionRPCClient       rpcclient.ClientConfig           `koanf:"execution-rpc-client" reload:"hot"`
	VersionAlerterServer     nitroversionalerter.ServerConfig `koanf:"version-alerter-server" reload:"hot"`
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
	if c.MessageExtraction.Enable && c.MessageExtraction.ReadMode != "latest" {
		if c.Sequencer {
			return errors.New("cannot enable message extraction in safe or finalized mode along with sequencer")
		}
		c.Feed.Output.Enable = false
		c.Feed.Input.URL = []string{}
	}
	if err := c.BlockValidator.Validate(); err != nil {
		return err
	}
	if err := c.MessageExtraction.Validate(); err != nil {
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
	if err := c.SeqCoordinator.Validate(); err != nil {
		return err
	}
	if err := c.DA.Validate(); err != nil {
		return err
	}
	if c.TransactionStreamer.TrackBlockMetadataFrom != 0 && !c.BlockMetadataFetcher.Enable {
		log.Warn("track-block-metadata-from is set but blockMetadata fetcher is not enabled")
	}
	if err := c.ExecutionRPCClient.Validate(); err != nil {
		return fmt.Errorf("error validating Client config: %w", err)
	}
	// Check that sync-interval is not more than msg-lag / 2
	if c.ConsensusExecutionSyncer.SyncInterval > c.SyncMonitor.MsgLag/2 {
		log.Warn("consensus-execution-syncer.sync-interval is more than half of sync-monitor.msg-lag, which may cause sync issues",
			"sync-interval", c.ConsensusExecutionSyncer.SyncInterval,
			"msg-lag", c.SyncMonitor.MsgLag)
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

// MigrateDeprecatedConfig migrates deprecated DataAvailability config to DA.AnyTrust.
// This allows operators to continue using --node.data-availability.* flags while
// transitioning to the new --node.da.anytrust.* flags.
func (c *Config) MigrateDeprecatedConfig() {
	if c.DataAvailability.Enable {
		log.Error("DEPRECATED: --node.data-availability.* flags are deprecated and will " +
			"overwrite any --node.da.anytrust.* settings; please migrate to --node.da.anytrust.*")
		c.DA.AnyTrust = c.DataAvailability
	}
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	f.Bool(prefix+".sequencer", ConfigDefault.Sequencer, "enable sequencer")
	headerreader.AddOptions(prefix+".parent-chain-reader", f)
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	BatchPosterConfigAddOptions(prefix+".batch-poster", f)
	MessagePrunerConfigAddOptions(prefix+".message-pruner", f)
	melrunner.MessageExtractionConfigAddOptions(prefix+".message-extraction", f)
	staker.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, feedInputEnable, feedOutputEnable)
	legacystaker.L1ValidatorConfigAddOptions(prefix+".staker", f)
	bold.BoldConfigAddOptions(prefix+".bold", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	anytrust.ConfigAddNodeOptions(prefix+".data-availability", f)
	daconfig.DAConfigAddOptions(prefix+".da", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	DangerousConfigAddOptions(prefix+".dangerous", f)
	TransactionStreamerConfigAddOptions(prefix+".transaction-streamer", f)
	MaintenanceConfigAddOptions(prefix+".maintenance", f)
	resourcemanager.ConfigAddOptions(prefix+".resource-mgmt", f)
	BlockMetadataFetcherConfigAddOptions(prefix+".block-metadata-fetcher", f)
	ConsensusExecutionSyncerConfigAddOptions(prefix+".consensus-execution-syncer", f)
	rpcserver.ConfigAddOptions(prefix+".rpc-server", "consensus", f)
	rpcclient.RPCClientAddOptions(prefix+".execution-rpc-client", f, &ConfigDefault.ExecutionRPCClient)
	nitroversionalerter.ServerConfigAddOptions(prefix+".version-alerter-server", f)
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
	MessageExtraction:        melrunner.DefaultMessageExtractionConfig,
	Bold:                     bold.DefaultBoldConfig,
	SeqCoordinator:           DefaultSeqCoordinatorConfig,
	DataAvailability:         anytrust.DefaultConfigForNode,
	DA:                       daconfig.DefaultDAConfig,
	SyncMonitor:              DefaultSyncMonitorConfig,
	Dangerous:                DefaultDangerousConfig,
	TransactionStreamer:      DefaultTransactionStreamerConfig,
	ResourceMgmt:             resourcemanager.DefaultConfig,
	BlockMetadataFetcher:     DefaultBlockMetadataFetcherConfig,
	Maintenance:              DefaultMaintenanceConfig,
	ConsensusExecutionSyncer: DefaultConsensusExecutionSyncerConfig,
	VersionAlerterServer:     nitroversionalerter.DefaultServerConfig,
	RPCServer:                rpcserver.DefaultConfig,
	ExecutionRPCClient: rpcclient.ClientConfig{
		URL:                       "",
		JWTSecret:                 "",
		Retries:                   3,
		RetryErrors:               "websocket: close.*|dial tcp .*|.*i/o timeout|.*connection reset by peer|.*connection refused",
		ArgLogLimit:               2048,
		WebsocketMessageSizeLimit: 256 * 1024 * 1024,
	},
}

func ConfigDefaultL1Test() *Config {
	config := ConfigDefaultL1NonSequencerTest()
	config.DelayedSequencer = TestDelayedSequencerConfig
	config.BatchPoster = TestBatchPosterConfig
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Sequencer = true
	config.Dangerous.NoSequencerCoordinator = true
	config.DA.ExternalProvider.DataStream = data_streaming.TestDataStreamerConfig(daclient.DefaultStreamRpcMethods)

	return config
}

func ConfigDefaultL1NonSequencerTest() *Config {
	config := ConfigDefault
	config.MessageExtraction = melrunner.TestMessageExtractionConfig
	config.Dangerous = TestDangerousConfig
	config.ParentChainReader = headerreader.TestConfig
	config.InboxReader = TestInboxReaderConfig
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.SeqCoordinator.Enable = false
	config.BlockValidator = staker.TestBlockValidatorConfig
	config.SyncMonitor = TestSyncMonitorConfig
	config.ConsensusExecutionSyncer = TestConsensusExecutionSyncerConfig
	config.Staker = legacystaker.TestL1ValidatorConfig
	config.Staker.Enable = false
	config.BlockValidator.ValidationServerConfigs = []rpcclient.ClientConfig{{URL: ""}}
	config.Bold.MinimumGapToParentAssertion = 0

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.MessageExtraction = melrunner.TestMessageExtractionConfig
	config.Dangerous = TestDangerousConfig
	config.ParentChainReader.Enable = false
	config.SeqCoordinator = TestSeqCoordinatorConfig
	config.Feed.Input.Verify.Dangerous.AcceptMissing = true
	config.Feed.Output.Signed = false
	config.SeqCoordinator.Signer.ECDSA.AcceptSequencer = false
	config.SeqCoordinator.Signer.ECDSA.Dangerous.AcceptMissing = true
	config.Staker = legacystaker.TestL1ValidatorConfig
	config.SyncMonitor = TestSyncMonitorConfig
	config.ConsensusExecutionSyncer = TestConsensusExecutionSyncerConfig
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

func DangerousConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".no-l1-listener", DefaultDangerousConfig.NoL1Listener, "DANGEROUS! disables listening to L1. To be used in test nodes only")
	f.Bool(prefix+".no-sequencer-coordinator", DefaultDangerousConfig.NoSequencerCoordinator, "DANGEROUS! allows sequencing without sequencer-coordinator")
	f.Bool(prefix+".disable-blob-reader", DefaultDangerousConfig.DisableBlobReader, "DANGEROUS! disables the EIP-4844 blob reader, which is necessary to read batches")
}

type Node struct {
	ConsensusDB              ethdb.Database
	Stack                    *node.Node
	ExecutionClient          execution.ExecutionClient
	ExecutionSequencer       execution.ExecutionSequencer
	ExecutionRecorder        execution.ExecutionRecorder
	L1Reader                 *headerreader.HeaderReader
	ParentChain              *parent.ParentChain
	TxStreamer               *TransactionStreamer
	DeployInfo               *chaininfo.RollupAddresses
	BlobReader               daprovider.BlobReader
	MessageExtractor         *melrunner.MessageExtractor
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
	providerServerCloseFn    func()
	AnyTrustLifecycleManager *anytrust.LifecycleManager
	SyncMonitor              *SyncMonitor
	blockMetadataFetcher     *BlockMetadataFetcher
	configFetcher            ConfigFetcher
	ctx                      context.Context
	ConsensusExecutionSyncer *ConsensusExecutionSyncer
	sequencerInbox           *SequencerInbox
}

var ErrNoBatchDataReader = errors.New("node has no batch data reader")

// BatchDataReader extends BatchMetadataFetcher with read-only access to message
// counts and batch/message lookups, abstracting over MessageExtractor (MEL) vs
// InboxTracker. Both satisfy this interface.
type BatchDataReader interface {
	BatchMetadataFetcher
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetDelayedCount() (uint64, error)
	GetBatchParentChainBlock(seqNum uint64) (uint64, error)
	FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error)
}

// Compile-time interface satisfaction checks.
var (
	_ BatchDataReader      = (*InboxTracker)(nil)
	_ BatchDataReader      = (*melrunner.MessageExtractor)(nil)
	_ BatchDataProvider    = (*melrunner.MessageExtractor)(nil)
	_ BatchMetadataFetcher = (*melrunner.MessageExtractor)(nil)
)

// BatchDataSource returns the node's active BatchDataReader, preferring
// MessageExtractor over InboxTracker.
func (n *Node) BatchDataSource() (BatchDataReader, error) {
	if n.MessageExtractor != nil {
		return n.MessageExtractor, nil
	}
	if n.InboxTracker != nil {
		return n.InboxTracker, nil
	}
	return nil, ErrNoBatchDataReader
}

type ConfigFetcher interface {
	Get() *Config
	Start(context.Context)
	StopAndWait()
	Started() bool
}

func checkConsensusDBSchemaVersion(consensusDB ethdb.Database) error {
	var version uint64
	hasVersion, err := consensusDB.Has(schema.DbSchemaVersion)
	if err != nil {
		return err
	}
	if hasVersion {
		versionBytes, err := consensusDB.Get(schema.DbSchemaVersion)
		if err != nil {
			return err
		}
		version = binary.BigEndian.Uint64(versionBytes)
	}
	for version != schema.CurrentDbSchemaVersion {
		batch := consensusDB.NewBatch()
		switch version {
		case 0:
			// No database updates are necessary for database format version 0->1.
			// This version adds a new format for delayed messages in the inbox tracker,
			// but it can still read the old format for old messages.
		case 1:
			// No database updates are necessary for database format version 1->0.
			// This version adds a new optional field to L1IncomingMessages,
			// but it can still read the old format for old messages.
		default:
			return fmt.Errorf("unsupported database format version %v", version)
		}

		// Increment version and flush the batch
		version++
		versionBytes := make([]uint8, 8)
		binary.BigEndian.PutUint64(versionBytes, version)
		err = batch.Put(schema.DbSchemaVersion, versionBytes)
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
			ParentChain: parent.NewParentChain(ctx, parentChainID, l1Reader),
		},
	)
}

func StakerDataposter(
	ctx context.Context, db ethdb.Database, l1Reader *headerreader.HeaderReader,
	transactOpts *bind.TransactOpts, cfgFetcher ConfigFetcher, syncMonitor *SyncMonitor,
	parentChain *parent.ParentChain,
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
			ParentChain:       parentChain,
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
	configFetcher ConfigFetcher,
	coordinator *SeqCoordinator,
	exec execution.ExecutionClient,
) (*MaintenanceRunner, error) {
	return NewMaintenanceRunner(func() *MaintenanceConfig { return &configFetcher.Get().Maintenance }, coordinator, exec)
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
	l2Config *params.ChainConfig,
	consensusDB ethdb.Database,
	exec execution.ExecutionClient,
	expectedChainId uint64,
) (*BlockMetadataFetcher, error) {
	config := configFetcher.Get()

	var blockMetadataFetcher *BlockMetadataFetcher
	if config.BlockMetadataFetcher.Enable {
		var err error
		blockMetadataFetcher, err = NewBlockMetadataFetcher(ctx, config.BlockMetadataFetcher, consensusDB, l2Config.ArbitrumChainParams.GenesisBlockNum, exec, config.TransactionStreamer.TrackBlockMetadataFrom, expectedChainId)
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

func getDAProviders(
	ctx context.Context,
	config *Config,
	txStreamer *TransactionStreamer,
	blobReader daprovider.BlobReader,
	l1Reader *headerreader.HeaderReader,
	deployInfo *chaininfo.RollupAddresses,
	dataSigner signature.DataSignerFunc,
	l1client *ethclient.Client,
) ([]daprovider.Writer, func(), *daprovider.DAProviderRegistry, error) {
	var writers []daprovider.Writer
	var cleanupFuncs []func()
	var dapRegistry = daprovider.NewDAProviderRegistry()

	// Priority order for writers:
	// 1. External DA (if enabled)
	// 2. AnyTrust (if enabled)

	// Create external DA client if enabled
	if config.DA.ExternalProvider.Enable {
		providerConfig := &config.DA.ExternalProvider

		log.Info("Creating external DA client", "url", providerConfig.RPC.URL, "withWriter", providerConfig.WithWriter)
		externalDAClient, err := daclient.NewClient(ctx, providerConfig, data_streaming.PayloadCommiter())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create external DA client: %w", err)
		}

		// Add to writers array if batch poster is enabled and WithWriter is true
		if providerConfig.WithWriter && config.BatchPoster.Enable {
			writers = append(writers, externalDAClient)
			log.Info("Added external DA writer")
		}

		// Register external DA client as both reader and validator
		result, err := externalDAClient.GetSupportedHeaderBytes().Await(ctx)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get supported header bytes from external DA client: %w", err)
		}
		for _, hb := range result.HeaderBytes {
			if err := dapRegistry.Register(hb, externalDAClient, externalDAClient); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to register DA provider: %w", err)
			}
		}
	}

	// Create AnyTrust DA provider if enabled (can coexist with external DA)
	if config.DA.AnyTrust.Enable {
		// Map deprecated BatchPoster.MaxSize to DA.AnyTrust.MaxBatchSize for backward compatibility
		if config.BatchPoster.MaxSize != 0 && config.DA.AnyTrust.MaxBatchSize == anytrust.DefaultConfig.MaxBatchSize {
			log.Warn("Using deprecated batch-poster.max-size for AnyTrust max batch size; please migrate to da.anytrust.max-batch-size")
			config.DA.AnyTrust.MaxBatchSize = config.BatchPoster.MaxSize
		}

		log.Info("Creating AnyTrust DA provider", "batchPosterEnabled", config.BatchPoster.Enable)

		// Create AnyTrust factory
		daFactory := anytrust.NewFactory(
			&config.DA.AnyTrust,
			dataSigner,
			l1client,
			l1Reader,
			deployInfo.SequencerInbox,
			config.BatchPoster.Enable,
		)
		log.Info("Created AnyTrust DA factory")

		if err := daFactory.ValidateConfig(); err != nil {
			return nil, nil, nil, err
		}

		var localCleanupFuncs []func()
		reader, readerCleanup, err := daFactory.CreateReader(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		if readerCleanup != nil {
			localCleanupFuncs = append(localCleanupFuncs, readerCleanup)
		}

		var writer daprovider.Writer
		if config.BatchPoster.Enable {
			var writerCleanup func()
			writer, writerCleanup, err = daFactory.CreateWriter(ctx)
			if err != nil {
				return nil, nil, nil, err
			}
			if writerCleanup != nil {
				localCleanupFuncs = append(localCleanupFuncs, writerCleanup)
			}
			if writer != nil {
				writers = append(writers, writer)
				log.Info("Added AnyTrust writer", "writerIndex", len(writers)-1, "totalWriters", len(writers))
			}
		}

		headerBytes := daFactory.GetSupportedHeaderBytes()
		// Register AnyTrust reader directly (no validator for AnyTrust)
		for _, hb := range headerBytes {
			if err := dapRegistry.Register(hb, reader, nil); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to register anytrust reader: %w", err)
			}
		}

		// Create cleanup function for AnyTrust
		anytrustCleanup := func() {
			for _, cleanup := range localCleanupFuncs {
				cleanup()
			}
		}
		cleanupFuncs = append(cleanupFuncs, anytrustCleanup)
	}

	// Check if chain requires AnyTrust but none is configured
	// We support a nil txStreamer for the pruning code
	if txStreamer != nil && txStreamer.chainConfig.ArbitrumChainParams.DataAvailabilityCommittee {
		if !config.DA.AnyTrust.Enable {
			return nil, nil, nil, errors.New("AnyTrust DA service required but unconfigured")
		}
	}

	if blobReader != nil {
		if err := dapRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(blobReader)); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to register blob reader: %w", err)
		}
	}

	// Register a fallback DACert reader to treat DACert batches as empty
	// if no real provider was configured, matching replay behavior.
	if dapRegistry.GetReader(daprovider.DACertificateMessageHeaderFlag) == nil {
		if err := dapRegistry.SetupDACertificateReader(&daprovider.FallbackDACertReader{}, nil); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to register fallback DACert reader: %w", err)
		}
	}

	// Combine all cleanup functions
	combinedCleanup := func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}

	log.Info("DA providers configured", "totalWriters", len(writers))
	return writers, combinedCleanup, dapRegistry, nil
}

func getInboxTrackerAndReader(
	config *Config,
	consensusDB ethdb.Database,
	txStreamer *TransactionStreamer,
	dapReaders *daprovider.DAProviderRegistry,
	configFetcher ConfigFetcher,
	l1client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	deployInfo *chaininfo.RollupAddresses,
	delayedBridge *DelayedBridge,
	sequencerInbox *SequencerInbox,
) (*InboxTracker, *InboxReader, error) {
	if config.MessageExtraction.Enable {
		return nil, nil, nil
	}
	inboxTracker, err := NewInboxTracker(consensusDB, txStreamer, dapReaders)
	if err != nil {
		return nil, nil, err
	}
	firstMessageBlock := new(big.Int).SetUint64(deployInfo.DeployedAt)
	inboxReader, err := NewInboxReader(inboxTracker, l1client, l1Reader, firstMessageBlock, delayedBridge, sequencerInbox, func() *InboxReaderConfig { return &configFetcher.Get().InboxReader })
	if err != nil {
		return nil, nil, err
	}
	return inboxTracker, inboxReader, nil
}

// computeMigrationStartBlock determines the parent chain block number to anchor
// the initial MEL state during legacy migration. Uses the last batch's parent
// chain block, capped at the finalized block number, to ensure the initial state
// cannot be reorged out.
// For Arbitrum parent chains (no native finality), uses the last batch's block directly.
func computeMigrationStartBlock(
	ctx context.Context,
	l1client *ethclient.Client,
	consensusDB ethdb.Database,
	deployInfo *chaininfo.RollupAddresses,
	parentChainIsArbitrum bool,
) (uint64, error) {
	totalBatchCount, err := read.Value[uint64](consensusDB, schema.SequencerBatchCountKey)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			totalBatchCount = 0
		} else {
			return 0, fmt.Errorf("failed to read legacy batch count: %w", err)
		}
	}
	if totalBatchCount == 0 {
		if deployInfo.DeployedAt == 0 {
			return 0, errors.New("cannot compute migration start block: DeployedAt is 0 and no batches exist")
		}
		return deployInfo.DeployedAt - 1, nil
	}
	lastBatchMeta, err := read.BatchMetadata(consensusDB, totalBatchCount-1)
	if err != nil {
		return 0, fmt.Errorf("failed to read last legacy batch metadata: %w", err)
	}
	startBlockNum := lastBatchMeta.ParentChainBlock
	if !parentChainIsArbitrum {
		finalizedHeader, err := l1client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
		if err != nil {
			return 0, fmt.Errorf("failed to get finalized block: %w", err)
		}
		if finalizedHeader == nil {
			return 0, errors.New("finalized block header not available on parent chain")
		}
		startBlockNum = min(startBlockNum, finalizedHeader.Number.Uint64())
	}
	return startBlockNum, nil
}

// migrateLegacyDBToMEL creates the initial MEL state from pre-MEL inbox reader/tracker
// data and persists it. Called once during the first MEL startup on a legacy node.
func migrateLegacyDBToMEL(
	ctx context.Context,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	consensusDB ethdb.Database,
	melDB *melrunner.Database,
	parentChainIsArbitrum bool,
) error {
	log.Info("Migrating legacy inbox reader/tracker data to MEL")
	chainId, err := l1client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	startBlockNum, err := computeMigrationStartBlock(ctx, l1client, consensusDB, deployInfo, parentChainIsArbitrum)
	if err != nil {
		return fmt.Errorf("failed to compute migration start block: %w", err)
	}
	delayedBridge, err := NewDelayedBridge(l1client, deployInfo.Bridge, deployInfo.DeployedAt)
	if err != nil {
		return fmt.Errorf("failed to create delayed bridge: %w", err)
	}
	delayedSeenAtBlock, err := delayedBridge.GetMessageCount(ctx, new(big.Int).SetUint64(startBlockNum))
	if err != nil {
		return fmt.Errorf("failed to get on-chain delayed message count at block %d: %w", startBlockNum, err)
	}
	initialState, err := melrunner.CreateInitialMELStateFromLegacyDB(
		consensusDB,
		deployInfo.SequencerInbox,
		deployInfo.Bridge,
		chainId.Uint64(),
		func(blockNum uint64) (common.Hash, common.Hash, error) {
			header, err := l1client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNum))
			if err != nil {
				return common.Hash{}, common.Hash{}, err
			}
			if header == nil {
				return common.Hash{}, common.Hash{}, fmt.Errorf("block %d not found on parent chain", blockNum)
			}
			return header.Hash(), header.ParentHash, nil
		},
		startBlockNum,
		delayedSeenAtBlock,
	)
	if err != nil {
		return fmt.Errorf("failed to create initial MEL state from legacy DB: %w", err)
	}
	if err = melDB.SaveInitialMelState(initialState); err != nil {
		return fmt.Errorf("failed to save initial mel state: %w", err)
	}
	log.Info("MEL migration from legacy data complete",
		"delayedSeen", initialState.DelayedMessagesSeen,
		"delayedRead", initialState.DelayedMessagesRead,
		"batchCount", initialState.BatchCount,
		"msgCount", initialState.MsgCount,
		"parentChainBlock", initialState.ParentChainBlockNumber,
	)
	return nil
}

func validateAndInitializeDBForMEL(
	ctx context.Context,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	consensusDB ethdb.Database,
	parentChainIsArbitrum bool,
) (*melrunner.Database, error) {
	melDB, err := melrunner.NewDatabase(consensusDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create MEL database: %w", err)
	}
	_, err = melDB.GetHeadMelState()
	if err == nil {
		return melDB, nil
	}
	if !rawdb.IsDbErrNotFound(err) {
		return nil, err
	}
	// No existing MEL state. Check if this is a legacy node (has inbox reader/tracker keys).
	hasSequencerBatchCountKey, err := consensusDB.Has(schema.SequencerBatchCountKey)
	if err != nil {
		return nil, err
	}
	hasDelayedMessageCountKey, err := consensusDB.Has(schema.DelayedMessageCountKey)
	if err != nil {
		return nil, err
	}
	if hasSequencerBatchCountKey || hasDelayedMessageCountKey {
		if err := migrateLegacyDBToMEL(ctx, l1client, deployInfo, consensusDB, melDB, parentChainIsArbitrum); err != nil {
			return nil, err
		}
		return melDB, nil
	}
	// Fresh node: no legacy keys exist.
	msgCount, err := read.Value[uint64](consensusDB, schema.MessageCountKey)
	if err != nil {
		return nil, err
	}
	if msgCount != 0 {
		return nil, errors.New("MEL being initialized when DB already has stale msgs")
	}
	initialState, err := createInitialMELState(ctx, deployInfo, l1client)
	if err != nil {
		return nil, err
	}
	if err = melDB.SaveState(initialState); err != nil {
		return nil, fmt.Errorf("failed to save initial mel state: %w", err)
	}
	return melDB, nil
}

func getMessageExtractor(
	ctx context.Context,
	config *Config,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	consensusDB ethdb.Database,
	dapRegistry *daprovider.DAProviderRegistry,
	sequencerInbox *SequencerInbox,
	l1Reader *headerreader.HeaderReader,
	fatalErrChan chan error,
) (*melrunner.MessageExtractor, error) {
	if !config.MessageExtraction.Enable {
		// Prevent database corruption. If HeadMelStateBlockNumKey exists,
		// it indicates this node was previously run with Message Extraction (MEL) enabled.
		// Switching back to the standard inbox reader/tracker is not allowed.
		hasHeadMelStateBlockNumKey, err := consensusDB.Has(schema.HeadMelStateBlockNumKey)
		if err != nil {
			return nil, err
		}
		if hasHeadMelStateBlockNumKey {
			return nil, errors.New("node already has MEL related database entries and is trying to start inbox reader and tracker, not allowed")
		}
		return nil, nil
	}
	melDB, err := validateAndInitializeDBForMEL(ctx, l1client, deployInfo, consensusDB, l1Reader.IsParentChainArbitrum())
	if err != nil {
		return nil, err
	}
	return melrunner.NewMessageExtractor(
		config.MessageExtraction,
		l1client,
		l2Config,
		deployInfo,
		melDB,
		dapRegistry,
		sequencerInbox,
		l1Reader,
		nil,
		fatalErrChan,
	)
}

func createInitialMELState(
	ctx context.Context,
	deployInfo *chaininfo.RollupAddresses,
	client *ethclient.Client,
) (*mel.State, error) {
	if deployInfo.DeployedAt == 0 {
		return nil, errors.New("DeployedAt is 0; cannot create initial MEL state before the genesis block")
	}
	// Create an initial MEL state anchored at the block before the rollup deployment block.
	startHeader, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(deployInfo.DeployedAt-1))
	if err != nil {
		return nil, err
	}
	if startHeader == nil {
		return nil, fmt.Errorf("block %d not found on parent chain", deployInfo.DeployedAt-1)
	}
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	return &mel.State{
		BatchPostingTargetAddress:          deployInfo.SequencerInbox,
		DelayedMessagePostingTargetAddress: deployInfo.Bridge,
		ParentChainId:                      chainId.Uint64(),
		ParentChainBlockNumber:             startHeader.Number.Uint64(),
		ParentChainBlockHash:               startHeader.Hash(),
		ParentChainPreviousBlockHash:       startHeader.ParentHash,
	}, nil
}

func getBlockValidator(
	config *Config,
	configFetcher ConfigFetcher,
	statelessBlockValidator *staker.StatelessBlockValidator,
	inboxTracker staker.InboxTrackerInterface,
	txStreamer *TransactionStreamer,
	fatalErrChan chan error,
) (*staker.BlockValidator, error) {
	if !config.ValidatorRequired() {
		return nil, nil
	}
	return staker.NewBlockValidator(
		statelessBlockValidator,
		inboxTracker,
		txStreamer,
		func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
		fatalErrChan,
	)
}

func getStaker(
	ctx context.Context,
	config *Config,
	configFetcher ConfigFetcher,
	consensusDB ethdb.Database,
	l1Reader *headerreader.HeaderReader,
	txOptsValidator *bind.TransactOpts,
	syncMonitor *SyncMonitor,
	parentChain *parent.ParentChain,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txStreamer *TransactionStreamer,
	inboxReader staker.InboxReaderInterface,
	inboxTracker staker.InboxTrackerInterface,
	batchMetaFetcher BatchMetadataFetcher,
	stack *node.Node,
	fatalErrChan chan error,
	statelessBlockValidator *staker.StatelessBlockValidator,
	blockValidator *staker.BlockValidator,
	dapRegistry *daprovider.DAProviderRegistry,
) (*multiprotocolstaker.MultiProtocolStaker, *MessagePruner, common.Address, error) {
	var stakerObj *multiprotocolstaker.MultiProtocolStaker
	var messagePruner *MessagePruner
	var stakerAddr common.Address

	if config.Staker.Enable {
		dp, err := StakerDataposter(
			ctx,
			rawdb.NewTable(consensusDB, storage.StakerPrefix),
			l1Reader,
			txOptsValidator,
			configFetcher,
			syncMonitor,
			parentChain,
		)
		if err != nil {
			return nil, nil, common.Address{}, err
		}
		getExtraGas := func() uint64 { return configFetcher.Get().Staker.ExtraGas }
		// TODO: factor this out into separate helper, and split rest of node
		// creation into multiple helpers.
		var wallet legacystaker.ValidatorWalletInterface = validatorwallet.NewNoOp(l1client)
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
				wallet, err = validatorwallet.NewContract(dp, existingWalletAddress, deployInfo.ValidatorWalletCreator, l1Reader, txOptsValidator, int64(deployInfo.DeployedAt), func(common.Address) {}, getExtraGas)
				if err != nil {
					return nil, nil, common.Address{}, err
				}
			} else {
				if len(config.Staker.ContractWalletAddress) > 0 {
					return nil, nil, common.Address{}, errors.New("validator contract wallet specified but flag to use a smart contract wallet was not specified")
				}
				wallet, err = validatorwallet.NewEOA(dp, l1client, getExtraGas)
				if err != nil {
					return nil, nil, common.Address{}, err
				}
			}
		}

		var confirmedNotifiers []legacystaker.LatestConfirmedNotifier
		if config.MessagePruner.Enable {
			messagePruner = NewMessagePruner(consensusDB, txStreamer, batchMetaFetcher, func() *MessagePrunerConfig { return &configFetcher.Get().MessagePruner })
			confirmedNotifiers = append(confirmedNotifiers, messagePruner)
		}

		stakerObj, err = multiprotocolstaker.NewMultiProtocolStaker(stack, l1Reader, wallet, bind.CallOpts{}, func() *legacystaker.L1ValidatorConfig { return &configFetcher.Get().Staker }, &configFetcher.Get().Bold, blockValidator, statelessBlockValidator, nil, deployInfo.StakeToken, deployInfo.Rollup, confirmedNotifiers, deployInfo.ValidatorUtils, deployInfo.Bridge, txStreamer, inboxTracker, inboxReader, dapRegistry, fatalErrChan)
		if err != nil {
			return nil, nil, common.Address{}, err
		}
		if config.Staker.UseSmartContractWallet {
			if !l1Reader.Started() {
				l1Reader.Start(ctx)
			}
			err = wallet.InitializeAndCreateSCW(ctx)
		} else {
			err = wallet.Initialize(ctx)
		}
		if err != nil {
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
	consensusDB ethdb.Database,
	l2Config *params.ChainConfig,
	exec execution.ExecutionClient,
	broadcastServer *broadcaster.Broadcaster,
	configFetcher ConfigFetcher,
	fatalErrChan chan error,
) (*TransactionStreamer, error) {
	transactionStreamerConfigFetcher := func() *TransactionStreamerConfig { return &configFetcher.Get().TransactionStreamer }
	return NewTransactionStreamer(ctx, consensusDB, l2Config, exec, broadcastServer, fatalErrChan, transactionStreamerConfigFetcher)
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
	inboxReader staker.InboxReaderInterface,
	inboxTracker staker.InboxTrackerInterface,
	txStreamer *TransactionStreamer,
	exec execution.ExecutionRecorder,
	consensusDB ethdb.Database,
	dapReaders *daprovider.DAProviderRegistry,
	stack *node.Node,
	latestWasmModuleRoot common.Hash,
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
			rawdb.NewTable(consensusDB, storage.BlockValidatorPrefix),
			dapReaders,
			func() *staker.BlockValidatorConfig { return &configFetcher.Get().BlockValidator },
			stack,
			latestWasmModuleRoot,
		)
	} else {
		err = errors.New("no validator url specified")
	}
	if err != nil {
		if config.ValidatorRequired() {
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
	l2Config *params.ChainConfig,
	txOptsBatchPoster *bind.TransactOpts,
	dapWriters []daprovider.Writer,
	l1Reader *headerreader.HeaderReader,
	batchMetaFetcher BatchMetadataFetcher,
	txStreamer *TransactionStreamer,
	arbOSVersionGetter execution.ArbOSVersionGetter,
	consensusDB ethdb.Database,
	syncMonitor *SyncMonitor,
	deployInfo *chaininfo.RollupAddresses,
	parentChain *parent.ParentChain,
	dapReaders *daprovider.DAProviderRegistry,
	stakerAddr common.Address,
) (*BatchPoster, error) {
	var batchPoster *BatchPoster
	if config.BatchPoster.Enable {
		if batchMetaFetcher == nil {
			return nil, errors.New("batch poster requires either an inbox tracker or a message extractor")
		}
		if arbOSVersionGetter == nil {
			return nil, errors.New("batch poster requires ArbOS version getter")
		}

		if txOptsBatchPoster == nil && config.BatchPoster.DataPoster.ExternalSigner.URL == "" {
			return nil, errors.New("batchposter, but no TxOpts")
		}
		if len(dapWriters) > 0 && !config.BatchPoster.CheckBatchCorrectness {
			return nil, errors.New("when da-provider is used by batch-poster for posting, check-batch-correctness needs to be enabled")
		}
		var err error
		batchPoster, err = NewBatchPoster(ctx, &BatchPosterOpts{
			DataPosterDB:         rawdb.NewTable(consensusDB, storage.BatchPosterPrefix),
			L1Reader:             l1Reader,
			BatchMetadataFetcher: batchMetaFetcher,
			Streamer:             txStreamer,
			VersionGetter:        arbOSVersionGetter,
			SyncMonitor:          syncMonitor,
			Config:               func() *BatchPosterConfig { return &configFetcher.Get().BatchPoster },
			DeployInfo:           deployInfo,
			TransactOpts:         txOptsBatchPoster,
			DAPWriters:           dapWriters,
			ParentChain:          parentChain,
			DAPReaders:           dapReaders,
			ChainConfig:          l2Config,
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
	delayedMessageFetcher DelayedMessageFetcher,
	delayedBridge *DelayedBridge,
	exec execution.ExecutionSequencer,
	configFetcher ConfigFetcher,
	coordinator *SeqCoordinator,
) (*DelayedSequencer, error) {
	if exec == nil {
		// No ExecutionSequencer means delayed messages cannot be sequenced.
		return nil, nil
	}
	if delayedMessageFetcher == nil {
		return nil, errors.New("delayed sequencer requires either an inbox tracker or a message extractor")
	}
	// always create DelayedSequencer if exec is non nil, it won't do anything if it is disabled
	return NewDelayedSequencer(l1Reader, delayedMessageFetcher, delayedBridge, exec, coordinator, func() *DelayedSequencerConfig { return &configFetcher.Get().DelayedSequencer })
}

func getNodeParentChainReaderDisabled(
	ctx context.Context,
	consensusDB ethdb.Database,
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
	// Create ConsensusExecutionSyncer even in L2-only mode to push sync data
	consensusExecutionSyncerConfigFetcher := func() *ConsensusExecutionSyncerConfig {
		return &configFetcher.Get().ConsensusExecutionSyncer
	}
	consensusExecutionSyncer := NewConsensusExecutionSyncer(
		consensusExecutionSyncerConfigFetcher,
		nil, // inboxReader
		executionClient,
		nil, // blockValidator
		txStreamer,
		syncMonitor,
	)

	return &Node{
		ConsensusDB:              consensusDB,
		Stack:                    stack,
		ExecutionClient:          executionClient,
		ExecutionSequencer:       executionSequencer,
		ExecutionRecorder:        executionRecorder,
		L1Reader:                 nil,
		TxStreamer:               txStreamer,
		DeployInfo:               nil,
		BlobReader:               blobReader,
		InboxReader:              nil,
		InboxTracker:             nil,
		DelayedSequencer:         nil,
		BatchPoster:              nil,
		MessagePruner:            nil,
		BlockValidator:           nil,
		StatelessBlockValidator:  nil,
		Staker:                   nil,
		BroadcastServer:          broadcastServer,
		BroadcastClients:         broadcastClients,
		SeqCoordinator:           coordinator,
		MaintenanceRunner:        maintenanceRunner,
		SyncMonitor:              syncMonitor,
		configFetcher:            configFetcher,
		ctx:                      ctx,
		blockMetadataFetcher:     blockMetadataFetcher,
		ConsensusExecutionSyncer: consensusExecutionSyncer,
	}
}

func createNodeImpl(
	ctx context.Context,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	executionSequencer execution.ExecutionSequencer,
	executionRecorder execution.ExecutionRecorder,
	arbOSVersionGetter execution.ArbOSVersionGetter,
	consensusDB ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	blobReader daprovider.BlobReader,
	latestWasmModuleRoot common.Hash,
	parentChain *parent.ParentChain,
) (*Node, error) {
	config := configFetcher.Get()

	err := checkConsensusDBSchemaVersion(consensusDB)
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

	txStreamer, err := getTransactionStreamer(ctx, consensusDB, l2Config, executionClient, broadcastServer, configFetcher, fatalErrChan)
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

	maintenanceRunner, err := getMaintenanceRunner(configFetcher, coordinator, executionClient)
	if err != nil {
		return nil, err
	}

	broadcastClients, err := getBroadcastClients(config, configFetcher, txStreamer, l2Config.ChainID.Uint64(), bpVerifier, fatalErrChan)
	if err != nil {
		return nil, err
	}

	blockMetadataFetcher, err := getBlockMetadataFetcher(ctx, configFetcher, l2Config, consensusDB, executionClient, l2Config.ChainID.Uint64())
	if err != nil {
		return nil, err
	}

	if !config.ParentChainReader.Enable {
		return getNodeParentChainReaderDisabled(ctx, consensusDB, stack, executionClient, executionSequencer, executionRecorder, txStreamer, blobReader, broadcastServer, broadcastClients, coordinator, maintenanceRunner, syncMonitor, configFetcher, blockMetadataFetcher), nil
	}

	delayedBridge, sequencerInbox, err := getDelayedBridgeAndSequencerInbox(deployInfo, l1client)
	if err != nil {
		return nil, err
	}

	dapWriters, providerServerCloseFn, dapRegistry, err := getDAProviders(ctx, config, txStreamer, blobReader, l1Reader, deployInfo, dataSigner, l1client)
	if err != nil {
		return nil, err
	}

	inboxTracker, inboxReader, err := getInboxTrackerAndReader(config, consensusDB, txStreamer, dapRegistry, configFetcher, l1client, l1Reader, deployInfo, delayedBridge, sequencerInbox)
	if err != nil {
		return nil, err
	}

	messageExtractor, err := getMessageExtractor(ctx, config, l2Config, l1client, deployInfo, consensusDB, dapRegistry, sequencerInbox, l1Reader, fatalErrChan)
	if err != nil {
		return nil, err
	}
	if messageExtractor != nil {
		if err := messageExtractor.SetMessageConsumer(txStreamer); err != nil {
			return nil, err
		}
	}

	var batchDataProvider BatchDataProvider
	if inboxReader != nil && inboxTracker != nil {
		batchDataProvider = inboxReader.GetParentChainDataSource()
	} else if messageExtractor != nil {
		batchDataProvider = messageExtractor
	}
	if batchDataProvider != nil {
		if err := txStreamer.SetBatchDataProvider(batchDataProvider, delayedBridge); err != nil {
			return nil, err
		}
	}

	// TODO: rename staker.InboxReaderInterface and staker.InboxTrackerInterface to a better name
	var validatorInboxReader staker.InboxReaderInterface
	var validatorInboxTracker staker.InboxTrackerInterface
	if messageExtractor != nil {
		validatorInboxReader = messageExtractor
		validatorInboxTracker = messageExtractor
	} else {
		validatorInboxReader = inboxReader
		validatorInboxTracker = inboxTracker
	}

	statelessBlockValidator, err := getStatelessBlockValidator(config, configFetcher, validatorInboxReader, validatorInboxTracker, txStreamer, executionRecorder, consensusDB, dapRegistry, stack, latestWasmModuleRoot)
	if err != nil {
		return nil, err
	}

	blockValidator, err := getBlockValidator(config, configFetcher, statelessBlockValidator, validatorInboxTracker, txStreamer, fatalErrChan)
	if err != nil {
		return nil, err
	}

	var batchMetaFetcher BatchMetadataFetcher
	if inboxTracker != nil {
		batchMetaFetcher = inboxTracker
	} else if messageExtractor != nil {
		batchMetaFetcher = messageExtractor
	}

	stakerObj, messagePruner, stakerAddr, err := getStaker(ctx, config, configFetcher, consensusDB, l1Reader, txOptsValidator, syncMonitor, parentChain, l1client, deployInfo, txStreamer, validatorInboxReader, validatorInboxTracker, batchMetaFetcher, stack, fatalErrChan, statelessBlockValidator, blockValidator, dapRegistry)
	if err != nil {
		return nil, err
	}

	batchPoster, err := getBatchPoster(ctx, config, configFetcher, l2Config, txOptsBatchPoster, dapWriters, l1Reader, batchMetaFetcher, txStreamer, arbOSVersionGetter, consensusDB, syncMonitor, deployInfo, parentChain, dapRegistry, stakerAddr)
	if err != nil {
		return nil, err
	}

	// Convert typed nil *MessageExtractor to untyped nil so the interface parameter
	// in NewDelayedSequencer is properly nil (Go nil-interface semantics).
	var delayedMessageFetcher DelayedMessageFetcher
	if inboxTracker != nil {
		delayedMessageFetcher = inboxTracker
	} else if messageExtractor != nil {
		delayedMessageFetcher = messageExtractor
	}
	delayedSequencer, err := getDelayedSequencer(l1Reader, delayedMessageFetcher, delayedBridge, executionSequencer, configFetcher, coordinator)
	if err != nil {
		return nil, err
	}

	consensusExecutionSyncerConfigFetcher := func() *ConsensusExecutionSyncerConfig {
		return &configFetcher.Get().ConsensusExecutionSyncer
	}
	var msgCountFetcher MessageCountFetcher
	if messageExtractor != nil {
		msgCountFetcher = messageExtractor
	} else {
		msgCountFetcher = inboxReader
	}
	consensusExecutionSyncer := NewConsensusExecutionSyncer(consensusExecutionSyncerConfigFetcher, msgCountFetcher, executionClient, blockValidator, txStreamer, syncMonitor)

	if messagePruner != nil && messageExtractor != nil {
		messagePruner.SetLegacyDelayedBound(messageExtractor.LegacyDelayedBound())
	}

	return &Node{
		ConsensusDB:              consensusDB,
		Stack:                    stack,
		ExecutionClient:          executionClient,
		ExecutionSequencer:       executionSequencer,
		ExecutionRecorder:        executionRecorder,
		L1Reader:                 l1Reader,
		ParentChain:              parentChain,
		TxStreamer:               txStreamer,
		DeployInfo:               deployInfo,
		BlobReader:               blobReader,
		InboxReader:              inboxReader,
		InboxTracker:             inboxTracker,
		MessageExtractor:         messageExtractor,
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
		providerServerCloseFn:    providerServerCloseFn,
		SyncMonitor:              syncMonitor,
		blockMetadataFetcher:     blockMetadataFetcher,
		configFetcher:            configFetcher,
		ctx:                      ctx,
		ConsensusExecutionSyncer: consensusExecutionSyncer,
		sequencerInbox:           sequencerInbox,
	}, nil
}

func (n *Node) OnConfigReload(_ *Config, _ *Config) error {
	// TODO: Implement hot reload for MEL config fields marked with reload:"hot"
	// (RetryInterval, BlocksToPrefetch, StallTolerance). Also propagate reloads
	// to MessagePruner and other subsystems that support hot config changes.
	return nil
}

func registerAPIs(currentNode *Node, stack *node.Node, genesisBlockNum uint64) {
	var apis []rpc.API
	apis = append(apis, rpc.API{
		Namespace: "arb",
		Version:   "1.0",
		Service:   NewArbAPI(currentNode, genesisBlockNum),
		Public:    true,
	})

	if currentNode.BlockValidator != nil {
		apis = append(apis, rpc.API{
			Namespace: "arb",
			Version:   "1.0",
			Service:   NewBlockValidatorAPI(currentNode.BlockValidator),
			Public:    false,
		})
	}
	if currentNode.StatelessBlockValidator != nil {
		apis = append(apis, rpc.API{
			Namespace: "arbdebug",
			Version:   "1.0",
			Service:   NewBlockValidatorDebugAPI(currentNode.StatelessBlockValidator),
			Public:    false,
		})
	}
	config := currentNode.configFetcher.Get()
	if config.RPCServer.Enable {
		apis = append(apis, rpc.API{
			Namespace:     consensus.RPCNamespace,
			Version:       "1.0",
			Service:       consensusrpcserver.NewConsensusRPCServer(currentNode),
			Public:        config.RPCServer.Public,
			Authenticated: config.RPCServer.Authenticated,
		})
	}
	versionAlerterServerCfg := func() *nitroversionalerter.ServerConfig { return &currentNode.configFetcher.Get().VersionAlerterServer }
	if versionAlerterServerCfg().Enable {
		apis = append(apis, rpc.API{
			Namespace: "arb",
			Version:   "1.0",
			Service:   nitroversionalerter.NewServer(versionAlerterServerCfg),
			Public:    true,
		})
	}
	stack.RegisterAPIs(apis)
}

func CreateConsensusNodeConnectedWithSimpleExecutionClient(
	ctx context.Context,
	stack *node.Node,
	executionClient execution.ExecutionClient,
	consensusDB ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	blobReader daprovider.BlobReader,
	latestWasmModuleRoot common.Hash,
	parentChain *parent.ParentChain,
) (*Node, error) {
	if configFetcher.Get().ExecutionRPCClient.URL != "" {
		execConfigFetcher := func() *rpcclient.ClientConfig { return &configFetcher.Get().ExecutionRPCClient }
		executionClient = executionrpcclient.NewClient(execConfigFetcher, stack)
	}
	if executionClient == nil {
		return nil, errors.New("execution client must be non-nil")
	}
	currentNode, err := createNodeImpl(ctx, stack, executionClient, nil, nil, executionClient, consensusDB, configFetcher, l2Config, l1client, deployInfo, txOptsValidator, txOptsBatchPoster, dataSigner, fatalErrChan, blobReader, latestWasmModuleRoot, parentChain)
	if err != nil {
		return nil, err
	}
	registerAPIs(currentNode, stack, l2Config.ArbitrumChainParams.GenesisBlockNum)
	return currentNode, nil
}

func CreateConsensusNode(
	ctx context.Context,
	stack *node.Node,
	fullExecutionClient execution.FullExecutionClient,
	consensusDB ethdb.Database,
	configFetcher ConfigFetcher,
	l2Config *params.ChainConfig,
	l1client *ethclient.Client,
	deployInfo *chaininfo.RollupAddresses,
	txOptsValidator *bind.TransactOpts,
	txOptsBatchPoster *bind.TransactOpts,
	dataSigner signature.DataSignerFunc,
	fatalErrChan chan error,
	blobReader daprovider.BlobReader,
	latestWasmModuleRoot common.Hash,
	parentChain *parent.ParentChain,
) (*Node, error) {
	var executionClient execution.ExecutionClient
	var executionRecorder execution.ExecutionRecorder
	var executionSequencer execution.ExecutionSequencer
	var arbOSVersionGetter execution.ArbOSVersionGetter
	if configFetcher.Get().ExecutionRPCClient.URL != "" {
		execConfigFetcher := func() *rpcclient.ClientConfig { return &configFetcher.Get().ExecutionRPCClient }
		rpcClient := executionrpcclient.NewClient(execConfigFetcher, stack)
		executionClient = rpcClient
		executionRecorder = rpcClient
		arbOSVersionGetter = rpcClient
		// executionSequencer intentionally left nil - RPC client does not implement ExecutionSequencer
	} else {
		executionClient = fullExecutionClient
		executionRecorder = fullExecutionClient
		executionSequencer = fullExecutionClient
		arbOSVersionGetter = fullExecutionClient
	}
	currentNode, err := createNodeImpl(ctx, stack, executionClient, executionSequencer, executionRecorder, arbOSVersionGetter, consensusDB, configFetcher, l2Config, l1client, deployInfo, txOptsValidator, txOptsBatchPoster, dataSigner, fatalErrChan, blobReader, latestWasmModuleRoot, parentChain)
	if err != nil {
		return nil, err
	}
	registerAPIs(currentNode, stack, l2Config.ArbitrumChainParams.GenesisBlockNum)
	return currentNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	var err error
	if execRPCClient, ok := n.ExecutionClient.(*executionrpcclient.Client); ok {
		if err = execRPCClient.Start(ctx); err != nil {
			return fmt.Errorf("error starting exec rpc client: %w", err)
		}
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
	// Even if the sequencer coordinator will populate this backlog,
	// we want to make sure it's populated before any clients connect.
	if err = n.TxStreamer.PopulateFeedBacklog(ctx); err != nil {
		return fmt.Errorf("error populating feed backlog on startup: %w", err)
	}
	if n.MessageExtractor != nil {
		err = n.MessageExtractor.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting message extractor: %w", err)
		}
	}
	// must init broadcast server before trying to sequence anything
	if n.BroadcastServer != nil {
		// PopulateFeedBacklog is a synchronous operation, hence we first
		// call it to populate the backlog and then start the broadcastServer
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
	if n.ParentChain != nil {
		n.ParentChain.Start(ctx)
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
	if n.L1Reader != nil && !n.L1Reader.Started() {
		n.L1Reader.Start(ctx)
	}
	if n.BroadcastClients != nil {
		go func() {
			var caughtUpChan <-chan struct{}
			if n.MessageExtractor != nil {
				caughtUpChan = n.MessageExtractor.CaughtUp()
			} else if n.InboxReader != nil {
				caughtUpChan = n.InboxReader.CaughtUp()
			}
			if caughtUpChan != nil {
				select {
				case <-caughtUpChan:
				case <-ctx.Done():
					return
				}
			}
			n.BroadcastClients.Start(ctx)
		}()
	}
	if n.blockMetadataFetcher != nil {
		err = n.blockMetadataFetcher.Start(ctx)
		if err != nil {
			return fmt.Errorf("error starting block metadata fetcher: %w", err)
		}
	}
	if n.configFetcher != nil {
		n.configFetcher.Start(ctx)
	}
	// Also make sure to call initialize on the sync monitor after the inbox reader (or message extractor),
	// tx streamer, and block validator are started. Else sync might call them before they are started.
	var syncFetcher MessageSyncProgressFetcher
	if n.MessageExtractor != nil {
		syncFetcher = n.MessageExtractor
	} else if n.InboxReader != nil {
		syncFetcher = n.InboxReader
	}
	n.SyncMonitor.Initialize(syncFetcher, n.TxStreamer, n.SeqCoordinator)
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
	if n.blockMetadataFetcher != nil {
		n.blockMetadataFetcher.StopAndWait()
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
	if n.ParentChain != nil && n.ParentChain.Started() {
		n.ParentChain.StopAndWait()
	}
	if n.InboxReader != nil && n.InboxReader.Started() {
		n.InboxReader.StopAndWait()
	}
	if n.MessageExtractor != nil && n.MessageExtractor.Started() {
		n.MessageExtractor.StopAndWait()
	}
	if n.L1Reader != nil && n.L1Reader.Started() {
		n.L1Reader.StopAndWait()
	}
	if n.TxStreamer.Started() {
		n.TxStreamer.StopAndWait()
	}
	// n.BroadcastServer is stopped after txStreamer and inboxReader because if done before it would lead to a deadlock, as the threads from these two components
	// attempt to Broadcast i.e send feedMessage to clientManager's broadcastChan when there won't be any reader to read it as n.BroadcastServer would've been stopped
	if n.BroadcastServer != nil && n.BroadcastServer.Started() {
		n.BroadcastServer.StopAndWait()
	}
	if n.SeqCoordinator != nil && n.SeqCoordinator.Started() {
		// Just stops the redis client (most other stuff was stopped earlier)
		n.SeqCoordinator.StopAndWait()
	}
	n.SyncMonitor.StopAndWait()
	if n.providerServerCloseFn != nil {
		n.providerServerCloseFn()
	}
	if n.ExecutionClient != nil {
		if _, ok := n.ExecutionClient.(*executionrpcclient.Client); ok {
			n.ExecutionClient.StopAndWait()
		}
	}
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

func (n *Node) GetParentChainDataSource() (ParentChainDataSource, error) {
	if n.MessageExtractor != nil {
		return n.MessageExtractor, nil
	}
	if n.InboxReader != nil {
		return n.InboxReader.GetParentChainDataSource(), nil
	}
	return nil, errors.New("no parent chain data source available: neither MessageExtractor nor InboxReader is set")
}

func (n *Node) GetL1Confirmations(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	if n.L1Reader == nil {
		return containers.NewReadyPromise(uint64(0), nil)
	}

	reader, err := n.BatchDataSource()
	if err != nil {
		return containers.NewReadyPromise(uint64(0), err)
	}
	batchNum, found, err := reader.FindInboxBatchContainingMessage(msgIdx)
	if err != nil {
		return containers.NewReadyPromise(uint64(0), err)
	}
	// batches not yet posted have 0 confirmations but no error
	if !found {
		return containers.NewReadyPromise(uint64(0), nil)
	}
	parentChainBlockNum, err := reader.GetBatchParentChainBlock(batchNum)
	if err != nil {
		return containers.NewReadyPromise(uint64(0), err)
	}

	if n.L1Reader.IsParentChainArbitrum() {
		return stopwaiter.LaunchPromiseThread(n.L1Reader, func(ctx context.Context) (uint64, error) {
			parentChainClient := n.L1Reader.Client()
			parentChainBlock, err := parentChainClient.BlockByNumber(ctx, new(big.Int).SetUint64(parentChainBlockNum))
			if err != nil {
				// Hide the parent chain RPC error from the client in case it contains sensitive information.
				// Likely though, this error is just "not found" because the block got reorg'd.
				return 0, fmt.Errorf("failed to get parent chain block %v containing batch", parentChainBlockNum)
			}

			var confs uint64
			err = parentChainClient.Client().CallContext(ctx, &confs, "arb_getL1Confirmations", parentChainBlock.Number())
			if err != nil {
				// falls back to node interface method
				log.Debug(FailedToUseArbGetL1ConfirmationsRPCFromParentChainLogMsg, "blockNumber", parentChainBlockNum, "blockHash", parentChainBlock.Hash(), "err", err)

				parentNodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, parentChainClient)
				if err != nil {
					return 0, err
				}
				confs, err = parentNodeInterface.GetL1Confirmations(&bind.CallOpts{Context: ctx}, parentChainBlock.Hash())
				if err != nil {
					log.Warn(
						"Failed to get L1 confirmations from parent chain",
						"blockNumber", parentChainBlockNum,
						"blockHash", parentChainBlock.Hash(), "err", err,
					)
					return 0, fmt.Errorf("failed to get L1 confirmations from parent chain for block %v", parentChainBlock.Hash())
				}
			}
			return confs, nil
		})
	}
	latestHeader, err := n.L1Reader.LastHeaderWithError()
	if err != nil {
		return containers.NewReadyPromise(uint64(0), err)
	}
	if latestHeader == nil {
		return containers.NewReadyPromise(uint64(0), errors.New("no headers read from l1"))
	}
	latestBlockNum := latestHeader.Number.Uint64()
	if latestBlockNum < parentChainBlockNum {
		return containers.NewReadyPromise(uint64(0), nil)
	}
	return containers.NewReadyPromise(latestBlockNum-parentChainBlockNum, nil)
}

func (n *Node) FindBatchContainingMessage(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	reader, err := n.BatchDataSource()
	if err != nil {
		return containers.NewReadyPromise(uint64(0), err)
	}
	batchNum, found, err := reader.FindInboxBatchContainingMessage(msgIdx)
	if err == nil && !found {
		return containers.NewReadyPromise(uint64(0), errors.New("block not yet found on any batch"))
	}
	return containers.NewReadyPromise(batchNum, err)
}
