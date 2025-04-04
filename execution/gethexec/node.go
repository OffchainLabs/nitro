package gethexec

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type StylusTargetConfig struct {
	Arm64      string   `koanf:"arm64"`
	Amd64      string   `koanf:"amd64"`
	Host       string   `koanf:"host"`
	ExtraArchs []string `koanf:"extra-archs"`

	wasmTargets []ethdb.WasmTarget
}

func (c *StylusTargetConfig) WasmTargets() []ethdb.WasmTarget {
	return c.wasmTargets
}

func (c *StylusTargetConfig) Validate() error {
	targetsSet := make(map[ethdb.WasmTarget]bool, len(c.ExtraArchs))
	for _, arch := range c.ExtraArchs {
		target := ethdb.WasmTarget(arch)
		if !rawdb.IsSupportedWasmTarget(target) {
			return fmt.Errorf("unsupported architecture: %v, possible values: %s, %s, %s, %s", arch, rawdb.TargetWavm, rawdb.TargetArm64, rawdb.TargetAmd64, rawdb.TargetHost)
		}
		targetsSet[target] = true
	}
	targetsSet[rawdb.LocalTarget()] = true
	targets := make([]ethdb.WasmTarget, 0, len(c.ExtraArchs)+1)
	for target := range targetsSet {
		targets = append(targets, target)
	}
	sort.Slice(
		targets,
		func(i, j int) bool {
			return targets[i] < targets[j]
		})
	c.wasmTargets = targets
	return nil
}

var DefaultStylusTargetConfig = StylusTargetConfig{
	Arm64:      programs.DefaultTargetDescriptionArm,
	Amd64:      programs.DefaultTargetDescriptionX86,
	Host:       "",
	ExtraArchs: []string{string(rawdb.TargetWavm)},
}

func StylusTargetConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".arm64", DefaultStylusTargetConfig.Arm64, "stylus programs compilation target for arm64 linux")
	f.String(prefix+".amd64", DefaultStylusTargetConfig.Amd64, "stylus programs compilation target for amd64 linux")
	f.String(prefix+".host", DefaultStylusTargetConfig.Host, "stylus programs compilation target for system other than 64-bit ARM or 64-bit x86")
	f.StringSlice(prefix+".extra-archs", DefaultStylusTargetConfig.ExtraArchs, fmt.Sprintf("Comma separated list of extra architectures to cross-compile stylus program to and cache in wasm store (additionally to local target). Currently must include at least %s. (supported targets: %s, %s, %s, %s)", rawdb.TargetWavm, rawdb.TargetWavm, rawdb.TargetArm64, rawdb.TargetAmd64, rawdb.TargetHost))
}

type Config struct {
	ParentChainReader           headerreader.Config `koanf:"parent-chain-reader" reload:"hot"`
	Sequencer                   SequencerConfig     `koanf:"sequencer" reload:"hot"`
	RecordingDatabase           BlockRecorderConfig `koanf:"recording-database"`
	TxPreChecker                TxPreCheckerConfig  `koanf:"tx-pre-checker" reload:"hot"`
	Forwarder                   ForwarderConfig     `koanf:"forwarder"`
	ForwardingTarget            string              `koanf:"forwarding-target"`
	SecondaryForwardingTarget   []string            `koanf:"secondary-forwarding-target"`
	Caching                     CachingConfig       `koanf:"caching"`
	RPC                         arbitrum.Config     `koanf:"rpc"`
	TxLookupLimit               uint64              `koanf:"tx-lookup-limit"`
	EnablePrefetchBlock         bool                `koanf:"enable-prefetch-block"`
	SyncMonitor                 SyncMonitorConfig   `koanf:"sync-monitor"`
	StylusTarget                StylusTargetConfig  `koanf:"stylus-target"`
	BlockMetadataApiCacheSize   uint64              `koanf:"block-metadata-api-cache-size"`
	BlockMetadataApiBlocksLimit uint64              `koanf:"block-metadata-api-blocks-limit"`

	forwardingTarget string
}

