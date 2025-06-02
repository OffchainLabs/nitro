package mel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode"
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

// initializeSeenDelayedMsgInfoQueue is to be only called by the Start fsm step of MEL
func (d *Database) initializeSeenDelayedMsgInfoQueue(ctx context.Context, state *meltypes.State) error {
	if state.DelayedMessagedSeen == state.DelayedMessagesRead {
		return nil
	}
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
		if prev.DelayedMessagedSeen <= state.DelayedMessagesRead {
			break
		}
		curr = prev
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		meltypes.ToPtrSlice(prev.DelayedMessageMerklePartials),
	)
	if err != nil {
		return err
	}
	for index := prev.DelayedMessagedSeen; index < state.DelayedMessagesRead; index++ {
		msg, err := d.fetchDelayedMessage(ctx, index)
		if err != nil {
			return err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return err
		}
	}
	var seenDelayedMsgInfoQueue []*meltypes.DelayedMsgInfoQueueItem
	for index := state.DelayedMessagesRead; index < state.DelayedMessagedSeen; index++ {
		msg, err := d.fetchDelayedMessage(ctx, index)
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
		seenDelayedMsgInfoQueue = append(seenDelayedMsgInfoQueue, &meltypes.DelayedMsgInfoQueueItem{
			Index:                       index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: delayedMsgIndexToParentChainBlockNum[index],
		})
	}
	state.SetSeenDelayedMsgInfoQueue(seenDelayedMsgInfoQueue)
	return nil
}

// GetState method of the StateFetcher interface is implemented by the database as it would be used after the initial fetch
func (d *Database) GetState(ctx context.Context, parentChainBlockHash common.Hash) (*meltypes.State, error) {
	headMelStateBlockNum, err := d.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, fmt.Errorf("error getting HeadMelStateBlockNum from database: %w", err)
	}
	state, err := d.State(ctx, headMelStateBlockNum)
	if err != nil {
		return nil, err
	}
	// We check if our current head mel state corresponds to this parentChainBlockHash
	if state.ParentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("head mel state's parentChainBlockHash in db: %v doesnt match the given parentChainBlockHash: %v ", state.ParentChainBlockHash, parentChainBlockHash)
	}
	if err = d.initializeSeenDelayedMsgInfoQueue(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (d *Database) setHeadMelStateBlockNum(batch ethdb.KeyValueWriter, parentChainBlockNumber uint64) error {
	parentChainBlockNumberBytes, err := rlp.EncodeToBytes(parentChainBlockNumber)
	if err != nil {
		return err
	}
	err = batch.Put(arbnode.HeadMelStateBlockNumKey, parentChainBlockNumberBytes)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetHeadMelStateBlockNum() (uint64, error) {
	parentChainBlockNumberBytes, err := d.db.Get(arbnode.HeadMelStateBlockNumKey)
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
	key := dbKey(arbnode.MelStatePrefix, parentChainBlockNumber)
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
	key := dbKey(arbnode.MelStatePrefix, parentChainBlockNumber)
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

func (d *Database) SaveDelayedMessages(ctx context.Context, state *meltypes.State, delayedMessages []*arbnode.DelayedInboxMessage) error {
	dbBatch := d.db.NewBatch()
	if state.DelayedMessagedSeen < uint64(len(delayedMessages)) {
		return fmt.Errorf("mel state's DelayedMessagedSeen: %d is lower than number of delayed messages: %d queued to be added", state.DelayedMessagedSeen, len(delayedMessages))
	}
	firstPos := state.DelayedMessagedSeen - uint64(len(delayedMessages))
	for i, msg := range delayedMessages {
		key := dbKey(arbnode.MelDelayedMessagePrefix, firstPos+uint64(i)) // #nosec G115
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

func (d *Database) checkAgainstAccumulator(ctx context.Context, state *meltypes.State, msg *arbnode.DelayedInboxMessage, index uint64) (bool, error) {
	seenDelayedInfoQueue := state.GetSeenDelayedMsgInfoQueue()
	pos := index - seenDelayedInfoQueue[0].Index
	delayedInfo := seenDelayedInfoQueue[pos]
	acc := state.GetReadDelayedMsgsAcc()
	if acc == nil {
		melStateParentChainBlockNum := delayedInfo.MelStateParentChainBlockNum
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
			delayed, err := d.fetchDelayedMessage(ctx, i)
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
	if merkleRoot == delayedInfo.MerkleRoot {
		delayedInfo.Read = true
		return true, nil
	}
	return false, nil
}

func (d *Database) fetchDelayedMessage(ctx context.Context, index uint64) (*arbnode.DelayedInboxMessage, error) {
	key := dbKey(arbnode.MelDelayedMessagePrefix, index)
	delayedBytes, err := d.db.Get(key)
	if err != nil {
		return nil, err
	}
	var delayed arbnode.DelayedInboxMessage
	if err = rlp.DecodeBytes(delayedBytes, &delayed); err != nil {
		return nil, err
	}
	return &delayed, nil
}

func (d *Database) ReadDelayedMessage(ctx context.Context, state *meltypes.State, index uint64) (*arbnode.DelayedInboxMessage, error) {
	delayed, err := d.fetchDelayedMessage(ctx, index)
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
