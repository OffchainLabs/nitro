//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package rentableStorage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

const RentableStorageLifetimeSeconds = 7 * 24 * 60 * 60

type RentableStorage struct {
	backingStorage *storage.Storage
}

func InitializeRentableStorage(sto *storage.Storage) {
	// no initialization needed
}

func OpenRentableStorage(sto *storage.Storage) *RentableStorage {
	return &RentableStorage{sto}
}

const (
	timestampOffset = 0
	numSlotsOffset  = 1
	numBytesOffset  = 2

	renewChargePer32Bytes = params.SstoreSetGas / 100
	renewChargePerBin     = 4 * renewChargePer32Bytes
	renewChargePerSlot    = 2 * renewChargePer32Bytes
	renewChargePerByte    = 1 + (renewChargePer32Bytes / 32)
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
	slots          uint64
	nbytes         uint64
	memberSet      *storage.HashSet
}

func (rs *RentableStorage) AllocateBin(binId common.Hash, currentTimestamp uint64) *RentableBin {
	if rs.BinExists(binId) {
		rs.OpenBin(binId, currentTimestamp).Delete()
	}

	backingStorage := rs.backingStorage.OpenSubStorage(binId.Bytes())
	timeout := currentTimestamp + RentableStorageLifetimeSeconds
	backingStorage.SetByInt64(0, util.IntToHash(int64(timeout)))
	memberSetStorage := backingStorage.OpenSubStorage(memberSetKey)
	storage.InitializeHashSet(memberSetStorage)
	return &RentableBin{
		rs,
		backingStorage,
		binId,
		timeout,
		0,
		0,
		storage.OpenHashSet(memberSetStorage),
	}
}

func (rs *RentableStorage) OpenBin(binId common.Hash, currentTimestamp uint64) *RentableBin {
	backingStorage := rs.backingStorage.OpenSubStorage(binId.Bytes())
	ret := &RentableBin{
		rs,
		backingStorage,
		binId,
		backingStorage.GetByInt64(0).Big().Uint64(),
		backingStorage.GetByInt64(1).Big().Uint64(),
		backingStorage.GetByInt64(2).Big().Uint64(),
		storage.OpenHashSet(backingStorage.OpenSubStorage(memberSetKey)),
	}
	if ret.timeout < currentTimestamp {
		ret.Delete()
		return nil
	}
	return ret
}

func (rs *RentableStorage) BinExists(binId common.Hash) bool {
	return rs.backingStorage.OpenSubStorage(binId.Bytes()).GetByInt64(timestampOffset).Big().Uint64() != 0
}

func (bin *RentableBin) GetTimeout() *big.Int {
	return bin.backingStorage.GetByInt64(0).Big()
}

func (bin *RentableBin) GetRenewGas() *big.Int {
	numSlots := bin.backingStorage.GetByInt64(numSlotsOffset).Big().Uint64()
	numBytes := bin.backingStorage.GetByInt64(numBytesOffset).Big().Uint64()
	return big.NewInt(int64(renewChargePerBin + numSlots*renewChargePerSlot + numBytes*renewChargePerByte))
}

func (bin *RentableBin) GetSlot(slotId common.Hash) []byte {
	return bin.backingStorage.OpenSubStorage(slotId.Bytes()).GetBytes()
}

func (bin *RentableBin) GetSlotDataSize(slotId common.Hash) uint64 {
	return bin.backingStorage.OpenSubStorage(slotId.Bytes()).GetBytesSize(false)
}

func (bin *RentableBin) SetSlot(slotId common.Hash, value []byte) {
	bin.DeleteSlot(slotId)
	bin.backingStorage.OpenSubStorage(slotId.Bytes()).WriteBytes(value)
	bin.modifyStorageCount(1, int64(len(value)))
}

func (bin *RentableBin) Delete() {
	backingStorage := bin.backingStorage
	memberSet := storage.OpenHashSet(backingStorage.OpenSubStorage(memberSetKey))
	for _, member := range memberSet.AllMembers() {
		bin.DeleteSlot(member)
	}
	backingStorage.SetByInt64(timestampOffset, common.Hash{})
}

func (bin *RentableBin) DeleteSlot(slotId common.Hash) {
	thisSlotStorage := bin.backingStorage.OpenSubStorage(contentsKey).OpenSubStorage(slotId.Bytes())
	thisSlotBytes := thisSlotStorage.GetBytesSize(true)
	thisSlotStorage.DeleteBytes()
	bin.modifyStorageCount(-1, -int64(thisSlotBytes))
}

func (bin *RentableBin) modifyStorageCount(slotsDelta int64, bytesDelta int64) {
	binStorage := bin.backingStorage
	binStorage.SetByInt64(numSlotsOffset, util.IntToHash(binStorage.GetByInt64(numSlotsOffset).Big().Int64()+slotsDelta))
	binStorage.SetByInt64(numBytesOffset, util.IntToHash(binStorage.GetByInt64(numBytesOffset).Big().Int64()+bytesDelta))
}