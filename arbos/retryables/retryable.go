//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package retryables

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
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
	numTries       *big.Int
	timeout        *uint64
	from           *common.Address
	to             *util.OptionAddress
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
	fmt.Println("Created retryable with ID ", id)
	rs.TryToReapOneRetryable(currentTimestamp)
	sto := rs.retryables.OpenSubStorage(id.Bytes())
	ret := &Retryable{
		id,
		sto,
		big.NewInt(0),
		&timeout,
		&from,
		util.NewOptionAddress(to),
		callvalue,
		&beneficiary,
		calldata,
	}
	sto.SetByUint64(numTriesOffset, common.Hash{})
	sto.SetByUint64(timeoutOffset, util.IntToHash(int64(timeout)))
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
	if sto.GetByUint64(timeoutOffset) == (common.Hash{}) {
		// no retryable here (real retryable never has a zero timeout)
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
	return 6*32 + retryable.CalldataSize()
}

func (rs *RetryableState) DeleteRetryable(id common.Hash) bool {
	retStorage := rs.retryables.OpenSubStorage(id.Bytes())
	if retStorage.GetByUint64(timeoutOffset) == (common.Hash{}) {
		return false
	}
	retStorage.SetByUint64(numTriesOffset, common.Hash{})
	retStorage.SetByUint64(timeoutOffset, common.Hash{})
	retStorage.SetByUint64(fromOffset, common.Hash{})
	retStorage.SetByUint64(toOffset, common.Hash{})
	retStorage.SetByUint64(callvalueOffset, common.Hash{})
	retStorage.SetByUint64(beneficiaryOffset, common.Hash{})
	retStorage.OpenSubStorage(calldataKey).DeleteBytes()
	return true
}

func (retryable *Retryable) NumTries() *big.Int {
	if retryable.numTries == nil {
		retryable.numTries = retryable.backingStorage.GetByUint64(numTriesOffset).Big()
	}
	return retryable.numTries
}

func (retryable *Retryable) SetNumTries(nt *big.Int) {
	retryable.numTries = nt
	retryable.backingStorage.SetByUint64(numTriesOffset, common.BigToHash(nt))
}

func (retryable *Retryable) IncrementNumTries() *big.Int {
	seqNum := retryable.NumTries()
	incr := new(big.Int).Add(seqNum, big.NewInt(1))
	retryable.SetNumTries(incr)
	return seqNum
}

func TxIdForRedeemAttempt(ticketId common.Hash, trySequenceNum *big.Int) common.Hash {
	// zero byte is included to prevent collision with a txId used by the Arbitrum Classic retryables API
	return crypto.Keccak256Hash(ticketId.Bytes(), []byte{0}, trySequenceNum.Bytes())
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
		t := retryable.backingStorage.GetByUint64(timeoutOffset).Big().Uint64()
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
		retryable.to = util.OptionAddressFromHash(retryable.backingStorage.GetByUint64(toOffset))
	}
	return retryable.to.ToAddressRef()
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
	retryable.backingStorage.SetByUint64(timeoutOffset, util.IntToHash(int64(timeout)))
}

func (rs *RetryableState) Keepalive(ticketId common.Hash, currentTimestamp, limitBeforeAdd, timeToAdd uint64) bool {
	retryable := rs.OpenRetryable(ticketId, currentTimestamp)
	if retryable == nil {
		return false
	}
	timeout := retryable.Timeout()
	if timeout > limitBeforeAdd {
		return false
	}
	retryable.SetTimeout(timeout + timeToAdd)
	return true
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

func (rs *RetryableState) MakeRetryTx(retry QueuedRetry, currentTimestamp uint64, chainId *big.Int, gasPrice *big.Int) *types.Transaction {
	retryable := rs.OpenRetryable(retry.TicketId, currentTimestamp)
	if retryable == nil {
		return nil
	}
	return types.NewTx(&types.ArbitrumRetryTx{
		ArbitrumContractTx: types.ArbitrumContractTx{
			ChainId:   chainId,
			RequestId: retry.RetryId,
			From:      retryable.From(),
			GasPrice:  gasPrice,
			Gas:       retry.Gas.Uint64(),
			To:        retryable.To(),
			Value:     retryable.Callvalue(),
			Data:      retryable.Calldata(),
		},
		TicketId: retry.TicketId,
		RefundTo: retry.RefundTo,
	})
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

type QueuedRetry struct {
	TicketId common.Hash
	RetryId  common.Hash
	SeqNum   uint64
	Gas      *big.Int
	RefundTo common.Address
}
