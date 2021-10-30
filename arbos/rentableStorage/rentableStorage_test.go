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

	bin := rs.OpenBin(common.Hash{}, timestamp)
	if bin != nil {
		t.Fatal()
	}
}

func TestAllocDeallocRentableBin(t *testing.T) {
	sto := storage.NewMemoryBacked()
	InitializeRentableStorage(sto)
	rs := OpenRentableStorage(sto)
	timestamp := uint64(1)
	binId1 := crypto.Keccak256Hash([]byte{1})
	binId2 := crypto.Keccak256Hash([]byte{2})

	rs.AllocateBin(binId1, timestamp)
	if !rs.BinExists(binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(binId1, timestamp) == nil {
		t.Fatal()
	}

	rs.AllocateBin(binId2, timestamp)
	if !rs.BinExists(binId2, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(binId2, timestamp) == nil {
		t.Fatal()
	}
	if !rs.BinExists(binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(binId1, timestamp) == nil {
		t.Fatal()
	}

	rs.OpenBin(binId1, timestamp).Delete()
	if rs.BinExists(binId1, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(binId1, timestamp) != nil {
		t.Fatal()
	}
	if !rs.BinExists(binId2, timestamp) {
		t.Fatal()
	}
	if rs.OpenBin(binId2, timestamp) == nil {
		t.Fatal()
	}

	rs.AllocateBin(binId1, timestamp)
	if !rs.BinExists(binId1, timestamp) {
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
	buf1 := []byte{3, 1, 4, 1, 5, 9, 2, 6, 3, 7}
	buf2 := []byte{2, 7, 1}

	rs.AllocateBin(binId, timestamp)
	bin := rs.OpenBin(binId, timestamp)

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
	binId := crypto.Keccak256Hash([]byte{1})
	slotId := crypto.Keccak256Hash([]byte{1, 1})
	buf := []byte{3, 1, 4, 1, 5, 9, 2, 6, 3, 7}

	rs.AllocateBin(binId, initialTimestamp)
	bin := rs.OpenBin(binId, initialTimestamp)
	renewCostForEmptyBin := bin.GetRenewGas()

	bin.SetSlot(slotId, buf)

	if !rs.BinExists(binId, initialTimestamp) {
		t.Fatal()
	}

	bin = rs.OpenBin(binId, muchLaterTimestamp)
	if bin != nil {
		t.Fatal()
	}
	if rs.BinExists(binId, muchLaterTimestamp) {
		t.Fatal()
	}

	rs.AllocateBin(binId, muchLaterTimestamp)
	bin = rs.OpenBin(binId, muchLaterTimestamp)
	if !rs.BinExists(binId, muchLaterTimestamp) {
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
