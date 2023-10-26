package gethexec

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	flag "github.com/spf13/pflag"
)

type DangerousConfig struct {
	ReorgToBlock int64 `koanf:"reorg-to-block"`
}

var DefaultDangerousConfig = DangerousConfig{
	ReorgToBlock: -1,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int64(prefix+".reorg-to-block", DefaultDangerousConfig.ReorgToBlock, "DANGEROUS! forces a reorg to an old block height. To be used for testing only. -1 to disable")
}

type Config struct {
	ParentChainReader headerreader.Config              `koanf:"parent-chain-reader" reload:"hot"`
	Sequencer         SequencerConfig                  `koanf:"sequencer" reload:"hot"`
	RecordingDatabase arbitrum.RecordingDatabaseConfig `koanf:"recording-database"`
	TxPreChecker      TxPreCheckerConfig               `koanf:"tx-pre-checker" reload:"hot"`
	Forwarder         ForwarderConfig                  `koanf:"forwarder"`
	ForwardingTarget  string                           `koanf:"forwarding-target"`
	Caching           CachingConfig                    `koanf:"caching"`
	RPC               arbitrum.Config                  `koanf:"rpc"`
	TxLookupLimit     uint64                           `koanf:"tx-lookup-limit"`
	Dangerous         DangerousConfig                  `koanf:"dangerous"`

	forwardingTarget string
}

func (c *Config) Validate() error {
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
	return nil
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions(prefix+".sequencer", f)
	headerreader.AddOptions(prefix+".parent-chain-reader", f)
	arbitrum.RecordingDatabaseConfigAddOptions(prefix+".recording-database", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTarget, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	TxPreCheckerConfigAddOptions(prefix+".tx-pre-checker", f)
	CachingConfigAddOptions(prefix+".caching", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")
	DangerousConfigAddOptions(prefix+".dangerous", f)
}

var ConfigDefault = Config{
	RPC:               arbitrum.DefaultConfig,
	Sequencer:         DefaultSequencerConfig,
	ParentChainReader: headerreader.DefaultConfig,
	RecordingDatabase: arbitrum.DefaultRecordingDatabaseConfig,
	ForwardingTarget:  "",
	TxPreChecker:      DefaultTxPreCheckerConfig,
	TxLookupLimit:     126_230_400, // 1 year at 4 blocks per second
	Caching:           DefaultCachingConfig,
	Dangerous:         DefaultDangerousConfig,
	Forwarder:         DefaultNodeForwarderConfig,
}

func ConfigDefaultNonSequencerTest() *Config {
	config := ConfigDefault
	config.ParentChainReader = headerreader.Config{OldHeaderTimeout: 5 * time.Minute}
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig
	config.ForwardingTarget = "null"

	_ = config.Validate()

	return &config
}

func ConfigDefaultTest() *Config {
	config := ConfigDefault
	config.ParentChainReader = headerreader.Config{}
	config.Sequencer = TestSequencerConfig
	config.ForwardingTarget = "null"
	config.ParentChainReader = headerreader.TestConfig

	_ = config.Validate()

	return &config
}

type ConfigFetcher func() *Config

type ExecutionNode struct {
	ChainDB           ethdb.Database
	Backend           *arbitrum.Backend
	FilterSystem      *filters.FilterSystem
	ArbInterface      *ArbInterface
	ExecEngine        *ExecutionEngine
	Recorder          *BlockRecorder
	Sequencer         *Sequencer // either nil or same as TxPublisher
	TxPublisher       TransactionPublisher
	ConfigFetcher     ConfigFetcher
	ParentChainReader *headerreader.HeaderReader
	started           atomic.Bool
}

func CreateExecutionNode(
	ctx context.Context,
	stack *node.Node,
	chainDB ethdb.Database,
	l2BlockChain *core.BlockChain,
	l1client arbutil.L1Interface,
	configFetcher ConfigFetcher,
) (*ExecutionNode, error) {
	config := configFetcher()
	execEngine, err := NewExecutionEngine(l2BlockChain)
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
			txPublisher = NewForwarder(config.forwardingTarget, &config.Forwarder)
		}
	}

	txprecheckConfigFetcher := func() *TxPreCheckerConfig { return &configFetcher().TxPreChecker }

	txPublisher = NewTxPreChecker(txPublisher, l2BlockChain, txprecheckConfigFetcher)
	arbInterface, err := NewArbInterface(execEngine, txPublisher)
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

	apis := []rpc.API{{
		Namespace: "arb",
		Version:   "1.0",
		Service:   NewArbAPI(txPublisher),
		Public:    false,
	}}
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
		ChainDB:           chainDB,
		Backend:           backend,
		FilterSystem:      filterSystem,
		ArbInterface:      arbInterface,
		ExecEngine:        execEngine,
		Recorder:          recorder,
		Sequencer:         sequencer,
		TxPublisher:       txPublisher,
		ConfigFetcher:     configFetcher,
		ParentChainReader: parentChainReader,
	}, nil

}

func (n *ExecutionNode) Initialize(ctx context.Context, arbnode interface{}, sync arbitrum.SyncProgressBackend) error {
	n.ArbInterface.Initialize(n)
	err := n.Backend.Start()
	if err != nil {
		return fmt.Errorf("error starting geth backend: %w", err)
	}
	err = n.TxPublisher.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("error initializing transaction publisher: %w", err)
	}
	err = n.Backend.APIBackend().SetSyncBackend(sync)
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
	return nil
}

func (n *ExecutionNode) StopAndWait() {
	if !n.started.Load() {
		return
	}
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

func (n *ExecutionNode) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) error {
	return n.ExecEngine.DigestMessage(num, msg)
}
func (n *ExecutionNode) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) error {
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
func (n *ExecutionNode) SetTransactionStreamer(streamer execution.TransactionStreamer) {
	n.ExecEngine.SetTransactionStreamer(streamer)
}
func (n *ExecutionNode) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64 {
	return n.ExecEngine.MessageIndexToBlockNumber(messageNum)
}

func (n *ExecutionNode) Maintenance() error {
	return n.ChainDB.Compact(nil, nil)
}
