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

type InboxReader struct {
	// Only in run thread
	caughtUp          bool
	firstMessageBlock *big.Int

	// Thread safe
	db             *InboxReaderDb
	delayedBridge  *DelayedBridge
	sequencerInbox *SequencerInbox
	caughtUpChan   chan bool
	client         L1Interface
}

func NewInboxReader(rawDb ethdb.Database, client L1Interface, firstMessageBlock *big.Int, delayedBridge *DelayedBridge, sequencerInbox *SequencerInbox) (*InboxReader, error) {
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

const inboxReaderDelay int64 = 4

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

		if inboxReaderDelay > 0 {
			currentHeight = new(big.Int).Sub(currentHeight, big.NewInt(inboxReaderDelay))
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
			if checkingDelayedCount > 0 {
				ourLatestDelayedCount, err := ir.db.GetDelayedCount()
				if err != nil {
					return err
				}
				if ourLatestDelayedCount < checkingDelayedCount {
					checkingDelayedCount = ourLatestDelayedCount
					missingSequencer = true
				}
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
			if checkingBatchCount.Sign() > 0 {
				ourLatestBatchCount, err := ir.db.GetBatchCount()
				if err != nil {
					return err
				}
				if ourLatestBatchCount.Cmp(checkingBatchCount) < 0 {
					checkingBatchCount = ourLatestBatchCount
					missingDelayed = true
				}
				checkingBatchSeqNum := new(big.Int).Sub(checkingBatchCount, big.NewInt(1))
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
				var matching int
				for i, batch := range sequencerBatches {
					start := new(big.Int).Sub(batch.SequenceNumber, big.NewInt(1))
					haveAcc, err := ir.db.GetBatchAcc(start)
					if errors.Is(err, accumulatorNotFound) {
						if i == 0 {
							reorgingSequencer = true
						}
						break
					} else if err != nil {
						return err
					} else if haveAcc != batch.BeforeInboxAcc {
						reorgingSequencer = true
						break
					} else {
						matching++
					}
				}
				sequencerBatches = sequencerBatches[matching:]
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
		case <-time.After(2 * time.Second):
		}
	}
}

func (r *InboxReader) addMessages(sequencerBatches []*SequencerInboxBatch, delayedMessages []*DelayedInboxMessage) (bool, error) {
	err := r.db.addDelayedMessages(delayedMessages)
	if err != nil {
		return false, err
	}
	if len(sequencerBatches) != 0 {
		panic("TODO: sequencer batches")
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
