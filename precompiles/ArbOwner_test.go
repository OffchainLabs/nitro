// Copyright 2021-2022, Offchain Labs, Inc.
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

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
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
	evm.StateDB.AddBalance(l1pricing.L1PricerFundsPoolAddress, uint256.MustFromBig(deposited), tracing.BalanceChangeUnspecified)
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
	if err := params.Save(true); err != nil {
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

// setupArbOwnerTestWithRunMode creates an EVM with the given run context,
// opens ArbOS state, adds a chain owner, and returns everything needed
// for testing ArbOwner precompile methods.
func setupArbOwnerTestWithRunMode(t *testing.T, runCtx *core.MessageRunContext) (*vm.EVM, *arbosState.ArbosState, *Context, *ArbOwner) {
	t.Helper()
	evm := newMockEVMForTestingWithVersionAndRunMode(nil, runCtx)
	caller := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tracer := util.NewTracingInfo(evm, testhelpers.RandomAddress(), types.ArbosAddress, util.TracingDuringEVM)
	state, err := arbosState.OpenArbosState(evm.StateDB, burn.NewSystemBurner(tracer, false))
	Require(t, err)
	Require(t, state.ChainOwners().Add(caller))
	callCtx := testContext(caller, evm)
	return evm, state, callCtx, &ArbOwner{}
}

func TestSaveReallyStoreTrue(t *testing.T) {
	// When reallyStore=true, Save should actually persist params to storage
	_, state, _, _ := setupArbOwnerTestWithRunMode(t, core.NewMessageCommitContext(nil))

	p, err := state.Programs().Params()
	Require(t, err)
	originalInkPrice := p.InkPrice

	// Modify and save with reallyStore=true
	newInkPrice := originalInkPrice + 1000
	p.InkPrice = newInkPrice
	Require(t, p.Save(true))

	// Read back — should see the new value
	p2, err := state.Programs().Params()
	Require(t, err)
	if p2.InkPrice != newInkPrice {
		Fail(t, "Expected InkPrice", newInkPrice, "got", p2.InkPrice)
	}
}

func TestSaveReallyStoreFalse(t *testing.T) {
	// When reallyStore=false, Save should NOT persist params, but should not error
	_, state, _, _ := setupArbOwnerTestWithRunMode(t, core.NewMessageEthcallContext())

	p, err := state.Programs().Params()
	Require(t, err)
	originalInkPrice := p.InkPrice

	// Modify and save with reallyStore=false
	p.InkPrice = originalInkPrice + 5000
	Require(t, p.Save(false))

	// Read back — should still see the original value
	p2, err := state.Programs().Params()
	Require(t, err)
	if p2.InkPrice != originalInkPrice {
		Fail(t, "Expected InkPrice to remain", originalInkPrice, "got", p2.InkPrice)
	}
}

func TestSaveReallyStoreFalseMultipleFields(t *testing.T) {
	// Verify that Save(false) does not persist any field changes
	_, state, _, _ := setupArbOwnerTestWithRunMode(t, core.NewMessageEthcallContext())

	p, err := state.Programs().Params()
	Require(t, err)
	origInkPrice := p.InkPrice
	origFreePages := p.FreePages
	origPageGas := p.PageGas
	origPageLimit := p.PageLimit
	origExpiryDays := p.ExpiryDays

	// Modify multiple fields
	p.InkPrice = origInkPrice + 1
	p.FreePages = origFreePages + 10
	p.PageGas = origPageGas + 100
	p.PageLimit = origPageLimit + 50
	p.ExpiryDays = origExpiryDays + 100

	Require(t, p.Save(false))

	// All fields should remain at original values
	p2, err := state.Programs().Params()
	Require(t, err)
	if p2.InkPrice != origInkPrice {
		Fail(t, "InkPrice changed: expected", origInkPrice, "got", p2.InkPrice)
	}
	if p2.FreePages != origFreePages {
		Fail(t, "FreePages changed: expected", origFreePages, "got", p2.FreePages)
	}
	if p2.PageGas != origPageGas {
		Fail(t, "PageGas changed: expected", origPageGas, "got", p2.PageGas)
	}
	if p2.PageLimit != origPageLimit {
		Fail(t, "PageLimit changed: expected", origPageLimit, "got", p2.PageLimit)
	}
	if p2.ExpiryDays != origExpiryDays {
		Fail(t, "ExpiryDays changed: expected", origExpiryDays, "got", p2.ExpiryDays)
	}
}

func TestArbOwnerSetInkPriceOnChain(t *testing.T) {
	// When executed on-chain (commit mode), SetInkPrice should persist the change
	evm, state, callCtx, prec := setupArbOwnerTestWithRunMode(t, core.NewMessageCommitContext(nil))

	Require(t, prec.SetInkPrice(callCtx, evm, 20000))

	p, err := state.Programs().Params()
	Require(t, err)
	if uint32(p.InkPrice) != 20000 {
		Fail(t, "Expected InkPrice 20000, got", p.InkPrice)
	}
}

func TestArbOwnerSetInkPriceEthCall(t *testing.T) {
	// When executed via eth_call, SetInkPrice should NOT persist the change
	evm, state, callCtx, prec := setupArbOwnerTestWithRunMode(t, core.NewMessageEthcallContext())

	p, err := state.Programs().Params()
	Require(t, err)
	originalInkPrice := p.InkPrice

	Require(t, prec.SetInkPrice(callCtx, evm, 20000))

	// Ink price should NOT have been persisted
	p2, err := state.Programs().Params()
	Require(t, err)
	if p2.InkPrice != originalInkPrice {
		Fail(t, "InkPrice should not change during eth_call: expected", originalInkPrice, "got", p2.InkPrice)
	}
}

func TestArbOwnerSetInkPriceGasEstimation(t *testing.T) {
	// When executed via gas estimation, SetInkPrice should NOT persist the change
	evm, state, callCtx, prec := setupArbOwnerTestWithRunMode(t, core.NewMessageGasEstimationContext())

	p, err := state.Programs().Params()
	Require(t, err)
	originalInkPrice := p.InkPrice

	Require(t, prec.SetInkPrice(callCtx, evm, 20000))

	// Ink price should NOT have been persisted
	p2, err := state.Programs().Params()
	Require(t, err)
	if p2.InkPrice != originalInkPrice {
		Fail(t, "InkPrice should not change during gas estimation: expected", originalInkPrice, "got", p2.InkPrice)
	}
}

func TestArbOwnerSettersNotPersistedOffChain(t *testing.T) {
	// Verify that ALL ArbOwner Stylus setters do not persist in eth_call mode
	evm, state, callCtx, prec := setupArbOwnerTestWithRunMode(t, core.NewMessageEthcallContext())

	// Get original params
	orig, err := state.Programs().Params()
	Require(t, err)

	// Call all setters with new values
	Require(t, prec.SetInkPrice(callCtx, evm, 20000))
	Require(t, prec.SetWasmMaxStackDepth(callCtx, evm, 999999))
	Require(t, prec.SetWasmFreePages(callCtx, evm, 10))
	Require(t, prec.SetWasmPageGas(callCtx, evm, 5000))
	Require(t, prec.SetWasmPageLimit(callCtx, evm, 256))
	Require(t, prec.SetWasmMinInitGas(callCtx, evm, 20000, 1000))
	Require(t, prec.SetWasmInitCostScalar(callCtx, evm, 80))
	Require(t, prec.SetWasmExpiryDays(callCtx, evm, 730))
	Require(t, prec.SetWasmKeepaliveDays(callCtx, evm, 60))
	Require(t, prec.SetWasmBlockCacheSize(callCtx, evm, 64))
	Require(t, prec.SetWasmMaxSize(callCtx, evm, 256*1024))

	// All params should still be at original values
	after, err := state.Programs().Params()
	Require(t, err)

	if after.InkPrice != orig.InkPrice {
		Fail(t, "InkPrice changed off-chain")
	}
	if after.MaxStackDepth != orig.MaxStackDepth {
		Fail(t, "MaxStackDepth changed off-chain")
	}
	if after.FreePages != orig.FreePages {
		Fail(t, "FreePages changed off-chain")
	}
	if after.PageGas != orig.PageGas {
		Fail(t, "PageGas changed off-chain")
	}
	if after.PageLimit != orig.PageLimit {
		Fail(t, "PageLimit changed off-chain")
	}
	if after.MinInitGas != orig.MinInitGas {
		Fail(t, "MinInitGas changed off-chain")
	}
	if after.InitCostScalar != orig.InitCostScalar {
		Fail(t, "InitCostScalar changed off-chain")
	}
	if after.ExpiryDays != orig.ExpiryDays {
		Fail(t, "ExpiryDays changed off-chain")
	}
	if after.KeepaliveDays != orig.KeepaliveDays {
		Fail(t, "KeepaliveDays changed off-chain")
	}
	if after.BlockCacheSize != orig.BlockCacheSize {
		Fail(t, "BlockCacheSize changed off-chain")
	}
	if after.MaxWasmSize != orig.MaxWasmSize {
		Fail(t, "MaxWasmSize changed off-chain")
	}
}

func TestArbOwnerSettersPersistedOnChain(t *testing.T) {
	// Verify that ALL ArbOwner Stylus setters DO persist in commit mode
	evm, state, callCtx, prec := setupArbOwnerTestWithRunMode(t, core.NewMessageCommitContext(nil))

	// Call all setters with new values
	Require(t, prec.SetInkPrice(callCtx, evm, 20000))
	Require(t, prec.SetWasmMaxStackDepth(callCtx, evm, 999999))
	Require(t, prec.SetWasmFreePages(callCtx, evm, 10))
	Require(t, prec.SetWasmPageGas(callCtx, evm, 5000))
	Require(t, prec.SetWasmPageLimit(callCtx, evm, 256))
	Require(t, prec.SetWasmMinInitGas(callCtx, evm, 20000, 1000))
	Require(t, prec.SetWasmInitCostScalar(callCtx, evm, 80))
	Require(t, prec.SetWasmExpiryDays(callCtx, evm, 730))
	Require(t, prec.SetWasmKeepaliveDays(callCtx, evm, 60))
	Require(t, prec.SetWasmBlockCacheSize(callCtx, evm, 64))
	Require(t, prec.SetWasmMaxSize(callCtx, evm, 256*1024))

	// All params should be at new values
	after, err := state.Programs().Params()
	Require(t, err)

	if uint32(after.InkPrice) != 20000 {
		Fail(t, "InkPrice not persisted on-chain: got", after.InkPrice)
	}
	if after.MaxStackDepth != 999999 {
		Fail(t, "MaxStackDepth not persisted on-chain: got", after.MaxStackDepth)
	}
	if after.FreePages != 10 {
		Fail(t, "FreePages not persisted on-chain: got", after.FreePages)
	}
	if after.PageGas != 5000 {
		Fail(t, "PageGas not persisted on-chain: got", after.PageGas)
	}
	if after.PageLimit != 256 {
		Fail(t, "PageLimit not persisted on-chain: got", after.PageLimit)
	}
	if after.ExpiryDays != 730 {
		Fail(t, "ExpiryDays not persisted on-chain: got", after.ExpiryDays)
	}
	if after.KeepaliveDays != 60 {
		Fail(t, "KeepaliveDays not persisted on-chain: got", after.KeepaliveDays)
	}
	if after.BlockCacheSize != 64 {
		Fail(t, "BlockCacheSize not persisted on-chain: got", after.BlockCacheSize)
	}
	if after.MaxWasmSize != 256*1024 {
		Fail(t, "MaxWasmSize not persisted on-chain: got", after.MaxWasmSize)
	}
}
