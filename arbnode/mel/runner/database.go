package melrunner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
)

// Database holds an ethdb.Database underneath and implements StateDatabase interface defined in 'mel'
type Database struct {
	db ethdb.Database
}

func NewDatabase(db ethdb.Database) *Database {
	return &Database{db}
}

func (d *Database) GetHeadMelState(ctx context.Context) (*mel.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	return d.State(ctx, headMelStateBlockNum)
}

// SaveState should exclusively be called for saving the recently generated "head" MEL state
func (d *Database) SaveState(ctx context.Context, state *mel.State) error {
	dbBatch := d.db.NewBatch()
	if err := d.setMelState(dbBatch, state.ParentChainBlockNumber, *state); err != nil {
		return err
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, state.ParentChainBlockNumber); err != nil {
		return err
	}
	return dbBatch.Write()
}

func (d *Database) setMelState(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64, state mel.State) error {
	key := dbKey(dbschema.MelStatePrefix, parentChainBlockNumber)
	melStateBytes, err := rlp.EncodeToBytes(state)
	if err != nil {
		return err
	}
	if err := batch.Put(key, melStateBytes); err != nil {
		return err
	}
	return nil
}

func (d *Database) setHeadMelStateBlockNum(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64) error {
	parentChainBlockNumberBytes, err := rlp.EncodeToBytes(parentChainBlockNumber)
	if err != nil {
		return err
	}
	err = batch.Put(dbschema.HeadMelStateBlockNumKey, parentChainBlockNumberBytes)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetHeadMelStateBlockNum() (uint64, error) {
	parentChainBlockNumberBytes, err := d.db.Get(dbschema.HeadMelStateBlockNumKey)
	if err != nil {
		return 0, err
	}
	var parentChainBlockNumber uint64
	err = rlp.DecodeBytes(parentChainBlockNumberBytes, &parentChainBlockNumber)
	if err != nil {
		return 0, err
	}
	return parentChainBlockNumber, nil
}

func (d *Database) State(ctx context.Context, parentChainBlockNumber uint64) (*mel.State, error) {
	key := dbKey(dbschema.MelStatePrefix, parentChainBlockNumber)
	data, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var state mel.State
	err = rlp.DecodeBytes(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (d *Database) SaveDelayedMessages(ctx context.Context, state *mel.State, delayedMessages []*mel.DelayedInboxMessage) error {
	dbBatch := d.db.NewBatch()
	if state.DelayedMessagesSeen < uint64(len(delayedMessages)) {
		return fmt.Errorf("mel state's DelayedMessagesSeen: %d is lower than number of delayed messages: %d queued to be added", state.DelayedMessagesSeen, len(delayedMessages))
	}
	firstPos := state.DelayedMessagesSeen - uint64(len(delayedMessages))
	for i, msg := range delayedMessages {
		key := dbKey(dbschema.MelDelayedMessagePrefix, firstPos+uint64(i)) // #nosec G115
		delayedBytes, err := rlp.EncodeToBytes(*msg)
		if err != nil {
			return err
		}
		err = dbBatch.Put(key, delayedBytes)
		if err != nil {
			return err
		}

	}
	return dbBatch.Write()
}

func (d *Database) ReadDelayedMessage(ctx context.Context, state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	if index == 0 { // Init message
		// TODO: to be implemented
		return nil, nil
	}
	delayed, err := d.fetchDelayedMessage(index)
	if err != nil {
		return nil, err
	}
	if ok, err := d.checkAgainstAccumulator(ctx, state, delayed, index); err != nil {
		return nil, fmt.Errorf("error checking if delayed message is part of the mel state accumulator: %w", err)
	} else if !ok {
		return nil, errors.New("delayed message message not part of the mel state accumulator")
	}
	return delayed, nil
}

func (d *Database) fetchDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	key := dbKey(dbschema.MelDelayedMessagePrefix, index)
	delayedBytes, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var delayed mel.DelayedInboxMessage
	if err = rlp.DecodeBytes(delayedBytes, &delayed); err != nil {
		return nil, err
	}
	return &delayed, nil
}

func (d *Database) checkAgainstAccumulator(ctx context.Context, state *mel.State, msg *mel.DelayedInboxMessage, index uint64) (bool, error) {
	// TODO: to be implemented
	return true, nil
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pos)
	key = append(key, data...)
	return key
}
