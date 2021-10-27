//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos"
	"testing"
)

func TestAddressSet(t *testing.T) {
	evm := newMockEVMForTesting(t)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	arbos.OpenArbosState(evm.StateDB).ChainOwners().Add(caller)

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	prec := &ArbOwner{}
	callCtx := testContext(caller)

	if err := prec.AddChainOwner(callCtx, evm, addr1); err != nil {
		t.Fatal(err)
	}
	if err := prec.AddChainOwner(callCtx, evm, addr2); err != nil {
		t.Fatal(err)
	}
	if err := prec.AddChainOwner(callCtx, evm, addr1); err != nil {
		t.Fatal(err)
	}
	member, err := prec.IsChainOwner(callCtx, evm, addr1)
	if err != nil {
		t.Fatal(err)
	}
	if !member {
		t.Fatal()
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	if err != nil {
		t.Fatal(err)
	}
	if !member {
		t.Fatal()
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr3)
	if err != nil {
		t.Fatal(err)
	}
	if member {
		t.Fatal()
	}

	if err := prec.RemoveChainOwner(callCtx, evm, addr1); err != nil {
		t.Fatal(err)
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr1)
	if err != nil {
		t.Fatal(err)
	}
	if member {
		t.Fatal()
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	if err != nil {
		t.Fatal(err)
	}
	if !member {
		t.Fatal()
	}

	if err := prec.AddChainOwner(callCtx, evm, addr1); err != nil {
		t.Fatal(err)
	}
	all, err := prec.GetAllChainOwners(callCtx, evm)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatal()
	}
	if all[0] == all[1] || all[1] == all[2] || all[0] == all[2] {
		t.Fatal()
	}
	if all[0] != addr1 && all[1] != addr1 && all[2] != addr1 {
		t.Fatal()
	}
	if all[0] != addr2 && all[1] != addr2 && all[2] != addr2 {
		t.Fatal()
	}
	if all[0] != caller && all[1] != caller && all[2] != caller {
		t.Fatal()
	}
}
