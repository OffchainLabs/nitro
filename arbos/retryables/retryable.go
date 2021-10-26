//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package retryables

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"io"
	"math/big"
)

type RetryableState struct {
	retryables   *storage.Storage
	timeoutQueue *storage.Queue
}

func InitializeRetryableState(sto *storage.Storage) {
	storage.InitializeQueue(sto.OpenSubStorage([]byte{}))
}

func OpenRetryableState(sto *storage.Storage) *RetryableState {
	return &RetryableState{
		sto,
		storage.OpenQueue(sto.OpenSubStorage([]byte{})),
	}
}

type Retryable struct {
	id          common.Hash // the retryable's ID is also the key that determines where it lives in storage
	numTries    *big.Int
	timeout     uint64
	from        common.Address
	to          common.Address
	callvalue   *big.Int
	beneficiary common.Address
	calldata    []byte
}

func (rs *RetryableState) CreateRetryable(
	currentTimestamp uint64,
	id common.Hash, // we assume that the id is unique and hasn't been used before
	timeout uint64,
	from common.Address,
	to common.Address,
	callvalue *big.Int,
	beneficiary common.Address,
	calldata []byte,
) *Retryable {
	rs.TryToReapOneRetryable(currentTimestamp)
	ret := &Retryable{
		id,
		big.NewInt(0),
		timeout,
		from,
		to,
		callvalue,
		beneficiary,
		calldata,
	}
	buf := bytes.Buffer{}
	if err := ret.serialize(&buf); err != nil {
		panic(err)
	}

	// set up a storage to hold the retryable
	rs.retryables.OpenSubStorage(id.Bytes()).WriteBytes(buf.Bytes())

	// insert the new retryable into the queue so it can be reaped later
	rs.timeoutQueue.Put(id)

	return ret
}

func (rs *RetryableState) OpenRetryable(id common.Hash, currentTimestamp uint64) *Retryable {
	seg := rs.retryables.OpenSubStorage(id.Bytes())
	contents := seg.GetBytes()
	if len(contents) == 0 {
		// no valid retryable with that ID
		return nil
	}
	ret, err := NewRetryableFromReader(bytes.NewReader(contents), id)
	if err != nil {
		return nil
	}
	if ret.timeout < currentTimestamp {
		// retryable has expired, so delete it
		seg.DeleteBytes()
		return nil
	}
	return ret
}

func (rs *RetryableState) RetryableSizeBytes(id common.Hash) uint64 {
	return rs.retryables.OpenSubStorage(id.Bytes()).GetBytesSize()
}

func (rs *RetryableState) DeleteRetryable(id common.Hash) {
	rs.retryables.OpenSubStorage(id.Bytes()).DeleteBytes()
}

func NewRetryableFromReader(rd io.Reader, id common.Hash) (*Retryable, error) {
	numTries, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
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
	beneficiary, err := util.AddressFromReader(rd)
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
		beneficiary,
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
	if _, err := wr.Write(retryable.beneficiary[:]); err != nil {
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

func (retryable *Retryable) Beneficiary() common.Address {
	return retryable.beneficiary
}

func (retryable *Retryable) Timeout() *big.Int {
	return big.NewInt(int64(retryable.timeout))
}

func (rs *RetryableState) Keepalive(ticketId common.Hash, currentTimestamp, limitBeforeAdd, timeToAdd uint64) bool {
	retryable := rs.OpenRetryable(ticketId, currentTimestamp)
	if retryable == nil {
		return false
	}
	if retryable.timeout > limitBeforeAdd {
		return false
	}
	retryable.timeout += timeToAdd

	// write the retryable back to storage
	buf := bytes.Buffer{}
	if err := retryable.serialize(&buf); err != nil {
		panic(err)
	}

	// set up a storage to hold the retryable
	rs.retryables.OpenSubStorage(ticketId.Bytes()).WriteBytes(buf.Bytes())

	return true
}

func (retryable *Retryable) Equals(other *Retryable) bool { // for testing
	if retryable.id != other.id {
		return false
	}
	if retryable.timeout != other.timeout {
		return false
	}
	if retryable.from != other.from {
		return false
	}
	if retryable.to != other.to {
		return false
	}
	if retryable.callvalue.Cmp(other.callvalue) != 0 {
		return false
	}
	if retryable.beneficiary != other.beneficiary {
		return false
	}
	return bytes.Equal(retryable.calldata, other.calldata)
}

func (rs *RetryableState) TryToReapOneRetryable(currentTimestamp uint64) {
	if !rs.timeoutQueue.IsEmpty() {
		id := rs.timeoutQueue.Get()
		retryable := rs.OpenRetryable(*id, currentTimestamp)
		if retryable != nil {
			// OpenRetryable returned non-nil, so we know the retryable hasn't expired
			rs.timeoutQueue.Put(*id)
		}
	}
}