func (c *Config) Validate() error {
	if err := c.Caching.Validate(); err != nil {
		return err
	}
	if err := c.Sequencer.Validate(); err != nil {
		return err
	}
	if !c.Sequencer.Enable && c.ForwardingTarget == "" {
		return errors.New("ForwardingTarget not set and not sequencer (can use \"null\")")
	}
	if c.ForwardingTarget == "null" {
		c.forwardingTarget = ""
	} else {
		c.forwardingTarget = c.ForwardingTarget
	}
	if c.forwardingTarget != "" && c.Sequencer.Enable {
		return errors.New("ForwardingTarget set and sequencer enabled")
	}
	if err := c.StylusTarget.Validate(); err != nil {
		return err
	}
	return nil
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions(prefix+".sequencer", f)
	headerreader.AddOptions(prefix+".parent-chain-reader", f)
	BlockRecorderConfigAddOptions(prefix+".recording-database", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTarget, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	f.StringSlice(prefix+".secondary-forwarding-target", ConfigDefault.SecondaryForwardingTarget, "secondary transaction forwarding target URL")
	AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	TxPreCheckerConfigAddOptions(prefix+".tx-pre-checker", f)
	CachingConfigAddOptions(prefix+".caching", f)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")
	f.Bool(prefix+".enable-prefetch-block", ConfigDefault.EnablePrefetchBlock, "enable prefetching of blocks")
	StylusTargetConfigAddOptions(prefix+".stylus-target", f)
	f.Uint64(prefix+".block-metadata-api-cache-size", ConfigDefault.BlockMetadataApiCacheSize, "size (in bytes) of lru cache storing the blockMetadata to service arb_getRawBlockMetadata")
	f.Uint64(prefix+".block-metadata-api-blocks-limit", ConfigDefault.BlockMetadataApiBlocksLimit, "maximum number of blocks allowed to be queried for blockMetadata per arb_getRawBlockMetadata query. Enabled by default, set 0 to disable the limit")
}

var ConfigDefault = Config{
	RPC:                         arbitrum.DefaultConfig,
	Sequencer:                   DefaultSequencerConfig,
	ParentChainReader:           headerreader.DefaultConfig,
	RecordingDatabase:           DefaultBlockRecorderConfig,
	ForwardingTarget:            "",
	SecondaryForwardingTarget:   []string{},
	TxPreChecker:                DefaultTxPreCheckerConfig,
	TxLookupLimit:               126_230_400, // 1 year at 4 blocks per second
	Caching:                     DefaultCachingConfig,
	Forwarder:                   DefaultNodeForwarderConfig,
	EnablePrefetchBlock:         true,
	StylusTarget:                DefaultStylusTargetConfig,
	BlockMetadataApiCacheSize:   100 * 1024 * 1024,
	BlockMetadataApiBlocksLimit: 100,
}

type ConfigFetcher func() *Config

type ExecutionNode struct {
	ChainDB                  ethdb.Database
	Backend                  *arbitrum.Backend
	FilterSystem             *filters.FilterSystem
	ArbInterface             *ArbInterface
	ExecEngine               *ExecutionEngine
	Recorder                 *BlockRecorder
	Sequencer                *Sequencer // either nil or same as TxPublisher
	TxPreChecker             *TxPreChecker
	TxPublisher              TransactionPublisher
	ExpressLaneService       *expressLaneService
	ConfigFetcher            ConfigFetcher
	SyncMonitor              *SyncMonitor
	ParentChainReader        *headerreader.HeaderReader
	ClassicOutbox            *ClassicOutboxRetriever
	started                  atomic.Bool
	bulkBlockMetadataFetcher *BulkBlockMetadataFetcher
}

func CreateExecutionNode(
	ctx context.Context,
	stack *node.Node,
	chainDB ethdb.Database,
	l2BlockChain *core.BlockChain,
	l1client *ethclient.Client,
	configFetcher ConfigFetcher,
) (*ExecutionNode, error) {
	config := configFetcher()
	execEngine, err := NewExecutionEngine(l2BlockChain)
	if config.EnablePrefetchBlock {
		execEngine.EnablePrefetchBlock()
	}
	if config.Caching.DisableStylusCacheMetricsCollection {
		execEngine.DisableStylusCacheMetricsCollection()
	}
	if err != nil {
		return nil, err
	}
	recorder := NewBlockRecorder(&config.RecordingDatabase, execEngine, chainDB)
	var txPublisher TransactionPublisher
	var sequencer *Sequencer

	var parentChainReader *headerreader.HeaderReader
	if l1client != nil && !reflect.ValueOf(l1client).IsNil() {
		arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1client)
		parentChainReader, err = headerreader.New(ctx, l1client, func() *headerreader.Config { return &configFetcher().ParentChainReader }, arbSys)
		if err != nil {
			return nil, err
		}
	} else if config.Sequencer.Enable {
		log.Warn("sequencer enabled without l1 client")
	}

	if config.Sequencer.Enable {
		seqConfigFetcher := func() *SequencerConfig { return &configFetcher().Sequencer }
		sequencer, err = NewSequencer(execEngine, parentChainReader, seqConfigFetcher)
		if err != nil {
			return nil, err
		}
		txPublisher = sequencer
	} else {
		if config.Forwarder.RedisUrl != "" {
			txPublisher = NewRedisTxForwarder(config.forwardingTarget, &config.Forwarder)
		} else if config.forwardingTarget == "" {
			txPublisher = NewTxDropper()
		} else {
			targets := append([]string{config.forwardingTarget}, config.SecondaryForwardingTarget...)
			txPublisher = NewForwarder(targets, &config.Forwarder)
		}
	}

	txprecheckConfigFetcher := func() *TxPreCheckerConfig { return &configFetcher().TxPreChecker }

	txPreChecker := NewTxPreChecker(txPublisher, l2BlockChain, txprecheckConfigFetcher)
	txPublisher = txPreChecker
	arbInterface, err := NewArbInterface(l2BlockChain, txPublisher)
	if err != nil {
		return nil, err
	}
	filterConfig := filters.Config{
		LogCacheSize: config.RPC.FilterLogCacheSize,
		Timeout:      config.RPC.FilterTimeout,
	}
	backend, filterSystem, err := arbitrum.NewBackend(stack, &config.RPC, chainDB, arbInterface, filterConfig)
	if err != nil {
		return nil, err
	}

	syncMon := NewSyncMonitor(&config.SyncMonitor, execEngine)

	var classicOutbox *ClassicOutboxRetriever

	if l2BlockChain.Config().ArbitrumChainParams.GenesisBlockNum > 0 {
		classicMsgDb, err := stack.OpenDatabase("classic-msg", 0, 0, "classicmsg/", true)
		if dbutil.IsNotExistError(err) {
			log.Warn("Classic Msg Database not found", "err", err)
			classicOutbox = nil
		} else if err != nil {
			return nil, fmt.Errorf("Failed to open classic-msg database: %w", err)
		} else {
			if err := dbutil.UnfinishedConversionCheck(classicMsgDb); err != nil {
				return nil, fmt.Errorf("classic-msg unfinished database conversion check error: %w", err)
			}
			classicOutbox = NewClassicOutboxRetriever(classicMsgDb)
		}
	}

	bulkBlockMetadataFetcher := NewBulkBlockMetadataFetcher(l2BlockChain, execEngine, config.BlockMetadataApiCacheSize, config.BlockMetadataApiBlocksLimit)

	apis := []rpc.API{{
		Namespace: "arb",
		Version:   "1.0",
		Service:   NewArbAPI(txPublisher, bulkBlockMetadataFetcher),
		Public:    false,
	}}
	apis = append(apis, rpc.API{
		Namespace:     "auctioneer",
		Version:       "1.0",
		Service:       NewArbTimeboostAuctioneerAPI(txPublisher),
		Public:        false,
		Authenticated: false,
	})
	apis = append(apis, rpc.API{
		Namespace: "timeboost",
		Version:   "1.0",
		Service:   NewArbTimeboostAPI(txPublisher),
		Public:    false,
	})
	apis = append(apis, rpc.API{
		Namespace: "arbdebug",
		Version:   "1.0",
		Service: NewArbDebugAPI(
			l2BlockChain,
			config.RPC.ArbDebug.BlockRangeBound,
			config.RPC.ArbDebug.TimeoutQueueBound,
		),
		Public: false,
	})
	apis = append(apis, rpc.API{
		Namespace: "arbtrace",
		Version:   "1.0",
		Service: NewArbTraceForwarderAPI(
			l2BlockChain.Config(),
			config.RPC.ClassicRedirect,
			config.RPC.ClassicRedirectTimeout,
		),
		Public: false,
	})
	apis = append(apis, rpc.API{
		Namespace: "debug",
		Service:   eth.NewDebugAPI(eth.NewArbEthereum(l2BlockChain, chainDB)),
		Public:    false,
	})

	stack.RegisterAPIs(apis)

	return &ExecutionNode{
		ChainDB:                  chainDB,
		Backend:                  backend,
		FilterSystem:             filterSystem,
		ArbInterface:             arbInterface,
		ExecEngine:               execEngine,
		Recorder:                 recorder,
		Sequencer:                sequencer,
		TxPreChecker:             txPreChecker,
		TxPublisher:              txPublisher,
		ConfigFetcher:            configFetcher,
		SyncMonitor:              syncMon,
		ParentChainReader:        parentChainReader,
		ClassicOutbox:            classicOutbox,
		bulkBlockMetadataFetcher: bulkBlockMetadataFetcher,
	}, nil

}

