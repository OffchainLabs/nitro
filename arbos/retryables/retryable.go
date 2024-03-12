// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package retryables

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const RetryableLifetimeSeconds = 7 * 24 * 60 * 60 // one week
const RetryableReapPrice = 58000

type RetryableState struct {
	retryables   *storage.Storage
	TimeoutQueue *storage.Queue
}

var (
	timeoutQueueKey = []byte{0}
	calldataKey     = []byte{1}
)

func InitializeRetryableState(sto *storage.Storage) error {
	return storage.InitializeQueue(sto.OpenCachedSubStorage(timeoutQueueKey))
}

func OpenRetryableState(sto *storage.Storage, statedb vm.StateDB) *RetryableState {
	return &RetryableState{
		sto,
		storage.OpenQueue(sto.OpenCachedSubStorage(timeoutQueueKey)),
	}
}

type Retryable struct {
	id                 common.Hash // not backed by storage; this key determines where it lives in storage
	backingStorage     *storage.Storage
	numTries           storage.StorageBackedUint64
	from               storage.StorageBackedAddress
	to                 storage.StorageBackedAddressOrNil
	callvalue          storage.StorageBackedBigUint
	beneficiary        storage.StorageBackedAddress
	calldata           storage.StorageBackedBytes
	timeout            storage.StorageBackedUint64
	timeoutWindowsLeft storage.StorageBackedUint64
}

const (
	numTriesOffset uint64 = iota
	fromOffset
	toOffset
	callvalueOffset
	beneficiaryOffset
	timeoutOffset
	timeoutWindowsLeftOffset
)

func (rs *RetryableState) CreateRetryable(
	id common.Hash, // we assume that the id is unique and hasn't been used before
	timeout uint64,
	from common.Address,
	to *common.Address,
	callvalue *big.Int,
	beneficiary common.Address,
	calldata []byte,
) (*Retryable, error) {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	ret := &Retryable{
		id,
		sto,
		sto.OpenStorageBackedUint64(numTriesOffset),
		sto.OpenStorageBackedAddress(fromOffset),
		sto.OpenStorageBackedAddressOrNil(toOffset),
		sto.OpenStorageBackedBigUint(callvalueOffset),
		sto.OpenStorageBackedAddress(beneficiaryOffset),
		sto.OpenStorageBackedBytes(calldataKey),
		sto.OpenStorageBackedUint64(timeoutOffset),
		sto.OpenStorageBackedUint64(timeoutWindowsLeftOffset),
	}
	_ = ret.numTries.Set(0)
	_ = ret.from.Set(from)
	_ = ret.to.Set(to)
	_ = ret.callvalue.SetChecked(callvalue)
	_ = ret.beneficiary.Set(beneficiary)
	_ = ret.calldata.Set(calldata)
	_ = ret.timeout.Set(timeout)
	_ = ret.timeoutWindowsLeft.Set(0)

	// insert the new retryable into the queue so it can be reaped later
	return ret, rs.TimeoutQueue.Put(id)
}

func (rs *RetryableState) OpenRetryable(id common.Hash, currentTimestamp uint64) (*Retryable, error) {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	timeoutStorage := sto.OpenStorageBackedUint64(timeoutOffset)
	timeout, err := timeoutStorage.Get()
	if timeout == 0 || timeout < currentTimestamp || err != nil {
		// Either no retryable here (real retryable never has a zero timeout),
		// Or the timeout has expired and the retryable will soon be reaped,
		// Or the user is out of gas
		return nil, err
	}
	return &Retryable{
		id:                 id,
		backingStorage:     sto,
		numTries:           sto.OpenStorageBackedUint64(numTriesOffset),
		from:               sto.OpenStorageBackedAddress(fromOffset),
		to:                 sto.OpenStorageBackedAddressOrNil(toOffset),
		callvalue:          sto.OpenStorageBackedBigUint(callvalueOffset),
		beneficiary:        sto.OpenStorageBackedAddress(beneficiaryOffset),
		calldata:           sto.OpenStorageBackedBytes(calldataKey),
		timeout:            timeoutStorage,
		timeoutWindowsLeft: sto.OpenStorageBackedUint64(timeoutWindowsLeftOffset),
	}, nil
}

