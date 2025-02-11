package gethexec

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-sequencer-go/client"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

/*
Caff Node creates blocks with finalized hotshot transactions
*/
type CaffNode struct {
	stopwaiter.StopWaiter

	config                  SequencerConfigFetcher
	namespace               uint64
	executionEngine         *ExecutionEngine
	espressoClient          *espressoClient.Client
	nextHotshotBlockNum     uint64
	messagesWithMetadata    []*arbostypes.MessageWithMetadata
	messagesWithMetadataPos []uint64
	messagesStateMutex      sync.Mutex
	skippedBlockPos         *uint64
	retryInterval           time.Duration
	hotshotPollingInterval  time.Duration
}

func NewCaffNode(configFetcher SequencerConfigFetcher, execEngine *ExecutionEngine) *CaffNode {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		log.Crit("Failed to validate caff  node config", "err", err)
	}
	return &CaffNode{
		config:                 configFetcher,
		namespace:              config.CaffNodeConfig.Namespace,
		espressoClient:         espressoClient.NewClient(config.CaffNodeConfig.HotShotUrl),
		nextHotshotBlockNum:    config.CaffNodeConfig.StartBlock,
		executionEngine:        execEngine,
		retryInterval:          config.CaffNodeConfig.RetryInterval,
		hotshotPollingInterval: config.CaffNodeConfig.HotshotPollingInterval,
	}
}

// TODO: For future versions, we should check the attestation quote to check if its from a valid TEE
// TODO: This machine should run in TEE and submit blocks to espresso only if the block is valid with an attestation.
/**
 * This function will create a block with the finalized hotshot transactions
 * It will first remove duplicates and ensure the ordering of messages is correct
 * Then it will run the STF using the `Produce Block`function and finally store the block in the database
 */
func (n *CaffNode) createBlock() (returnValue bool) {

	n.messagesStateMutex.Lock()
	defer n.messagesStateMutex.Unlock()

	//  If we have no messages to process, return
	if len(n.messagesWithMetadata) == 0 {
		return false
	}
	messageWithMetadata := n.messagesWithMetadata[0]
	messageWithMetadataPos := n.messagesWithMetadataPos[0]

	// Get the last block header stored in the database
	if n.executionEngine.bc == nil {
		log.Error("execution engine bc not initialized")
		return false
	}

	lastBlockHeader := n.executionEngine.bc.CurrentBlock()

	currentMsgPos, err := n.executionEngine.BlockNumberToMessageIndex(lastBlockHeader.Number.Uint64())
	if err != nil {
		log.Error("failed to convert block number to message index")
		return false
	}
	currentPos := uint64(currentMsgPos)

	// Check for duplicates and remove them
	if messageWithMetadataPos <= currentPos {
		log.Error("message has already been processed, removing duplicate",
			"messageWithMetadataPos", messageWithMetadataPos, "currentMessageCount", currentPos)
		n.messagesWithMetadata = n.messagesWithMetadata[1:]
		n.messagesWithMetadataPos = n.messagesWithMetadataPos[1:]
		return false
	}

	// Check if the message is in the correct order, it should be sequentially increasing
	if (messageWithMetadataPos != currentPos+1) && n.skippedBlockPos == nil {
		log.Error("order of message is incorrect", "currentPos", currentPos,
			"messageWithMetadataPos", messageWithMetadataPos)
		return false
	}

	// If a message was skipped, check if the message is in the correct order
	if n.skippedBlockPos != nil && messageWithMetadataPos != *n.skippedBlockPos+1 {
		log.Error("order of message is incorrect", "skippedBlockPos", *n.skippedBlockPos,
			"messageWithMetadataPos", messageWithMetadataPos)
		return false
	}

	// Get the state of the database at the last block
	statedb, err := n.executionEngine.bc.StateAt(lastBlockHeader.Root)
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
		n.executionEngine.bc,
		n.executionEngine.bc.Config(),
		false,
		core.MessageReplayMode)

	if err != nil {
		log.Error("Failed to produce block", "err", err)
		// if we fail to produce a block, we should remove this message from the queue
		// and set skippedBlockPos to the current messageWithMetadataPos
		n.skippedBlockPos = &messageWithMetadataPos
		n.messagesWithMetadata = n.messagesWithMetadata[1:]
		n.messagesWithMetadataPos = n.messagesWithMetadataPos[1:]
		return false
	}

	// If block is nil or receipts is empty, return false
	if len(receipts) == 0 || block == nil {
		log.Error("Failed to produce block, no receipts or block")
		// if we fail to produce a block, we should remove this message from the queue
		// and set skippedBlockPos to the current messageWithMetadataPos
		n.skippedBlockPos = &messageWithMetadataPos
		n.messagesWithMetadata = n.messagesWithMetadata[1:]
		n.messagesWithMetadataPos = n.messagesWithMetadataPos[1:]
		return false
	}

	// Reset the skippedBlockPos
	n.skippedBlockPos = nil

	blockCalcTime := time.Since(startTime)

	log.Info("Produced block", "block", block.Hash(), "blockNumber", block.Number(), "receipts", len(receipts))

	err = n.executionEngine.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		log.Error("Failed to append block", "err", err)
		return false
	}

	// Pop the message from the front of the queue at the end.
	n.messagesWithMetadata = n.messagesWithMetadata[1:]
	n.messagesWithMetadataPos = n.messagesWithMetadataPos[1:]

	return true
}

