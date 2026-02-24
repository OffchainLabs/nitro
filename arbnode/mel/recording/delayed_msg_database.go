// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// DelayedMsgDatabase holds an ethdb.KeyValueStore that contains delayed messages stored by native MEL and implements DelayedMessageDatabase
// interface defined in 'mel'. It is solely used for recording of preimages relating to delayed messages needed for MEL validation
type DelayedMsgDatabase struct {
	db          ethdb.KeyValueStore
	preimages   map[common.Hash][]byte
	initialized bool
}

// NewDelayedMsgDatabase returns DelayedMsgDatabase that records preimages related
// to the delayed messages needed for MEL validation into the given preimages map
func NewDelayedMsgDatabase(db ethdb.KeyValueStore, preimages daprovider.PreimagesMap) (*DelayedMsgDatabase, error) {
	if preimages == nil {
		return nil, errors.New("preimages recording destination cannot be nil")
	}
	if _, ok := preimages[arbutil.Keccak256PreimageType]; !ok {
		preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	return &DelayedMsgDatabase{
		db:          db,
		preimages:   preimages[arbutil.Keccak256PreimageType],
		initialized: false,
	}, nil
}

// ReadDelayedMessage allows for retrieving a delayed message that has been observed by MEL but not yet consumed in a batch
// at a specific index. Underneath the hood, delayed messages are stored in a binary Merkle tree representation to make
// retrieval possible in WASM mode. In recording mode, reading a delayed message records its access in a preimages mapping
func (r *DelayedMsgDatabase) ReadDelayedMessage(ctx context.Context, state *mel.State, index uint64) (*mel.DelayedInboxMessage, error) {
	// if index == 0 { // Init message
	// 	// This message cannot be found in the database as it is supposed to be seen and read in the same block, so we persist that in DelayedMessageBacklog
	// 	delayed := state.GetDelayedMessageBacklog().GetInitMsg()
	// 	delayedMsgBytes, err := rlp.EncodeToBytes(delayed)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	hashDelayedHash := crypto.Keccak256(delayed.Hash().Bytes())
	// 	r.preimages[common.BytesToHash(hashDelayedHash)] = delayed.Hash().Bytes()
	// 	r.preimages[delayed.Hash()] = delayedMsgBytes
	// 	return delayed, nil
	// }
	if !r.initialized {
		if err := r.initialize(state); err != nil {
			return nil, fmt.Errorf("error initializing recording database for MEL validation: %w", err)
		}
		r.initialized = true
	}
	// Lightweight operation that is needed as state.Clone() clears the seenDelayedMsgsAcc
	if err := r.initSeenDelayedMsgsAccForRecording(state); err != nil {
		return nil, fmt.Errorf("error initializing seenDelayedMsgsAcc for recording: %w", err)
	}
	delayed, err := fetchDelayedMessage(r.db, index)
	if err != nil {
		return nil, err
	}
	delayedMsgBytes, err := rlp.EncodeToBytes(delayed)
	if err != nil {
		return nil, err
	}
	// Leaves in the Merkle tree are double hashed to prevent against second preimage attacks
	// or length extension attacks, which is why we need to add both the preimage of the hash
	// and the preimage of the hash of the hash to the mapping
	hashDelayedHash := crypto.Keccak256(delayed.Hash().Bytes())
	r.preimages[common.BytesToHash(hashDelayedHash)] = delayed.Hash().Bytes()
	r.preimages[delayed.Hash()] = delayedMsgBytes
	return delayed, nil
}

func fetchDelayedMessage(db ethdb.KeyValueStore, index uint64) (*mel.DelayedInboxMessage, error) {
	delayed, err := read.Value[mel.DelayedInboxMessage](db, read.Key(schema.MelDelayedMessagePrefix, index))
	if err != nil {
		return nil, err
	}
	return &delayed, nil
}

func getState(db ethdb.KeyValueStore, parentChainBlockNumber uint64) (*mel.State, error) {
	state, err := read.Value[mel.State](db, read.Key(schema.MelStatePrefix, parentChainBlockNumber))
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *DelayedMsgDatabase) initialize(state *mel.State) error {
	var acc *merkleAccumulator.MerkleAccumulator
	for i := state.ParentChainBlockNumber; i > 0; i-- {
		seenState, err := getState(r.db, i)
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
	return nil
}

func (r *DelayedMsgDatabase) initSeenDelayedMsgsAccForRecording(state *mel.State) error {
	var err error
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
