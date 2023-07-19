// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type InboxReaderConfig struct {
	DelayBlocks         uint64        `koanf:"delay-blocks" reload:"hot"`
	CheckDelay          time.Duration `koanf:"check-delay" reload:"hot"`
	HardReorg           bool          `koanf:"hard-reorg" reload:"hot"`
	MinBlocksToRead     uint64        `koanf:"min-blocks-to-read" reload:"hot"`
	DefaultBlocksToRead uint64        `koanf:"default-blocks-to-read" reload:"hot"`
	TargetMessagesRead  uint64        `koanf:"target-messages-read" reload:"hot"`
	MaxBlocksToRead     uint64        `koanf:"max-blocks-to-read" reload:"hot"`
}

type InboxReaderConfigFetcher func() *InboxReaderConfig

func (c *InboxReaderConfig) Validate() error {
	if c.MaxBlocksToRead == 0 || c.MaxBlocksToRead < c.DefaultBlocksToRead {
		return errors.New("inbox reader max-blocks-to-read cannot be zero or less than default-blocks-to-read")
	}
	return nil
}

func InboxReaderConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".delay-blocks", DefaultInboxReaderConfig.DelayBlocks, "number of latest blocks to ignore to reduce reorgs")
	f.Duration(prefix+".check-delay", DefaultInboxReaderConfig.CheckDelay, "the maximum time to wait between inbox checks (if not enough new blocks are found)")
	f.Bool(prefix+".hard-reorg", DefaultInboxReaderConfig.HardReorg, "erase future transactions in addition to overwriting existing ones on reorg")
	f.Uint64(prefix+".min-blocks-to-read", DefaultInboxReaderConfig.MinBlocksToRead, "the minimum number of blocks to read at once (when caught up lowers load on L1)")
	f.Uint64(prefix+".default-blocks-to-read", DefaultInboxReaderConfig.DefaultBlocksToRead, "the default number of blocks to read at once (will vary based on traffic by default)")
	f.Uint64(prefix+".target-messages-read", DefaultInboxReaderConfig.TargetMessagesRead, "if adjust-blocks-to-read is enabled, the target number of messages to read at once")
	f.Uint64(prefix+".max-blocks-to-read", DefaultInboxReaderConfig.MaxBlocksToRead, "if adjust-blocks-to-read is enabled, the maximum number of blocks to read at once")
}

var DefaultInboxReaderConfig = InboxReaderConfig{
	DelayBlocks:         0,
	CheckDelay:          time.Minute,
	HardReorg:           false,
	MinBlocksToRead:     1,
	DefaultBlocksToRead: 100,
	TargetMessagesRead:  500,
	MaxBlocksToRead:     2000,
}

var TestInboxReaderConfig = InboxReaderConfig{
	DelayBlocks:         0,
	CheckDelay:          time.Millisecond * 10,
	HardReorg:           false,
	MinBlocksToRead:     1,
	DefaultBlocksToRead: 100,
	TargetMessagesRead:  500,
	MaxBlocksToRead:     2000,
}

type InboxReader struct {
	stopwaiter.StopWaiter

	// Only in run thread
	caughtUp          bool
	firstMessageBlock *big.Int
	config            InboxReaderConfigFetcher

	// Thread safe
	tracker        *InboxTracker
	delayedBridge  *DelayedBridge
	sequencerInbox *SequencerInbox
	caughtUpChan   chan struct{}
	client         arbutil.L1Interface
	l1Reader       *headerreader.HeaderReader

	// Atomic
	lastSeenBatchCount uint64

	// Behind the mutex
	lastReadMutex      sync.RWMutex
	lastReadBlock      uint64
	lastReadBatchCount uint64
}

func NewInboxReader(tracker *InboxTracker, client arbutil.L1Interface, l1Reader *headerreader.HeaderReader, firstMessageBlock *big.Int, delayedBridge *DelayedBridge, sequencerInbox *SequencerInbox, config InboxReaderConfigFetcher) (*InboxReader, error) {
	err := config().Validate()
	if err != nil {
		return nil, err
	}
	return &InboxReader{
		tracker:           tracker,
		delayedBridge:     delayedBridge,
		sequencerInbox:    sequencerInbox,
		client:            client,
		l1Reader:          l1Reader,
		firstMessageBlock: firstMessageBlock,
		caughtUpChan:      make(chan struct{}),
		config:            config,
	}, nil
}

