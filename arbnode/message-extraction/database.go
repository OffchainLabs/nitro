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

// initializeSeenUnreadDelayedMetaDeque is to be only called by the Start fsm step of MEL
func (d *Database) initializeSeenUnreadDelayedMetaDeque(ctx context.Context, state *meltypes.State, finalizedBlock uint64) error {
	if state.DelayedMessagedSeen == state.DelayedMessagesRead && state.ParentChainBlockNumber <= finalizedBlock {
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
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		meltypes.ToPtrSlice(prev.DelayedMessageMerklePartials),
	)
	if err != nil {
		return err
	}
	// We then walk forward the merkleAccumulator till targetDelayedMessagesRead
	for index := prev.DelayedMessagedSeen; index < targetDelayedMessagesRead; index++ {
		msg, err := d.fetchDelayedMessage(ctx, index)
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
		seenUnreadDelayedMetaDeque.Add(&meltypes.DelayedMeta{
			Index:                       index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: delayedMsgIndexToParentChainBlockNum[index],
		})
	}
	state.SetSeenUnreadDelayedMetaDeque(seenUnreadDelayedMetaDeque)
	return nil
}

// FetchInitialState method of the StateFetcher interface is implemented by the database as it would be used after the initial fetch
func (d *Database) FetchInitialState(ctx context.Context, parentChainBlockHash common.Hash, finalizedBlock uint64) (*meltypes.State, error) {
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
	return merkleRoot == delayedMeta.MerkleRoot, nil
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
