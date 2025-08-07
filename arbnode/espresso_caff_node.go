package arbnode

import (
	"context"
	"errors"
	"fmt"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network/sdks/go/client"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/espressostreamer"
	"github.com/offchainlabs/nitro/espressotee"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type EspressoCaffNodeConfig struct {
	Enable                  bool                    `koanf:"enable"`
	HotShotUrls             []string                `koanf:"hotshot-urls"`
	NextHotshotBlock        uint64                  `koanf:"next-hotshot-block"`
	FromBlock               uint64                  `koanf:"from-block"`
	Namespace               uint64                  `koanf:"namespace"`
	RetryTime               time.Duration           `koanf:"retry-time"`
	HotshotPollingInterval  time.Duration           `koanf:"hotshot-polling-interval"`
	HotshotPollingTimeout   time.Duration           `koanf:"hotshot-polling-timeout"`
	EspressoSGXVerifierAddr string                  `koanf:"espresso-sgx-verifier-addr"`
	BatchPosterAddr         string                  `koanf:"batch-poster-addr"`
	RecordPerformance       bool                    `koanf:"record-performance"`
	WaitForFinalization     bool                    `koanf:"wait-for-finalization"`
	WaitForConfirmations    bool                    `koanf:"wait-for-confirmations"`
	RequiredBlockDepth      uint64                  `koanf:"required-block-depth"`
	BlocksToRead            uint64                  `koanf:"blocks-to-read"`
	Dangerous               DangerousCaffNodeConfig `koanf:"dangerous"`

	// Force Inclusion Checker
	ForceInclusionChecker ForceInclusionCheckerConfig `koanf:"force-inclusion-checker"`
	StateChecker          StateCheckerConfig          `koanf:"state-checker"`
}

type DangerousCaffNodeConfig struct {
	IgnoreDatabaseHotshotBlock bool `koanf:"ignore-database-hotshot-block"`
	IgnoreDatabaseFromBlock    bool `koanf:"ignore-database-from-block"`
}

var DefaultDangerousCaffNodeConfig = DangerousCaffNodeConfig{
	IgnoreDatabaseHotshotBlock: false,
}

var DefaultEspressoCaffNodeConfig = EspressoCaffNodeConfig{
	Enable:                  false,
	HotShotUrls:             []string{},
	NextHotshotBlock:        1,
	Namespace:               0,
	RetryTime:               time.Second * 2,
	HotshotPollingInterval:  time.Millisecond * 100,
	HotshotPollingTimeout:   time.Minute * 2,
	EspressoSGXVerifierAddr: "",
	BatchPosterAddr:         "",
	RecordPerformance:       false,
	// Setting these values to the default
	// values set by Arbitrum
	WaitForFinalization:  false,
	WaitForConfirmations: true,
	RequiredBlockDepth:   20,
	BlocksToRead:         10000,
	Dangerous:            DefaultDangerousCaffNodeConfig,
	FromBlock:            1,
}

func EspressoCaffNodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultEspressoCaffNodeConfig.Enable, "enable espresso Caff node")
	f.StringSlice(prefix+".hotshot-urls", DefaultEspressoCaffNodeConfig.HotShotUrls, "Hotshot urls")
	f.Uint64(prefix+".next-hotshot-block", DefaultEspressoCaffNodeConfig.NextHotshotBlock, "the Hotshot block number from which the Caff node will read")
	f.Uint64(prefix+".namespace", DefaultEspressoCaffNodeConfig.Namespace, "the namespace of the chain in Espresso Network, usually the chain id")
	f.Duration(prefix+".retry-time", DefaultEspressoCaffNodeConfig.RetryTime, "retry time after a failure")
	f.Duration(prefix+".hotshot-polling-interval", DefaultEspressoCaffNodeConfig.HotshotPollingInterval, "time after a success")
	f.Duration(prefix+".hotshot-polling-timeout", DefaultEspressoCaffNodeConfig.HotshotPollingTimeout, "timeout for hotshot polling")
	f.String(prefix+".espresso-sgx-verifier-addr", DefaultEspressoCaffNodeConfig.EspressoSGXVerifierAddr, "espresso legacy SGX verifier address that is used to verify the signature of the Hotshot transactions")
	f.String(prefix+".batch-poster-addr", DefaultEspressoCaffNodeConfig.BatchPosterAddr, "batch poster address that is used to verify the signature of the Hotshot transactions")
	f.Bool(prefix+".record-performance", DefaultEspressoCaffNodeConfig.RecordPerformance, "record performance of the Caff node")
	f.Bool(prefix+".wait-for-finalization", DefaultEspressoCaffNodeConfig.WaitForFinalization, "Configures the Caff node to only produce blocks from delayed messages if they are finalized on the parent chain")
	f.Bool(prefix+".wait-for-confirmations", DefaultEspressoCaffNodeConfig.WaitForConfirmations, "Configures the Caff node to only produce blocks from delayed messages if they have atleast requiredBlockDepth confirmations on the parent chain")
	f.Uint64(prefix+".required-block-depth", DefaultEspressoCaffNodeConfig.RequiredBlockDepth, "Configures the required block depth/number of confirmations on the parent chain that a delayed message is required to have before this Caff node will add it to it's state")
	f.Uint64(prefix+".blocks-to-read", DefaultEspressoCaffNodeConfig.BlocksToRead, "Configures the number of blocks to read from the parent chain for delayed messages")
	f.Uint64(prefix+".from-block", DefaultEspressoCaffNodeConfig.FromBlock, "Configures the block number to start reading delayed messages from")
	DangerousCaffNodeConfigAddOptions(prefix+".dangerous", f)

	EspressoForceInclusionConfigAddOptions(prefix+".force-inclusion-checker", f)
	EspressoStateCheckerConfigAddOptions(prefix+".state-checker", f)
}

func DangerousCaffNodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".ignore-database-hotshot-block", DefaultDangerousCaffNodeConfig.IgnoreDatabaseHotshotBlock, "Ignores the database hotshot block and starts from the next block specified in the config by the user")
	f.Bool(prefix+".ignore-database-from-block", DefaultDangerousCaffNodeConfig.IgnoreDatabaseFromBlock, "Ignores the database from block and starts from the next block specified in the config by the user")
}

type EspressoCaffNodeConfigFetcher func() *EspressoCaffNodeConfig

type EspressoCaffNode struct {
	stopwaiter.StopWaiter

	executionEngine  *gethexec.ExecutionEngine
	espressoStreamer espressostreamer.EspressoStreamerInterface

	configFetcher EspressoCaffNodeConfigFetcher
	db            ethdb.Database

	delayedMessageFetcher DelayedMessageFetcherInterface

	l1Reader *headerreader.HeaderReader

	forceInclusionChecker *ForceInclusionChecker
	stateChecker          *StateChecker

	batcherAddrMonitor *BatcherAddrMonitor
}

func NewEspressoCaffNode(
	configFetcher EspressoCaffNodeConfigFetcher,
	execEngine *gethexec.ExecutionEngine,
	delayedBridge *DelayedBridge,
	l1Reader *headerreader.HeaderReader,
	db ethdb.Database,
	recordPerformance bool,
	blocksToRead uint64,
	sequencerInbox *SequencerInbox,
	fatalErrChan chan error,
	httpPort int,
) *EspressoCaffNode {
	if !configFetcher().Enable {
		return nil
	}

	if l1Reader == nil {
		log.Crit("l1Reader is nil")
		return nil
	}

	// For backward compatibility, the espresso streamer should be able to verify legacy where we signed
	// hotshot transactions using SGX quote. Therefore we create a SGX TEE verifier here.
	sgxVerifier, err := espressotee.NewEspressoSGXVerifier(
		l1Reader.Client(),
		common.HexToAddress(configFetcher().EspressoSGXVerifierAddr),
	)
	if err != nil {
		log.Crit("failed to create espressoTEEVerifier", "err", err)
		return nil
	}
	client, err := espressoClient.NewMultipleNodesClient(configFetcher().HotShotUrls)
	if err != nil {
		log.Crit("Failed to create hotshot client", "err", err)
	}

	batcherAddrMonitor := NewBatcherAddrMonitor(
		[]common.Address{common.HexToAddress(configFetcher().BatchPosterAddr)},
		db,
		l1Reader,
		sequencerInbox.address,
		delayedBridge.fromBlock,
		configFetcher().FromBlock,
	)
	espressoStreamer := espressostreamer.NewEspressoStreamer(configFetcher().Namespace,
		configFetcher().NextHotshotBlock,
		sgxVerifier,
		client,
		recordPerformance,
		batcherAddrMonitor.GetValidAddresses,
		configFetcher().RetryTime,
	)

	fromBlock := configFetcher().FromBlock
	if !configFetcher().Dangerous.IgnoreDatabaseFromBlock {
		fromBlock, err = readCurrentFromBlockFromDb(db)
		if err != nil {
			log.Crit("failed to read l1 block from db", "err", err)
		}
	}

	if fromBlock == 0 {
		fromBlock = configFetcher().FromBlock
		if fromBlock == 0 {
			log.Crit("fromBlock is 0, please provide a valid block number")
		}
	}

	delayedMessageFetcher := NewDelayedMessageFetcher(delayedBridge, l1Reader, db, blocksToRead,
		configFetcher().WaitForFinalization, configFetcher().WaitForConfirmations, configFetcher().RequiredBlockDepth, fromBlock, sequencerInbox)

	seqInbox, err := bridgegen.NewSequencerInbox(sequencerInbox.address, l1Reader.Client())
	if err != nil {
		log.Crit("failed to create sequencer inbox", "err", err)
		return nil
	}

	forceInclusionChecker := NewForceInclusionChecker(
		&SeqInbox{seqInbox: seqInbox},
		configFetcher().ForceInclusionChecker,
		l1Reader,
		delayedMessageFetcher,
		fatalErrChan,
	)

	stateChecker := NewStateChecker(
		configFetcher().StateChecker,
		httpPort,
		fatalErrChan,
	)

	return &EspressoCaffNode{
		configFetcher:         configFetcher,
		executionEngine:       execEngine,
		delayedMessageFetcher: delayedMessageFetcher,
		espressoStreamer:      espressoStreamer,
		db:                    db,
		l1Reader:              l1Reader,
		forceInclusionChecker: forceInclusionChecker,
		stateChecker:          stateChecker,
		batcherAddrMonitor:    batcherAddrMonitor,
	}
}

