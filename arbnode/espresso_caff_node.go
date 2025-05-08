package arbnode

import (
	"context"
	"fmt"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network-go/client"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/espressostreamer"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type EspressoCaffNodeConfig struct {
	Enable                  bool          `koanf:"enable"`
	HotShotUrls             []string      `koanf:"hotshot-urls"`
	NextHotshotBlock        uint64        `koanf:"next-hotshot-block"`
	Namespace               uint64        `koanf:"namespace"`
	RetryTime               time.Duration `koanf:"retry-time"`
	HotshotPollingInterval  time.Duration `koanf:"hotshot-polling-interval"`
	EspressoTEEVerifierAddr string        `koanf:"espresso-tee-verifier-addr"`
	BatchPosterAddr         string        `koanf:"batch-poster-addr"`
	RecordPerformance       bool          `koanf:"record-performance"`
	WaitForFinalization     bool          `koanf:"wait-for-finalization"`
	WaitForConfirmations    bool          `koanf:"wait-for-confirmations"`
	RequiredBlockDepth      uint64        `koanf:"required-block-depth"`
	BlocksToRead            uint64        `koanf:"blocks-to-read"`
}

var DefaultEspressoCaffNodeConfig = EspressoCaffNodeConfig{
	Enable:                  false,
	HotShotUrls:             []string{},
	NextHotshotBlock:        1,
	Namespace:               0,
	RetryTime:               time.Second * 2,
	HotshotPollingInterval:  time.Millisecond * 100,
	EspressoTEEVerifierAddr: "",
	BatchPosterAddr:         "",
	RecordPerformance:       false,
	WaitForFinalization:     true,
	WaitForConfirmations:    false,
	RequiredBlockDepth:      6,
	BlocksToRead:            100,
}

func EspressoCaffNodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultEspressoCaffNodeConfig.Enable, "enable espresso Caff node")
	f.StringSlice(prefix+".hotshot-urls", DefaultEspressoCaffNodeConfig.HotShotUrls, "Hotshot urls")
	f.Uint64(prefix+".next-hotshot-block", DefaultEspressoCaffNodeConfig.NextHotshotBlock, "the Hotshot block number from which the Caff node will read")
	f.Uint64(prefix+".namespace", DefaultEspressoCaffNodeConfig.Namespace, "the namespace of the chain in Espresso Network, usually the chain id")
	f.Duration(prefix+".retry-time", DefaultEspressoCaffNodeConfig.RetryTime, "retry time after a failure")
	f.Duration(prefix+".hotshot-polling-interval", DefaultEspressoCaffNodeConfig.HotshotPollingInterval, "time after a success")
	f.String(prefix+".espresso-tee-verifier-addr", "", "tee verifier address")
	f.String(prefix+".batch-poster-addr", DefaultEspressoCaffNodeConfig.BatchPosterAddr, "batch poster address that is used to verify the signature of the Hotshot transactions")
	f.Bool(prefix+".record-performance", DefaultEspressoCaffNodeConfig.RecordPerformance, "record performance of the Caff node")
	f.Bool(prefix+".wait-for-finalization", DefaultEspressoCaffNodeConfig.WaitForFinalization, "Configures the Caff node to only produce blocks from delayed messages if they are finalized on the parent chain")
	f.Bool(prefix+".wait-for-confirmations", DefaultEspressoCaffNodeConfig.WaitForConfirmations, "Configures the Caff node to only produce blocks from delayed messages if they have atleast requiredBlockDepth confirmations on the parent chain")
	f.Uint64(prefix+".required-block-depth", DefaultEspressoCaffNodeConfig.RequiredBlockDepth, "Configures the required block depth/number of confirmations on the parent chain that a delayed message is required to have before this Caff node will add it to it's state")
	f.Uint64(prefix+".blocks-to-read", DefaultEspressoCaffNodeConfig.BlocksToRead, "Configures the number of blocks to read from the parent chain for delayed messages")
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
}

