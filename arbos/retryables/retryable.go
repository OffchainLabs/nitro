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
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

// set by the precompile module, to avoid a package dependence cycle

const RetryableLifetimeSeconds = 7 * 24 * 60 * 60 // one week
const RetryableReapPrice = 58000
const ExpiredSnapshotsCapacity = 2
const ExpiredSnapshotsRotationIntervalSeconds = 30 * 60 // 30 minutes

var ErrTicketNotFound = errors.New("ticketId not found")
var ErrInvalidRoot = errors.New("invalid root hash")
var ErrWrongProof = errors.New("wrong proof")
var ErrAlreadyRevived = errors.New("already revived")

type RetryableState struct {
	retryables       *storage.Storage
	TimeoutQueue     *storage.Queue
	Expired          *merkleAccumulator.MerkleAccumulator
	expiredSnapshots *storage.RingBuffer
	revived          *storage.Uint64Set
}

var (
	timeoutQueueKey     = []byte{0}
	calldataKey         = []byte{1}
	archiveKey          = []byte{2}
	expiredSnapshotsKey = []byte{3}
	revivedKey          = []byte{4}
)

func InitializeRetryableState(sto *storage.Storage) error {
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenSubStorage(archiveKey))
	storage.InitializeUint64Set(sto.OpenSubStorage(revivedKey))
	if err := storage.InitializeQueue(sto.OpenSubStorage(timeoutQueueKey)); err != nil {
		return err
	}
	return storage.InitializeRingBuffer(sto.OpenSubStorage(expiredSnapshotsKey), ExpiredSnapshotsCapacity)
}

func OpenRetryableState(sto *storage.Storage) *RetryableState {
	return &RetryableState{
		sto,
		storage.OpenQueue(sto.OpenSubStorage(timeoutQueueKey)),
		merkleAccumulator.OpenMerkleAccumulator(sto.OpenSubStorage(archiveKey)),
		storage.OpenRingBuffer(sto.OpenSubStorage(expiredSnapshotsKey)),
		storage.OpenUint64Set(sto.OpenSubStorage(revivedKey)),
	}
}

type Retryable struct {
	id                 common.Hash // not backed by storage; this key determines where it lives in storage
	backingStorage     *storage.Storage
	numTries           storage.StorageBackedUint64
	from               storage.StorageBackedAddress
	to                 storage.StorageBackedAddressOrNil
	callValue          storage.StorageBackedBigUint
	beneficiary        storage.StorageBackedAddress
	callData           storage.StorageBackedBytes
	timeout            storage.StorageBackedUint64
	timeoutWindowsLeft storage.StorageBackedUint64
}

type TestRetryableData struct {
	Id          common.Hash
	NumTries    uint64
	From        common.Address
	To          common.Address
	CallValue   *big.Int
	Beneficiary common.Address
	CallData    []byte
}

func (r *TestRetryableData) Hash() common.Hash {
	return RetryableHash(r.Id, r.NumTries, r.From, r.To, r.CallValue, r.Beneficiary, r.CallData)
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
	callValue *big.Int,
	beneficiary common.Address,
	callData []byte,
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
	_ = ret.callValue.SetChecked(callValue)
	_ = ret.beneficiary.Set(beneficiary)
	_ = ret.callData.Set(callData)
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
		callValue:          sto.OpenStorageBackedBigUint(callvalueOffset),
		beneficiary:        sto.OpenStorageBackedAddress(beneficiaryOffset),
		callData:           sto.OpenStorageBackedBytes(calldataKey),
		timeout:            timeoutStorage,
		timeoutWindowsLeft: sto.OpenStorageBackedUint64(timeoutWindowsLeftOffset),
	}, nil
}

func (rs *RetryableState) OpenPotentialyExpiredRetryable(id common.Hash) *Retryable {
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	return &Retryable{
		id:                 id,
		backingStorage:     sto,
		numTries:           sto.OpenStorageBackedUint64(numTriesOffset),
		from:               sto.OpenStorageBackedAddress(fromOffset),
		to:                 sto.OpenStorageBackedAddressOrNil(toOffset),
		callValue:          sto.OpenStorageBackedBigUint(callvalueOffset),
		beneficiary:        sto.OpenStorageBackedAddress(beneficiaryOffset),
		callData:           sto.OpenStorageBackedBytes(calldataKey),
		timeout:            sto.OpenStorageBackedUint64(timeoutOffset),
		timeoutWindowsLeft: sto.OpenStorageBackedUint64(timeoutWindowsLeftOffset),
	}
}

func CalculateRetryableSizeWords(calldataSize uint64) uint64 {
	callData := 32 + 32*arbmath.WordsForBytes(calldataSize) // length + contents
	return arbmath.WordsForBytes(6*32 + callData)
}

