// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package addressSet

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestEmptyAddressSet(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)
	version := params.ArbitrumDevTestParams().InitialArbOSVersion

	if size(t, aset) != 0 {
		Fail(t)
	}
	if isMember(t, aset, common.Address{}) {
		Fail(t)
	}
	err := aset.Remove(common.Address{}, version)
	Require(t, err)
	if size(t, aset) != 0 {
		Fail(t)
	}
	if isMember(t, aset, common.Address{}) {
		Fail(t)
	}
}

func TestAddressSet(t *testing.T) {
	db := storage.NewMemoryBackedStateDB()
	sto := storage.NewGeth(db, burn.NewSystemBurner(nil, false))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)
	version := params.ArbitrumDevTestParams().InitialArbOSVersion

	statedb, _ := (db).(*state.StateDB)
	stateHashBeforeChanges := statedb.IntermediateRoot(false)

	addr1 := testhelpers.RandomAddress()
	addr2 := testhelpers.RandomAddress()
	addr3 := testhelpers.RandomAddress()
	possibleAddresses := []common.Address{addr1, addr2, addr3}

	Require(t, aset.Add(addr1))
	if size(t, aset) != 1 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Add(addr2))
	if size(t, aset) != 2 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Add(addr1))
	if size(t, aset) != 2 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)
	if !isMember(t, aset, addr1) {
		Fail(t)
	}
	if !isMember(t, aset, addr2) {
		Fail(t)
	}
	if isMember(t, aset, addr3) {
		Fail(t)
	}

	Require(t, aset.Remove(addr1, version))
	if size(t, aset) != 1 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)
	if isMember(t, aset, addr1) {
		Fail(t)
	}
	if !isMember(t, aset, addr2) {
		Fail(t)
	}

	Require(t, aset.Add(addr3))
	if size(t, aset) != 2 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Remove(addr3, version))
	if size(t, aset) != 1 {
		Fail(t)
	}
	checkAllMembers(t, aset, possibleAddresses)

	Require(t, aset.Add(addr1))
	all, err := aset.AllMembers(math.MaxUint64)
	Require(t, err)
	if len(all) != 2 {
		Fail(t)
	}
	if all[0] == addr1 {
		if all[1] != addr2 {
			Fail(t)
		}
	} else {
		if (all[0] != addr2) || (all[1] != addr1) {
			Fail(t)
		}
	}

	stateHashAfterChanges := statedb.IntermediateRoot(false)
	Require(t, aset.Clear())
	stateHashAfterClear := statedb.IntermediateRoot(false)

	colors.PrintBlue("prior ", stateHashBeforeChanges)
	colors.PrintGrey("after ", stateHashAfterChanges)
	colors.PrintBlue("clear ", stateHashAfterClear)

	if stateHashAfterClear != stateHashBeforeChanges {
		Fail(t, "Clear() left data in the statedb")
	}
	if stateHashAfterChanges == stateHashBeforeChanges {
		Fail(t, "set-operations didn't change the underlying statedb")
	}
}

func TestAddressSetAllMembers(t *testing.T) {
	db := storage.NewMemoryBackedStateDB()
	sto := storage.NewGeth(db, burn.NewSystemBurner(nil, false))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)
	version := params.ArbitrumDevTestParams().InitialArbOSVersion

	addr1 := testhelpers.RandomAddress()
	addr2 := testhelpers.RandomAddress()
	addr3 := testhelpers.RandomAddress()
	possibleAddresses := []common.Address{addr1, addr2, addr3}

	Require(t, aset.Add(addr1))
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Add(addr2))
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Remove(addr1, version))
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Add(addr3))
	checkAllMembers(t, aset, possibleAddresses)
	Require(t, aset.Remove(addr2, version))
	checkAllMembers(t, aset, possibleAddresses)

	for i := 0; i < 512; i++ {
		rem := rand.Intn(2) == 1
		addr := possibleAddresses[rand.Intn(len(possibleAddresses))]
		if rem {
			fmt.Printf("removing %v\n", addr)
			Require(t, aset.Remove(addr, version))
		} else {
			fmt.Printf("adding %v\n", addr)
			Require(t, aset.Add(addr))
		}
		checkAllMembers(t, aset, possibleAddresses)
	}
}