func (r *InboxReader) Start(ctxIn context.Context) error {
	r.StopWaiter.Start(ctxIn, r)
	hadError := false
	r.CallIteratively(func(ctx context.Context) time.Duration {
		err := r.run(ctx, hadError)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "header not found") {
			log.Warn("error reading inbox", "err", err)
			hadError = true
		} else {
			hadError = false
		}
		return time.Second
	})

	// Ensure we read the init message before other things start up
	for i := 0; ; i++ {
		batchCount, err := r.tracker.GetBatchCount()
		if err != nil {
			return err
		}
		if batchCount > 0 {
			// Validate the init message matches our L2 blockchain
			message, err := r.tracker.GetDelayedMessage(0)
			if err != nil {
				return err
			}
			initMessage, err := message.ParseInitMessage()
			if err != nil {
				return err
			}
			chainConfig := r.tracker.txStreamer.chainConfig
			configChainId := chainConfig.ChainID
			if initMessage.ChainId.Cmp(configChainId) != 0 {
				return fmt.Errorf("expected L2 chain ID %v but read L2 chain ID %v from init message in L1 inbox", configChainId, initMessage.ChainId)
			}
			if initMessage.ChainConfig != nil {
				if err := initMessage.ChainConfig.CheckCompatible(chainConfig, chainConfig.ArbitrumChainParams.GenesisBlockNum, 0); err != nil {
					return fmt.Errorf("incompatible chain config read from init message in L1 inbox: %w", err)
				}
			}
			break
		}
		if i == 30*10 {
			return errors.New("failed to read init message")
		}
		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

// assumes l1block is recent so we could do a simple-search from the end
func (r *InboxReader) recentParentChainBlockToMsg(ctx context.Context, parentChainBlock uint64) (arbutil.MessageIndex, error) {
	batch, err := r.tracker.GetBatchCount()
	if err != nil {
		return 0, err
	}
	for {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if batch == 0 {
			return 0, nil
		}
		batch -= 1
		meta, err := r.tracker.GetBatchMetadata(batch)
		if err != nil {
			return 0, err
		}
		if meta.ParentChainBlock <= parentChainBlock {
			return meta.MessageCount, nil
		}
	}
}

func (r *InboxReader) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	l1block, err := r.l1Reader.LatestSafeBlockNr(ctx)
	if err != nil {
		return 0, err
	}
	return r.recentParentChainBlockToMsg(ctx, l1block)
}

func (r *InboxReader) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	l1block, err := r.l1Reader.LatestFinalizedBlockNr(ctx)
	if err != nil {
		return 0, err
	}
	return r.recentParentChainBlockToMsg(ctx, l1block)
}

func (r *InboxReader) Tracker() *InboxTracker {
	return r.tracker
}

func (r *InboxReader) DelayedBridge() *DelayedBridge {
	return r.delayedBridge
}

func (r *InboxReader) CaughtUp() chan struct{} {
	return r.caughtUpChan
}

