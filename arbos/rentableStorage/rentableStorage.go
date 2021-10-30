//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package rentableStorage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

const RentableStorageLifetimeSeconds = 7 * 24 * 60 * 60

var (
	binsStorageKey  = []byte{0}
	timeoutQueueKey = []byte{1}
)

type RentableStorage struct {
	binsStorage  *storage.Storage
	timeoutQueue *storage.Queue
}

func InitializeRentableStorage(sto *storage.Storage) {
	storage.InitializeQueue(sto.OpenSubStorage(timeoutQueueKey))
}

func OpenRentableStorage(sto *storage.Storage) *RentableStorage {
	return &RentableStorage{
		sto.OpenSubStorage(binsStorageKey),
		storage.OpenQueue(sto.OpenSubStorage(timeoutQueueKey)),
	}
}

const (
	timeoutOffset  = 0
	numBytesOffset = 1

	renewChargePer32Bytes = params.SstoreSetGas / 100
	renewChargePerBin     = 4 * renewChargePer32Bytes
	renewChargePerSlot    = 2 * renewChargePer32Bytes
	renewChargePerByte    = (renewChargePer32Bytes + 31) / 32
)

var (
	memberSetKey = []byte{0}
	contentsKey  = []byte{1}
)

type RentableBin struct {
	parent         *RentableStorage
	backingStorage *storage.Storage
	binId          common.Hash
	timeout        uint64
	totalBytes     uint64
	memberSet      *storage.HashSet
	slotsStorage   *storage.Storage
}

func (rs *RentableStorage) AllocateBin(binId common.Hash, currentTimestamp uint64) *RentableBin {
	rs.TryToReapOneBin(currentTimestamp)
	if rs.BinExists(binId, currentTimestamp) {
		rs.OpenBin(binId, currentTimestamp).Delete()
	}
	rs.timeoutQueue.Put(binId)

	backingStorage := rs.binsStorage.OpenSubStorage(binId.Bytes())
	timeout := currentTimestamp + RentableStorageLifetimeSeconds
	backingStorage.SetByInt64(timeoutOffset, util.IntToHash(int64(timeout)))
	backingStorage.SetByInt64(numBytesOffset, common.Hash{})
	memberSetStorage := backingStorage.OpenSubStorage(memberSetKey)
	storage.InitializeHashSet(memberSetStorage)
	return &RentableBin{
		rs,
		backingStorage,
		binId,
		timeout,
		0,
		storage.OpenHashSet(memberSetStorage),
		backingStorage.OpenSubStorage(contentsKey),
	}
}

func (rs *RentableStorage) OpenBin(binId common.Hash, currentTimestamp uint64) *RentableBin {
	backingStorage := rs.binsStorage.OpenSubStorage(binId.Bytes())
	ret := &RentableBin{
		rs,
		backingStorage,
		binId,
		backingStorage.GetByInt64(timeoutOffset).Big().Uint64(),
		backingStorage.GetByInt64(numBytesOffset).Big().Uint64(),
		storage.OpenHashSet(backingStorage.OpenSubStorage(memberSetKey)),
		backingStorage.OpenSubStorage(contentsKey),
	}
	if ret.timeout < currentTimestamp {
		ret.Delete()
		return nil
	}
	return ret
}

func (rs *RentableStorage) BinExists(binId common.Hash, currentTimestamp uint64) bool {
	binTimestamp := rs.binsStorage.OpenSubStorage(binId.Bytes()).GetByInt64(timeoutOffset).Big().Uint64()
	if binTimestamp == 0 {
		return false
	}
	if binTimestamp < currentTimestamp {
		_ = rs.OpenBin(binId, currentTimestamp) // this will delete the bin
		return false
	}
	return true
}

func (rs *RentableStorage) TryToReapOneBin(currentTimestamp uint64) {
	binId := rs.timeoutQueue.Get()
	if binId != nil && rs.BinExists(*binId, currentTimestamp) {
		rs.timeoutQueue.Put(*binId)
	}
}

func (bin *RentableBin) Delete() {
	backingStorage := bin.backingStorage
	for _, member := range bin.memberSet.AllMembers() {
		bin.DeleteSlot(member)
	}
	backingStorage.SetByInt64(timeoutOffset, common.Hash{})
	backingStorage.SetByInt64(numBytesOffset, common.Hash{})
}

func (bin *RentableBin) GetTimeout() uint64 {
	return bin.timeout
}

func (bin *RentableBin) CanBeRenewedNow(currentTimestamp uint64) bool {
	return bin.timeout < currentTimestamp+RentableStorageLifetimeSeconds
}

func (bin *RentableBin) GetRenewGas() uint64 {
	numSlots := bin.memberSet.Size()
	numBytes := bin.totalBytes
	return renewChargePerBin + numSlots*renewChargePerSlot + numBytes*renewChargePerByte
}

func (bin *RentableBin) Renew(currentTimestamp uint64) {
	if bin.CanBeRenewedNow(currentTimestamp) {
		bin.timeout += RentableStorageLifetimeSeconds
		bin.backingStorage.SetByInt64(timeoutOffset, util.IntToHash(int64(bin.timeout)))
	}
}

func (bin *RentableBin) storageForSlot(slotId common.Hash) *storage.Storage {
	return bin.slotsStorage.OpenSubStorage(slotId.Bytes())
}

func (bin *RentableBin) GetSlot(slotId common.Hash) []byte {
	return bin.storageForSlot(slotId).GetBytes()
}

func (bin *RentableBin) GetSlotDataSize(slotId common.Hash) uint64 {
	return bin.storageForSlot(slotId).GetBytesSize()
}

func (bin *RentableBin) SetSlot(slotId common.Hash, value []byte) {
	bin.DeleteSlot(slotId)
	if len(value) > 0 {
		bin.storageForSlot(slotId).WriteBytes(value)
		bin.modifyStorageCount(int64(len(value)))
		bin.memberSet.Add(slotId)
	}
}

func (bin *RentableBin) DeleteSlot(slotId common.Hash) {
	thisSlotStorage := bin.storageForSlot(slotId)
	thisSlotBytes := thisSlotStorage.GetBytesSize()
	thisSlotStorage.DeleteBytes()
	bin.modifyStorageCount(-int64(thisSlotBytes))
	bin.memberSet.Remove(slotId)
}

func (bin *RentableBin) modifyStorageCount(bytesDelta int64) {
	binStorage := bin.backingStorage
	bin.totalBytes = uint64(int64(bin.totalBytes) + bytesDelta)
	binStorage.SetByInt64(numBytesOffset, util.IntToHash(int64(bin.totalBytes)))
}