// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package addressSet

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
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
