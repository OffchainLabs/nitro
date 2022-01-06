//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package retryables

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
)

const RetryableLifetimeSeconds = 7 * 24 * 60 * 60 // one week

type RetryableState struct {
	retryables   *storage.Storage
	timeoutQueue *storage.Queue
}

var (
	timeoutQueueKey = []byte{0}
	calldataKey     = []byte{1}
)

func InitializeRetryableState(sto *storage.Storage) {
	storage.InitializeQueue(sto.OpenSubStorage(timeoutQueueKey))
}

func OpenRetryableState(sto *storage.Storage) *RetryableState {
	return &RetryableState{
		sto,
		storage.OpenQueue(sto.OpenSubStorage(timeoutQueueKey)),
	}
}

type Retryable struct {
	id             common.Hash // the retryable's ID is also the key that determines where it lives in storage
	backingStorage *storage.Storage
	numTries       *uint64
	timeout        *uint64
	from           *common.Address
	to             *common.Address // potentially nil
	callvalue      *big.Int
	beneficiary    *common.Address
	calldata       []byte
}

const (
	numTriesOffset uint64 = iota
	timeoutOffset
	fromOffset
	toOffset
	callvalueOffset
	beneficiaryOffset
)

func (rs *RetryableState) CreateRetryable(
	currentTimestamp uint64,
	id common.Hash, // we assume that the id is unique and hasn't been used before
	timeout uint64,
	from common.Address,
	to *common.Address,
	callvalue *big.Int,
	beneficiary common.Address,
	calldata []byte,
) *Retryable {
	rs.TryToReapOneRetryable(currentTimestamp)
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	seq := uint64(0)
	ret := &Retryable{
		id,
		sto,
		&seq,
		&timeout,
		&from,
		to,
		callvalue,
		&beneficiary,
		calldata,
	}
	sto.SetByUint64(numTriesOffset, common.Hash{})
	sto.SetUint64ByUint64(timeoutOffset, timeout)
	sto.SetByUint64(fromOffset, common.BytesToHash(from.Bytes()))
	sto.SetByUint64(toOffset, common.BytesToHash(to.Bytes()))
	sto.SetByUint64(callvalueOffset, common.BigToHash(callvalue))
	sto.SetByUint64(beneficiaryOffset, common.BytesToHash(beneficiary.Bytes()))
	sto.OpenSubStorage(calldataKey).WriteBytes(calldata)

	// insert the new retryable into the queue so it can be reaped later
	rs.timeoutQueue.Put(id)

	return ret
}

func (rs *RetryableState) OpenRetryable(id common.Hash, currentTimestamp uint64) *Retryable {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	timeout := sto.GetByUint64(timeoutOffset)
	if timeout == (common.Hash{}) {
		// no retryable here (real retryable never has a zero timeout)
		return nil
	}
	if timeout.Big().Uint64() < currentTimestamp {
		// the timeout has expired and will soon be reaped
		return nil
	}
	return &Retryable{
		id:             id,
		backingStorage: sto,
	}
}

func (rs *RetryableState) RetryableSizeBytes(id common.Hash, currentTime uint64) uint64 {
	retryable := rs.OpenRetryable(id, currentTime)
	if retryable == nil {
		return 0
	}
	calldata := 32 + 32*util.WordsForBytes(retryable.CalldataSize()) // length + contents
	return 6*32 + calldata
}

func (rs *RetryableState) DeleteRetryable(id common.Hash) bool {
	retStorage := rs.retryables.OpenSubStorage(id.Bytes())
	if retStorage.GetByUint64(timeoutOffset) == (common.Hash{}) {
		return false
	}
	retStorage.SetUint64ByUint64(numTriesOffset, 0)
	retStorage.SetByUint64(timeoutOffset, common.Hash{})
	retStorage.SetByUint64(fromOffset, common.Hash{})
	retStorage.SetByUint64(toOffset, common.Hash{})
	retStorage.SetByUint64(callvalueOffset, common.Hash{})
	retStorage.SetByUint64(beneficiaryOffset, common.Hash{})
	retStorage.OpenSubStorage(calldataKey).DeleteBytes()
	return true
}

func (retryable *Retryable) NumTries() uint64 {
	if retryable.numTries == nil {
		numTries := retryable.backingStorage.GetUint64ByUint64(numTriesOffset)
		retryable.numTries = &numTries
	}
	return *retryable.numTries
}

