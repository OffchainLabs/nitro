// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

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

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a cold, virgin address, no value transfer.
func TestDimTxOpSelfdestructColdNoTransferVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a cold, funded address, no value transfer.
func TestDimTxOpSelfdestructColdNoTransferFunded(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a cold, virgin address with value transfer.
func TestDimTxOpSelfdestructColdPayingVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.SelfDestruct(emptyAccountAddress) - which is cold
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.SelfDestruct, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a cold, funded address with value transfer.
func TestDimTxOpSelfdestructColdPayingFunded(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a warm, virgin address, no value transfer.
func TestDimTxOpSelfdestructWarmNoTransferVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	_, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a warm, funded address, no value transfer.
func TestDimTxOpSelfdestructWarmNoTransferFunded(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a warm, virgin address with value transfer.
func TestDimTxOpSelfdestructWarmPayingVirgin(t *testing.T) {
	ctx, cancel, builder, auth, cleanup := gasDimensionTestSetup(t, false)
	defer cancel()
	defer cleanup()

	selfDestructorAddress, selfDestructor := deployGasDimensionTestContract(t, builder, auth, gas_dimensionsgen.DeploySelfDestructor)
	emptyAccountAddress := common.HexToAddress("0x00000000000000000000000000000000DeaDBeef")

	// the TransferBalanceTo helper function does the require statements and waiting etc for us
	builder.L2.TransferBalanceTo(t, "Owner", selfDestructorAddress, big.NewInt(1e17), builder.L2Info)

	// call selfDestructor.warmSelfDestructor(0xdeadbeef)
	receipt := callOnContractWithOneArg(t, builder, auth, selfDestructor.WarmEmptySelfDestructor, emptyAccountAddress)

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}

// Tests that the total gas used by the transaction matches the expected value in the receipt,
// and that all gas dimension components sum to the total gas consumed.
// Scenario: SELFDESTRUCT to a warm, funded address with value transfer.
func TestDimTxOpSelfdestructWarmPayingFunded(t *testing.T) {
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

	TxOpTraceAndCheck(t, ctx, builder, receipt)
}
