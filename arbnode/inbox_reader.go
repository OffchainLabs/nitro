//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type InboxReaderConfig struct {
	DelayBlocks int64
	CheckDelay  time.Duration
	HardReorg   bool // erase future transactions in addition to overwriting existing ones
}

var DefaultInboxReaderConfig = InboxReaderConfig{
	DelayBlocks: 4,
	CheckDelay:  2 * time.Second,
	HardReorg:   true,
}

var TestInboxReaderConfig = InboxReaderConfig{
	DelayBlocks: 0,
	CheckDelay:  time.Millisecond * 10,
	HardReorg:   true,
}

type InboxReader struct {
	// Only in run thread
	caughtUp          bool
	firstMessageBlock *big.Int
	config            *InboxReaderConfig

	// Thread safe
	tracker        *InboxTracker
	delayedBridge  *DelayedBridge
	sequencerInbox *SequencerInbox
	caughtUpChan   chan bool
	client         L1Interface
}

func NewInboxReader(rawDb ethdb.Database, txStreamer *TransactionStreamer, client L1Interface, firstMessageBlock *big.Int, delayedBridge *DelayedBridge, sequencerInbox *SequencerInbox, config *InboxReaderConfig) (*InboxReader, error) {
	tracker, err := NewInboxTracker(rawDb, txStreamer)
	if err != nil {
		return nil, err
	}
	return &InboxReader{
		tracker:           tracker,
		delayedBridge:     delayedBridge,
		sequencerInbox:    sequencerInbox,
		client:            client,
		firstMessageBlock: firstMessageBlock,
		caughtUpChan:      make(chan bool, 1),
		config:            config,
	}, nil
}