func TestRectifyMappingAgainstHistory(t *testing.T) {
	db := storage.NewMemoryBackedStateDB()
	sto := storage.NewGeth(db, burn.NewSystemBurner(nil, false))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)
	version := uint64(10)

	// Test Nova history
	addr1 := common.HexToAddress("0x9C040726F2A657226Ed95712245DeE84b650A1b5")
	addr2 := common.HexToAddress("0xd345e41ae2cb00311956aa7109fc801ae8c81a52")
	addr3 := common.HexToAddress("0xd0749b3e537ed52de4e6a3ae1eb6fc26059d0895")
	addr4 := common.HexToAddress("0x86a02dd71363c440b21f4c0e5b2ad01ffe1a7482")
	// Follow logs
	Require(t, aset.Add(addr1))
	Require(t, aset.Add(addr2))
	Require(t, aset.Remove(addr1, version))
	Require(t, aset.Add(addr3))
	Require(t, aset.Add(addr4))
	Require(t, aset.Remove(addr2, version))
	Require(t, aset.Remove(addr3, version))
	// Check if history's correct
	CurrentOwner, _ := aset.backingStorage.GetByUint64(uint64(1))
	isOwner, _ := aset.IsMember(addr2)
	correctOwner, _ := aset.IsMember(addr4)
	if size(t, aset) != uint64(1) || CurrentOwner != common.BytesToHash(addr2.Bytes()) || isOwner || !correctOwner {
		Fail(t, "Logs and current state did not match")
	}
	// Run RectifyMapping to fix the issue
	checkIfRectifyMappingWorks(t, aset, []common.Address{addr4}, true)
	Require(t, aset.Clear())

	// Test Arb1 history
	addr1 = common.HexToAddress("0xd345e41ae2cb00311956aa7109fc801ae8c81a52")
	addr2 = common.HexToAddress("0x98e4db7e07e584f89a2f6043e7b7c89dc27769ed")
	addr3 = common.HexToAddress("0xcf57572261c7c2bcf21ffd220ea7d1a27d40a827")
	// Follow logs
	Require(t, aset.Add(addr1))
	Require(t, aset.Add(addr2))
	Require(t, aset.Add(addr3))
	Require(t, aset.Remove(addr1, version))
	Require(t, aset.Remove(addr2, version))
	// Check if history's correct
	CurrentOwner, _ = aset.backingStorage.GetByUint64(uint64(1))
	correctOwner, _ = aset.IsMember(addr3)
	index, _ := aset.byAddress.GetUint64(common.BytesToHash(addr3.Bytes()))
	if size(t, aset) != uint64(1) || index == 1 || CurrentOwner != common.BytesToHash(addr3.Bytes()) || !correctOwner {
		Fail(t, "Logs and current state did not match")
	}
	// Run RectifyMapping to fix the issue
	checkIfRectifyMappingWorks(t, aset, []common.Address{addr3}, true)
	Require(t, aset.Clear())

	// Test Goerli history
	addr1 = common.HexToAddress("0x186B56023d42B2B4E7616589a5C62EEf5FCa21DD")
	addr2 = common.HexToAddress("0xc8efdb677afeb775ce1617dd976b56b3a6e95bba")
	addr3 = common.HexToAddress("0xc3f86bb81e32295d29c288ffb4828936538cf326")
	addr4 = common.HexToAddress("0x67acb531a05160a81dcd03079347f264c4fa2da3")
	// Follow logs
	Require(t, aset.Add(addr1))
	Require(t, aset.Add(addr2))
	Require(t, aset.Add(addr3))
	Require(t, aset.Remove(addr1, version))
	Require(t, aset.Add(addr4))
	Require(t, aset.Remove(addr3, version))
	Require(t, aset.Remove(addr2, version))
	// Check if history's correct
	CurrentOwner, _ = aset.backingStorage.GetByUint64(uint64(1))
	isOwner, _ = aset.IsMember(addr3)
	correctOwner, _ = aset.IsMember(addr4)
	if size(t, aset) != uint64(1) || CurrentOwner != common.BytesToHash(addr3.Bytes()) || isOwner || !correctOwner {
		Fail(t, "Logs and current state did not match")
	}
	// Run RectifyMapping to fix the issue
	checkIfRectifyMappingWorks(t, aset, []common.Address{addr4}, true)
}

