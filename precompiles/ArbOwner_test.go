// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestArbOwner(t *testing.T) {
	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	prec := &ArbOwner{}
	gasInfo := &ArbGasInfo{}
	callCtx := testContext(caller, evm)

	// the zero address is an owner by default
	Require(t, prec.RemoveChainOwner(callCtx, evm, common.Address{}))

	Require(t, prec.AddChainOwner(callCtx, evm, addr1))
	Require(t, prec.AddChainOwner(callCtx, evm, addr2))
	Require(t, prec.AddChainOwner(callCtx, evm, addr1))

	member, err := prec.IsChainOwner(callCtx, evm, addr1)
	Require(t, err)
	if !member {
		Fail(t)
	}

	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	Require(t, err)
	if !member {
		Fail(t)
	}

	member, err = prec.IsChainOwner(callCtx, evm, addr3)
	Require(t, err)
	if member {
		Fail(t)
	}

	Require(t, prec.RemoveChainOwner(callCtx, evm, addr1))
	member, err = prec.IsChainOwner(callCtx, evm, addr1)
	Require(t, err)
	if member {
		Fail(t)
	}
	member, err = prec.IsChainOwner(callCtx, evm, addr2)
	Require(t, err)
	if !member {
		Fail(t)
	}

	Require(t, prec.AddChainOwner(callCtx, evm, addr1))
	all, err := prec.GetAllChainOwners(callCtx, evm)
	Require(t, err)
	if len(all) != 3 {
		Fail(t)
	}
	if all[0] == all[1] || all[1] == all[2] || all[0] == all[2] {
		Fail(t)
	}
	if all[0] != addr1 && all[1] != addr1 && all[2] != addr1 {
		Fail(t)
	}
	if all[0] != addr2 && all[1] != addr2 && all[2] != addr2 {
		Fail(t)
	}
	if all[0] != caller && all[1] != caller && all[2] != caller {
		Fail(t)
	}

	costCap, err := gasInfo.GetAmortizedCostCapBips(callCtx, evm)
	Require(t, err)
	if costCap != 0 {
		Fail(t, costCap)
	}
	newCostCap := uint64(77734)
	Require(t, prec.SetAmortizedCostCapBips(callCtx, evm, newCostCap))
	costCap, err = gasInfo.GetAmortizedCostCapBips(callCtx, evm)
	Require(t, err)
	if costCap != newCostCap {
		Fail(t)
	}

	avail, err := gasInfo.GetL1FeesAvailable(callCtx, evm)
	Require(t, err)
	if avail.Sign() != 0 {
		Fail(t, avail)
	}
	deposited := big.NewInt(1000000)
	evm.StateDB.AddBalance(l1pricing.L1PricerFundsPoolAddress, deposited)
	avail, err = gasInfo.GetL1FeesAvailable(callCtx, evm)
	Require(t, err)
	if avail.Sign() != 0 {
		Fail(t, avail)
	}
	requested := big.NewInt(200000)
	x, err := prec.ReleaseL1PricerSurplusFunds(callCtx, evm, requested)
	Require(t, err)
	if x.Cmp(requested) != 0 {
		Fail(t, x, requested)
	}
	avail, err = gasInfo.GetL1FeesAvailable(callCtx, evm)
	Require(t, err)
	if avail.Cmp(requested) != 0 {
		Fail(t, avail, requested)
	}
	x, err = prec.ReleaseL1PricerSurplusFunds(callCtx, evm, deposited)
	Require(t, err)
	if x.Cmp(new(big.Int).Sub(deposited, requested)) != 0 {
		Fail(t, x, deposited, requested)
	}
	avail, err = gasInfo.GetL1FeesAvailable(callCtx, evm)
	Require(t, err)
	if avail.Cmp(deposited) != 0 {
		Fail(t, avail, deposited)
	}
	x, err = prec.ReleaseL1PricerSurplusFunds(callCtx, evm, deposited)
	Require(t, err)
	if x.Sign() != 0 {
		Fail(t, x)
	}
	avail, err = gasInfo.GetL1FeesAvailable(callCtx, evm)
	Require(t, err)
	if avail.Cmp(deposited) != 0 {
		Fail(t, avail, deposited)
	}
}

func TestArbOwnerSetChainConfig(t *testing.T) {
	evm := newMockEVMForTestingWithVersionAndRunMode(nil, core.MessageGasEstimationMode)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))
	prec := &ArbOwner{}
	callCtx := testContext(caller, evm)

	chainConfig := params.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams.AllowDebugPrecompiles = false
	serializedChainConfig, err := json.Marshal(chainConfig)
	Require(t, err)
	err = prec.SetChainConfig(callCtx, evm, serializedChainConfig)
	Require(t, err)
	config, err := state.ChainConfig()
	Require(t, err)
	if !bytes.Equal(config, serializedChainConfig) {
		Fail(t, config, serializedChainConfig)
	}

	chainConfig.ArbitrumChainParams.AllowDebugPrecompiles = true
	serializedChainConfig, err = json.Marshal(chainConfig)
	Require(t, err)
	err = prec.SetChainConfig(callCtx, evm, serializedChainConfig)
	Require(t, err)
	config, err = state.ChainConfig()
	Require(t, err)
	if !bytes.Equal(config, serializedChainConfig) {
		Fail(t, config, serializedChainConfig)
	}
}

func TestArbInfraFeeAccount(t *testing.T) {
	version0 := uint64(0)
	evm := newMockEVMForTestingWithVersion(&version0)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	newAddr := common.BytesToAddress(crypto.Keccak256([]byte{0})[:20])
	callCtx := testContext(caller, evm)
	prec := &ArbOwner{}
	_, err := prec.GetInfraFeeAccount(callCtx, evm)
	Require(t, err)
	err = prec.SetInfraFeeAccount(callCtx, evm, newAddr) // this should be a no-op (because ArbOS version 0)
	Require(t, err)

	version5 := uint64(5)
	evm = newMockEVMForTestingWithVersion(&version5)
	callCtx = testContext(caller, evm)
	prec = &ArbOwner{}
	precPublic := &ArbOwnerPublic{}
	addr, err := prec.GetInfraFeeAccount(callCtx, evm)
	Require(t, err)
	if addr != (common.Address{}) {
		t.Fatal()
	}
	addr, err = precPublic.GetInfraFeeAccount(callCtx, evm)
	Require(t, err)
	if addr != (common.Address{}) {
		t.Fatal()
	}

	err = prec.SetInfraFeeAccount(callCtx, evm, newAddr)
	Require(t, err)
	addr, err = prec.GetInfraFeeAccount(callCtx, evm)
	Require(t, err)
	if addr != newAddr {
		t.Fatal()
	}
	addr, err = precPublic.GetInfraFeeAccount(callCtx, evm)
	Require(t, err)
	if addr != newAddr {
		t.Fatal()
	}
}
