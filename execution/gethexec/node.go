package gethexec

import (
	"context"
	"errors"
	"fmt"

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
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
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
	SyncMonitor            SyncMonitorConfig   `koanf:"sync-monitor" reload:"hot"`
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

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions(prefix+".sequencer", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTargetImpl, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	AddOptionsForNodeForwarderConfig(prefix+".forwarder", f)
	txPreCheckerDescription := "how strict to be when checking txs before forwarding them. 0 = accept anything, " +
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, " +
		"30 = full validation which may reject txs that would succeed"
	f.Uint(prefix+".tx-pre-checker-strictness", ConfigDefault.TxPreCheckerStrictness, txPreCheckerDescription)
	SyncMonitorConfigAddOptions(prefix+".sync-monitor", f)
	CachingConfigAddOptions(prefix+".caching", f)
	f.Uint64(prefix+".tx-lookup-limit", ConfigDefault.TxLookupLimit, "retain the ability to lookup transactions by hash for the past N blocks (0 = all blocks)")

	archiveMsg := fmt.Sprintf("retain past block state (deprecated, please use %v.caching.archive)", prefix)
	f.Bool(prefix+".archive", ConfigDefault.Archive, archiveMsg)
	DangerousConfigAddOptions(prefix+".dangerous", f)
}

var ConfigDefault = Config{
	L1Reader:               headerreader.DefaultConfig,
	RPC:                    arbitrum.DefaultConfig,
	Sequencer:              DefaultSequencerConfig,
	ForwardingTargetImpl:   "",
	TxPreCheckerStrictness: TxPreCheckerStrictnessNone,
	Archive:                false,
	TxLookupLimit:          126_230_400, // 1 year at 4 blocks per second
	Caching:                DefaultCachingConfig,
}

func ConfigDefaultNonSequencerTest() *Config {
	config := ConfigDefault
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig

	return &config
}

func ConfigDefaultTest() *Config {
	config := ConfigDefault
	config.Sequencer = TestSequencerConfig
	config.L1Reader = headerreader.TestConfig
	return &config
}

type ConfigFetcher func() *Config

type ExecutionNode struct {
	ChainDB       ethdb.Database
	Backend       *arbitrum.Backend
	FilterSystem  *filters.FilterSystem
	ArbInterface  *ArbInterface
	ExecEngine    *ExecutionEngine
	Recorder      *BlockRecorder
	Sequencer     *Sequencer // either nil or same as TxPublisher
	TxPublisher   TransactionPublisher
	ConfigFetcher ConfigFetcher
	SyncMonitor   *SyncMonitor
	L1Reader      *headerreader.HeaderReader
	ClassicOutbox *ClassicOutboxRetriever
}

func CreateExecutionNode(
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
	recorder := NewBlockRecorder(execEngine, chainDB)
	var txPublisher TransactionPublisher
	var sequencer *Sequencer

	var l1Reader *headerreader.HeaderReader

	fwTarget := config.ForwardingTarget()
	if config.Sequencer.Enable {
		if fwTarget != "" {
			return nil, errors.New("sequencer and forwarding target both set")
		}
		if l1client != nil {
			l1Reader = headerreader.New(l1client, func() *headerreader.Config { return &configFetcher().L1Reader })
		} else {
			log.Warn("sequencer enabled without l1 client")
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

	syncMonFetcher := func() *SyncMonitorConfig { return &configFetcher().SyncMonitor }
	syncMon := NewSyncMonitor(execEngine, syncMonFetcher)

	var classicOutbox *ClassicOutboxRetriever
	classicMsgDb, err := stack.OpenDatabase("classic-msg", 0, 0, "", true)
	if err != nil {
		if l2BlockChain.Config().ArbitrumChainParams.GenesisBlockNum > 0 {
			log.Warn("Classic Msg Database not found", "err", err)
		}
		classicOutbox = nil
	} else {
		classicOutbox = NewClassicOutboxRetriever(classicMsgDb)
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
		configFetcher,
		syncMon,
		l1Reader,
		classicOutbox,
	}, nil

}

func (n *ExecutionNode) Initialize(ctx context.Context) error {
	n.ArbInterface.Initialize(n)
	err := n.Backend.Start()
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
	if n.L1Reader != nil {
		n.L1Reader.Start(ctx)
	}
	n.SyncMonitor.Start(ctx)
	return nil
}

func (n *ExecutionNode) StopAndWait() {
	// TODO after separation
	// n.Stack.StopRPC() // does nothing if not running
	if n.TxPublisher.Started() {
		n.TxPublisher.StopAndWait()
	}
	n.Recorder.OrderlyShutdown()
	if n.L1Reader != nil && n.L1Reader.Started() {
		n.L1Reader.StopAndWait()
	}
	if n.ExecEngine.Started() {
		n.ExecEngine.StopAndWait()
	}
	n.ArbInterface.BlockChain().Stop() // does nothing if not running
	if err := n.Backend.Stop(); err != nil {
		log.Error("backend stop", "err", err)
	}
	n.SyncMonitor.StopAndWait()
	// TODO after separation
	// if err := n.Stack.Close(); err != nil {
	// 	log.Error("error on stak close", "err", err)
	// }
}

func (n *ExecutionNode) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return n.ExecEngine.DigestMessage(num, msg)
}
func (n *ExecutionNode) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}] {
	return n.ExecEngine.Reorg(count, newMessages, oldMessages)
}
func (n *ExecutionNode) HeadMessageNumber() containers.PromiseInterface[arbutil.MessageIndex] {
	return n.ExecEngine.HeadMessageNumber()
}
func (n *ExecutionNode) NextDelayedMessageNumber() containers.PromiseInterface[uint64] {
	return n.ExecEngine.NextDelayedMessageNumber()
}
func (n *ExecutionNode) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}] {
	return n.ExecEngine.SequenceDelayedMessage(message, delayedSeqNum)
}
func (n *ExecutionNode) ResultAtPos(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return n.ExecEngine.ResultAtPos(pos)
}

func (n *ExecutionNode) RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.RecordResult] {
	return n.Recorder.RecordBlockCreation(pos, msg)
}

func (n *ExecutionNode) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	n.Recorder.MarkValid(pos, resultHash)
}

func (n *ExecutionNode) PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return n.Recorder.PrepareForRecord(start, end)
}

func (n *ExecutionNode) Pause() containers.PromiseInterface[struct{}] {
	if n.Sequencer != nil {
		return n.Sequencer.Pause()
	}
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

func (n *ExecutionNode) Activate() containers.PromiseInterface[struct{}] {
	if n.Sequencer != nil {
		return n.Sequencer.Activate()
	}
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

func (n *ExecutionNode) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	if n.Sequencer != nil {
		return n.Sequencer.ForwardTo(url)
	} else {
		return containers.NewReadyPromise[struct{}](struct{}{}, errors.New("forwardTo not supported - sequencer not acrtive"))
	}
}

func (n *ExecutionNode) SetConsensusClient(consensus consensus.FullConsensusClient) {
	n.ExecEngine.SetTransactionStreamer(consensus)
	n.SyncMonitor.SetConsensusInfo(consensus)
}

func (n *ExecutionNode) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64 {
	return n.ExecEngine.MessageIndexToBlockNumber(messageNum)
}

func (n *ExecutionNode) Maintenance() containers.PromiseInterface[struct{}] {
	err := n.ChainDB.Compact(nil, nil)
	return containers.NewReadyPromise[struct{}](struct{}{}, err)
}
