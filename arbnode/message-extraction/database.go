package mel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	dbschema "github.com/offchainlabs/nitro/arbnode/db-schema"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pos)
	key = append(key, data...)
	return key
}

// Database is a wrapper around arbDB to avoid import cycle issue between arbnode package and mel
type Database struct {
	db ethdb.Database
}

func NewDatabase(db ethdb.Database) *Database {
	return &Database{db}
}

// initializeSeenUnreadDelayedMetaDeque is to be only called by the Start fsm step of MEL
func (d *Database) initializeSeenUnreadDelayedMetaDeque(ctx context.Context, state *meltypes.State, finalizedBlock uint64) error {
	if state.DelayedMessagedSeen == state.DelayedMessagesRead &&
		(state.DelayedMessagedSeen == 0 || // this is the first mel state so no need to initialize deque even if the state isnt finalized yet. TODO: during upgrade we would want the initial mel state's parentchainblocknumber to be finalized
			state.ParentChainBlockNumber <= finalizedBlock) {
		return nil
	}
	// To make the deque reorg resistant we will need to add more delayedMeta even though those messages are `Read`
	// this is only relevant if finalizedBlock is behind the current head Mel state's ParentChainBlockNumber
	targetDelayedMessagesRead := state.DelayedMessagesRead
	if finalizedBlock > 0 && state.ParentChainBlockNumber > finalizedBlock {
		finalizedMelState, err := d.State(ctx, finalizedBlock)
		if err != nil {
			return err
		}
		targetDelayedMessagesRead = finalizedMelState.DelayedMessagesRead
	}
	// We first find the melState whose DelayedMessagedSeen is just before the targetDelayedMessagesRead, so that we can construct a merkleAccumulator
	// thats relevant to us
	var err error
	var prev *meltypes.State
	delayedMsgIndexToParentChainBlockNum := make(map[uint64]uint64)
	curr := state
	for i := state.ParentChainBlockNumber - 1; i > 0; i-- {
		prev, err = d.State(ctx, i)
		if err != nil {
			return err
		}
		if curr.DelayedMessagedSeen > prev.DelayedMessagedSeen { // Meaning the 'curr' melState has seen some delayed messages
			for j := prev.DelayedMessagedSeen; j < curr.DelayedMessagedSeen; j++ {
				delayedMsgIndexToParentChainBlockNum[j] = curr.ParentChainBlockNumber
			}
		}
		if prev.DelayedMessagedSeen <= targetDelayedMessagesRead {
			break
		}
		curr = prev
	}
	if prev == nil {
		return nil
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		meltypes.ToPtrSlice(prev.DelayedMessageMerklePartials),
	)
	if err != nil {
		return err
	}
	// We then walk forward the merkleAccumulator till targetDelayedMessagesRead
	for index := prev.DelayedMessagedSeen; index < targetDelayedMessagesRead; index++ {
		msg, err := d.fetchDelayedMessage(index)
		if err != nil {
			return err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return err
		}
	}
	// Accumulator is now at the step we need, hence we start creating DelayedMeta for all the delayed messages that are seen but not read
	seenUnreadDelayedMetaDeque := &meltypes.DelayedMetaDeque{}
	for index := targetDelayedMessagesRead; index < state.DelayedMessagedSeen; index++ {
		msg, err := d.fetchDelayedMessage(index)
		if err != nil {
			return err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return err
		}
		merkleRoot, err := acc.Root()
		if err != nil {
			return err
		}
		seenUnreadDelayedMetaDeque.Add(&meltypes.DelayedMeta{
			Index:                       index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: delayedMsgIndexToParentChainBlockNum[index],
		})
	}
	state.SetSeenUnreadDelayedMetaDeque(seenUnreadDelayedMetaDeque)
	return nil
}

func (d *Database) GetHeadMelState(ctx context.Context) (*meltypes.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	return d.State(ctx, headMelStateBlockNum)
}

