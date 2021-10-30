//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package rentableStorage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"testing"
)

func TestEmptyRentableStorage(t *testing.T) {
	sto := storage.NewMemoryBacked()
	InitializeRentableStorage(sto)
	rs := OpenRentableStorage(sto)
	timestamp := uint64(1)

	bin := rs.OpenBin(common.Address{}, common.Hash{}, timestamp)
	if bin != nil {
		t.Fatal()
	}
}

func TestAllocDeallocRentableBin(t *testing.T) {
	sto := storage.NewMemoryBacked()
	InitializeRentableStorage(sto)
	rs := OpenRentableStorage(sto)
	timestamp := uint64(1)
	owner := common.BytesToAddress([]byte{3, 1, 4, 1, 5, 9})
	binId1 := crypto.Keccak256Hash([]byte{1})
	binId2 := crypto.Keccak256Hash([]byte{2})

	rs.AllocateBin(owner, binId1, timestamp)
	if !rs.BinExists(owner, binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(owner, binId1, timestamp) == nil {
		t.Fatal()
	}

	rs.AllocateBin(owner, binId2, timestamp)
	if !rs.BinExists(owner, binId2, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(owner, binId2, timestamp) == nil {
		t.Fatal()
	}
	if !rs.BinExists(owner, binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(owner, binId1, timestamp) == nil {
		t.Fatal()
	}

	rs.OpenBin(owner, binId1, timestamp).Delete()
	if rs.BinExists(owner, binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(owner, binId1, timestamp) != nil {
		t.Fatal()
	}
	if !rs.BinExists(owner, binId2, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(owner, binId2, timestamp) == nil {
		t.Fatal()
	}

	rs.AllocateBin(owner, binId1, timestamp)
	if !rs.BinExists(owner, binId1, timestamp) {
		t.Fatal()
	}
}

func TestRentableStorageSlot(t *testing.T) {
	sto := storage.NewMemoryBacked()
	InitializeRentableStorage(sto)
	rs := OpenRentableStorage(sto)
	timestamp := uint64(1)
	binId := crypto.Keccak256Hash([]byte{1})
	slotId := crypto.Keccak256Hash([]byte{1, 1})
	owner := common.BytesToAddress([]byte{9, 8, 7, 6, 5})
	buf1 := []byte{3, 1, 4, 1, 5, 9, 2, 6, 3, 7}
	buf2 := []byte{2, 7, 1}

	rs.AllocateBin(owner, binId, timestamp)
	bin := rs.OpenBin(owner, binId, timestamp)

	if len(bin.GetSlot(slotId)) != 0 {
		t.Fatal()
	}

	bin.SetSlot(slotId, buf1)
	if len(bin.GetSlot(slotId)) != len(buf1) {
		t.Fatal()
	}
	if bin.GetSlotDataSize(slotId) != uint64(len(buf1)) {
		t.Fatal()
	}

	bin.SetSlot(slotId, buf2)
	if len(bin.GetSlot(slotId)) != len(buf2) {
		t.Fatal()
	}
	if bin.GetSlotDataSize(slotId) != uint64(len(buf2)) {
		t.Fatal()
	}
}

func TestRentableExpiration(t *testing.T) {
	sto := storage.NewMemoryBacked()
	InitializeRentableStorage(sto)
	rs := OpenRentableStorage(sto)
	initialTimestamp := uint64(1)
	muchLaterTimestamp := initialTimestamp + RentableStorageLifetimeSeconds + 2
	owner := common.BytesToAddress([]byte{9, 8, 7, 6, 5})
	binId := crypto.Keccak256Hash([]byte{1})
	slotId := crypto.Keccak256Hash([]byte{1, 1})
	buf := []byte{3, 1, 4, 1, 5, 9, 2, 6, 3, 7}

	rs.AllocateBin(owner, binId, initialTimestamp)
	bin := rs.OpenBin(owner, binId, initialTimestamp)
	renewCostForEmptyBin := bin.GetRenewGas()

	bin.SetSlot(slotId, buf)

	if !rs.BinExists(owner, binId, initialTimestamp) {
		t.Fatal()
	}

	bin = rs.OpenBin(owner, binId, muchLaterTimestamp)
	if bin != nil {
		t.Fatal()
	}
	if rs.BinExists(owner, binId, muchLaterTimestamp) {
		t.Fatal()
	}

	rs.AllocateBin(owner, binId, muchLaterTimestamp)
	bin = rs.OpenBin(owner, binId, muchLaterTimestamp)
	if !rs.BinExists(owner, binId, muchLaterTimestamp) {
		t.Fatal()
	}

	// make sure the slot was deleted when the bin timed out
	if bin.GetRenewGas() != renewCostForEmptyBin {
		t.Fatal()
	}

	if len(bin.GetSlot(slotId)) != 0 {
		t.Fatal(len(bin.GetSlot(slotId)))
	}
}
