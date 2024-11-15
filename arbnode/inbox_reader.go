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
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

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
	ReadMode            string        `koanf:"read-mode" reload:"hot"`
}

type InboxReaderConfigFetcher func() *InboxReaderConfig

func (c *InboxReaderConfig) Validate() error {
	if c.MaxBlocksToRead == 0 || c.MaxBlocksToRead < c.DefaultBlocksToRead {
		return errors.New("inbox reader max-blocks-to-read cannot be zero or less than default-blocks-to-read")
	}
	c.ReadMode = strings.ToLower(c.ReadMode)
	if c.ReadMode != "latest" && c.ReadMode != "safe" && c.ReadMode != "finalized" {
		return fmt.Errorf("inbox reader read-mode is invalid, want: latest or safe or finalized, got: %s", c.ReadMode)
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
	f.String(prefix+".read-mode", DefaultInboxReaderConfig.ReadMode, "mode to only read latest or safe or finalized L1 blocks. Enabling safe or finalized disables feed input and output. Defaults to latest. Takes string input, valid strings- latest, safe, finalized")
}

var DefaultInboxReaderConfig = InboxReaderConfig{
	DelayBlocks:         0,
	CheckDelay:          time.Minute,
	HardReorg:           false,
	MinBlocksToRead:     1,
	DefaultBlocksToRead: 100,
	TargetMessagesRead:  500,
	MaxBlocksToRead:     2000,
	ReadMode:            "latest",
}

var TestInboxReaderConfig = InboxReaderConfig{
	DelayBlocks:         0,
	CheckDelay:          time.Millisecond * 10,
	HardReorg:           false,
	MinBlocksToRead:     1,
	DefaultBlocksToRead: 100,
	TargetMessagesRead:  500,
	MaxBlocksToRead:     2000,
	ReadMode:            "latest",
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
	client         *ethclient.Client
	l1Reader       *headerreader.HeaderReader

	// Atomic
	lastSeenBatchCount atomic.Uint64
	lastReadBatchCount atomic.Uint64
}

func NewInboxReader(tracker *InboxTracker, client *ethclient.Client, l1Reader *headerreader.HeaderReader, firstMessageBlock *big.Int, delayedBridge *DelayedBridge, sequencerInbox *SequencerInbox, config InboxReaderConfigFetcher) (*InboxReader, error) {
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
			if r.tracker.snapSyncConfig.Enabled {
				break
			}
			// Validate the init message matches our L2 blockchain
			ctx, err := r.StopWaiter.GetContextSafe()
			if err != nil {
				return err
			}
			message, err := r.tracker.GetDelayedMessage(ctx, 0)
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

type lazyHashLogging struct {
	f func() common.Hash
}

func (l lazyHashLogging) String() string {
	return l.f().String()
}

func (l lazyHashLogging) TerminalString() string {
	return l.f().TerminalString()
}

func (l lazyHashLogging) MarshalText() ([]byte, error) {
	return l.f().MarshalText()
}

func (l lazyHashLogging) Format(s fmt.State, c rune) {
	l.f().Format(s, c)
}

func (r *InboxReader) run(ctx context.Context, hadError bool) error {
	readMode := r.config().ReadMode
	from, err := r.getNextBlockToRead(ctx)
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
			r.lastSeenBatchCount.Store(seenBatchCount)
			seenBatchCountStored = seenBatchCount
		}
	}
	defer storeSeenBatchCount() // in case of error
	for {
		config := r.config()
		currentHeight := big.NewInt(0)
		if readMode != "latest" {
			var blockNum uint64
			fetchLatestSafeOrFinalized := func() {
				if readMode == "safe" {
					blockNum, err = r.l1Reader.LatestSafeBlockNr(ctx)
				} else {
					blockNum, err = r.l1Reader.LatestFinalizedBlockNr(ctx)
				}
			}
			fetchLatestSafeOrFinalized()
			if err != nil || blockNum == 0 {
				return fmt.Errorf("inboxreader running in read only %s mode and unable to fetch latest %s block. err: %w", readMode, readMode, err)
			}
			currentHeight.SetUint64(blockNum)
			// latest block in our db is newer than the latest safe/finalized block hence reset 'from' to match the last safe/finalized block number
			if from.Uint64() > currentHeight.Uint64()+1 {
				from.Set(currentHeight)
			}
			for currentHeight.Cmp(from) <= 0 {
				select {
				case <-newHeaders:
					fetchLatestSafeOrFinalized()
					if err != nil || blockNum == 0 {
						return fmt.Errorf("inboxreader waiting for recent %s block and unable to fetch its block number. err: %w", readMode, err)
					}
					currentHeight.SetUint64(blockNum)
				case <-ctx.Done():
					return nil
				}
			}
		} else {

			latestHeader, err := r.l1Reader.LastHeader(ctx)
			if err != nil {
				return err
			}
			currentHeight = latestHeader.Number

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
				log.Debug("Expecting to find delayed messages", "checkingDelayedCount", checkingDelayedCount, "ourLatestDelayedCount", ourLatestDelayedCount, "currentHeight", currentHeight)
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
				l1DelayedAcc, err := r.delayedBridge.GetAccumulator(ctx, checkingDelayedSeqNum, currentHeight, common.Hash{})
				if err != nil {
					return err
				}
				dbDelayedAcc, err := r.tracker.GetDelayedAcc(checkingDelayedSeqNum)
				if err != nil {
					return err
				}
				if dbDelayedAcc != l1DelayedAcc {
					log.Debug("Latest delayed accumulator mismatch", "delayedSeqNum", checkingDelayedSeqNum, "dbDelayedAcc", dbDelayedAcc, "l1DelayedAcc", l1DelayedAcc)
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
				log.Debug("Expecting to find sequencer batches", "checkingBatchCount", checkingBatchCount, "ourLatestBatchCount", ourLatestBatchCount, "currentHeight", currentHeight)
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
					log.Debug("Latest sequencer batch accumulator mismatch", "batchSeqNum", checkingBatchSeqNum, "dbBatchAcc", dbBatchAcc, "l1BatchAcc", l1BatchAcc)
					reorgingSequencer = true
				}
			}
		}

		if !missingDelayed && !reorgingDelayed && !missingSequencer && !reorgingSequencer {
			// There's nothing to do
			from = arbmath.BigAddByUint(currentHeight, 1)
			blocksToFetch = config.DefaultBlocksToRead
			r.lastReadBatchCount.Store(checkingBatchCount)
			storeSeenBatchCount()
			if !r.caughtUp && readMode == "latest" {
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
			log.Debug(
				"Looking up messages",
				"from", from.String(),
				"to", to.String(),
				"missingDelayed", missingDelayed,
				"missingSequencer", missingSequencer,
				"reorgingDelayed", reorgingDelayed,
				"reorgingSequencer", reorgingSequencer,
			)
			sequencerBatches, err := r.sequencerInbox.LookupBatchesInRange(ctx, from, to)
			if err != nil {
				return err
			}
			delayedMessages, err := r.delayedBridge.LookupMessagesInRange(ctx, from, to, func(batchNum uint64) ([]byte, error) {
				if len(sequencerBatches) > 0 && batchNum >= sequencerBatches[0].SequenceNumber {
					idx := batchNum - sequencerBatches[0].SequenceNumber
					if idx < uint64(len(sequencerBatches)) {
						return sequencerBatches[idx].Serialize(ctx, r.l1Reader.Client())
					}
					log.Warn("missing mentioned batch in L1 message lookup", "batch", batchNum)
				}
				data, _, err := r.GetSequencerMessageBytes(ctx, batchNum)
				return data, err
			})
			if err != nil {
				return err
			}
			if !r.caughtUp && to.Cmp(currentHeight) == 0 && readMode == "latest" {
				r.caughtUp = true
				close(r.caughtUpChan)
			}
			if len(sequencerBatches) > 0 {
				missingSequencer = false
				reorgingSequencer = false
				var havePrevAcc common.Hash
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
					havePrevAcc = haveAcc
				}
				readLastAcc := sequencerBatches[len(sequencerBatches)-1].AfterInboxAcc
				var duplicateBatches int
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
						} else if haveAcc == batch.AfterInboxAcc {
							// Skip this batch, as we already have it in the database
							sequencerBatches = sequencerBatches[1:]
							duplicateBatches++
						} else {
							// The first batch AfterInboxAcc matches, but this batch doesn't,
							// so we'll successfully reorg it when we hit the addMessages
							break
						}
					}
				}
				log.Debug(
					"Found sequencer batches",
					"firstSequenceNumber", firstBatch.SequenceNumber,
					"newBatchesCount", len(sequencerBatches),
					"duplicateBatches", duplicateBatches,
					"reorgingSequencer", reorgingSequencer,
					"readBeforeAcc", firstBatch.BeforeInboxAcc,
					"haveBeforeAcc", havePrevAcc,
					"readLastAcc", readLastAcc,
				)
			} else if missingSequencer && to.Cmp(currentHeight) >= 0 {
				log.Debug("Didn't find expected sequencer batches", "from", from, "to", to, "currentHeight", currentHeight)
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
				var havePrevAcc common.Hash
				if beforeCount > 0 {
					haveAcc, err := r.tracker.GetDelayedAcc(beforeCount - 1)
					if errors.Is(err, AccumulatorNotFoundErr) {
						reorgingDelayed = true
					} else if err != nil {
						return err
					} else if haveAcc != beforeAcc {
						reorgingDelayed = true
					}
					havePrevAcc = haveAcc
				}
				log.Debug(
					"Found delayed messages",
					"firstSequenceNumber", beforeCount,
					"count", len(delayedMessages),
					"reorgingDelayed", reorgingDelayed,
					"readBeforeAcc", beforeAcc,
					"haveBeforeAcc", havePrevAcc,
					"readLastAcc", lazyHashLogging{func() common.Hash {
						// Only compute this if we need to log it, as it's somewhat expensive
						return delayedMessages[len(delayedMessages)-1].AfterInboxAcc()
					}},
				)
			} else if missingDelayed && to.Cmp(currentHeight) >= 0 {
				log.Debug("Didn't find expected delayed messages", "from", from, "to", to, "currentHeight", currentHeight)
				// We were missing delayed messages but didn't find any.
				// This must mean that the delayed messages are in the past.
				reorgingDelayed = true
			}

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
					r.lastReadBatchCount.Store(sequencerBatches[len(sequencerBatches)-1].SequenceNumber + 1)
					storeSeenBatchCount()
				}
			}
			// #nosec G115
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
			if reorgingDelayed || reorgingSequencer {
				from, err = r.getPrevBlockForReorg(from, blocksToFetch)
				if err != nil {
					return err
				}
			} else {
				from = arbmath.BigAddByUint(to, 1)
			}
		}

		if !readAnyBatches {
			r.lastReadBatchCount.Store(checkingBatchCount)
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

func (r *InboxReader) getPrevBlockForReorg(from *big.Int, maxBlocksBackwards uint64) (*big.Int, error) {
	if from.Cmp(r.firstMessageBlock) <= 0 {
		return nil, errors.New("can't get older messages")
	}
	newFrom := arbmath.BigSub(from, new(big.Int).SetUint64(maxBlocksBackwards))
	if newFrom.Cmp(r.firstMessageBlock) < 0 {
		newFrom = new(big.Int).Set(r.firstMessageBlock)
	}
	return newFrom, nil
}

func (r *InboxReader) getNextBlockToRead(ctx context.Context) (*big.Int, error) {
	delayedCount, err := r.tracker.GetDelayedCount()
	if err != nil {
		return nil, err
	}
	if delayedCount == 0 {
		return new(big.Int).Set(r.firstMessageBlock), nil
	}
	_, _, parentChainBlockNumber, err := r.tracker.GetDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, delayedCount-1)
	if err != nil {
		return nil, err
	}
	msgBlock := new(big.Int).SetUint64(parentChainBlockNumber)
	if arbmath.BigLessThan(msgBlock, r.firstMessageBlock) {
		msgBlock.Set(r.firstMessageBlock)
	}
	return msgBlock, nil
}

