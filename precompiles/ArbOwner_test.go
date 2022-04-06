// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/util"
)

func TestAddressSet(t *testing.T) {
	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	prec := &ArbOwner{}
	callCtx := testContext(caller, evm)

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
