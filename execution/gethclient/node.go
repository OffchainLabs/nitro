package gethclient

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
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
	L1Reader               headerreader.Config `koanf:"l1-reader" reload:"hot"`
	Sequencer              SequencerConfig     `koanf:"sequencer" reload:"hot"`
	TxPreCheckerStrictness uint                `koanf:"tx-pre-checker-strictness" reload:"hot"`
	Forwarder              ForwarderConfig     `koanf:"forwarder"`
	ForwardingTargetImpl   string              `koanf:"forwarding-target"`
	Caching                CachingConfig       `koanf:"caching"`
	RPC                    arbitrum.Config     `koanf:"rpc"`
	Archive                bool                `koanf:"archive"`
	TxLookupLimit          uint64              `koanf:"tx-lookup-limit"`
	Dangerous              DangerousConfig     `koanf:"dangerous"`
}

func (c *Config) ForwardingTarget() string {
	if c.ForwardingTargetImpl == "null" {
		return ""
	}

	return c.ForwardingTargetImpl
}

func (c *Config) Validate() error {
	if err := c.Sequencer.Validate(); err != nil {
		return err
	}
	return nil
}

func ConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions(prefix+".sequencer", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTargetImpl, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	txPreCheckerDescription := "how strict to be when checking txs before forwarding them. 0 = accept anything, " +
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, " +
		"30 = full validation which may reject txs that would succeed"
	f.Uint(prefix+".tx-pre-checker-strictness", ConfigDefault.TxPreCheckerStrictness, txPreCheckerDescription)
	CachingConfigAddOptions(prefix+".caching", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")

	archiveMsg := fmt.Sprintf("retain past block state (deprecated, please use %v.caching.archive)", prefix)
	f.Bool(prefix+".archive", ConfigDefault.Archive, archiveMsg)
	DangerousConfigAddOptions(prefix+".dangerous", f)
}

var ConfigDefault = Config{
	RPC:                    arbitrum.DefaultConfig,
	Sequencer:              DefaultSequencerConfig,
	ForwardingTargetImpl:   "",
	TxPreCheckerStrictness: TxPreCheckerStrictnessNone,
	Archive:                false,
	TxLookupLimit:          126_230_400, // 1 year at 4 blocks per second
	Caching:                DefaultCachingConfig,
}

func ConfigDefaultL1Test() *Config {
	config := ConfigDefaultL1NonSequencerTest()
	config.Sequencer = TestSequencerConfig

	return config
}

func ConfigDefaultL1NonSequencerTest() *Config {
	config := ConfigDefault
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig

	return &config
}

func ConfigDefaultL2Test() *Config {
	config := ConfigDefault
	config.Sequencer = TestSequencerConfig

	return &config
}

type ConfigFetcher func() *Config

type ExecutionNode struct {
	ChainDB      ethdb.Database
	Backend      *arbitrum.Backend
	FilterSystem *filters.FilterSystem
	ArbInterface *ArbInterface
	ExecEngine   *ExecutionEngine
	Recorder     *BlockRecorder
	Sequencer    *Sequencer // either nil or same as TxPublisher
	TxPublisher  TransactionPublisher
}

func CreateExecutionNode(
	stack *node.Node,
	chainDB ethdb.Database,
	l2BlockChain *core.BlockChain,
	l1client arbutil.L1Interface,
	syncMonitor arbitrum.SyncProgressBackend,
	configFetcher ConfigFetcher,
) (*ExecutionNode, error) {
	config := configFetcher()
	execEngine, err := NewExecutionEngine(l2BlockChain)
	if err != nil {
		return nil, err
	}
	recorder := NewBlockRecorder(execEngine, chainDB)
	var txPublisher TransactionPublisher
	var sequencer *Sequencer

	l1Reader := headerreader.New(l1client, func() *headerreader.Config { return &configFetcher().L1Reader })

	fwTarget := config.ForwardingTarget()
	if config.Sequencer.Enable {
		if fwTarget != "" {
			return nil, errors.New("sequencer and forwarding target both set")
		}
		seqConfigFetcher := func() *SequencerConfig { return &configFetcher().Sequencer }
		sequencer, err = NewSequencer(execEngine, l1Reader, seqConfigFetcher)
		if err != nil {
			return nil, err
		}
		txPublisher = sequencer
	} else {
		if config.Forwarder.RedisUrl != "" {
			txPublisher = NewRedisTxForwarder(fwTarget, &config.Forwarder)
		} else if fwTarget == "" {
			txPublisher = NewTxDropper()
		} else {
			txPublisher = NewForwarder(fwTarget, &config.Forwarder)
		}
	}

	strictnessFetcher := func() uint { return configFetcher().TxPreCheckerStrictness }
	txPublisher = NewTxPreChecker(txPublisher, l2BlockChain, strictnessFetcher)
	arbInterface, err := NewArbInterface(execEngine, txPublisher)
	if err != nil {
		return nil, err
	}
	filterConfig := filters.Config{
		LogCacheSize: config.RPC.FilterLogCacheSize,
		Timeout:      config.RPC.FilterTimeout,
	}
	backend, filterSystem, err := arbitrum.NewBackend(stack, &config.RPC, chainDB, arbInterface, syncMonitor, filterConfig)
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
		chainDB,
		backend,
		filterSystem,
		arbInterface,
		execEngine,
		recorder,
		sequencer,
		txPublisher,
	}, nil

}

func (n *ExecutionNode) Initialize(ctx context.Context, arbnode interface{}) error {
	n.ArbInterface.Initialize(n)
	err := n.Backend.Start()
	if err != nil {
		return fmt.Errorf("error starting geth backend: %w", err)
	}
	err = n.TxPublisher.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("error initializing transaction publisher: %w", err)
	}
	return nil
}

func (n *ExecutionNode) Start(ctx context.Context) error {
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
	// TODO after separation
	// if n.L1Reader != nil {
	// 	n.L1Reader.Start(ctx)
	// }
	return nil
}

func (n *ExecutionNode) StopAndWait() {
	// TODO after separation
	// n.Stack.StopRPC() // does nothing if not running
	if n.TxPublisher.Started() {
		n.TxPublisher.StopAndWait()
	}
	n.Recorder.OrderlyShutdown()
	// TODO after separation
	// if n.L1Reader != nil && n.L1Reader.Started() {
	// 	n.L1Reader.StopAndWait()
	// }
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
	n.Sequencer.Pause()
}
func (n *ExecutionNode) Activate() {
	n.Sequencer.Activate()
}
func (n *ExecutionNode) ForwardTo(url string) error {
	return n.Sequencer.ForwardTo(url)
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