func NewEspressoCaffNode(
	configFetcher EspressoCaffNodeConfigFetcher,
	execEngine *gethexec.ExecutionEngine,
	delayedBridge *DelayedBridge,
	l1Reader *headerreader.HeaderReader,
	db ethdb.Database,
	recordPerformance bool,
	blocksToRead uint64,
) *EspressoCaffNode {
	if !configFetcher().Enable {
		return nil
	}

	if l1Reader == nil {
		log.Crit("l1Reader is nil")
		return nil
	}

	espressoTEEVerifierCaller, err := bridgegen.NewEspressoTEEVerifier(
		common.HexToAddress(configFetcher().EspressoTEEVerifierAddr),
		l1Reader.Client())

	if err != nil || espressoTEEVerifierCaller == nil {
		log.Crit("failed to create espressoTEEVerifierCaller", "err", err)
		return nil
	}

	espressoStreamer := espressostreamer.NewEspressoStreamer(configFetcher().Namespace,
		configFetcher().NextHotshotBlock,
		configFetcher().RetryTime,
		configFetcher().HotshotPollingInterval,
		espressoTEEVerifierCaller,
		espressoClient.NewMultipleNodesClient(configFetcher().HotShotUrls),
		recordPerformance,
		common.HexToAddress(configFetcher().BatchPosterAddr),
	)

	delayedMessageFetcher := NewDelayedMessageFetcher(delayedBridge, l1Reader, db, blocksToRead,
		configFetcher().WaitForFinalization, configFetcher().WaitForConfirmations, configFetcher().RequiredBlockDepth)

	return &EspressoCaffNode{
		configFetcher:         configFetcher,
		executionEngine:       execEngine,
		delayedMessageFetcher: delayedMessageFetcher,
		espressoStreamer:      espressoStreamer,
		db:                    db,
		l1Reader:              l1Reader,
	}
}

// nextMessage wraps the espressoStreamer.Next() method, to handle producing delayed messages by checking they are within the nodes safety tolerance.
// Returns:
//   - MessageWithMetadataAndPos: A message, delayed or normally sequenced, that is for the next position in the chain.
//   - error: If any error is encountered during this function it is propegated to the caller.
//
// Semantics:
//
//	This function will either produce a message, or an error. When an error is produced, the messageWithMetadataAndPos will be nil.
//	If the message is populated, the error will be nil.
func (n *EspressoCaffNode) nextMessage() (*espressostreamer.MessageWithMetadataAndPos, error) {
	messageWithMetadataAndPos, err := n.espressoStreamer.Next()
	if err != nil {
		return nil, err
	}

	if messageWithMetadataAndPos == nil {
		return nil, nil
	}

	messageWithMetadataAndPos, err = n.delayedMessageFetcher.processDelayedMessage(messageWithMetadataAndPos)
	if err != nil {
		log.Error("unable to get the next delayed message", "err", err)
		n.reset(messageWithMetadataAndPos)
		return nil, err
	}

	return messageWithMetadataAndPos, nil
}

/*
Resets the espresso streamer to the given message and hotshot height.
*/
func (n *EspressoCaffNode) reset(messageWithMetadataAndPos *espressostreamer.MessageWithMetadataAndPos) {
	n.espressoStreamer.Reset(messageWithMetadataAndPos.Pos, messageWithMetadataAndPos.HotshotHeight)
}

