package melrunner

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

type RecordingDatabase struct {
	db        ethdb.KeyValueStore
	preimages map[common.Hash][]byte
}

func NewRecordingDatabase(db ethdb.KeyValueStore) *RecordingDatabase {
	return &RecordingDatabase{db, make(map[common.Hash][]byte)}
}

func (r *RecordingDatabase) Initialize(ctx context.Context, state *mel.State) error {
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