func (r *InboxReader) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	metadata, err := r.tracker.GetBatchMetadata(seqNum)
	if err != nil {
		return nil, common.Hash{}, err
	}
	blockNum := arbmath.UintToBig(metadata.ParentChainBlock)
	seqBatches, err := r.sequencerInbox.LookupBatchesInRange(ctx, blockNum, blockNum)
	if err != nil {
		return nil, common.Hash{}, err
	}
	var seenBatches []uint64
	for _, batch := range seqBatches {
		if batch.SequenceNumber == seqNum {
			data, err := batch.Serialize(ctx, r.client)
			return data, batch.BlockHash, err
		}
		seenBatches = append(seenBatches, batch.SequenceNumber)
	}
	return nil, common.Hash{}, fmt.Errorf("sequencer batch %v not found in L1 block %v (found batches %v)", seqNum, metadata.ParentChainBlock, seenBatches)
}

func (r *InboxReader) GetLastReadBatchCount() uint64 {
	return r.lastReadBatchCount.Load()
}

// GetLastSeenBatchCount returns how many sequencer batches the inbox reader has read in from L1.
// Return values:
// >0 - last batchcount seen in run() - only written after lastReadBatchCount updated
// 0 - no batch seen, error
func (r *InboxReader) GetLastSeenBatchCount() uint64 {
	return r.lastSeenBatchCount.Load()
}

func (r *InboxReader) GetDelayBlocks() uint64 {
	return r.config().DelayBlocks
}
