// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/espressostreamer"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	DelayedFetcherCurrentL1BlockKey = []byte("espressoDelayedFetcherCurrentL1Block")
	DelayedMessageCountKey          = []byte("espressoDelayedMessageCount")
	// To not to mess with the existing schema, we use another prefix
	DelayedMessagePrefix = []byte("espressoDelayed")
)

type DelayedMessageFetcherInterface interface {
	reset(seqNum uint64)
	getDelayedMessageCountAtBlock(blockNumber uint64) (uint64, error)
	processDelayedMessage(messageWithMetadataAndPos *espressostreamer.MessageWithMetadataAndPos) (*espressostreamer.MessageWithMetadataAndPos, error)
}

type DelayedMessageFetcher struct {
	fromBlock            uint64
	delayedBridge        *DelayedBridge
	l1Reader             *headerreader.HeaderReader
	blocksToRead         uint64
	db                   ethdb.Database
	delayedCount         uint64
	waitForFinalization  bool
	waitForConfirmations bool
	requiredBlockDepth   uint64
}

func NewDelayedMessageFetcher(
	delayedBridge *DelayedBridge,
	l1Reader *headerreader.HeaderReader,
	db ethdb.Database,
	blocksToRead uint64,
	waitForFinalization bool,
	waitForConfirmations bool,
	requiredBlockDepth uint64,
	fromBlock uint64,
) *DelayedMessageFetcher {

	delayedCount, err := readDelayedMessageCount(db)
	if err != nil && !dbutil.IsErrNotFound(err) {
		log.Crit("failed to read delayed message count from db", "err", err)
		return nil
	}

	if delayedCount == 0 {
		delayedCount = 1
	}

	return &DelayedMessageFetcher{
		fromBlock:            fromBlock,
		delayedBridge:        delayedBridge,
		l1Reader:             l1Reader,
		db:                   db,
		blocksToRead:         blocksToRead,
		delayedCount:         delayedCount,
		waitForFinalization:  waitForFinalization,
		waitForConfirmations: waitForConfirmations,
		requiredBlockDepth:   requiredBlockDepth,
	}
}

func (f *DelayedMessageFetcher) reset(seqNum uint64) {
	f.delayedCount = seqNum
}

