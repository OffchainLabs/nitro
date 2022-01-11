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

func InitializeRetryableState(sto *storage.Storage) error {
	return storage.InitializeQueue(sto.OpenSubStorage(timeoutQueueKey))
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
) (*Retryable, error) {
	err := rs.TryToReapOneRetryable(currentTimestamp)
	if err != nil {
		return nil, err
	}
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	ret := &Retryable{
		id,
		sto,
		sto.OpenStorageBackedUint64(numTriesOffset),
		sto.OpenStorageBackedUint64(timeoutOffset),
		sto.OpenStorageBackedAddress(fromOffset),
		sto.OpenStorageBackedAddressOrNil(toOffset),
		sto.OpenStorageBackedBigInt(callvalueOffset),
		sto.OpenStorageBackedAddress(beneficiaryOffset),
		sto.OpenStorageBackedBytes(calldataKey),
	}
	_ = ret.numTries.Set(0)
	_ = ret.timeout.Set(timeout)
	_ = ret.from.Set(from)
	_ = ret.to.Set(to)
	_ = ret.callvalue.Set(callvalue)
	_ = ret.beneficiary.Set(beneficiary)
	_ = ret.calldata.Set(calldata)

	// insert the new retryable into the queue so it can be reaped later
	err = rs.timeoutQueue.Put(id)
	return ret, err
}

func (rs *RetryableState) OpenRetryable(id common.Hash, currentTimestamp uint64) (*Retryable, error) {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	timeoutStorage := sto.OpenStorageBackedUint64(timeoutOffset)
	timeout, err := timeoutStorage.Get()
	if timeout == 0 || timeout < currentTimestamp || err != nil {
		// Either no retryable here (real retryable never has a zero timeout),
		// Or the timeout has expired and the retryable will soon be reaped
		return nil, err
	}
	return &Retryable{
		id:             id,
		backingStorage: sto,
		numTries:       sto.OpenStorageBackedUint64(numTriesOffset),
		timeout:        timeoutStorage,
		from:           sto.OpenStorageBackedAddress(fromOffset),
		to:             sto.OpenStorageBackedAddressOrNil(toOffset),
		callvalue:      sto.OpenStorageBackedBigInt(callvalueOffset),
		beneficiary:    sto.OpenStorageBackedAddress(beneficiaryOffset),
		calldata:       sto.OpenStorageBackedBytes(calldataKey),
	}, nil
}

func (rs *RetryableState) RetryableSizeBytes(id common.Hash, currentTime uint64) (uint64, error) {
	retryable, err := rs.OpenRetryable(id, currentTime)
	if retryable == nil || err != nil {
		return 0, err
	}
	size, err := retryable.CalldataSize()
	if err != nil {
		return 0, err
	}
	calldata := 32 + 32*util.WordsForBytes(size) // length + contents
	return 6*32 + calldata, nil
}

func (rs *RetryableState) DeleteRetryable(id common.Hash) (bool, error) {
	retStorage := rs.retryables.OpenSubStorage(id.Bytes())
	timeout, err := retStorage.GetByUint64(timeoutOffset)
	if timeout == (common.Hash{}) || err != nil {
		return false, err
	}
	_ = retStorage.SetUint64ByUint64(numTriesOffset, 0)
	_ = retStorage.SetByUint64(timeoutOffset, common.Hash{})
	_ = retStorage.SetByUint64(fromOffset, common.Hash{})
	_ = retStorage.SetByUint64(toOffset, common.Hash{})
	_ = retStorage.SetByUint64(callvalueOffset, common.Hash{})
	_ = retStorage.SetByUint64(beneficiaryOffset, common.Hash{})
	err = retStorage.OpenSubStorage(calldataKey).ClearBytes()
	return true, err
}

func (retryable *Retryable) NumTries() (uint64, error) {
	return retryable.numTries.Get()
}

func (retryable *Retryable) IncrementNumTries() (uint64, error) {
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

func (retryable *Retryable) Beneficiary() (common.Address, error) {
	return retryable.beneficiary.Get()
}

func (retryable *Retryable) Timeout() (uint64, error) {
	return retryable.timeout.Get()
}

func (retryable *Retryable) SetTimeout(val uint64) error {
	return retryable.timeout.Set(val)
}

func (retryable *Retryable) From() (common.Address, error) {
	return retryable.from.Get()
}

func (retryable *Retryable) To() (*common.Address, error) {
	return retryable.to.Get()
}

func (retryable *Retryable) Callvalue() (*big.Int, error) {
	return retryable.callvalue.Get()
}

func (retryable *Retryable) Calldata() ([]byte, error) {
	return retryable.calldata.Get()
}

// efficiently gets size of calldata without loading all of it
func (retryable *Retryable) CalldataSize() (uint64, error) {
	return retryable.calldata.Size()
}

func (rs *RetryableState) Keepalive(ticketId common.Hash, currentTimestamp, limitBeforeAdd, timeToAdd uint64) error {
	retryable, err := rs.OpenRetryable(ticketId, currentTimestamp)
	if err != nil {
		return err
	}
	if retryable == nil {
		return errors.New("ticketId not found")
	}
	timeout, err := retryable.Timeout()
	if err != nil {
		return err
	}
	if timeout > limitBeforeAdd {
		return errors.New("timeout too far into the future")
	}
	return retryable.SetTimeout(timeout + timeToAdd)
}

func (retryable *Retryable) Equals(other *Retryable) (bool, error) { // for testing
	if retryable.id != other.id {
		return false, nil
	}
	rTries, _ := retryable.NumTries()
	oTries, _ := other.NumTries()
	rTimeout, _ := retryable.Timeout()
	oTimeout, _ := other.Timeout()
	rFrom, _ := retryable.From()
	oFrom, _ := other.From()
	rTo, _ := retryable.To()
	oTo, _ := other.To()
	rCallvalue, _ := retryable.Callvalue()
	oCallvalue, _ := other.Callvalue()
	rBeneficiary, _ := retryable.Beneficiary()
	oBeneficiary, _ := other.Beneficiary()
	rBytes, _ := retryable.Calldata()
	oBytes, err := other.Calldata()

	diff := rTries != oTries || rTimeout != oTimeout || rFrom != oFrom || rBeneficiary != oBeneficiary
	diff = diff || rCallvalue.Cmp(oCallvalue) != 0 || !bytes.Equal(rBytes, oBytes)
	if diff {
		return false, err
	}

	if rTo == nil {
		if oTo != nil {
			return false, err
		}
	} else if oTo == nil {
		return false, err
	} else if *rTo != *oTo {
		return false, err
	}
	return true, err
}

func (rs *RetryableState) TryToReapOneRetryable(currentTimestamp uint64) error {
	empty, err := rs.timeoutQueue.IsEmpty()
	if err != nil {
		return err
	}
	if !empty {
		id, err := rs.timeoutQueue.Get()
		if err != nil {
			return err
		}
		slot := rs.retryables.OpenSubStorage(id.Bytes()).OpenStorageBackedUint64(timeoutOffset)
		timeout, err := slot.Get()
		if err != nil {
			return err
		}
		if timeout != 0 {
			// retryables always have a non-zero timeout, so we know one exists here

			if timeout < currentTimestamp {
				// the retryable has expired, time to reap
				_, err = rs.DeleteRetryable(*id)
			} else {
				// the retryable has not expired, but we'll check back later
				// to preserve round-robin ordering, we put this at the end
				err = rs.timeoutQueue.Put(*id)
			}
			return err
		}
	}
	return nil
}
