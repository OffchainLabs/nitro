// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDeploy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	auth := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	_, simple := testNode.DeploySimple(t, auth)

	tx, err := simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fatal(t, "Unexpected counter value", counter)
	}
}

func TestEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	auth := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	gasPrice := big.NewInt(params.GWei / 10)

	// set the gas price
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), testNode.L2Client)
	Require(t, err, "could not deploy ArbOwner contract")
	tx, err := arbOwner.SetMinimumL2BaseFee(&auth, gasPrice)
	Require(t, err, "could not set L2 gas price")
	_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	// connect to arbGasInfo precompile
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), testNode.L2Client)
	Require(t, err, "could not deploy contract")

	// wait for price to come to equilibrium
	equilibrated := false
	numTriesLeft := 20
	for !equilibrated && numTriesLeft > 0 {
		// make an empty block to let the gas price update
		testNode.L2Info.GasPrice = new(big.Int).Mul(testNode.L2Info.GasPrice, big.NewInt(2))
		TransferBalance(t, "Owner", "Owner", common.Big0, testNode.L2Info, testNode.L2Client, ctx)

		// check if the price has equilibrated
		_, _, _, _, _, setPrice, err := arbGasInfo.GetPricesInWei(&bind.CallOpts{})
		Require(t, err, "could not get L2 gas price")
		if gasPrice.Cmp(setPrice) == 0 {
			equilibrated = true
		}
		numTriesLeft--
	}
	if !equilibrated {
		Fatal(t, "L2 gas price did not converge", gasPrice)
	}

	initialBalance, err := testNode.L2Client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")

	// deploy a test contract
	_, tx, simple, err := mocksgen.DeploySimple(&auth, testNode.L2Client)
	Require(t, err, "could not deploy contract")
	receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	header, err := testNode.L2Client.HeaderByNumber(ctx, receipt.BlockNumber)
	Require(t, err, "could not get header")
	if header.BaseFee.Cmp(gasPrice) != 0 {
		Fatal(t, "Header has wrong basefee", header.BaseFee, gasPrice)
	}

	balance, err := testNode.L2Client.BalanceAt(ctx, auth.From, nil)
	Require(t, err, "could not get balance")
	expectedCost := receipt.GasUsed * gasPrice.Uint64()
	observedCost := initialBalance.Uint64() - balance.Uint64()
	if expectedCost != observedCost {
		Fatal(t, "Expected deployment to cost", expectedCost, "instead of", observedCost)
	}

	tx, err = simple.Increment(&auth)
	Require(t, err, "failed to call Increment()")
	_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err, "failed to get counter")

	if counter != 1 {
		Fatal(t, "Unexpected counter value", counter)
	}
}

func TestComponentEstimate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	l1BaseFee := new(big.Int).Set(arbostypes.DefaultInitialL1BaseFee)
	l2BaseFee := GetBaseFee(t, testNode.L2Client, ctx)

	colors.PrintGrey("l1 basefee ", l1BaseFee)
	colors.PrintGrey("l2 basefee ", l2BaseFee)

	userBalance := big.NewInt(1e16)
	maxPriorityFeePerGas := big.NewInt(0)
	maxFeePerGas := arbmath.BigMulByUfrac(l2BaseFee, 3, 2)

	testNode.L2Info.GenerateAccount("User")
	TransferBalance(t, "Owner", "User", userBalance, testNode.L2Info, testNode.L2Client, ctx)

	from := testNode.L2Info.GetAddress("User")
	to := testhelpers.RandomAddress()
	gas := uint64(100000000)
	calldata := []byte{0x00, 0x12}
	value := big.NewInt(4096)

	nodeAbi, err := node_interfacegen.NodeInterfaceMetaData.GetAbi()
	Require(t, err)

	nodeMethod := nodeAbi.Methods["gasEstimateComponents"]
	estimateCalldata := append([]byte{}, nodeMethod.ID...)
	packed, err := nodeMethod.Inputs.Pack(to, false, calldata)
	Require(t, err)
	estimateCalldata = append(estimateCalldata, packed...)

	msg := ethereum.CallMsg{
		From:      from,
		To:        &types.NodeInterfaceAddress,
		Gas:       gas,
		GasFeeCap: maxFeePerGas,
		GasTipCap: maxPriorityFeePerGas,
		Value:     value,
		Data:      estimateCalldata,
	}
	returnData, err := testNode.L2Client.CallContract(ctx, msg, nil)
	Require(t, err)

	outputs, err := nodeMethod.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 4 {
		Fatal(t, "expected 4 outputs from gasEstimateComponents, got", len(outputs))
	}

	gasEstimate, _ := outputs[0].(uint64)
	gasEstimateForL1, _ := outputs[1].(uint64)
	baseFee, _ := outputs[2].(*big.Int)
	l1BaseFeeEstimate, _ := outputs[3].(*big.Int)

	tx := testNode.L2Info.SignTxAs("User", &types.DynamicFeeTx{
		ChainID:   testNode.L2Node.Execution.ArbInterface.BlockChain().Config().ChainID,
		Nonce:     0,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasEstimate,
		To:        &to,
		Value:     value,
		Data:      calldata,
	})

	l2Estimate := gasEstimate - gasEstimateForL1

	colors.PrintBlue("Est. ", gasEstimate, " - ", gasEstimateForL1, " = ", l2Estimate)

	if !arbmath.BigEquals(l1BaseFeeEstimate, l1BaseFee) {
		Fatal(t, l1BaseFeeEstimate, l1BaseFee)
	}
	if !arbmath.BigEquals(baseFee, l2BaseFee) {
		Fatal(t, baseFee, l2BaseFee.Uint64())
	}

	Require(t, testNode.L2Client.SendTransaction(ctx, tx))
	receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	l2Used := receipt.GasUsed - receipt.GasUsedForL1
	colors.PrintMint("True ", receipt.GasUsed, " - ", receipt.GasUsedForL1, " = ", l2Used)

	if l2Estimate != l2Used {
		Fatal(t, l2Estimate, l2Used)
	}
}

func TestDisableL1Charging(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()
	addr := common.HexToAddress("0x12345678")

	gasWithL1Charging, err := testNode.L2Client.EstimateGas(ctx, ethereum.CallMsg{To: &addr})
	Require(t, err)

	gasWithoutL1Charging, err := testNode.L2Client.EstimateGas(ctx, ethereum.CallMsg{To: &addr, SkipL1Charging: true})
	Require(t, err)

	if gasWithL1Charging <= gasWithoutL1Charging {
		Fatal(t, "SkipL1Charging didn't disable L1 charging")
	}
	if gasWithoutL1Charging != params.TxGas {
		Fatal(t, "Incorrect gas estimate with disabled L1 charging")
	}

	_, err = testNode.L2Client.CallContract(ctx, ethereum.CallMsg{To: &addr, Gas: gasWithL1Charging}, nil)
	Require(t, err)

	_, err = testNode.L2Client.CallContract(ctx, ethereum.CallMsg{To: &addr, Gas: gasWithoutL1Charging}, nil)
	if err == nil {
		Fatal(t, "CallContract passed with insufficient gas")
	}

	_, err = testNode.L2Client.CallContract(ctx, ethereum.CallMsg{To: &addr, Gas: gasWithoutL1Charging, SkipL1Charging: true}, nil)
	Require(t, err)
}