func (r *InboxReader) run(ctx context.Context, hadError bool) error {
	from, err := r.getNextBlockToRead()
	if err != nil {
		return err
	}
	newHeaders, unsubscribe := r.l1Reader.Subscribe(false)
	defer unsubscribe()
	blocksToFetch := r.config().DefaultBlocksToRead
	if hadError {
		blocksToFetch = 1
	}
	seenBatchCount := uint64(0)
	seenBatchCountStored := uint64(math.MaxUint64)
	storeSeenBatchCount := func() {
		if seenBatchCountStored != seenBatchCount {
			atomic.StoreUint64(&r.lastSeenBatchCount, seenBatchCount)
			seenBatchCountStored = seenBatchCount
		}
	}
	defer storeSeenBatchCount() // in case of error
	for {

		latestHeader, err := r.l1Reader.LastHeader(ctx)
		if err != nil {
			return err
		}
		config := r.config()
		currentHeight := latestHeader.Number

		neededBlockAdvance := config.DelayBlocks + arbmath.SaturatingUSub(config.MinBlocksToRead, 1)
		neededBlockHeight := arbmath.BigAddByUint(from, neededBlockAdvance)
		checkDelayTimer := time.NewTimer(config.CheckDelay)
	WaitForHeight:
		for arbmath.BigLessThan(currentHeight, neededBlockHeight) {
			select {
			case latestHeader = <-newHeaders:
				if latestHeader == nil {
					// shutting down
					return nil
				}
				currentHeight = new(big.Int).Set(latestHeader.Number)
			case <-ctx.Done():
				return nil
			case <-checkDelayTimer.C:
				break WaitForHeight
			}
		}
		checkDelayTimer.Stop()

		if config.DelayBlocks > 0 {
			currentHeight = new(big.Int).Sub(currentHeight, new(big.Int).SetUint64(config.DelayBlocks))
			if currentHeight.Cmp(r.firstMessageBlock) < 0 {
				currentHeight = new(big.Int).Set(r.firstMessageBlock)
			}
		}

		reorgingDelayed := false
		reorgingSequencer := false
		missingDelayed := false
		missingSequencer := false

		{
			checkingDelayedCount, err := r.delayedBridge.GetMessageCount(ctx, currentHeight)
			if err != nil {
				return err
			}
			ourLatestDelayedCount, err := r.tracker.GetDelayedCount()
			if err != nil {
				return err
			}
			if ourLatestDelayedCount < checkingDelayedCount {
				checkingDelayedCount = ourLatestDelayedCount
				missingDelayed = true
			} else if ourLatestDelayedCount > checkingDelayedCount {
				log.Info("backwards reorg of delayed messages", "from", ourLatestDelayedCount, "to", checkingDelayedCount)
				err = r.tracker.ReorgDelayedTo(checkingDelayedCount, config.HardReorg)
				if err != nil {
					return err
				}
			}
			if checkingDelayedCount > 0 {
				checkingDelayedSeqNum := checkingDelayedCount - 1
				l1DelayedAcc, err := r.delayedBridge.GetAccumulator(ctx, checkingDelayedSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbDelayedAcc, err := r.tracker.GetDelayedAcc(checkingDelayedSeqNum)
				if err != nil {
					return err
				}
				if dbDelayedAcc != l1DelayedAcc {
					reorgingDelayed = true
				}
			}
		}

		seenBatchCount, err = r.sequencerInbox.GetBatchCount(ctx, currentHeight)
		if err != nil {
			seenBatchCount = 0
			return err
		}
		checkingBatchCount := seenBatchCount
		{
			ourLatestBatchCount, err := r.tracker.GetBatchCount()
			if err != nil {
				return err
			}
			if ourLatestBatchCount < checkingBatchCount {
				checkingBatchCount = ourLatestBatchCount
				missingSequencer = true
			} else if ourLatestBatchCount > checkingBatchCount && config.HardReorg {
				err = r.tracker.ReorgBatchesTo(checkingBatchCount)
				if err != nil {
					return err
				}
			}
			if checkingBatchCount > 0 {
				checkingBatchSeqNum := checkingBatchCount - 1
				l1BatchAcc, err := r.sequencerInbox.GetAccumulator(ctx, checkingBatchSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbBatchAcc, err := r.tracker.GetBatchAcc(checkingBatchSeqNum)
				if err != nil {
					return err
				}
				if dbBatchAcc != l1BatchAcc {
					reorgingSequencer = true
				}
			}
		}

		if !missingDelayed && !reorgingDelayed && !missingSequencer && !reorgingSequencer {
			// There's nothing to do
			from = arbmath.BigAddByUint(currentHeight, 1)
			blocksToFetch = config.DefaultBlocksToRead
			r.lastReadMutex.Lock()
			r.lastReadBlock = currentHeight.Uint64()
			r.lastReadBatchCount = checkingBatchCount
			r.lastReadMutex.Unlock()
			storeSeenBatchCount()
			if !r.caughtUp {
				r.caughtUp = true
				close(r.caughtUpChan)
			}
			continue
		}

		readAnyBatches := false
		for {
			if ctx.Err() != nil {
				// the context is done, shut down
				// nolint:nilerr
				return nil
			}
			if from.Cmp(currentHeight) > 0 {
				if missingDelayed {
					reorgingDelayed = true
				}
				if missingSequencer {
					reorgingSequencer = true
				}
				if !reorgingDelayed && !reorgingSequencer {
					break
				} else {
					from = new(big.Int).Set(currentHeight)
				}
			}
			to := new(big.Int).Add(from, new(big.Int).SetUint64(blocksToFetch))
			if to.Cmp(currentHeight) > 0 {
				to.Set(currentHeight)
			}
			sequencerBatches, err := r.sequencerInbox.LookupBatchesInRange(ctx, from, to)
			if err != nil {
				return err
			}
			delayedMessages, err := r.delayedBridge.LookupMessagesInRange(ctx, from, to, func(batchNum uint64) ([]byte, error) {
				if len(sequencerBatches) > 0 && batchNum >= sequencerBatches[0].SequenceNumber {
					idx := int(batchNum - sequencerBatches[0].SequenceNumber)
					if idx < len(sequencerBatches) {
						return sequencerBatches[idx].Serialize(ctx, r.l1Reader.Client())
					}
					log.Warn("missing mentioned batch in L1 message lookup", "batch", batchNum)
				}
				return r.GetSequencerMessageBytes(ctx, batchNum)
			})
			if err != nil {
				return err
			}
			if !r.caughtUp && to.Cmp(currentHeight) == 0 {
				r.caughtUp = true
				close(r.caughtUpChan)
			}
			if len(sequencerBatches) > 0 {
				missingSequencer = false
				reorgingSequencer = false
				firstBatch := sequencerBatches[0]
				if firstBatch.SequenceNumber > 0 {
					haveAcc, err := r.tracker.GetBatchAcc(firstBatch.SequenceNumber - 1)
					if errors.Is(err, AccumulatorNotFoundErr) {
						reorgingSequencer = true
					} else if err != nil {
						return err
					} else if haveAcc != firstBatch.BeforeInboxAcc {
						reorgingSequencer = true
					}
				}
				if !reorgingSequencer {
					// Skip any batches we already have in the database
					for len(sequencerBatches) > 0 {
						batch := sequencerBatches[0]
						haveAcc, err := r.tracker.GetBatchAcc(batch.SequenceNumber)
						if errors.Is(err, AccumulatorNotFoundErr) {
							// This batch is new
							break
						} else if err != nil {
							// Unknown error (database error?)
							return err
						} else if haveAcc == batch.BeforeInboxAcc {
							// Skip this batch, as we already have it in the database
							sequencerBatches = sequencerBatches[1:]
						} else {
							// The first batch BeforeInboxAcc matches, but this batch doesn't,
							// so we'll successfully reorg it when we hit the addMessages
							break
						}
					}
				}
			} else if missingSequencer && to.Cmp(currentHeight) >= 0 {
				// We were missing sequencer batches but didn't find any.
				// This must mean that the sequencer batches are in the past.
				reorgingSequencer = true
			}

			if len(delayedMessages) > 0 {
				missingDelayed = false
				reorgingDelayed = false
				firstMsg := delayedMessages[0]
				beforeAcc := firstMsg.BeforeInboxAcc
				beforeCount, err := firstMsg.Message.Header.SeqNum()
				if err != nil {
					return err
				}
				if beforeCount > 0 {
					haveAcc, err := r.tracker.GetDelayedAcc(beforeCount - 1)
					if errors.Is(err, AccumulatorNotFoundErr) {
						reorgingDelayed = true
					} else if err != nil {
						return err
					} else if haveAcc != beforeAcc {
						reorgingDelayed = true
					}
				}
			} else if missingDelayed && to.Cmp(currentHeight) >= 0 {
				// We were missing delayed messages but didn't find any.
				// This must mean that the delayed messages are in the past.
				reorgingDelayed = true
			}

			log.Trace("looking up messages", "from", from.String(), "to", to.String(), "missingDelayed", missingDelayed, "missingSequencer", missingSequencer, "reorgingDelayed", reorgingDelayed, "reorgingSequencer", reorgingSequencer)
			if !reorgingDelayed && !reorgingSequencer && (len(delayedMessages) != 0 || len(sequencerBatches) != 0) {
				delayedMismatch, err := r.addMessages(ctx, sequencerBatches, delayedMessages)
				if err != nil {
					return err
				}
				if delayedMismatch {
					reorgingDelayed = true
				}
				if len(sequencerBatches) > 0 {
					readAnyBatches = true
					r.lastReadMutex.Lock()
					r.lastReadBlock = to.Uint64()
					r.lastReadBatchCount = sequencerBatches[len(sequencerBatches)-1].SequenceNumber + 1
					r.lastReadMutex.Unlock()
					storeSeenBatchCount()
				}
			}
			if reorgingDelayed || reorgingSequencer {
				from, err = r.getPrevBlockForReorg(from)
				if err != nil {
					return err
				}
			} else {
				from = arbmath.BigAddByUint(to, 1)
			}
			haveMessages := uint64(len(delayedMessages) + len(sequencerBatches))
			if haveMessages <= (config.TargetMessagesRead / 2) {
				blocksToFetch += (blocksToFetch + 4) / 5
			} else if haveMessages >= (config.TargetMessagesRead * 3 / 2) {
				// This cannot overflow, as it'll never try to subtract more than blocksToFetch
				blocksToFetch -= (blocksToFetch + 4) / 5
			}
			if blocksToFetch < 1 {
				blocksToFetch = 1
			} else if blocksToFetch > config.MaxBlocksToRead {
				blocksToFetch = config.MaxBlocksToRead
			}
		}

		if !readAnyBatches {
			r.lastReadMutex.Lock()
			r.lastReadBlock = currentHeight.Uint64()
			r.lastReadBatchCount = checkingBatchCount
			r.lastReadMutex.Unlock()
			storeSeenBatchCount()
		}
	}
}

func (r *InboxReader) addMessages(ctx context.Context, sequencerBatches []*SequencerInboxBatch, delayedMessages []*DelayedInboxMessage) (bool, error) {
	err := r.tracker.AddDelayedMessages(delayedMessages, r.config().HardReorg)
	if err != nil {
		return false, err
	}
	err = r.tracker.AddSequencerBatches(ctx, r.client, sequencerBatches)
	if errors.Is(err, delayedMessagesMismatch) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
}

func (r *InboxReader) getPrevBlockForReorg(from *big.Int) (*big.Int, error) {
	if from.Cmp(r.firstMessageBlock) <= 0 {
		return nil, errors.New("can't get older messages")
	}
	newFrom := arbmath.BigSub(from, big.NewInt(10))
	if newFrom.Cmp(r.firstMessageBlock) < 0 {
		newFrom = new(big.Int).Set(r.firstMessageBlock)
	}
	return newFrom, nil
}

func (r *InboxReader) getNextBlockToRead() (*big.Int, error) {
	delayedCount, err := r.tracker.GetDelayedCount()
	if err != nil {
		return nil, err
	}
	if delayedCount == 0 {
		return new(big.Int).Set(r.firstMessageBlock), nil
	}
	_, _, parentChainBlockNumber, err := r.tracker.GetDelayedMessageAccumulatorAndParentChainBlockNumber(delayedCount - 1)
	if err != nil {
		return nil, err
	}
	msgBlock := new(big.Int).SetUint64(parentChainBlockNumber)
	if arbmath.BigLessThan(msgBlock, r.firstMessageBlock) {
		msgBlock.Set(r.firstMessageBlock)
	}
	return msgBlock, nil
}

func (r *InboxReader) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	metadata, err := r.tracker.GetBatchMetadata(seqNum)
	if err != nil {
		return nil, err
	}
	blockNum := arbmath.UintToBig(metadata.ParentChainBlock)
	seqBatches, err := r.sequencerInbox.LookupBatchesInRange(ctx, blockNum, blockNum)
	if err != nil {
		return nil, err
	}
	var seenBatches []uint64
	for _, batch := range seqBatches {
		if batch.SequenceNumber == seqNum {
			return batch.Serialize(ctx, r.client)
		}
		seenBatches = append(seenBatches, batch.SequenceNumber)
	}
	return nil, fmt.Errorf("sequencer batch %v not found in L1 block %v (found batches %v)", seqNum, metadata.ParentChainBlock, seenBatches)
}

func (r *InboxReader) GetLastReadBlockAndBatchCount() (uint64, uint64) {
	r.lastReadMutex.RLock()
	defer r.lastReadMutex.RUnlock()
	return r.lastReadBlock, r.lastReadBatchCount
}

// GetLastSeenBatchCount returns how many sequencer batches the inbox reader has read in from L1.
// Return values:
// >0 - last batchcount seen in run() - only written after lastReadBatchCount updated
// 0 - no batch seen, error
func (r *InboxReader) GetLastSeenBatchCount() uint64 {
	return atomic.LoadUint64(&r.lastSeenBatchCount)
}

func (r *InboxReader) GetDelayBlocks() uint64 {
	return r.config().DelayBlocks
}
