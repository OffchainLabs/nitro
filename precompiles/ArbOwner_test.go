// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
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
	evm.StateDB.AddBalance(types.L1PricerFundsPoolAddress, uint256.MustFromBig(deposited), tracing.BalanceChangeUnspecified)
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

	err = prec.SetNetworkFeeAccount(callCtx, evm, addr1)
	Require(t, err)
	retrievedNetworkFeeAccount, err := prec.GetNetworkFeeAccount(callCtx, evm)
	Require(t, err)
	if retrievedNetworkFeeAccount.Cmp(addr1) != 0 {
		Fail(t, "Expected", addr1, "got", retrievedNetworkFeeAccount)
	}

	l2BaseFee := big.NewInt(123)
	err = prec.SetL2BaseFee(callCtx, evm, l2BaseFee)
	Require(t, err)
	retrievedL2BaseFee, err := state.L2PricingState().BaseFeeWei()
	Require(t, err)
	if l2BaseFee.Cmp(retrievedL2BaseFee) != 0 {
		Fail(t, "Expected", l2BaseFee, "got", retrievedL2BaseFee)
	}

	params, err := state.Programs().Params()
	Require(t, err)
	maxWasmSize := params.MaxWasmSize
	want := 128 * 1024 // Initial maxWasmSize
	if maxWasmSize != uint32(want) {
		Fail(t, "Got", maxWasmSize, "want", want)
	}

	want = 256 * 1024
	params.MaxWasmSize = uint32(want)
	if err := params.Save(); err != nil {
		Fail(t, err)
	}
	params, err = state.Programs().Params()
	Require(t, err)
	maxWasmSize = params.MaxWasmSize
	if maxWasmSize != uint32(want) {
		Fail(t, "Got", maxWasmSize, "want", want)
	}

	pubPrec := &ArbOwnerPublic{}

	cdpi, err := pubPrec.IsCalldataPriceIncreaseEnabled(callCtx, evm)
	Require(t, err)
	if cdpi {
		Fail(t)
	}
	err = prec.SetCalldataPriceIncrease(callCtx, evm, true)
	Require(t, err)
	cdpi, err = pubPrec.IsCalldataPriceIncreaseEnabled(callCtx, evm)
	Require(t, err)
	if !cdpi {
		Fail(t)
	}
}

func TestArbWasmActivationGas(t *testing.T) {
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams.InitialArbOSVersion = params.ArbosVersion_60
	evm := newMockEVMForTestingWithConfigs(chainConfig, chainConfig)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))

	arbOwner := &ArbOwner{}
	arbWasm := &ArbWasm{}
	callCtx := testContext(caller, evm)

	// default is zero
	gas, err := arbWasm.ActivationGas(callCtx, evm)
	Require(t, err)
	if gas != 0 {
		Fail(t, "expected default activation gas 0, got", gas)
	}

	// set and read back
	const testGas = uint64(5_000_000)
	Require(t, arbOwner.SetWasmActivationGas(callCtx, evm, testGas))

	gas, err = arbWasm.ActivationGas(callCtx, evm)
	Require(t, err)
	if gas != testGas {
		Fail(t, "expected activation gas", testGas, "got", gas)
	}
}

func TestArbOwnerSetChainConfig(t *testing.T) {
	evm := newMockEVMForTestingWithVersionAndRunMode(nil, core.NewMessageGasEstimationContext())
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))
	prec := &ArbOwner{}
	callCtx := testContext(caller, evm)

	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
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

func TestDisableOffchainArbOwner(t *testing.T) {
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()

	makeWrapper := func() *OwnerPrecompile {
		_, inner := MakePrecompile(precompilesgen.ArbOwnerMetaData, &ArbOwner{Address: types.ArbOwnerAddress})
		return &OwnerPrecompile{
			precompile:  inner,
			emitSuccess: func(evm mech, method bytes4, owner addr, data []byte) error { return nil },
		}
	}

	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	input := make([]byte, 4)
	expectedErr := "ArbOwner precompile is disabled outside on-chain execution"

	callWrapper := func(wrapper *OwnerPrecompile, evm mech) (retErr error) {
		defer func() {
			// Recover from panics deeper in Call (e.g. incomplete mock state).
			// We only care whether the disableOffchain guard returned its error.
			if r := recover(); r != nil {
				retErr = nil // not the disable error
			}
		}()
		_, _, _, err := wrapper.Call(input, caller, caller, common.Big0, false, 1000000, evm)
		return err
	}

	makeEVM := func(runCtx *core.MessageRunContext) mech {
		evm := newMockEVMForTestingWithConfigs(chainConfig, chainConfig)
		evm.ProcessingHook = arbos.NewTxProcessor(evm, &core.Message{TxRunContext: runCtx})
		return evm
	}

	// Flag enabled: ethcall and gas estimation should return the disable error
	wrapper := makeWrapper()
	wrapper.SetDisableOffchain(true)
	for _, tc := range []struct {
		name string
		evm  mech
	}{
		{"ethcall", makeEVM(core.NewMessageEthcallContext())},
		{"gas estimation", makeEVM(core.NewMessageGasEstimationContext())},
	} {
		err := callWrapper(wrapper, tc.evm)
		if err == nil || err.Error() != expectedErr {
			t.Fatalf("%s: expected error %q, got %v", tc.name, expectedErr, err)
		}
	}

	// Flag enabled: commit and replay should NOT return the disable error
	for _, tc := range []struct {
		name string
		evm  mech
	}{
		{"commit", makeEVM(core.NewMessageCommitContext(nil))},
		{"replay", makeEVM(core.NewMessageReplayContext())},
	} {
		err := callWrapper(wrapper, tc.evm)
		if err != nil && err.Error() == expectedErr {
			t.Fatalf("%s: should not return disable error for on-chain context", tc.name)
		}
	}

	// Flag enabled but ProcessingHook is not a TxProcessor: should return error
	evmNoHook := vm.NewEVM(vm.BlockContext{
		BlockNumber: big.NewInt(0),
		GasLimit:    ^uint64(0),
	}, nil, chainConfig, vm.Config{})
	err := callWrapper(wrapper, evmNoHook)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("no TxProcessor hook: expected error %q, got %v", expectedErr, err)
	}

	// Flag disabled (default): ethcall should NOT return the disable error
	wrapperDisabled := makeWrapper()
	err = callWrapper(wrapperDisabled, makeEVM(core.NewMessageEthcallContext()))
	if err != nil && err.Error() == expectedErr {
		t.Fatalf("ethcall with flag disabled: should not return disable error")
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

	version5 := params.ArbosVersion_5
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