// FetchInitialState method of the StateFetcher interface is implemented by the database as it would be used after the initial fetch
func (d *Database) FetchInitialState(ctx context.Context, parentChainBlockHash common.Hash, finalizedBlock uint64) (*meltypes.State, error) {
	state, err := d.GetHeadMelState(ctx)
	if err != nil {
		return nil, err
	}
	// We check if our current head mel state corresponds to this parentChainBlockHash
	if state.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("head mel state's parentChainBlockHash in db: %v doesnt match the given parentChainBlockHash: %v ", state.ParentChainBlockHash, parentChainBlockHash)
	}
	if err = d.initializeSeenUnreadDelayedMetaDeque(ctx, state, finalizedBlock); err != nil {
		return nil, err
	}
	return state, nil
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

func (d *Database) setMelState(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64, state meltypes.State) error {
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

func (d *Database) SaveBatchMetas(ctx context.Context, state *meltypes.State, batchMetas []*meltypes.BatchMetadata) error {
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

func (d *Database) fetchBatchMetadata(seqNum uint64) (*meltypes.BatchMetadata, error) {
	key := dbKey(dbschema.MelSequencerBatchMetaPrefix, seqNum)
	batchMetadataBytes, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var batchMetadata meltypes.BatchMetadata
	if err = rlp.DecodeBytes(batchMetadataBytes, &batchMetadata); err != nil {
		return nil, err
	}
	return &batchMetadata, nil
}

// SaveState should exclusively be called for saving the recently generated "head" MEL state
func (d *Database) SaveState(ctx context.Context, state *meltypes.State) error {
	dbBatch := d.db.NewBatch()
	if err := d.setMelState(dbBatch, state.ParentChainBlockNumber, *state); err != nil {
		return err
	}
	if err := d.setHeadMelStateBlockNum(dbBatch, state.ParentChainBlockNumber); err != nil {
		return err
	}
	return dbBatch.Write()
}

func (d *Database) State(ctx context.Context, parentChainBlockNumber uint64) (*meltypes.State, error) {
	key := dbKey(dbschema.MelStatePrefix, parentChainBlockNumber)
	data, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var state meltypes.State
	err = rlp.DecodeBytes(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (d *Database) SaveDelayedMessages(ctx context.Context, state *meltypes.State, delayedMessages []*meltypes.DelayedInboxMessage) error {
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

func (d *Database) checkAgainstAccumulator(ctx context.Context, state *meltypes.State, msg *meltypes.DelayedInboxMessage, index uint64) (bool, error) {
	delayedMeta := state.GetSeenUnreadDelayedMetaDeque().GetByIndex(index)
	acc := state.GetReadDelayedMsgsAcc()
	if acc == nil {
		melStateParentChainBlockNum := delayedMeta.MelStateParentChainBlockNum
		targetState, err := d.State(ctx, melStateParentChainBlockNum-1)
		if err != nil {
			return false, err
		}
		if acc, err = merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
			meltypes.ToPtrSlice(targetState.DelayedMessageMerklePartials),
		); err != nil {
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
		state.SetReadDelayedMsgsAcc(acc)
	}
	_, err := acc.Append(msg.Hash())
	if err != nil {
		return false, err
	}
	merkleRoot, err := acc.Root()
	if err != nil {
		return false, err
	}
	return merkleRoot == delayedMeta.MerkleRoot, nil
}

func (d *Database) fetchDelayedMessage(index uint64) (*meltypes.DelayedInboxMessage, error) {
	key := dbKey(dbschema.MelDelayedMessagePrefix, index)
	delayedBytes, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var delayed meltypes.DelayedInboxMessage
	if err = rlp.DecodeBytes(delayedBytes, &delayed); err != nil {
		return nil, err
	}
	return &delayed, nil
}

func (d *Database) ReadDelayedMessage(ctx context.Context, state *meltypes.State, index uint64) (*meltypes.DelayedInboxMessage, error) {
	if index == 0 { // Init message
		// This message cannot be found in the database as it is supposed to be seen and read in the same block, so we persist that in SeenUnreadDelayedMetaDeque
		return state.GetSeenUnreadDelayedMetaDeque().GetInitMsg(), nil
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
