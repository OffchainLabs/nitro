package melrunner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

// Database holds an ethdb.Database underneath and implements StateDatabase interface defined in 'mel'
type Database struct {
	db ethdb.KeyValueStore
}

func NewDatabase(db ethdb.KeyValueStore) *Database {
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

func (d *Database) SaveBatchMetas(ctx context.Context, state *mel.State, batchMetas []*mel.BatchMetadata) error {
	dbBatch := d.db.NewBatch()
	if state.BatchCount < uint64(len(batchMetas)) {
		return fmt.Errorf("mel state's BatchCount: %d is lower than number of batchMetadata: %d queued to be added", state.BatchCount, len(batchMetas))
	}
	firstPos := state.BatchCount - uint64(len(batchMetas))
	for i, batchMetadata := range batchMetas {
		key := dbKey(dbschema.MelSequencerBatchMetaPrefix, firstPos+uint64(i)) // #nosec G115
		batchMetadataBytes, err := rlp.EncodeToBytes(*batchMetadata)
		if err != nil {
			return err
		}
		err = dbBatch.Put(key, batchMetadataBytes)
		if err != nil {
			return err
		}

	}
	return dbBatch.Write()
}

func (d *Database) fetchBatchMetadata(seqNum uint64) (*mel.BatchMetadata, error) {
	key := dbKey(dbschema.MelSequencerBatchMetaPrefix, seqNum)
	batchMetadataBytes, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var batchMetadata mel.BatchMetadata
	if err = rlp.DecodeBytes(batchMetadataBytes, &batchMetadata); err != nil {
		return nil, err
	}
	return &batchMetadata, nil
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

// checkAgainstAccumulator is used to validate the fetched delayed inbox message from the database that is currently being READ. We do this by first checking
// if the message has already been pre-read via state.GetReadCountFromBacklog(), if it is then we simply check that the message hashes match. Else, we create a new
// merkle accumulator that has accumulated messages till the position 'index' and then accumulate all the messages in the backlog i.e pre-reading them and we
// update the readCountFromBacklog of the state accordingly. The optimization is done as it is unfeasible to store merkle partials for each delayed inbox message
// and accumulate all the future seen but not read messages every single time
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
	for i := targetState.DelayedMessagesSeen; i < index; i++ {
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
	for i := index + 1; i < state.DelayedMessagesSeen; i++ {
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
	seenAcc := state.GetSeenDelayedMsgsAcc()
	if seenAcc == nil {
		log.Debug("Initializing MelState's seenDelayedMsgsAcc, needed for validation")
		// This is very low cost hence better to reconstruct seenDelayedMsgsAcc from fresh partals instead of risking using a dirty acc
		seenAcc, err = merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(mel.ToPtrSlice(state.DelayedMessageMerklePartials))
		if err != nil {
			return false, err
		}
		state.SetSeenDelayedMsgsAcc(seenAcc)
	}
	want, err := seenAcc.Root()
	if err != nil {
		return false, err
	}
	if have == want {
		state.SetReadCountFromBacklog(state.DelayedMessagesSeen) // meaning all messages from index to state.DelayedMessagesSeen-1 inclusive have been pre-read
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