func (retryable *Retryable) SetNumTries(newNumTries uint64) {
	retryable.numTries = &newNumTries
	retryable.backingStorage.SetUint64ByUint64(numTriesOffset, newNumTries)
}

func (retryable *Retryable) IncrementNumTries() uint64 {
	newNumTries := retryable.NumTries() + 1
	retryable.SetNumTries(newNumTries)
	return newNumTries
}

func TxIdForRedeemAttempt(ticketId common.Hash, trySequenceNum uint64) common.Hash {
	// Since tickets & sequence numbers are assigned sequentially, each is expressible as a uint64.
	// Relying on this, we can set the upper and lower 8 bytes for the ticket & sequence number, respectively.

	bytes := make([]byte, 32)
	binary.BigEndian.PutUint64(bytes[0:], ticketId.Big().Uint64())
	binary.BigEndian.PutUint64(bytes[24:], trySequenceNum)
	return common.BytesToHash(bytes)
}

func (retryable *Retryable) Beneficiary() common.Address {
	if retryable.beneficiary == nil {
		b := common.BytesToAddress(retryable.backingStorage.GetByUint64(beneficiaryOffset).Bytes())
		retryable.beneficiary = &b
	}
	return *retryable.beneficiary
}

func (retryable *Retryable) Timeout() uint64 {
	if retryable.timeout == nil {
		t := retryable.backingStorage.GetUint64ByUint64(timeoutOffset)
		retryable.timeout = &t
	}
	return *retryable.timeout
}

func (retryable *Retryable) From() common.Address {
	if retryable.from == nil {
		a := common.BytesToAddress(retryable.backingStorage.GetByUint64(fromOffset).Bytes())
		retryable.from = &a
	}
	return *retryable.from
}

func (retryable *Retryable) To() *common.Address {
	if retryable.to == nil {
		to := common.BytesToAddress(retryable.backingStorage.GetByUint64(toOffset).Bytes())
		retryable.to = &to
	}
	return retryable.to
}

func (retryable *Retryable) Callvalue() *big.Int {
	if retryable.callvalue == nil {
		retryable.callvalue = retryable.backingStorage.GetByUint64(callvalueOffset).Big()
	}
	return retryable.callvalue
}

func (retryable *Retryable) Calldata() []byte {
	if retryable.calldata == nil {
		retryable.calldata = retryable.backingStorage.OpenSubStorage(calldataKey).GetBytes()
	}
	return retryable.calldata
}

func (retryable *Retryable) CalldataSize() uint64 { // efficiently gets size of calldata without loading all of it
	if retryable.calldata == nil {
		return retryable.backingStorage.OpenSubStorage(calldataKey).GetBytesSize()
	} else {
		return uint64(len(retryable.calldata))
	}
}

func (retryable *Retryable) SetTimeout(timeout uint64) {
	retryable.timeout = &timeout
	retryable.backingStorage.SetUint64ByUint64(timeoutOffset, timeout)
}

func (rs *RetryableState) Keepalive(ticketId common.Hash, currentTimestamp, limitBeforeAdd, timeToAdd uint64) error {
	retryable := rs.OpenRetryable(ticketId, currentTimestamp)
	if retryable == nil {
		return errors.New("ticketId not found")
	}
	timeout := retryable.Timeout()
	if timeout > limitBeforeAdd {
		return errors.New("timeout too far into the future")
	}
	retryable.SetTimeout(timeout + timeToAdd)
	return nil
}

func (retryable *Retryable) Equals(other *Retryable) bool { // for testing
	if retryable.id != other.id {
		return false
	}
	if retryable.Timeout() != other.Timeout() {
		return false
	}
	if retryable.From() != other.From() {
		return false
	}
	rto := retryable.To()
	oto := other.To()
	if rto == nil {
		if oto != nil {
			return false
		}
	} else if oto == nil {
		return false
	} else if *rto != *oto {
		return false
	}
	if retryable.Callvalue().Cmp(other.Callvalue()) != 0 {
		return false
	}
	if retryable.Beneficiary() != other.Beneficiary() {
		return false
	}
	return bytes.Equal(retryable.Calldata(), other.Calldata())
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