/**
* This function will create a queue of messages from the hotshot to be processed by the node
* It will sort the messages by the message index
* and store the messages in `messagesWithMetadata` queue
 */
func (n *CaffNode) queueMessagesFromHotshot(ctx context.Context) error {
	if n.nextHotshotBlockNum == 0 {
		latestBlock, err := n.espressoClient.FetchLatestBlockHeight(ctx)
		if err != nil {
			log.Warn("unable to fetch latest hotshot block", "err", err)
			return err
		}
		log.Info("Started node at the latest hotshot block", "block number", latestBlock)
		n.nextHotshotBlockNum = latestBlock
	}

	nextHotshotBlockNum := n.nextHotshotBlockNum
	header, err := n.espressoClient.FetchHeaderByHeight(ctx, nextHotshotBlockNum)
	if err != nil {
		log.Warn("failed to fetch header", "err", err)
		return err
	}
	height := header.Header.GetBlockHeight()
	arbTxns, err := n.espressoClient.FetchTransactionsInBlock(ctx, height, n.namespace)
	if err != nil {
		log.Warn("failed to fetch transactions", "err", err)
		return err
	}
	if len(arbTxns.Transactions) == 0 {
		return nil
	}

	n.messagesStateMutex.Lock()
	defer n.messagesStateMutex.Unlock()
	for _, tx := range arbTxns.Transactions {
		// Parse hotshot payload
		_, indices, messages, err := arbutil.ParseHotShotPayload(tx)
		if err != nil {
			log.Warn("failed to parse hotshot payload, will retry", "err", err)
			return err
		}
		// Parse the messages
		for i, message := range messages {
			var messageWithMetadata arbostypes.MessageWithMetadata
			err = rlp.DecodeBytes(message, &messageWithMetadata)
			if err != nil {
				log.Warn("failed to decode message, will retry", "err", err)
				return err
			}
			n.messagesWithMetadata = append(n.messagesWithMetadata, &messageWithMetadata)
			n.messagesWithMetadataPos = append(n.messagesWithMetadataPos, indices[i])

		}
	}
	// Sort the messagesWithMetadata and messagesWithMetadataPos based on ascending order
	// This is to ensure that we process messages in the correct order
	sort.SliceStable(n.messagesWithMetadata, func(i, j int) bool {
		return n.messagesWithMetadataPos[i] < n.messagesWithMetadataPos[j]
	})
	sort.SliceStable(n.messagesWithMetadataPos, func(i, j int) bool {
		return n.messagesWithMetadataPos[i] < n.messagesWithMetadataPos[j]
	})

	return nil
}

func (n *CaffNode) Start(ctx context.Context) error {
	n.StopWaiter.Start(ctx, n)

	err := n.CallIterativelySafe(func(ctx context.Context) time.Duration {
		err := n.queueMessagesFromHotshot(ctx)
		if err != nil {
			return n.retryInterval
		}
		n.nextHotshotBlockNum += 1
		log.Info("Now processing hotshot block", "block", n.nextHotshotBlockNum)
		return n.hotshotPollingInterval
	})
	if err != nil {
		return fmt.Errorf("failed to start  node, error in queueMessagesFromHotshot: %w", err)
	}

	err = n.CallIterativelySafe(func(ctx context.Context) time.Duration {
		madeBlock := n.createBlock()
		if madeBlock {
			return n.hotshotPollingInterval
		}
		return n.retryInterval
	})
	if err != nil {
		return fmt.Errorf("failed to start node, error in createBlock: %w", err)
	}
	return nil
}

func (n *CaffNode) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return nil
}

func (n *CaffNode) CheckHealth(ctx context.Context) error {
	return nil
}

func (n *CaffNode) Initialize(ctx context.Context) error {
	return nil
}
