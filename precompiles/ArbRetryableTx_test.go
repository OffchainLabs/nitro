// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos"
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

func TestRetryableRedeem(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

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

	if gasLeft != storage.StorageWriteCost-storage.StorageWriteZeroCost {
		// We expect to have some gas left over, because in this test we write a zero, but in other
		//     use cases the precompile would cause a non-zero write. So the precompile allocates enough gas
		//     to handle both cases, and some will be left over in this test's use case.
		Fail(t, "didn't consume all the expected gas")
	}
}

func TestRetryableRedeemWithGasConstraints(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

	for i := range l2pricing.GasConstraintsLimit {
		// #nosec G115
		target0 := uint64((i + 1) * 1000000)
		// #nosec G115
		window0 := uint64((i + 1) * 10)
		// #nosec G115
		backlog0 := uint64((i + 1) * 500000)

		err := precompileCtx.State.L2PricingState().AddGasConstraint(target0, window0, backlog0)
		Require(t, err)
	}

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
		// Say how much different the result is (and in which direction).
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
