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
	db            *InboxReaderDb
	delayedBridge *DelayedBridge
	caughtUpChan  chan bool
	client        L1Interface
}

func NewInboxReader(rawDb ethdb.Database, client L1Interface, firstMessageBlock *big.Int, delayedBridge *DelayedBridge) (*InboxReader, error) {
	db, err := NewInboxReaderDb(rawDb)
	if err != nil {
		return nil, err
	}
	return &InboxReader{
		db:                db,
		delayedBridge:     delayedBridge,
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
				missingDelayed = true
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
		// TODO the same as above but for sequencer messges

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
			/*
				sequencerBatches, err := ir.sequencerInbox.LookupBatchesInRange(ctx, from, to)
				if err != nil {
					return err
				}
			*/
			if !ir.caughtUp && to.Cmp(currentHeight) == 0 {
				// TODO better caught up tracking
				ir.caughtUp = true
				ir.caughtUpChan <- true
			}
			/*
				if len(sequencerBatches) > 0 {
					batchAccs := make([]common.Hash, 0, len(sequencerBatches)+1)
					start := sequencerBatches[0].GetBeforeCount()
					checkingStart := start.Sign() > 0
					if checkingStart {
						start.Sub(start, big.NewInt(1))
						batchAccs = append(batchAccs, sequencerBatches[0].GetBeforeAcc())
					}
					for _, batch := range sequencerBatches {
						if len(batchAccs) > 0 && batch.GetBeforeAcc() != batchAccs[len(batchAccs)-1] {
							return errors.New("Mismatching batch accumulators; reorg?")
						}
						batchAccs = append(batchAccs, batch.GetAfterAcc())
					}
					matching, err := ir.CountMatchingBatchAccs(start, batchAccs)
					if err != nil {
						return err
					}
					reorgingSequencer = false
					if checkingStart {
						if matching == 0 {
							reorgingSequencer = true
						} else {
							matching--
						}
					}
					sequencerBatches = sequencerBatches[matching:]
				}
			*/
			if len(delayedMessages) > 0 {
				missingDelayed = false
				firstMsg := delayedMessages[0]
				beforeAcc := firstMsg.BeforeInboxAcc
				beforeSeqNum, err := firstMsg.Message.Header.SeqNum()
				if err != nil {
					return err
				}
				reorgingDelayed = false
				if beforeSeqNum > 0 {
					haveAcc, err := ir.db.GetDelayedAcc(beforeSeqNum - 1)
					if err != nil || haveAcc != beforeAcc {
						reorgingDelayed = true
					}
				}
			} else if missingDelayed {
				// We were missing delayed messages but didn't find any.
				// This must mean that the delayed messages are in the past.
				reorgingDelayed = true
			}
			/*
				if len(sequencerBatches) < 5 {
					blocksToFetch += 20
				} else if len(sequencerBatches) > 10 {
					blocksToFetch /= 2
				}
				if blocksToFetch < 2 {
					blocksToFetch = 2
				}
			*/

			log.Trace("looking up messages", "from", from.String(), "to", to.String())
			var sequencerBatches []interface{} // TODO
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

func (r *InboxReader) addMessages(sequencerBatches []interface{}, delayedMessages []*DelayedInboxMessage) (bool, error) {
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