// peekMessage wraps the espressoStreamer.Peek() method, to handle producing delayed messages by checking they are within the nodes safety tolerance.
// Returns:
//   - MessageWithMetadataAndPos: A message, delayed or normally sequenced, that is for the next position in the chain.
//   - error: If any error is encountered during this function it is propegated to the caller.
//
// Semantics:
//
//	This function will either produce a message, or an error. When an error is produced, the messageWithMetadataAndPos will be nil.
//	If the message is populated, the error will be nil.
func (n *EspressoCaffNode) peekMessage(ctx context.Context) (*espressostreamer.MessageWithMetadataAndPos, error) {
	messageWithMetadataAndPos := n.espressoStreamer.Peek(ctx)

	if messageWithMetadataAndPos == nil {
		return nil, nil
	}

	// Check if its a delayed message, if so fetch from the database
	delayedMessageToProcessIndex, err := n.executionEngine.NextDelayedMessageNumber()
	if err != nil {
		log.Error("failed to get next delayed message number", "err", err)
		return nil, err
	}
	if delayedMessageToProcessIndex == messageWithMetadataAndPos.MessageWithMeta.DelayedMessagesRead-1 {
		messageWithMetadataAndPosDelayed, err := n.delayedMessageFetcher.processDelayedMessage(messageWithMetadataAndPos)
		if err != nil {
			log.Error("unable to get the next delayed message", "err", err)
			return nil, err
		}
		return messageWithMetadataAndPosDelayed, nil
	}

	return messageWithMetadataAndPos, nil
}

// Creates a block from the next message in the queue.
func (n *EspressoCaffNode) createBlock(ctx context.Context) (returnValue bool) {

	lastBlockHeader := n.executionEngine.Bc().CurrentBlock()

	messageWithMetadataAndPos, err := n.peekMessage(ctx)
	if err != nil {
		log.Warn("unable to get next message", "err", err)
		return false
	}

	if messageWithMetadataAndPos == nil {
		// No message found, so we need to wait for the next message
		return false
	}

	messageWithMetadata := messageWithMetadataAndPos.MessageWithMeta

	// Get the state of the database at the last block
	statedb, err := n.executionEngine.Bc().StateAt(lastBlockHeader.Root)
	if err != nil {
		log.Error("failed to get state at last block header", "err", err)
		return false
	}

	log.Info("Initial State", "lastBlockHash", lastBlockHeader.Hash(), "lastBlockStateRoot", lastBlockHeader.Root)
	startTime := time.Now()

	// Run the Produce block function in replay mode
	// This is the core function that is used by replay.wasm to validate the block
	block, receipts, err := arbos.ProduceBlock(messageWithMetadata.Message,
		messageWithMetadata.DelayedMessagesRead,
		lastBlockHeader,
		statedb,
		n.executionEngine.Bc(),
		false,
		core.MessageReplayMode)

	if err != nil || block == nil {
		log.Error("Failed to produce block", "err", err)
		return false
	}

	blockCalcTime := time.Since(startTime)

	log.Info("Produced block", "block", block.Hash(), "blockNumber", block.Number(), "receipts", len(receipts))

	hotshotBlockNumber := n.espressoStreamer.GetCurrentEarliestHotShotBlockNumber()
	err = n.espressoStreamer.StoreHotshotBlock(n.db, hotshotBlockNumber)
	if err != nil {
		log.Warn("Failed to store hotshot block. This should be an ephemeral error", "err", err)
		return false
	}

	err = n.executionEngine.AppendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		log.Error("Failed to append block", "err", err)
		return false
	}

	n.espressoStreamer.Advance()

	n.executionEngine.Bc().SetFinalized(block.Header())
	n.executionEngine.Bc().SetSafe(block.Header())
	n.espressoStreamer.RecordTimeDurationBetweenHotshotAndCurrentBlock(messageWithMetadataAndPos.HotshotHeight, time.Now())

	return true
}