func (r *InboxReader) Start(ctx context.Context) {
	go (func() {
		for {
			err := r.run(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("error reading inbox", "err", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	})()
}

func (r *InboxReader) Tracker() *InboxTracker {
	return r.tracker
}

func (r *InboxReader) DelayedBridge() *DelayedBridge {
	return r.delayedBridge
}

func (ir *InboxReader) run(ctx context.Context) error {
	from, err := ir.getNextBlockToRead()
	if err != nil {
		return err
	}
	blocksToFetch := uint64(100)
	for {
		currentHeightRaw, err := ir.client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		currentHeight := new(big.Int).SetUint64(currentHeightRaw)

		if ir.config.DelayBlocks > 0 {
			currentHeight = new(big.Int).Sub(currentHeight, big.NewInt(ir.config.DelayBlocks))
			if currentHeight.Cmp(ir.firstMessageBlock) < 0 {
				currentHeight = new(big.Int).Set(ir.firstMessageBlock)
			}
		}

		reorgingDelayed := false
		reorgingSequencer := false
		missingDelayed := false
		missingSequencer := false

		{
			checkingDelayedCount, err := ir.delayedBridge.GetMessageCount(ctx, currentHeight)
			if err != nil {
				return err
			}
			ourLatestDelayedCount, err := ir.tracker.GetDelayedCount()
			if err != nil {
				return err
			}
			if ourLatestDelayedCount < checkingDelayedCount {
				checkingDelayedCount = ourLatestDelayedCount
				missingDelayed = true
			} else if ourLatestDelayedCount > checkingDelayedCount && ir.config.HardReorg {
				log.Info("backwards reorg of delayed messages", "from", ourLatestDelayedCount, "to", checkingDelayedCount)
				err = ir.tracker.ReorgDelayedTo(checkingDelayedCount)
				if err != nil {
					return err
				}
			}
			if checkingDelayedCount > 0 {
				checkingDelayedSeqNum := checkingDelayedCount - 1
				l1DelayedAcc, err := ir.delayedBridge.GetAccumulator(ctx, checkingDelayedSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbDelayedAcc, err := ir.tracker.GetDelayedAcc(checkingDelayedSeqNum)
				if err != nil {
					return err
				}
				if dbDelayedAcc != l1DelayedAcc {
					reorgingDelayed = true
				}
			}
		}

		{
			checkingBatchCount, err := ir.sequencerInbox.GetBatchCount(ctx, currentHeight)
			if err != nil {
				return err
			}
			ourLatestBatchCount, err := ir.tracker.GetBatchCount()
			if err != nil {
				return err
			}
			if ourLatestBatchCount < checkingBatchCount {
				checkingBatchCount = ourLatestBatchCount
				missingSequencer = true
			} else if ourLatestBatchCount > checkingBatchCount && ir.config.HardReorg {
				err = ir.tracker.ReorgBatchesTo(checkingBatchCount)
				if err != nil {
					return err
				}
			}
			if checkingBatchCount > 0 {
				checkingBatchSeqNum := checkingBatchCount - 1
				l1BatchAcc, err := ir.sequencerInbox.GetAccumulator(ctx, checkingBatchSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbBatchAcc, err := ir.tracker.GetBatchAcc(checkingBatchSeqNum)
				if err != nil {
					return err
				}
				if dbBatchAcc != l1BatchAcc {
					reorgingSequencer = true
				}
			}
		}

		for {
			if ctx.Err() != nil {
				// the context is done, shut down
				// nolint:nilerr
				return nil
			}
			if from.Cmp(currentHeight) >= 0 {
				if missingDelayed {
					reorgingDelayed = true
				}
				if missingSequencer {
					reorgingSequencer = true
				}
				if !reorgingDelayed && !reorgingSequencer {
					break
				} else {
					from = currentHeight
				}
			}
			to := new(big.Int).Add(from, new(big.Int).SetUint64(blocksToFetch))
			if to.Cmp(currentHeight) > 0 {
				to = currentHeight
			}
			var delayedMessages []*DelayedInboxMessage
			delayedMessages, err := ir.delayedBridge.LookupMessagesInRange(ctx, from, to)
			if err != nil {
				return err
			}
			sequencerBatches, err := ir.sequencerInbox.LookupBatchesInRange(ctx, from, to)
			if err != nil {
				return err
			}
			if !ir.caughtUp && to.Cmp(currentHeight) == 0 {
				// TODO better caught up tracking
				ir.caughtUp = true
				ir.caughtUpChan <- true
			}
			if len(sequencerBatches) > 0 {
				missingSequencer = false
				reorgingSequencer = false
				firstBatch := sequencerBatches[0]
				if firstBatch.SequenceNumber > 0 {
					haveAcc, err := ir.tracker.GetBatchAcc(firstBatch.SequenceNumber - 1)
					if errors.Is(err, accumulatorNotFound) {
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
						haveAcc, err := ir.tracker.GetBatchAcc(batch.SequenceNumber)
						if errors.Is(err, accumulatorNotFound) {
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
					haveAcc, err := ir.tracker.GetDelayedAcc(beforeCount - 1)
					if errors.Is(err, accumulatorNotFound) {
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

			log.Trace("looking up messages", "from", from.String(), "to", to.String())
			if !reorgingDelayed && !reorgingSequencer && (len(delayedMessages) != 0 || len(sequencerBatches) != 0) {
				delayedMismatch, err := ir.addMessages(ctx, sequencerBatches, delayedMessages)
				if err != nil {
					return err
				}
				if delayedMismatch {
					reorgingDelayed = true
				}
			}
			if reorgingDelayed || reorgingSequencer {
				from, err = ir.getPrevBlockForReorg(from)
				if err != nil {
					return err
				}
			} else {
				delta := new(big.Int).SetUint64(blocksToFetch)
				if new(big.Int).Add(to, delta).Cmp(currentHeight) >= 0 {
					delta = delta.Div(delta, big.NewInt(2))
					from = from.Add(from, delta)
					if from.Cmp(to) > 0 {
						from = from.Set(to)
					}
				} else {
					from = from.Add(to, big.NewInt(1))
				}
			}
		}
		// TODO feed reading
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(ir.config.CheckDelay):
		}
	}
}

func (r *InboxReader) addMessages(ctx context.Context, sequencerBatches []*SequencerInboxBatch, delayedMessages []*DelayedInboxMessage) (bool, error) {
	err := r.tracker.addDelayedMessages(delayedMessages)
	if err != nil {
		return false, err
	}
	err = r.tracker.addSequencerBatches(ctx, r.client, sequencerBatches)
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
	newFrom := new(big.Int).Sub(from, big.NewInt(10))
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
	msg, err := r.tracker.GetDelayedMessage(delayedCount - 1)
	if err != nil {
		return nil, err
	}
	return msg.Header.RequestId.Big(), nil
}