func (n *ExecutionNode) MarkFeedStart(to arbutil.MessageIndex) {
	n.ExecEngine.MarkFeedStart(to)
}

func (n *ExecutionNode) Initialize(ctx context.Context) error {
	config := n.ConfigFetcher()
	err := n.ExecEngine.Initialize(config.Caching.StylusLRUCacheCapacity, &config.StylusTarget)
	if err != nil {
		return fmt.Errorf("error initializing execution engine: %w", err)
	}
	n.ArbInterface.Initialize(n)
	err = n.Backend.Start()
	if err != nil {
		return fmt.Errorf("error starting geth backend: %w", err)
	}
	err = n.TxPublisher.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("error initializing transaction publisher: %w", err)
	}
	err = n.Backend.APIBackend().SetSyncBackend(n.SyncMonitor)
	if err != nil {
		return fmt.Errorf("error setting sync backend: %w", err)
	}

	return nil
}

// not thread safe
func (n *ExecutionNode) Start(ctx context.Context) error {
	if n.started.Swap(true) {
		return errors.New("already started")
	}
	// TODO after separation
	// err := n.Stack.Start()
	// if err != nil {
	// 	return fmt.Errorf("error starting geth stack: %w", err)
	// }
	n.ExecEngine.Start(ctx)
	err := n.TxPublisher.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction puiblisher: %w", err)
	}
	if n.ParentChainReader != nil {
		n.ParentChainReader.Start(ctx)
	}
	n.bulkBlockMetadataFetcher.Start(ctx)
	return nil
}