/*
Creates a block from the next message in the queue.
*/
func (n *EspressoCaffNode) createBlock() (returnValue bool) {

	lastBlockHeader := n.executionEngine.Bc().CurrentBlock()

	messageWithMetadataAndPos, err := n.nextMessage()
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
		log.Debug("Resetting espresso streamer", "currentMessagePos",
			messageWithMetadataAndPos.Pos, "currentHostshotBlock",
			messageWithMetadataAndPos.HotshotHeight)
		n.espressoStreamer.Reset(messageWithMetadataAndPos.Pos, messageWithMetadataAndPos.HotshotHeight)
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
		n.executionEngine.Bc().Config(),
		false,
		core.MessageReplayMode)

	if err != nil || block == nil {
		log.Error("Failed to produce block", "err", err)
		log.Debug("Resetting espresso streamer", "currentMessagePos",
			messageWithMetadataAndPos.Pos, "currentHostshotBlock",
			messageWithMetadataAndPos.HotshotHeight)
		n.reset(messageWithMetadataAndPos)
		return false
	}

	blockCalcTime := time.Since(startTime)

	log.Info("Produced block", "block", block.Hash(), "blockNumber", block.Number(), "receipts", len(receipts))

	err = n.espressoStreamer.StoreHotshotBlock(n.db, messageWithMetadataAndPos.HotshotHeight)
	if err != nil {
		log.Error("Failed to store hotshot block", "err", err)
		log.Debug("Resetting espresso streamer", "currentMessagePos",
			messageWithMetadataAndPos.Pos, "currentHostshotBlock",
			messageWithMetadataAndPos.HotshotHeight)
		n.reset(messageWithMetadataAndPos)
		return false
	}

	err = n.executionEngine.AppendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		log.Error("Failed to append block", "err", err)
		log.Debug("Resetting espresso streamer", "currentMessagePos",
			messageWithMetadataAndPos.Pos, "currentHostshotBlock",
			messageWithMetadataAndPos.HotshotHeight)
		n.reset(messageWithMetadataAndPos)
		return false
	}

	n.espressoStreamer.RecordTimeDurationBetweenHotshotAndCurrentBlock(messageWithMetadataAndPos.HotshotHeight, time.Now())

	return true
}

func (n *EspressoCaffNode) Start(ctx context.Context) error {
	n.StopWaiter.Start(ctx, n)
	err := n.espressoStreamer.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start espresso streamer: %w", err)
	}
	// This is +1 because the current block is the block after the last processed block
	currentBlockNum := n.executionEngine.Bc().CurrentBlock().Number.Uint64() + 1
	currentMessagePos, err := n.executionEngine.BlockNumberToMessageIndex(currentBlockNum)
	if err != nil {
		return fmt.Errorf("failed to convert block number to message index: %w", err)
	}
	nextHotshotBlock, err := n.espressoStreamer.ReadNextHotshotBlockFromDb(n.db)
	if err != nil {
		log.Crit("failed to read next hotshot block", "err", err)
		return nil
	}

	if nextHotshotBlock == 0 {
		// No next hotshot block found, so we need to start from config.CaffNodeConfig.NextHotshotBlock
		nextHotshotBlock = n.configFetcher().NextHotshotBlock
		if nextHotshotBlock == 0 {
			log.Crit("No next hotshot block found in database, and no config.CaffNodeConfig.NextHotshotBlock set")
		}
	}
	// The reason we do the reset here is because database is only initialized after Caff node is initialized
	// so if we want to read the current position from the database, we need to reset the streamer
	// during the start of the espresso streamer and caff node
	log.Debug("Starting streamer at", "nextHotshotBlock", nextHotshotBlock, "currentMessagePos", currentMessagePos)
	n.espressoStreamer.Reset(uint64(currentMessagePos), nextHotshotBlock)

	// Deserialize the current block from the database to get the parent chain block number
	// and the delayed messages read. Note: the nonce in the header of the block contains the delayed messages read
	header := types.DeserializeHeaderExtraInformation(n.executionEngine.Bc().CurrentHeader())
	parentChainBlockNumber := header.L1BlockNumber
	// Nonce of the previous block is the number of delayed messages read
	// Check `NextDelayedMessageNumber` in execution node to confirm this
	delayedMessagesRead := n.executionEngine.Bc().CurrentBlock().Nonce.Uint64()
	n.delayedMessageFetcher.reset(parentChainBlockNumber, delayedMessagesRead)

	err = n.CallIterativelySafe(func(ctx context.Context) time.Duration {
		madeBlock := n.createBlock()
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
