//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/pkg/errors"
)

type InboxReaderDb struct {
	db    ethdb.Database
	mutex sync.Mutex
}

func NewInboxReaderDb(raw ethdb.Database) (*InboxReaderDb, error) {
	db := &InboxReaderDb{
		db: rawdb.NewTable(raw, arbitrumPrefix),
	}
	err := db.initialize()
	return db, err
}

func (d *InboxReaderDb) initialize() error {
	hasKey, err := d.db.Has(delayedMessageCountKey)
	if err != nil {
		return err
	}
	if !hasKey {
		value, err := rlp.EncodeToBytes(uint64(0))
		if err != nil {
			return err
		}
		err = d.db.Put(delayedMessageCountKey, value)
		if err != nil {
			return err
		}
	}
	return nil
}

var accumulatorNotFound error = errors.New("accumulator not found")

func (d *InboxReaderDb) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	hasKey, err := d.db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !hasKey {
		return common.Hash{}, accumulatorNotFound
	}
	data, err := d.db.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(data) < 32 {
		return common.Hash{}, errors.New("delayed message entry missing accumulator")
	}
	var hash common.Hash
	copy(hash[:], data[:32])
	return hash, nil
}

func (d *InboxReaderDb) GetDelayedCount() (uint64, error) {
	data, err := d.db.Get(delayedMessageCountKey)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = rlp.DecodeBytes(data, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *InboxReaderDb) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	key := dbKey(sequencerBatchMetaPrefix, seqNum)
	hasKey, err := d.db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !hasKey {
		return common.Hash{}, accumulatorNotFound
	}
	data, err := d.db.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(data) < 32 {
		return common.Hash{}, errors.New("delayed message entry missing accumulator")
	}
	var hash common.Hash
	copy(hash[:], data[:32])
	return hash, nil
}

func (d *InboxReaderDb) GetBatchCount() (uint64, error) {
	data, err := d.db.Get(sequencerBatchCountKey)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = rlp.DecodeBytes(data, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *InboxReaderDb) GetDelayedMessage(seqNum uint64) (*arbos.L1IncomingMessage, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	data, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	if len(data) < 32 {
		return nil, errors.New("delayed message entry missing accumulator")
	}
	data = data[32:]
	var message arbos.L1IncomingMessage
	err = rlp.DecodeBytes(data, &message)
	return &message, err
}

func (d *InboxReaderDb) addDelayedMessages(messages []*DelayedInboxMessage) error {
	if len(messages) == 0 {
		return nil
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	pos, err := messages[0].Message.Header.SeqNum()
	if err != nil {
		return err
	}
	var nextAcc common.Hash
	if pos > 0 {
		var err error
		nextAcc, err = d.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, accumulatorNotFound) {
				return errors.New("missing previous delayed message")
			} else {
				return err
			}
		}
	}

	batch := d.db.NewBatch()
	// TODO: remove sequencer batches whose delayed count is > pos
	for _, message := range messages {
		seqNum, err := message.Message.Header.SeqNum()
		if err != nil {
			return err
		}

		if seqNum != pos {
			return errors.New("unexpected delayed sequence number")
		}

		if nextAcc != message.BeforeInboxAcc {
			return errors.New("previous delayed accumulator mismatch")
		}
		nextAcc = message.AfterInboxAcc()

		msgKey := dbKey(delayedMessagePrefix, seqNum)

		msgData, err := rlp.EncodeToBytes(message.Message)
		if err != nil {
			return err
		}
		data := nextAcc.Bytes()
		data = append(data, msgData...)
		err = batch.Put(msgKey, data)
		if err != nil {
			return err
		}

		pos++
	}

	newDelayedCount := pos
	err = deleteStartingAt(d.db, batch, delayedMessagePrefix, uint64ToBytes(newDelayedCount))
	if err != nil {
		return err
	}
	countData, err := rlp.EncodeToBytes(newDelayedCount)
	if err != nil {
		return err
	}
	err = batch.Put(delayedMessageCountKey, countData)
	if err != nil {
		return err
	}

	return batch.Write()
}

var delayedMessagesMismatch = errors.New("sequencer batch delayed messages missing or different")

func (d *InboxReaderDb) addSequencerBatches(batches []*SequencerInboxBatch) error {
	if len(batches) == 0 {
		return nil
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	pos := batches[0].SequenceNumber
	var nextAcc common.Hash
	if pos > 0 {
		var err error
		nextAcc, err = d.GetBatchAcc(pos - 1)
		if err != nil {
			if errors.Is(err, accumulatorNotFound) {
				return errors.New("missing previous sequencer batch")
			} else {
				return err
			}
		}
	}

	dbBatch := d.db.NewBatch()
	for _, batch := range batches {
		if batch.SequenceNumber != pos {
			return errors.New("unexpected batch sequence number")
		}
		if nextAcc != batch.BeforeInboxAcc {
			return errors.New("previous batch accumulator mismatch")
		}

		haveDelayedAcc, err := d.GetDelayedAcc(batch.SequenceNumber)
		if errors.Is(err, accumulatorNotFound) {
			return delayedMessagesMismatch
		} else if err != nil {
			return err
		} else if haveDelayedAcc != batch.AfterDelayedAcc {
			return delayedMessagesMismatch
		}

		nextAcc = batch.AfterInboxAcc

		metaKey := dbKey(sequencerBatchMetaPrefix, pos)
		data := nextAcc.Bytes()
		err = dbBatch.Put(metaKey, data)
		if err != nil {
			return err
		}

		pos++
	}

	err := deleteStartingAt(d.db, dbBatch, sequencerBatchCountKey, uint64ToBytes(pos))
	if err != nil {
		return err
	}
	countData, err := rlp.EncodeToBytes(pos)
	if err != nil {
		return err
	}
	err = dbBatch.Put(sequencerBatchCountKey, countData)
	if err != nil {
		return err
	}

	// TODO create inbox messages from sequencer batches

	return dbBatch.Write()
}