func (n *ExecutionNode) StopAndWait() {
	if !n.started.Load() {
		return
	}
	n.bulkBlockMetadataFetcher.StopAndWait()
	// TODO after separation
	// n.Stack.StopRPC() // does nothing if not running
	if n.TxPublisher.Started() {
		n.TxPublisher.StopAndWait()
	}
	n.Recorder.OrderlyShutdown()
	if n.ParentChainReader != nil && n.ParentChainReader.Started() {
		n.ParentChainReader.StopAndWait()
	}
	if n.ExecEngine.Started() {
		n.ExecEngine.StopAndWait()
	}
	n.ArbInterface.BlockChain().Stop() // does nothing if not running
	if err := n.Backend.Stop(); err != nil {
		log.Error("backend stop", "err", err)
	}
	// TODO after separation
	// if err := n.Stack.Close(); err != nil {
	// 	log.Error("error on stak close", "err", err)
	// }
}

func (n *ExecutionNode) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	return n.ExecEngine.DigestMessage(num, msg, msgForPrefetch)
}
func (n *ExecutionNode) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	return n.ExecEngine.Reorg(count, newMessages, oldMessages)
}
func (n *ExecutionNode) HeadMessageNumber() (arbutil.MessageIndex, error) {
	return n.ExecEngine.HeadMessageNumber()
}
func (n *ExecutionNode) HeadMessageNumberSync(t *testing.T) (arbutil.MessageIndex, error) {
	return n.ExecEngine.HeadMessageNumberSync(t)
}
func (n *ExecutionNode) NextDelayedMessageNumber() (uint64, error) {
	return n.ExecEngine.NextDelayedMessageNumber()
}
func (n *ExecutionNode) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	return n.ExecEngine.SequenceDelayedMessage(message, delayedSeqNum)
}
func (n *ExecutionNode) ResultAtPos(pos arbutil.MessageIndex) (*execution.MessageResult, error) {
	return n.ExecEngine.ResultAtPos(pos)
}
func (n *ExecutionNode) ArbOSVersionForMessageNumber(messageNum arbutil.MessageIndex) (uint64, error) {
	return n.ExecEngine.ArbOSVersionForMessageNumber(messageNum)
}

