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

// initializeDelayedMetaBacklog is to be only called by the Start fsm step of MEL
func (d *Database) initializeDelayedMetaBacklog(ctx context.Context, state *mel.State, finalizedBlock uint64) error {
	if state.DelayedMessagedSeen == state.DelayedMessagesRead && state.ParentChainBlockNumber <= finalizedBlock {
		return nil // in this case initialization of backlog is handled later in the Start fsm step of mel runner
	}
	// To make the delayedMetaBacklog reorg resistant we will need to add more delayedMeta even though those messages are `Read`
	// this is only relevant if the current head Mel state's ParentChainBlockNumber is not yet finalized
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
	var prev *mel.State
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
		mel.ToPtrSlice(prev.DelayedMessageMerklePartials),
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
	delayedMetaBacklog := mel.NewDelayedMetaBacklog()
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
		delayedMetaBacklog.Add(&mel.DelayedMeta{
			Index:                       index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: delayedMsgIndexToParentChainBlockNum[index],
		})
	}
	state.SetDelayedMetaBacklog(delayedMetaBacklog)
	return nil
}

func (d *Database) GetHeadMelState(ctx context.Context) (*mel.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	return d.State(ctx, headMelStateBlockNum)
}

// FetchInitialState method of the StateFetcher interface is implemented by the database as it would be used after the initial fetch
func (d *Database) FetchInitialState(ctx context.Context, parentChainBlockHash common.Hash, finalizedBlock uint64) (*mel.State, error) {
	state, err := d.GetHeadMelState(ctx)
	if err != nil {
		return nil, err
	}
	// We check if our current head mel state corresponds to this parentChainBlockHash
	if state.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("head mel state's parentChainBlockHash in db: %v doesnt match the given parentChainBlockHash: %v ", state.ParentChainBlockHash, parentChainBlockHash)
	}
	if err = d.initializeDelayedMetaBacklog(ctx, state, finalizedBlock); err != nil {
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

func (d *Database) checkAgainstAccumulator(ctx context.Context, state *mel.State, msg *mel.DelayedInboxMessage, index uint64) (bool, error) {
	delayedMeta := state.GetDelayedMetaBacklog().GetByIndex(index)
	acc := state.GetReadDelayedMsgsAcc()
	if acc == nil {
		melStateParentChainBlockNum := delayedMeta.MelStateParentChainBlockNum
		targetState, err := d.State(ctx, melStateParentChainBlockNum-1)
		if err != nil {
			return false, err
		}
		if acc, err = merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
			mel.ToPtrSlice(targetState.DelayedMessageMerklePartials),
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

func (d *Database) ReadDelayedMessage(ctx context.Context, state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	if index == 0 { // Init message
		// This message cannot be found in the database as it is supposed to be seen and read in the same block, so we persist that in DelayedMetaBacklog
		return state.GetDelayedMetaBacklog().GetInitMsg(), nil
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
