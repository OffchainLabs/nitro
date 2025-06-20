// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/gas_dimensionsgen"
)

// #########################################################################################################
// #########################################################################################################
//                                              SELFDESTRUCT
// #########################################################################################################
// #########################################################################################################
//
// SELFDESTRUCT has many permutations
// Warm vs Cold (in the access list)
// Paying vs NoTransfer (is ether value being sent with this call?)
// Virgin vs Funded (has the target address ever been seen before / does it have code already / is the account already in the accounts tree?)
//
// `value_to_empty_account_cost` is storage growth (25000)
// `address_access_cost` for cold addresses is a read/write cost
// since all this operation does is send ether at this point
// then we assign the static gas of 5000 to read/write,
// since in the non-state-growth case this would be a read/write
// and the access cost is read/write
// in the case where state growth happens due to sending funds to
// a new empty account, then we assign that to state growth.

// in this test case, we self destruct and set the target of funds to be
// an empty address that has no code or value at that address. Normally
// that would trigger state growth, but we also have no money to send
// as part of the selfdestruct, so we don't trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructColdNoTransferVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that is not empty (i.e. it has code or value).
// but we also self destruct with no funds to send, so it's kind of moot.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructColdNoTransferFunded(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPayableCounter)

	// prefund the selfDestructor and payableCounter with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + ColdMinusWarmAccountAccessCost,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has no code or value at that address.
// this does trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600 + 25000,
// for the base selfdestruct cost and the access list cold read cost and the
// value to empty account cost,
// so we expect a computation of 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 25000,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructColdPayingVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.SelfDestruct(emptyAccountAddress) - which is cold
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + ColdMinusWarmAccountAccessCost,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that already has code or value at that address.
// since the address already has code or value, the operation is a state read/write
// rather than a state growth.
//
// in this case we expect the one-dimensional cost to be 5000 + 2600,
// for the base selfdestruct cost and the access list cold read cost,
// computation to be 100 (for the warm access list read),
// state access to be 5000+2500, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructColdPayingFunded(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccount := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// send some money to the self destructor address ahead of time
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.SelfDestruct(emptyAccountAddress) - which is cold
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccount)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas + params.ColdAccountAccessCostEIP2929,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 + ColdMinusWarmAccountAccessCost,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an empty address that has no code or value at that address. Normally
// that would trigger state growth, but we also have no money to send
// as part of the selfdestruct, so we don't trigger state growth.
//
// in this case we expect the one-dimensional cost to be 5000,
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructWarmNoTransferVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has some code at that address and some eth value already,
// but we don't have any funds to send, so we don't send any as part of the
// selfdestruct
//
// for this transaction we expect a one-dimensional cost of 5000
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructWarmNoTransferFunded(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_ /*selfDestructorAddress*/, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPayableCounter)

	// prefund the payableCounter with some funds, but not the selfDestructor
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmSelfDestructor, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case we self destruct and set the target of funds to be
// an address that neither has code nor any eth value yet. Which means
// the resulting selfdestruct will cause state growth, assigning value
// to a new account.
//
// in this case we expect the one-dimensional cost to be 30000,
// 5000 from static cost and 25000 from the value to empty account cost
// 100 for warm access list read
// that gives us a computation of 100, state access of 4900, state growth of 25000,
// history growth of 0, and state growth refund of 0
func TestDimLogSelfdestructWarmPayingVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150 + params.CreateBySelfdestructGas,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           params.CreateBySelfdestructGas,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}

// in this test case, we self destruct and set the target of funds to be
// an address that has some code at that address and some eth value already
//
// for this transaction we expect a one-dimensional cost of 5000
// computation to be 100 (for the warm access list read),
// state access to be 4900, state growth to be 0,
// history growth to be 0, and state growth refund to be 0
func TestDimLogSelfdestructWarmPayingFunded(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	payableCounterAddress, _ /*payableCounter*/ := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeployPayableCounter)

	// prefund the selfDestructor and payableCounter with some funds
	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)
	builder.L2.TransferBalanceTo(t, "Owner", payableCounterAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(payableCounterAddress)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmSelfDestructor, payableCounterAddress)

	traceResult := callDebugTraceTransactionWithLogger(t, ctx, builder, receipt.TxHash)
	selfDestructLog := getSpecificDimensionLog(t, traceResult.DimensionLogs, "SELFDESTRUCT")

	expected := ExpectedGasCosts{
		OneDimensionalGasCost: params.SelfdestructGasEIP150,
		Computation:           params.WarmStorageReadCostEIP2929,
		StateAccess:           params.SelfdestructGasEIP150 - params.WarmStorageReadCostEIP2929,
		StateGrowth:           0,
		HistoryGrowth:         0,
		StateGrowthRefund:     0,
	}
	checkGasDimensionsMatch(t, expected, selfDestructLog)
	checkGasDimensionsEqualOneDimensionalGas(t, selfDestructLog)
}