func (rs *RetryableState) RetryableSizeWords(id common.Hash, currentTime uint64) (uint64, error) {
	retryable, err := rs.OpenRetryable(id, currentTime)
	if retryable == nil || err != nil {
		return 0, err
	}
	size, err := retryable.CalldataSize()
	return CalculateRetryableSizeWords(size), err
}

func clearRetryable(retStorage *storage.Storage) error {
	_ = retStorage.ClearByUint64(numTriesOffset)
	_ = retStorage.ClearByUint64(fromOffset)
	_ = retStorage.ClearByUint64(toOffset)
	_ = retStorage.ClearByUint64(callvalueOffset)
	_ = retStorage.ClearByUint64(beneficiaryOffset)
	_ = retStorage.ClearByUint64(timeoutOffset)
	_ = retStorage.ClearByUint64(timeoutWindowsLeftOffset)
	return retStorage.OpenSubStorage(calldataKey).ClearBytes()
}

func (rs *RetryableState) DeleteRetryable(id common.Hash, evm *vm.EVM, scenario util.TracingScenario) (bool, error) {
	retStorage := rs.retryables.OpenSubStorage(id.Bytes())
	timeout, err := retStorage.GetByUint64(timeoutOffset)
	if timeout == (common.Hash{}) || err != nil {
		return false, err
	}

	// move any funds in escrow to the beneficiary (should be none if the retry succeeded -- see EndTxHook)
	beneficiary, _ := retStorage.GetByUint64(beneficiaryOffset)
	beneficiaryAddress := common.BytesToAddress(beneficiary[:])
	escrowAddress := RetryableEscrowAddress(id)
	amount := evm.StateDB.GetBalance(escrowAddress)
	if err := util.TransferBalance(&escrowAddress, &beneficiaryAddress, amount, evm, scenario, "escrow"); err != nil {
		return false, err
	}

	return true, clearRetryable(retStorage)
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
	return retryable.callValue.Get()
}

func (retryable *Retryable) Calldata() ([]byte, error) {
	return retryable.callData.Get()
}

// CalldataSize efficiently gets size of callData without loading all of it
func (retryable *Retryable) CalldataSize() (uint64, error) {
	return retryable.callData.Size()
}

func RetryableHash(id common.Hash, numTries uint64, from, to common.Address, callValue *big.Int, beneficiary common.Address, callData []byte) common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		id.Bytes(),
		arbmath.UintToBig(numTries).Bytes(),
		from.Bytes(),
		to.Bytes(),
		callValue.Bytes(),
		beneficiary.Bytes(),
		callData,
	))
}