// getDelayedMessageCountAtBlock is a wrapper function for the delayedBridge.GetMessageCount function. This allows users of the DelayedMessageFetcher
// to query for the message count at a block.
func (f *DelayedMessageFetcher) getDelayedMessageCountAtBlock(blockNumber uint64) (uint64, error) {
	count, err := f.delayedBridge.GetMessageCount(context.Background(), new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (f *DelayedMessageFetcher) getDelayedMessage(index uint64) (*arbostypes.L1IncomingMessage, error) {
	// Check if the delayed message at index exists in the database
	msg, err := f.readDelayedMessage(index)
	if err != nil && !dbutil.IsErrNotFound(err) {
		log.Error("Failed to read delayed message", "err", err, "msg", msg)
		return nil, err
	}
	// If the delayed message already exists in the database and we have already processed it
	// the parent block number then we can just return the message
	if msg != nil && f.fromBlock >= msg.ParentChainBlockNumber {
		log.Debug("Delayed message already exists in the database and we have already processed it", "msg", msg.ParentChainBlockNumber, "fromBlock", f.fromBlock)
		return msg.Message, nil
	}

	// get the current block number from the L1 Reader
	currL1, err := f.l1Reader.Client().BlockNumber(context.Background())
	if err != nil {
		return nil, err
	}
	// if the current L1 block is less than the from block this means no new L1 blocks have been added
	// since we did the last read, so we can just return nil
	if currL1 < f.fromBlock {
		return nil, fmt.Errorf("l1 block number %d is less than from block %d", currL1, f.fromBlock)
	}

	log.Debug("Current L1 block and from block:", "currL1", currL1, "fromBlock", f.fromBlock)

	startBlock := f.fromBlock
	endBlock := currL1
	hasFound := false

	batch := f.db.NewBatch()
	// Lookup `MessageDelivered` events from the `startBlock` to `startBlock + blocksToRead`
	// see if any of them have a `message` field that matches the `seqNum` we are looking for
	for startBlock <= endBlock && !hasFound {
		from := big.NewInt(0).SetUint64(startBlock)
		to := big.NewInt(0).SetUint64(startBlock + f.blocksToRead)

		log.Debug("Looking for delayed messages from range", "from", from, "to", to)
		msgs, err := f.delayedBridge.LookupMessagesInRange(context.Background(), from, to, nil)
		if err != nil {
			log.Error("Failed to lookup delayed messages", "err", err)
			return nil, err
		}
		for _, msg := range msgs {
			seqNum, err := msg.Message.Header.SeqNum()
			if err != nil {
				return nil, err
			}
			if seqNum == index {
				hasFound = true
			}
			err = f.storeDelayedMessage(batch, seqNum, *msg)
			if err != nil {
				return nil, err
			}
		}
		// Read the next `blocksToRead` blocks
		startBlock = startBlock + f.blocksToRead + 1
	}

	// if startBlock is less than the endBlock this means
	// we were able to find the delayed message number before the endBlock
	// so next time, we start from where we left off
	if startBlock <= endBlock {
		f.fromBlock = startBlock
	} else {
		f.fromBlock = endBlock + 1
	}

	err = storeCurrentL1Block(batch, f.fromBlock)
	if err != nil {
		log.Error("Failed to store current L1 block", "err", err)
		return nil, err
	}

	err = batch.Write()
	if err != nil {
		return nil, err
	}

	if !hasFound {
		return nil, fmt.Errorf("no message found for pos %d", index)
	}

	result, err := f.readDelayedMessage(index)
	if err != nil {
		log.Error("Failed to read delayed message", "err", err)
		return nil, err
	}

	return result.Message, nil
}

func (f *DelayedMessageFetcher) processDelayedMessage(messageWithMetadataAndPos *espressostreamer.MessageWithMetadataAndPos) (*espressostreamer.MessageWithMetadataAndPos, error) {
	delayedMessagesRead := messageWithMetadataAndPos.MessageWithMeta.DelayedMessagesRead
	if delayedMessagesRead > f.delayedCount+1 || delayedMessagesRead < f.delayedCount {
		log.Error("messages are not processed in order", "delayedMessagesRead", delayedMessagesRead, "delayedCount", f.delayedCount)
		return nil, fmt.Errorf("delayed message count is greater than the delayed count")
	}
	if delayedMessagesRead == f.delayedCount+1 {
		log.Debug("Getting delayed message", "delayedCount", f.delayedCount)
		// If this is delayed message, we need to get the message from L1
		// and replace the message in the messageWithMetadataAndPos
		// Note: here we are using DelayedMessagesRead - 1 because that is the index of the delayed message
		// that needs to be read
		message, err := f.getDelayedMessage(f.delayedCount)
		if err != nil {
			log.Error("failed to get delayed message", "err", err)
			return messageWithMetadataAndPos, err
		}
		messageWithMetadataAndPos.MessageWithMeta.Message = message
		isDelayedMessageWithinSafetyTolerance, err := f.isDelayedMessageWithinSafetyTolerance(messageWithMetadataAndPos)
		if err != nil {
			return messageWithMetadataAndPos, err
		}

		if !isDelayedMessageWithinSafetyTolerance {
			return messageWithMetadataAndPos, fmt.Errorf("delayed message was not within safety tolerance parameters, the node needs to wait until it is")
		}
		f.delayedCount++
		err = storeDelayedMessageCount(f.db, f.delayedCount)
		if err != nil {
			log.Error("Failed to store delayed message count", "err", err)
			return messageWithMetadataAndPos, err
		}
	}

	return messageWithMetadataAndPos, nil
}

// isDelayedMessageWithinSafetyTolerance determines if a delayed message given to it is within the configured safety tolerance of this Caff node
// Parameters:
//
//	message  - a Delayed message to check for compliance with the nodes safety tolerance strategy.
//
// Return values:
//
//	bool - representing if the delayed message is safe to add to this nodes internal state
//	error - any error that occurrs as a result of a different function call is propegated to the caller.
//
// Semantics:
//
//	If the boolean return value is true, error will always be nil. if err is populated, the boolean will always be false.
//	Any error calls to other functions makes it impossible to determine the safety of the message without retrying, therefore
//	the caller **MUST** assume error being non nil means that the message is unsafe to add to the nodes state.
func (f *DelayedMessageFetcher) isDelayedMessageWithinSafetyTolerance(message *espressostreamer.MessageWithMetadataAndPos) (bool, error) {
	var safeBlockNumber uint64
	if f.waitForFinalization {
		// if we have configured to wait for finalizations, fetch the latest finalized block number.
		blockNumber, err := f.l1Reader.LatestFinalizedBlockNr(context.Background())
		safeBlockNumber = blockNumber
		if err != nil {
			log.Warn("Error getting finalized block header to check safety tolerance of delayed message", "err", err)
			return false, err
		}

	} else if f.waitForConfirmations {
		// if we are waiting for block confirmations, get the latest header and subtract the required block depth.
		latestHeader, err := f.l1Reader.Client().HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Warn("Error getting finalized block header to check safety tolerance of delayed message", "err", err)
			return false, err
		}
		safeBlockNumber = latestHeader.Number.Sub(latestHeader.Number, new(big.Int).SetUint64(f.requiredBlockDepth)).Uint64()

	} else {
		// If we haven't configured a safety strategy, every delayed message is valid to include in the nodes state.
		log.Debug("No safety strategy configured, every delayed message is valid")
		return true, nil
	}

	// safeBlockNumber will be popluated from here onwards. Any code paths that don't set the variable, return from the function before they get here.
	delayCount, err := f.getDelayedMessageCountAtBlock(safeBlockNumber)
	if err != nil {
		log.Warn("Error getting the delayed message count while checking the delayed messages safety tolerance", "err", err)
		return false, err
	}
	if (message.MessageWithMeta.Message.Header.BlockNumber <= safeBlockNumber) && (message.MessageWithMeta.DelayedMessagesRead <= delayCount) {
		return true, nil
	}
	return false, nil

}

/*
Store the delayed message in the database
*/
func (f *DelayedMessageFetcher) storeDelayedMessage(batch ethdb.Batch, seqNum uint64, msg DelayedInboxMessage) error {
	key := dbKey(DelayedMessagePrefix, seqNum)
	encodedMsg, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return fmt.Errorf("failed to encode delayed message: %w", err)
	}
	return batch.Put(key, encodedMsg)
}

