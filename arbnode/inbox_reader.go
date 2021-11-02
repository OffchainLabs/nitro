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
}

var DefaultInboxReaderConfig = &InboxReaderConfig{
	DelayBlocks: 4,
	CheckDelay:  2 * time.Second,
}

type InboxReader struct {
	// Only in run thread
	caughtUp          bool
	firstMessageBlock *big.Int
	config            *InboxReaderConfig

	// Thread safe
	db             *InboxReaderDb
	delayedBridge  *DelayedBridge
	sequencerInbox *SequencerInbox
	caughtUpChan   chan bool
	client         L1Interface
}

func NewInboxReader(rawDb ethdb.Database, client L1Interface, firstMessageBlock *big.Int, delayedBridge *DelayedBridge, sequencerInbox *SequencerInbox, config *InboxReaderConfig) (*InboxReader, error) {
	db, err := NewInboxReaderDb(rawDb)
	if err != nil {
		return nil, err
	}
	return &InboxReader{
		db:                db,
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

func (r *InboxReader) Database() *InboxReaderDb {
	return r.db
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
		l1Header, err := ir.client.BlockByNumber(ctx, nil)
		if err != nil {
			return err
		}
		currentHeight := l1Header.Number()

		if ir.config.DelayBlocks > 0 {
			currentHeight = new(big.Int).Sub(currentHeight, big.NewInt(ir.config.DelayBlocks))
			if currentHeight.Sign() < 0 {
				currentHeight = currentHeight.SetInt64(0)
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
			ourLatestDelayedCount, err := ir.db.GetDelayedCount()
			if err != nil {
				return err
			}
			if ourLatestDelayedCount < checkingDelayedCount {
				checkingDelayedCount = ourLatestDelayedCount
				missingDelayed = true
			}
			if checkingDelayedCount > 0 {
				checkingDelayedSeqNum := checkingDelayedCount - 1
				l1DelayedAcc, err := ir.delayedBridge.GetAccumulator(ctx, checkingDelayedSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbDelayedAcc, err := ir.db.GetDelayedAcc(checkingDelayedSeqNum)
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
			ourLatestBatchCount, err := ir.db.GetBatchCount()
			if err != nil {
				return err
			}
			if ourLatestBatchCount < checkingBatchCount {
				checkingBatchCount = ourLatestBatchCount
				missingSequencer = true
			}
			if checkingBatchCount > 0 {
				checkingBatchSeqNum := checkingBatchCount - 1
				l1DelayedAcc, err := ir.sequencerInbox.GetAccumulator(ctx, checkingBatchSeqNum, currentHeight)
				if err != nil {
					return err
				}
				dbDelayedAcc, err := ir.db.GetBatchAcc(checkingBatchSeqNum)
				if err != nil {
					return err
				}
				if dbDelayedAcc != l1DelayedAcc {
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
				break
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
					haveAcc, err := ir.db.GetBatchAcc(firstBatch.SequenceNumber - 1)
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
						haveAcc, err := ir.db.GetBatchAcc(batch.SequenceNumber)
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
			} else if missingSequencer {
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
					haveAcc, err := ir.db.GetDelayedAcc(beforeCount - 1)
					if errors.Is(err, accumulatorNotFound) {
						reorgingDelayed = true
					} else if err != nil {
						return err
					} else if haveAcc != beforeAcc {
						reorgingDelayed = true
					}
				}
			} else if missingDelayed {
				// We were missing delayed messages but didn't find any.
				// This must mean that the delayed messages are in the past.
				reorgingDelayed = true
			}

			log.Trace("looking up messages", "from", from.String(), "to", to.String())
			if !reorgingDelayed && !reorgingSequencer && (len(delayedMessages) != 0 || len(sequencerBatches) != 0) {
				delayedMismatch, err := ir.addMessages(sequencerBatches, delayedMessages)
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

func (r *InboxReader) addMessages(sequencerBatches []*SequencerInboxBatch, delayedMessages []*DelayedInboxMessage) (bool, error) {
	err := r.db.addDelayedMessages(delayedMessages)
	if err != nil {
		return false, err
	}
	err = r.db.addSequencerBatches(sequencerBatches)
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
		newFrom = r.firstMessageBlock
	}
	return newFrom, nil
}

func (r *InboxReader) getNextBlockToRead() (*big.Int, error) {
	delayedCount, err := r.db.GetDelayedCount()
	if err != nil {
		return nil, err
	}
	if delayedCount == 0 {
		return r.firstMessageBlock, nil
	}
	msg, err := r.db.GetDelayedMessage(delayedCount - 1)
	if err != nil {
		return nil, err
	}
	return msg.Header.RequestId.Big(), nil
}
