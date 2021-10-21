//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/util"
	"io"
	"math/big"
)

type Retryable struct {
	id        common.Hash // the retryable's ID is also the offset where its storage lives in storage
	numTries  *big.Int
	timeout   uint64
	from      common.Address
	to        common.Address
	callvalue *big.Int
	calldata  []byte
}

func CreateRetryable(
	state *ArbosState,
	id common.Hash,
	timeout uint64,
	from common.Address,
	to common.Address,
	callvalue *big.Int,
	calldata []byte,
) *Retryable {
	state.TryToReapOneRetryable()
	ret := &Retryable{
		id,
		big.NewInt(0),
		timeout,
		from,
		to,
		callvalue,
		calldata,
	}
	buf := bytes.Buffer{}
	if err := ret.serialize(&buf); err != nil {
		panic(err)
	}

	// set up a storage to hold the retryable
	state.backingStorage.Open(id.Bytes()).WriteBytes(buf.Bytes())

	// insert the new retryable into the queue so it can be reaped later
	state.RetryableQueue().Put(id)

	// mark the new retryable as valid
	state.ValidRetryablesSet().Set(id, common.Hash{1})

	return ret
}

func OpenRetryable(state *ArbosState, id common.Hash) *Retryable {
	if state.ValidRetryablesSet().Get(id) == (common.Hash{}) {
		// that is not a valid retryable
		return nil
	}
	seg := state.backingStorage.Open(id.Bytes())
	contents := seg.GetBytes()
	ret, err := NewRetryableFromReader(bytes.NewReader(contents), id)
	if err != nil {
		panic(err)
	}
	if ret.timeout < state.LastTimestampSeen() {
		// retryable has expired, so delete it
		seg.DeleteBytes()
		state.ValidRetryablesSet().Set(id, common.Hash{})
		return nil
	}
	return ret
}

func DeleteRetryable(state *ArbosState, id common.Hash) {
	vrs := state.ValidRetryablesSet()
	if vrs.Get(id) != (common.Hash{}) {
		vrs.Set(id, common.Hash{})
		state.backingStorage.Open(id.Bytes()).DeleteBytes()
	}
}

func NewRetryableFromReader(rd io.Reader, id common.Hash) (*Retryable, error) {
	numTries, err := util.HashFromReader(rd)
	timeout, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	from, err := util.AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	to, err := util.AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	callvalue, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	sizeBuf := make([]byte, 8)
	if _, err := io.ReadFull(rd, sizeBuf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint64(sizeBuf)
	calldata := make([]byte, size)
	if _, err := io.ReadFull(rd, calldata); err != nil {
		return nil, err
	}

	return &Retryable{
		id,
		numTries.Big(),
		timeout,
		from,
		to,
		callvalue.Big(),
		calldata,
	}, nil
}

func (retryable *Retryable) serialize(wr io.Writer) error {
	if _, err := wr.Write(common.BigToHash(retryable.numTries).Bytes()); err != nil {
		return err
	}
	if err := util.Uint64ToWriter(retryable.timeout, wr); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.from[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.to[:]); err != nil {
		return err
	}
	if _, err := wr.Write(common.BigToHash(retryable.callvalue).Bytes()); err != nil {
		return err
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(len(retryable.calldata)))
	if _, err := wr.Write(b); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.calldata); err != nil {
		return err
	}
	return nil
}

func (state *ArbosState) TryToReapOneRetryable() {
	queue := state.RetryableQueue()
	if !queue.IsEmpty() {
		id := queue.Get()
		retryable := OpenRetryable(state, *id)
		if retryable != nil {
			// OpenRetryable returned non-nil, so we know the retryable hasn't expired
			queue.Put(*id)
		}
	}
}
