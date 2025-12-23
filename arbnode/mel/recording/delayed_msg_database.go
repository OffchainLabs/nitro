package melrecording

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

// RecordingDatabase holds an ethdb.KeyValueStore that contains delayed messages stored by native MEL and implements DelayedMessageDatabase
// interface defined in 'mel'. It is solely used for recording of preimages relating to delayed messages needed for MEL validation
type RecordingDatabase struct {
	db          ethdb.KeyValueStore
	preimages   map[common.Hash][]byte
	initialized bool
}

func NewRecordingDatabase(db ethdb.KeyValueStore) *RecordingDatabase {
	return &RecordingDatabase{db, make(map[common.Hash][]byte), false}
}

func (r *RecordingDatabase) initialize(ctx context.Context, state *mel.State) error {
	var acc *merkleAccumulator.MerkleAccumulator
	for i := state.ParentChainBlockNumber; i > 0; i-- {
		seenState, err := getState(ctx, r.db, i)
		if err != nil {
			return err
		}
		if seenState.DelayedMessagesSeen <= state.DelayedMessagesRead {
			acc, err = merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(mel.ToPtrSlice(seenState.DelayedMessageMerklePartials))
			if err != nil {
				return err
			}
			for j := seenState.DelayedMessagesSeen; j < state.DelayedMessagesRead; j++ {
				delayed, err := fetchDelayedMessage(r.db, j)
				if err != nil {
					return err
				}
				_, err = acc.Append(delayed.Hash())
				if err != nil {
					return err
				}
			}
			break
		}
	}
	if acc == nil {
		return errors.New("couldnt initialize the accumulator")
	}
	acc.RecordPreimagesTo(r.preimages)
	for i := state.DelayedMessagesRead; i < state.DelayedMessagesSeen; i++ {
		delayed, err := fetchDelayedMessage(r.db, i)
		if err != nil {
			return err
		}
		_, err = acc.Append(delayed.Hash())
		if err != nil {
			return err
		}
	}
	_, err := acc.Root()
	if err != nil {
		return err
	}
	seenAcc := state.GetSeenDelayedMsgsAcc()
	if seenAcc == nil {
		seenAcc, err = merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(mel.ToPtrSlice(state.DelayedMessageMerklePartials))
		if err != nil {
			return err
		}
	}
	seenAcc.RecordPreimagesTo(r.preimages)
	state.SetSeenDelayedMsgsAcc(seenAcc)
	return nil
}

func (r *RecordingDatabase) Preimages() map[common.Hash][]byte { return r.preimages }

func (r *RecordingDatabase) ReadDelayedMessage(ctx context.Context, state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	if index == 0 { // Init message
		// This message cannot be found in the database as it is supposed to be seen and read in the same block, so we persist that in DelayedMessageBacklog
		return state.GetDelayedMessageBacklog().GetInitMsg(), nil
	}
	if !r.initialized {
		if err := r.initialize(ctx, state); err != nil {
			return nil, fmt.Errorf("error initializing recording database for MEL validation: %w", err)
		}
		r.initialized = true
	}
	delayed, err := fetchDelayedMessage(r.db, index)
	if err != nil {
		return nil, err
	}
	delayedMsgBytes, err := rlp.EncodeToBytes(delayed)
	if err != nil {
		return nil, err
	}
	hashDelayedHash := crypto.Keccak256(delayed.Hash().Bytes())
	r.preimages[common.BytesToHash(hashDelayedHash)] = delayedMsgBytes
	return delayed, nil
}

func fetchDelayedMessage(db ethdb.KeyValueStore, index uint64) (*mel.DelayedInboxMessage, error) {
	delayed, err := read.Value[mel.DelayedInboxMessage](db, read.Key(schema.MelDelayedMessagePrefix, index))
	if err != nil {
		return nil, err
	}
	return &delayed, nil
}

func getState(ctx context.Context, db ethdb.KeyValueStore, parentChainBlockNumber uint64) (*mel.State, error) {
	state, err := read.Value[mel.State](db, read.Key(schema.MelStatePrefix, parentChainBlockNumber))
	if err != nil {
		return nil, err
	}
	return &state, nil
}