func (retryable *Retryable) GetHash() (common.Hash, error) {
	numTries, err := retryable.numTries.Get()
	if err != nil {
		return common.Hash{}, err
	}
	from, err := retryable.from.Get()
	if err != nil {
		return common.Hash{}, err
	}
	to, err := retryable.to.Get()
	if err != nil {
		return common.Hash{}, err
	}
	if to == nil {
		nilAddr := common.BytesToAddress(storage.NilAddressRepresentation.Bytes())
		to = &nilAddr
	}
	callValue, err := retryable.callValue.Get()
	if err != nil {
		return common.Hash{}, err
	}
	beneficiary, err := retryable.beneficiary.Get()
	if err != nil {
		return common.Hash{}, err
	}
	callData, err := retryable.callData.Get()
	if err != nil {
		return common.Hash{}, err
	}
	return RetryableHash(retryable.id, numTries, from, *to, callValue, beneficiary, callData), nil
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
		return 0, ErrTicketNotFound
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

func (rs *RetryableState) Revive(
	ticketId common.Hash,
	numTries uint64,
	from common.Address,
	to common.Address,
	callValue *big.Int,
	beneficiary common.Address,
	callData []byte,
	rootHash common.Hash,
	leafIndex uint64,
	proof []common.Hash,
	currentTimestamp,
	timeToAdd uint64,
) (uint64, error) {
	var found bool
	err := rs.expiredSnapshots.ForEach(func(_ uint64, previousRoot common.Hash) (bool, error) {
		if bytes.Equal(rootHash.Bytes(), previousRoot.Bytes()) {
			found = true
		}
		return !found, nil
	})
	if err != nil {
		return 0, err
	}
	if !found {
		currentRoot, err := rs.Expired.Root()
		if err != nil {
			return 0, err
		}
		if !bytes.Equal(rootHash.Bytes(), currentRoot.Bytes()) {
			return 0, ErrInvalidRoot
		}
	}
	retryableHash := RetryableHash(ticketId, numTries, from, to, callValue, beneficiary, callData)
	merkleProof := merkletree.MerkleProof{
		RootHash:  rootHash,
		LeafHash:  common.BytesToHash(crypto.Keccak256(retryableHash.Bytes())),
		LeafIndex: leafIndex,
		Proof:     proof,
	}
	if !merkleProof.IsCorrect() {
		return 0, ErrWrongProof
	}
	inserted, err := rs.revived.Add(leafIndex)
	if err != nil {
		return 0, err
	}
	if !inserted {
		return 0, ErrAlreadyRevived
	}
	ret := rs.OpenPotentialyExpiredRetryable(ticketId)
	timeout, err := ret.timeout.Get()
	if err != nil {
		return 0, err
	}
	// TODO(magic) do we want to skip this check?
	if timeout != 0 {
		// shouldn't ever happen
		return 0, ErrAlreadyRevived
	}
	newTimeout := currentTimestamp + timeToAdd
	if err = ret.numTries.Set(numTries); err != nil {
		return 0, err
	}
	if err = ret.from.Set(from); err != nil {
		return 0, err
	}
	if err = ret.to.Set(&to); err != nil {
		return 0, err
	}
	if err = ret.callValue.SetChecked(callValue); err != nil {
		return 0, err
	}
	if err = ret.beneficiary.Set(beneficiary); err != nil {
		return 0, err
	}
	if err = ret.callData.Set(callData); err != nil {
		return 0, err
	}
	if err = ret.timeout.Set(newTimeout); err != nil {
		return 0, err
	}
	if err = ret.timeoutWindowsLeft.Set(0); err != nil {
		return 0, err
	}
	if err = rs.TimeoutQueue.Put(ret.id); err != nil {
		return 0, err
	}
	// Pay in advance for the work needed to reap the retryable from the timeout queue
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

type ExpiredRetryableLeaf struct {
	TicketId common.Hash
	NumTries uint64
	Index    uint64
	Hash     common.Hash
}

func (rs *RetryableState) TryToReapOneRetryable(currentTimestamp uint64, evm *vm.EVM, scenario util.TracingScenario) ([]merkleAccumulator.MerkleTreeNodeEvent, *ExpiredRetryableLeaf, error) {
	id, err := rs.TimeoutQueue.Peek()
	if err != nil || id == nil {
		return nil, nil, err
	}
	retryableStorage := rs.retryables.OpenSubStorage(id.Bytes())
	timeoutStorage := retryableStorage.OpenStorageBackedUint64(timeoutOffset)
	timeout, err := timeoutStorage.Get()
	if err != nil {
		return nil, nil, err
	}
	if timeout == 0 {
		// The retryable has already been deleted, so discard the peeked entry
		_, err = rs.TimeoutQueue.Get()
		return nil, nil, err
	}

	windowsLeftStorage := retryableStorage.OpenStorageBackedUint64(timeoutWindowsLeftOffset)
	windowsLeft, err := windowsLeftStorage.Get()
	// TODO(magic) why can't we check the second cond earlier?
	if err != nil || timeout >= currentTimestamp {
		return nil, nil, err
	}

	// Either the retryable has expired, or it's lost a lifetime's worth of time
	_, err = rs.TimeoutQueue.Get()
	if err != nil {
		return nil, nil, err
	}

	if windowsLeft == 0 {
		// the retryable has expired, time to reap
		expired := rs.OpenPotentialyExpiredRetryable(*id)
		numTries, err := expired.NumTries()
		if err != nil {
			return nil, nil, err
		}
		expiredHash, err := expired.GetHash()
		if err != nil {
			return nil, nil, err
		}
		if err = clearRetryable(retryableStorage); err != nil {
			return nil, nil, err
		}
		merkleUpdateEvents, accumulatorSize, err := rs.Expired.Append(expiredHash)
		if err != nil {
			return nil, nil, err
		}
		return merkleUpdateEvents, &ExpiredRetryableLeaf{TicketId: *id, NumTries: numTries, Index: accumulatorSize - 1, Hash: common.BytesToHash(crypto.Keccak256(expiredHash.Bytes()))}, nil
	}

	// Consume a window, delaying the timeout one lifetime period
	if err := timeoutStorage.Set(timeout + RetryableLifetimeSeconds); err != nil {
		return nil, nil, err
	}
	return nil, nil, windowsLeftStorage.Set(windowsLeft - 1)
}

func (rs *RetryableState) TryRotatingExpiredRootSnapshots(currentTime uint64) (*common.Hash, error) {
	var rotatedRoot *common.Hash
	err := rs.expiredSnapshots.RotateAndSetExtraConditionaly(func(timestamp uint64) (bool, common.Hash, uint64, error) {
		var err error
		if timestamp < currentTime && currentTime-timestamp > ExpiredSnapshotsRotationIntervalSeconds {
			var root common.Hash
			root, err = rs.Expired.Root()
			if err == nil {
				rotatedRoot = &root
				return true, root, currentTime, nil
			}
		}
		return false, common.Hash{}, 0, err
	})
	if err != nil {
		return nil, err
	}
	return rotatedRoot, nil
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
	callValue, err := retryable.Callvalue()
	if err != nil {
		return nil, err
	}
	callData, err := retryable.Calldata()
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
		Value:               callValue,
		Data:                callData,
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