func (rs *RetryableState) RetryableSizeBytes(id common.Hash, currentTime uint64) (uint64, error) {
	retryable, err := rs.OpenRetryable(id, currentTime)
	if retryable == nil || err != nil {
		return 0, err
	}
	size, err := retryable.CalldataSize()
	calldata := 32 + 32*arbmath.WordsForBytes(size) // length + contents
	return 6*32 + calldata, err
}

func (rs *RetryableState) DeleteRetryable(id common.Hash, evm *vm.EVM, scenario util.TracingScenario) (bool, error) {
	retStorage := rs.retryables.OpenSubStorage(id.Bytes())
	timeout, err := retStorage.GetByUint64(timeoutOffset)
	if timeout == (common.Hash{}) || err != nil {
		return false, err
	}

	// move any funds in escrow to the beneficiary (should be none if the retry succeeded -- see EndTxHook)
	beneficiary, _ := retStorage.GetByUint64(beneficiaryOffset)
	escrowAddress := RetryableEscrowAddress(id)
	beneficiaryAddress := common.BytesToAddress(beneficiary[:])
	amount := evm.StateDB.GetBalance(escrowAddress)
	err = util.TransferBalance(&escrowAddress, &beneficiaryAddress, amount, evm, scenario, "escrow")
	if err != nil {
		return false, err
	}

	// we ignore returned error as we expect that if one ClearByUint64 fails, than all consecutive calls to ClearByUint64 will fail with the same error (not modifying state), and then ClearBytes will also fail with the same error (also not modifying state) - and this one we check and return
	_ = retStorage.ClearByUint64(numTriesOffset)
	_ = retStorage.ClearByUint64(fromOffset)
	_ = retStorage.ClearByUint64(toOffset)
	_ = retStorage.ClearByUint64(callvalueOffset)
	_ = retStorage.ClearByUint64(beneficiaryOffset)
	_ = retStorage.ClearByUint64(timeoutOffset)
	_ = retStorage.ClearByUint64(timeoutWindowsLeftOffset)
	err = retStorage.OpenSubStorage(calldataKey).ClearBytes()
	return true, err
}

func (retryable *Retryable) NumTries() (uint64, error) {
	return retryable.numTries.Get()
}

func (retryable *Retryable) IncrementNumTries() (uint64, error) {
	return retryable.numTries.Increment()
}

func (retryable *Retryable) Beneficiary() (common.Address, error) {
	return retryable.beneficiary.Get()
}

func (retryable *Retryable) CalculateTimeout() (uint64, error) {
	timeout, err := retryable.timeout.Get()
	if err != nil {
		return 0, err
	}
	windows, err := retryable.timeoutWindowsLeft.Get()
	return timeout + windows*RetryableLifetimeSeconds, err
}

func (retryable *Retryable) SetTimeout(val uint64) error {
	return retryable.timeout.Set(val)
}