/*
Reads the delayed message from the database
*/
func (f *DelayedMessageFetcher) readDelayedMessage(seqNum uint64) (*DelayedInboxMessage, error) {
	key := dbKey(DelayedMessagePrefix, seqNum)
	encodedMsg, err := f.db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get delayed message: %w", err)
	}
	var msg DelayedInboxMessage
	err = rlp.DecodeBytes(encodedMsg, &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode delayed message: %w", err)
	}
	return &msg, nil
}

/*
Stores the current L1 block in the database.
*/
func storeCurrentL1Block(batch ethdb.Batch, fromBlock uint64) error {
	blockNumberBytes, err := rlp.EncodeToBytes(fromBlock)
	if err != nil {
		return fmt.Errorf("failed to encode next hotshot block: %w", err)
	}

	err = batch.Put([]byte(DelayedFetcherCurrentL1BlockKey), blockNumberBytes)
	if err != nil {
		return fmt.Errorf("failed to put next hotshot block: %w", err)
	}

	return nil
}

/*
Reads the current L1 block from the database.
*/
func readCurrentL1BlockFromDb(db ethdb.Database) (uint64, error) {
	var blockNumber uint64
	blockNumberBytes, err := db.Get([]byte(DelayedFetcherCurrentL1BlockKey))
	if err != nil && !dbutil.IsErrNotFound(err) {
		return 0, fmt.Errorf("failed to get next hotshot block: %w", err)
	}
	if blockNumberBytes != nil {
		err = rlp.DecodeBytes(blockNumberBytes, &blockNumber)
		if err != nil {
			return 0, fmt.Errorf("failed to decode next hotshot block: %w", err)
		}
	}

	return blockNumber, nil
}

func readDelayedMessageCount(db ethdb.Database) (uint64, error) {
	var delayedCount uint64
	delayedCountBytes, err := db.Get([]byte(DelayedMessageCountKey))
	if err != nil {
		return 0, fmt.Errorf("failed to get delayed message count: %w", err)
	}
	err = rlp.DecodeBytes(delayedCountBytes, &delayedCount)
	if err != nil {
		return 0, fmt.Errorf("failed to decode delayed message count: %w", err)
	}
	return delayedCount, nil
}

func storeDelayedMessageCount(db ethdb.Database, count uint64) error {
	countBytes, err := rlp.EncodeToBytes(count)
	if err != nil {
		return fmt.Errorf("failed to encode delayed message count: %w", err)
	}
	return db.Put([]byte(DelayedMessageCountKey), countBytes)
}