func (n *ExecutionNode) RecordBlockCreation(
	ctx context.Context,
	pos arbutil.MessageIndex,
	msg *arbostypes.MessageWithMetadata,
) (*execution.RecordResult, error) {
	return n.Recorder.RecordBlockCreation(ctx, pos, msg)
}
func (n *ExecutionNode) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	n.Recorder.MarkValid(pos, resultHash)
}
func (n *ExecutionNode) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	return n.Recorder.PrepareForRecord(ctx, start, end)
}

func (n *ExecutionNode) Pause() {
	if n.Sequencer != nil {
		n.Sequencer.Pause()
	}
}

func (n *ExecutionNode) Activate() {
	if n.Sequencer != nil {
		n.Sequencer.Activate()
	}
}

func (n *ExecutionNode) ForwardTo(url string) error {
	if n.Sequencer != nil {
		return n.Sequencer.ForwardTo(url)
	} else {
		return errors.New("forwardTo not supported - sequencer not active")
	}
}

func (n *ExecutionNode) SetConsensusClient(consensus execution.FullConsensusClient) {
	n.ExecEngine.SetConsensus(consensus)
	n.SyncMonitor.SetConsensusInfo(consensus)
}

func (n *ExecutionNode) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64 {
	return n.ExecEngine.MessageIndexToBlockNumber(messageNum)
}
func (n *ExecutionNode) BlockNumberToMessageIndex(blockNum uint64) (arbutil.MessageIndex, error) {
	return n.ExecEngine.BlockNumberToMessageIndex(blockNum)
}

func (n *ExecutionNode) Maintenance() error {
	trieCapLimitBytes := arbmath.SaturatingUMul(uint64(n.ConfigFetcher().Caching.TrieCapLimit), 1024*1024)
	err := n.ExecEngine.Maintenance(trieCapLimitBytes)
	if err != nil {
		return err
	}
	return n.ChainDB.Compact(nil, nil)
}

func (n *ExecutionNode) Synced() bool {
	return n.SyncMonitor.Synced()
}

func (n *ExecutionNode) FullSyncProgressMap() map[string]interface{} {
	return n.SyncMonitor.FullSyncProgressMap()
}

func (n *ExecutionNode) InitializeTimeboost(ctx context.Context, chainConfig *params.ChainConfig) error {
	execNodeConfig := n.ConfigFetcher()
	if execNodeConfig.Sequencer.Timeboost.Enable {
		auctionContractAddr := common.HexToAddress(execNodeConfig.Sequencer.Timeboost.AuctionContractAddress)

		auctionContract, err := NewExpressLaneAuctionFromInternalAPI(
			n.Backend.APIBackend(),
			n.FilterSystem,
			auctionContractAddr)
		if err != nil {
			return err
		}

		roundTimingInfo, err := GetRoundTimingInfo(auctionContract)
		if err != nil {
			return err
		}

		expressLaneTracker := NewExpressLaneTracker(
			*roundTimingInfo,
			execNodeConfig.Sequencer.MaxBlockSpeed,
			n.Backend.APIBackend(),
			auctionContract,
			auctionContractAddr,
			chainConfig,
			execNodeConfig.Sequencer.Timeboost.EarlySubmissionGrace,
		)

		n.TxPreChecker.SetExpressLaneTracker(expressLaneTracker)

		if execNodeConfig.Sequencer.Enable {
			err := n.Sequencer.InitializeExpressLaneService(
				common.HexToAddress(execNodeConfig.Sequencer.Timeboost.AuctioneerAddress),
				roundTimingInfo,
				expressLaneTracker,
			)
			if err != nil {
				log.Error("failed to create express lane service", "err", err)
			}
			n.Sequencer.StartExpressLaneService(ctx)
		}

		expressLaneTracker.Start(ctx)
	}

	return nil
}