func (n *EspressoCaffNode) GetEspressoStreamer() espressostreamer.EspressoStreamerInterface {
	return n.espressoStreamer
}

func (n *EspressoCaffNode) Start(ctx context.Context) error {
	log.Info("Starting espresso caff node")
	n.StopWaiter.Start(ctx, n)
	err := n.espressoStreamer.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start espresso streamer: %w", err)
	}
	err = n.batcherAddrMonitor.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start batcher address monitor: %w", err)
	}
	err = n.forceInclusionChecker.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start force inclusion checker: %w", err)
	}

	if n.stateChecker != nil {
		err = n.stateChecker.Start(ctx)
		if err != nil {
			return fmt.Errorf("failed to start state checker: %w", err)
		}
	}

	// This is +1 because the current block is the block after the last processed block
	currentBlockNum := n.executionEngine.Bc().CurrentBlock().Number.Uint64() + 1
	currentMessagePos, err := n.executionEngine.BlockNumberToMessageIndex(currentBlockNum)
	if err != nil {
		return fmt.Errorf("failed to convert block number to message index: %w", err)
	}
	var nextHotshotBlock uint64

	if !n.configFetcher().Dangerous.IgnoreDatabaseHotshotBlock {
		nextHotshotBlock, err = n.espressoStreamer.ReadNextHotshotBlockFromDb(n.db)
		if err != nil {
			return fmt.Errorf("failed to read next hotshot block: %w", err)
		}
	}

	if nextHotshotBlock == 0 {
		// No next hotshot block found, so we need to start from config.CaffNodeConfig.NextHotshotBlock
		nextHotshotBlock = n.configFetcher().NextHotshotBlock
		if nextHotshotBlock == 0 {
			return errors.New("no next hotshot block found in database or dangerous.ignore-database-hotshot-block is set to true, please set config.CaffNodeConfig.NextHotshotBlock")
		}
	}
	// The reason we do the reset here is because database is only initialized after Caff node is initialized
	// so if we want to read the current position from the database, we need to reset the streamer
	// during the start of the espresso streamer and caff node
	log.Info("Starting streamer at", "nextHotshotBlock", nextHotshotBlock, "currentMessagePos", currentMessagePos)
	n.espressoStreamer.Reset(uint64(currentMessagePos), nextHotshotBlock)

	// Nonce of the previous block is the number of delayed messages read
	// Check `NextDelayedMessageNumber` in execution node to confirm this
	delayedMessagesRead := n.executionEngine.Bc().CurrentBlock().Nonce.Uint64()
	// we store delayedmessagecount-1 because that is the index of the delayed message
	// that needs to be read
	err = n.delayedMessageFetcher.storeDelayedMessageLatestIndex(n.db, delayedMessagesRead-1)
	if err != nil {
		log.Error("failed to store delayed message count", "err", err)
		return err
	}
	log.Debug("stored delayed message count", "delayedMessagesRead", delayedMessagesRead-1)

	// Start the delayed message fetcher
	started := n.delayedMessageFetcher.Start(ctx)
	if !started {
		return fmt.Errorf("failed to start delayed message fetcher")
	}

	log.Info("started delayed message fetcher")

	err = n.CallIterativelySafe(func(ctx context.Context) time.Duration {
		madeBlock := n.createBlock(ctx)
		if madeBlock {
			return n.configFetcher().HotshotPollingInterval
		}
		return n.configFetcher().RetryTime
	})
	if err != nil {
		return fmt.Errorf("failed to start node, error in createBlock: %w", err)
	}

	return nil
}
