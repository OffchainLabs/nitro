// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func newMockEVMForTestingWithCurrentRefundTo(currentRefundTo *common.Address) *vm.EVM {
	evm := newMockEVMForTesting()
	txProcessor := arbos.NewTxProcessor(evm, &core.Message{})
	txProcessor.CurrentRefundTo = currentRefundTo
	evm.ProcessingHook = txProcessor
	return evm
}

func TestGetCurrentRedeemer(t *testing.T) {
	currentRefundTo := common.HexToAddress("0x030405")

	evm := newMockEVMForTestingWithCurrentRefundTo(&currentRefundTo)
	retryableTx := ArbRetryableTx{}
	context := testContext(common.Address{}, evm)

	currentRedeemer, err := retryableTx.GetCurrentRedeemer(context, evm)
	Require(t, err)
	if currentRefundTo.Cmp(currentRedeemer) != 0 {
		t.Fatal("Expected to be ", currentRefundTo, " but got ", currentRedeemer)
	}
}

func testRetryableRedeem(t *testing.T, evm *vm.EVM, precompileCtx *Context) {
	t.Helper()

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := evm.Context.Time + 10000000
	from := common.HexToAddress("0x030405")
	to := common.HexToAddress("0x06070809")
	callvalue := big.NewInt(0)
	beneficiary := common.HexToAddress("0x0301040105090206")
	calldata := make([]byte, 42)
	for i := range calldata {
		calldata[i] = byte(i + 3)
	}
	_, err := precompileCtx.State.RetryableState().CreateRetryable(
		id,
		timeout,
		from,
		&to,
		callvalue,
		beneficiary,
		calldata,
	)
	Require(t, err)

	retryABI, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	redeemCalldata, err := retryABI.Pack("redeem", id)
	Require(t, err)

	retryAddress := common.HexToAddress("6e")
	_, gasLeft, _, err := Precompiles()[retryAddress].Call(
		redeemCalldata,
		retryAddress,
		common.Address{},
		big.NewInt(0),
		false,
		1000000,
		evm,
	)
	Require(t, err)

	expected := storage.StorageWriteCost - storage.StorageWriteZeroCost
	if gasLeft != expected {
		// We expect to have some gas left over, because in this test we write a zero, but in other
		//     use cases the precompile would cause a non-zero write. So the precompile allocates enough gas
		//     to handle both cases, and some will be left over in this test's use case.

		var delta uint64
		var relation string
		if gasLeft > expected {
			delta = gasLeft - expected
			relation = "more"
		} else {
			delta = expected - gasLeft
			relation = "less"
		}
		t.Fatalf("unexpected gas left: got %d, want %d (%d %s than expected)",
			gasLeft, expected, delta, relation)
	}
}

func TestRetryableRedeem(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

	model, err := precompileCtx.State.L2PricingState().GasModelToUse()
	Require(t, err)

	if model != l2pricing.GasModelLegacy {
		Fail(t, "should use legacy model")
	}

	testRetryableRedeem(t, evm, precompileCtx)
}

func TestRetryableRedeemWithSingleGasConstraints(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

	for i := range l2pricing.GasConstraintsMaxNum {
		// #nosec G115
		target0 := uint64((i + 1) * 1000000)
		// #nosec G115
		window0 := uint64((i + 1) * 10)
		// #nosec G115
		backlog0 := uint64((i + 1) * 500000)

		err := precompileCtx.State.L2PricingState().AddGasConstraint(target0, window0, backlog0)
		Require(t, err)
	}

	model, err := precompileCtx.State.L2PricingState().GasModelToUse()
	Require(t, err)

	if model != l2pricing.GasModelSingleGasConstraints {
		Fail(t, "should use single-gas constraints model")
	}

	testRetryableRedeem(t, evm, precompileCtx)
}

func TestRetryableRedeemWithMultiGasConstraints(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)
	precompileCtx.State.L2PricingState().ArbosVersion = l2pricing.ArbosMultiGasConstraintsVersion

	// Override default ArbOS varsion in the database
	versionSlot := uint64(0)
	version := new(big.Int).SetUint64(l2pricing.ArbosMultiGasConstraintsVersion)
	burner := burn.NewSystemBurner(nil, false)
	sto := storage.NewGeth(evm.StateDB, burner)
	err := sto.SetByUint64(versionSlot, common.BigToHash(version))
	Require(t, err)

	for i := range l2pricing.MultiGasConstraintsMaxNum {
		// #nosec G115
		target := uint64((i + 1) * 1000000)
		// #nosec G115
		window := uint32((i + 1) * 10)
		// #nosec G115
		backlog := uint64((i + 1) * 500000)

		weights := map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):     1,
			uint8(multigas.ResourceKindStorageAccess):   2,
			uint8(multigas.ResourceKindStorageGrowth):   3,
			uint8(multigas.ResourceKindL1Calldata):      4,
			uint8(multigas.ResourceKindL2Calldata):      5,
			uint8(multigas.ResourceKindWasmComputation): 6,
		}

		err = precompileCtx.State.L2PricingState().AddMultiGasConstraint(target, window, backlog, weights)
		Require(t, err)
	}

	model, err := precompileCtx.State.L2PricingState().GasModelToUse()
	Require(t, err)

	if model != l2pricing.GasModelMultiGasConstraints {
		Fail(t, "should use multi-gas constraints model")
	}

	testRetryableRedeem(t, evm, precompileCtx)
}
