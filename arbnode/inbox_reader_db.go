//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/pkg/errors"
)

type InboxReaderDb struct {
	db ethdb.Database
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

func (d *InboxReaderDb) GetDelayedAcc(seqNum *big.Int) (common.Hash, error) {
	if !seqNum.IsUint64() {
		return common.Hash{}, accumulatorNotFound
	}
	key := dbKey(delayedMessagePrefix, seqNum.Uint64())
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

func (d *InboxReaderDb) GetDelayedCount() (*big.Int, error) {
	data, err := d.db.Get(delayedMessageCountKey)
	if err != nil {
		return nil, err
	}
	var count uint64
	err = rlp.DecodeBytes(data, &count)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(count), nil
}

func (d *InboxReaderDb) GetDelayedMessage(seqNum *big.Int) (*arbos.L1IncomingMessage, error) {
	if !seqNum.IsUint64() {
		return nil, errors.New("delayed sequence number not a uint64")
	}
	key := dbKey(delayedMessagePrefix, seqNum.Uint64())
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
