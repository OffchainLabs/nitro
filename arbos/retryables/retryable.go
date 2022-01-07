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
	id             common.Hash // not backed by storage; this is the key that determines where it lives in storage
	backingStorage *storage.Storage
	numTries       storage.StorageBackedUint64
	timeout        storage.StorageBackedUint64
	from           storage.StorageBackedAddress
	to             storage.StorageBackedAddressOrNil
	callvalue      storage.StorageBackedBigInt
	beneficiary    storage.StorageBackedAddress
	calldata       storage.StorageBackedBytes
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
	ret := &Retryable{
		id,
		sto,
		sto.NewStorageBackedUint64(numTriesOffset),
		sto.NewStorageBackedUint64(timeoutOffset),
		sto.NewStorageBackedAddress(fromOffset),
		sto.NewStorageBackedAddressOrNil(toOffset),
		sto.NewStorageBackedBigInt(callvalueOffset),
		sto.NewStorageBackedAddress(beneficiaryOffset),
		sto.NewStorageBackedBytes(calldataKey),
	}
	ret.numTries.Set(0)
	ret.timeout.Set(timeout)
	ret.from.Set(from)
	ret.to.Set(to)
	ret.callvalue.Set(callvalue)
	ret.beneficiary.Set(beneficiary)
	ret.calldata.Set(calldata)

	// insert the new retryable into the queue so it can be reaped later
	rs.timeoutQueue.Put(id)

	return ret
}

func (rs *RetryableState) OpenRetryable(id common.Hash, currentTimestamp uint64) *Retryable {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	timeout := sto.NewStorageBackedUint64(timeoutOffset)
	if timeout.Get() == 0 {
		// no retryable here (real retryable never has a zero timeout)
		return nil
	}
	if timeout.Get() < currentTimestamp {
		// the timeout has expired and will soon be reaped
		return nil
	}
	return &Retryable{
		id:             id,
		backingStorage: sto,
		numTries:       sto.NewStorageBackedUint64(numTriesOffset),
		timeout:        timeout,
		from:           sto.NewStorageBackedAddress(fromOffset),
		to:             sto.NewStorageBackedAddressOrNil(toOffset),
		callvalue:      sto.NewStorageBackedBigInt(callvalueOffset),
		beneficiary:    sto.NewStorageBackedAddress(beneficiaryOffset),
		calldata:       sto.NewStorageBackedBytes(calldataKey),
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
	retStorage.OpenSubStorage(calldataKey).ClearBytes()
	return true
}

func (retryable *Retryable) NumTries() uint64 {
	return retryable.numTries.Get()
}

func (retryable *Retryable) IncrementNumTries() uint64 {
	return retryable.numTries.Increment()
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
	return retryable.beneficiary.Get()
}

func (retryable *Retryable) Timeout() uint64 {
	return retryable.timeout.Get()
}

func (retryable *Retryable) SetTimeout(val uint64) {
	retryable.timeout.Set(val)
}

func (retryable *Retryable) From() common.Address {
	return retryable.from.Get()
}

func (retryable *Retryable) To() *common.Address {
	return retryable.to.Get()
}

func (retryable *Retryable) Callvalue() *big.Int {
	return retryable.callvalue.Get()
}

func (retryable *Retryable) Calldata() []byte {
	return retryable.calldata.Get()
}

func (retryable *Retryable) CalldataSize() uint64 { // efficiently gets size of calldata without loading all of it
	return retryable.calldata.Size()
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
	if retryable.NumTries() != other.NumTries() {
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
