package melrunner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	dbschema "github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
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

// FetchInitialState method of the StateFetcher interface is implemented by the database as it would be used after the initial fetch
func (d *Database) FetchInitialState(ctx context.Context, parentChainBlockHash common.Hash) (*mel.State, error) {
	state, err := d.GetHeadMelState(ctx)
	if err != nil {
		return nil, err
	}
	// We check if our current head mel state corresponds to this parentChainBlockHash
	if state.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("head mel state's parentChainBlockHash in db: %v does not match the given parentChainBlockHash: %v ", state.ParentChainBlockHash, parentChainBlockHash)
	}
	return state, nil
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
	if state.DelayedMessagedSeen < uint64(len(delayedMessages)) {
		return fmt.Errorf("mel state's DelayedMessagedSeen: %d is lower than number of delayed messages: %d queued to be added", state.DelayedMessagedSeen, len(delayedMessages))
	}
	firstPos := state.DelayedMessagedSeen - uint64(len(delayedMessages))
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
		// This message cannot be found in the database as it is supposed to be seen and read in the same block, so we persist that in DelayedMessageBacklog
		return state.GetDelayedMessageBacklog().GetInitMsg(), nil
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
	delayedMessageBacklog := state.GetDelayedMessageBacklog()
	delayedMeta, err := delayedMessageBacklog.Get(index)
	if err != nil {
		return false, err
	}
	preReadCount := state.GetReadCountFromBacklog()
	if index < preReadCount {
		// Delayed message has already been verified with a merkle root, we just need to verify that the hash matches
		if msg.Hash() != delayedMeta.MsgHash {
			return false, nil
		}
		return true, nil
	}
	targetState, err := d.State(ctx, delayedMeta.MelStateParentChainBlockNum-1)
	if err != nil {
		return false, err
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		mel.ToPtrSlice(targetState.DelayedMessageMerklePartials),
	)
	if err != nil {
		return false, err
	}
	for i := targetState.DelayedMessagedSeen; i < index; i++ {
		delayed, err := d.fetchDelayedMessage(i)
		if err != nil {
			return false, err
		}
		_, err = acc.Append(delayed.Hash())
		if err != nil {
			return false, err
		}
	}
	// Accumulate this message
	_, err = acc.Append(msg.Hash())
	if err != nil {
		return false, err
	}
	// Accumulate rest of the message-hashes in backlog
	for i := index + 1; i < state.DelayedMessagedSeen; i++ {
		backlogEntry, err := delayedMessageBacklog.Get(i)
		if err != nil {
			return false, err
		}
		_, err = acc.Append(backlogEntry.MsgHash)
		if err != nil {
			return false, err
		}
	}
	have, err := acc.Root()
	if err != nil {
		return false, err
	}
	want, err := state.GetSeenDelayedMsgsAcc().Root()
	if err != nil {
		return false, err
	}
	if have == want {
		state.SetReadCountFromBacklog(state.DelayedMessagedSeen) // meaning all messages from index to state.DelayedMessagedSeen-1 inclusive have been pre-read
		return true, nil
	}
	return false, nil
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pos)
	key = append(key, data...)
	return key
}