func TestRectifyMapping(t *testing.T) {
	db := storage.NewMemoryBackedStateDB()
	sto := storage.NewGeth(db, burn.NewSystemBurner(nil, false))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)

	addr1 := testhelpers.RandomAddress()
	addr2 := testhelpers.RandomAddress()
	addr3 := testhelpers.RandomAddress()
	possibleAddresses := []common.Address{addr1, addr2, addr3}

	Require(t, aset.Add(addr1))
	Require(t, aset.Add(addr2))
	Require(t, aset.Add(addr3))

	// Non owner's should not be able to call RectifyMapping
	err := aset.RectifyMapping(testhelpers.RandomAddress())
	if err == nil {
		Fail(t, "RectifyMapping was succesfully called by non owner")
	}

	// Corrupt the list and verify if RectifyMapping fixes it
	addrHash := common.BytesToHash(addr2.Bytes())
	Require(t, aset.backingStorage.SetByUint64(uint64(1), addrHash))
	checkIfRectifyMappingWorks(t, aset, possibleAddresses, true)

	// Corrupt the map and verify if RectifyMapping fixes it
	addrHash = common.BytesToHash(addr2.Bytes())
	Require(t, aset.byAddress.Set(addrHash, util.UintToHash(uint64(6))))
	checkIfRectifyMappingWorks(t, aset, possibleAddresses, true)

	// Add a new owner to the map and verify if RectifyMapping syncs list with the map
	// to check for the case where list has fewer owners than expected
	addr4 := testhelpers.RandomAddress()
	addrHash = common.BytesToHash(addr4.Bytes())
	Require(t, aset.byAddress.Set(addrHash, util.UintToHash(uint64(1))))
	checkIfRectifyMappingWorks(t, aset, possibleAddresses, true)

	// RectifyMapping should not do anything if the mapping is correct
	// Check to verify functionality post fix
	err = aset.RectifyMapping(addr1)
	if err == nil {
		Fail(t, "RectifyMapping called by a correctly mapped owner")
	}

}

func checkIfRectifyMappingWorks(t *testing.T, aset *AddressSet, owners []common.Address, clearList bool) {
	t.Helper()
	if clearList {
		Require(t, aset.ClearList())
	}
	for index, owner := range owners {
		Require(t, aset.RectifyMapping(owner))

		addrAsHash := common.BytesToHash(owner.Bytes())
		slot, err := aset.byAddress.GetUint64(addrAsHash)
		Require(t, err)
		atSlot, err := aset.backingStorage.GetByUint64(slot)
		Require(t, err)
		if slot == 0 || atSlot != addrAsHash {
			Fail(t, "RectifyMapping did not fix the mismatch")
		}

		if clearList && int(size(t, aset)) != index+1 {
			Fail(t, "RectifyMapping did not fix the mismatch")
		}
	}
	allMembers, err := aset.AllMembers(size(t, aset))
	Require(t, err)
	less := func(a, b common.Address) bool { return a.String() < b.String() }
	if cmp.Diff(owners, allMembers, cmpopts.SortSlices(less)) != "" {
		Fail(t, "RectifyMapping did not fix the mismatch")
	}
}

func checkAllMembers(t *testing.T, aset *AddressSet, possibleAddresses []common.Address) {
	allMembers, err := aset.AllMembers(1024)
	Require(t, err)

	allMembersSet := make(map[common.Address]struct{})
	for _, addr := range allMembers {
		allMembersSet[addr] = struct{}{}
	}

	if len(allMembers) != len(allMembersSet) {
		Fail(t, "allMembers contains duplicates:", allMembers)
	}

	possibleAddressSet := make(map[common.Address]struct{})
	for _, addr := range possibleAddresses {
		possibleAddressSet[addr] = struct{}{}
	}
	for _, addr := range allMembers {
		_, isPossible := possibleAddressSet[addr]
		if !isPossible {
			Fail(t, "allMembers contains impossible address", addr)
		}
	}

	for _, possible := range possibleAddresses {
		isMember, err := aset.IsMember(possible)
		Require(t, err)
		_, inSet := allMembersSet[possible]
		if isMember != inSet {
			Fail(t, "IsMember", isMember, "does not match whether it's in the allMembers list", inSet)
		}
	}
}

func isMember(t *testing.T, aset *AddressSet, address common.Address) bool {
	t.Helper()
	present, err := aset.IsMember(address)
	Require(t, err)
	return present
}

func size(t *testing.T, aset *AddressSet) uint64 {
	t.Helper()
	size, err := aset.Size()
	Require(t, err)
	return size
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