func (retryable *Retryable) TimeoutWindowsLeft() (uint64, error) {
	return retryable.timeoutWindowsLeft.Get()
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

// CalldataSize efficiently gets size of calldata without loading all of it
func (retryable *Retryable) CalldataSize() (uint64, error) {
	return retryable.calldata.Size()
}

func (rs *RetryableState) Keepalive(
	ticketId common.Hash,
	currentTimestamp,
	limitBeforeAdd,
	timeToAdd uint64,
) (uint64, error) {
	retryable, err := rs.OpenRetryable(ticketId, currentTimestamp)
	if err != nil {
		return 0, err
	}
	if retryable == nil {
		return 0, errors.New("ticketId not found")
	}
	timeout, err := retryable.CalculateTimeout()
	if err != nil {
		return 0, err
	}
	if timeout > limitBeforeAdd {
		return 0, errors.New("timeout too far into the future")
	}

	// Add a duplicate entry to the end of the queue (only the last one deletes the retryable)
	err = rs.TimeoutQueue.Put(retryable.id)
	if err != nil {
		return 0, err
	}
	if _, err := retryable.timeoutWindowsLeft.Increment(); err != nil {
		return 0, err
	}
	newTimeout := timeout + RetryableLifetimeSeconds

	// Pay in advance for the work needed to reap the duplicate from the timeout queue
	return newTimeout, rs.retryables.Burner().Burn(RetryableReapPrice)
}

func (retryable *Retryable) Equals(other *Retryable) (bool, error) { // for testing
	if retryable.id != other.id {
		return false, nil
	}
	rTries, _ := retryable.NumTries()
	oTries, _ := other.NumTries()
	rTimeout, _ := retryable.timeout.Get()
	oTimeout, _ := other.timeout.Get()
	rWindows, _ := retryable.timeoutWindowsLeft.Get()
	oWindows, _ := other.timeoutWindowsLeft.Get()
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

	diff := rTries != oTries || rTimeout != oTimeout || rWindows != oWindows
	diff = diff || rFrom != oFrom || rBeneficiary != oBeneficiary
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

func (rs *RetryableState) TryToReapOneRetryable(currentTimestamp uint64, evm *vm.EVM, scenario util.TracingScenario) error {
	id, err := rs.TimeoutQueue.Peek()
	if err != nil || id == nil {
		return err
	}
	retryableStorage := rs.retryables.OpenSubStorage(id.Bytes())
	timeoutStorage := retryableStorage.OpenStorageBackedUint64(timeoutOffset)
	timeout, err := timeoutStorage.Get()
	if err != nil {
		return err
	}
	if timeout == 0 {
		// The retryable has already been deleted, so discard the peeked entry
		_, err = rs.TimeoutQueue.Get()
		return err
	}

	windowsLeftStorage := retryableStorage.OpenStorageBackedUint64(timeoutWindowsLeftOffset)
	windowsLeft, err := windowsLeftStorage.Get()
	if err != nil || timeout >= currentTimestamp {
		return err
	}

	// Either the retryable has expired, or it's lost a lifetime's worth of time
	_, err = rs.TimeoutQueue.Get()
	if err != nil {
		return err
	}

	if windowsLeft == 0 {
		// the retryable has expired, time to reap
		_, err = rs.DeleteRetryable(*id, evm, scenario)
		return err
	}

	// Consume a window, delaying the timeout one lifetime period
	if err := timeoutStorage.Set(timeout + RetryableLifetimeSeconds); err != nil {
		return err
	}
	return windowsLeftStorage.Set(windowsLeft - 1)
}

func (retryable *Retryable) MakeTx(chainId *big.Int, nonce uint64, gasFeeCap *big.Int, gas uint64, ticketId common.Hash, refundTo common.Address, maxRefund *big.Int, submissionFeeRefund *big.Int) (*types.ArbitrumRetryTx, error) {
	from, err := retryable.From()
	if err != nil {
		return nil, err
	}
	to, err := retryable.To()
	if err != nil {
		return nil, err
	}
	callvalue, err := retryable.Callvalue()
	if err != nil {
		return nil, err
	}
	calldata, err := retryable.Calldata()
	if err != nil {
		return nil, err
	}
	return &types.ArbitrumRetryTx{
		ChainId:             chainId,
		Nonce:               nonce,
		From:                from,
		GasFeeCap:           gasFeeCap,
		Gas:                 gas,
		To:                  to,
		Value:               callvalue,
		Data:                calldata,
		TicketId:            ticketId,
		RefundTo:            refundTo,
		MaxRefund:           maxRefund,
		SubmissionFeeRefund: submissionFeeRefund,
	}, nil
}

func RetryableEscrowAddress(ticketId common.Hash) common.Address {
	return common.BytesToAddress(crypto.Keccak256([]byte("retryable escrow"), ticketId.Bytes()))
}

func RetryableSubmissionFee(calldataLengthInBytes int, l1BaseFee *big.Int) *big.Int {
	return arbmath.BigMulByUint(l1BaseFee, uint64(1400+6*calldataLengthInBytes))
}
