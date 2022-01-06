//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/util"
)

func TestAddressSet(t *testing.T) {
	evm := newMockEVMForTesting(t)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	arbosState.OpenArbosState(evm.StateDB).ChainOwners().Add(caller)

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	prec := &ArbOwner{}
	callCtx := testContext(caller)

	// the zero address is an owner by default
	ZeroAddressL2 := util.RemapL1Address(common.Address{})
	Require(t, prec.RemoveChainOwner(callCtx, evm, ZeroAddressL2))

	Require(t, prec.AddChainOwner(callCtx, evm, addr1))
	Require(t, prec.AddChainOwner(callCtx, evm, addr2))
	Require(t, prec.AddChainOwner(callCtx, evm, addr1))

	member, err := prec.IsChainOwner(callCtx, evm, addr1)
	Require(t, err)
	if !member {
		t.Fatal()
	}

	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	Require(t, err)
	if !member {
		t.Fatal()
	}

	member, err = prec.IsChainOwner(callCtx, evm, addr3)
	Require(t, err)
	if member {
		t.Fatal()
	}

	Require(t, prec.RemoveChainOwner(callCtx, evm, addr1))
	member, err = prec.IsChainOwner(callCtx, evm, addr1)
	Require(t, err)
	if member {
		t.Fatal()
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	Require(t, err)
	if !member {
		t.Fatal()
	}

	Require(t, prec.AddChainOwner(callCtx, evm, addr1))
	all, err := prec.GetAllChainOwners(callCtx, evm)
	Require(t, err)
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
