// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gas

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

var estimationTests = []systest.Scenario{
	systest.Test(testRunDeploy),
	systest.Test(testRunEstimate),
	systest.Test(testRunBlockDifficulty, systest.MatrixArbOS(params.ArbosVersion_10, params.MaxArbosVersionSupported)),
	systest.Test(testRunBlobBasefeeReverts),
	systest.Test(testRunDisableL1Charging),
	systest.Test(testRunComponentEstimate),
	systest.Test(testEstimationWithRPCGasLimit(params.TxGas, false), systest.Named("GasEstimationWithRPCGasLimit"),
		systest.WithMultiNode(), systest.WithFollowerExecConfigOverride(func(cfg *gethexec.Config) { cfg.RPC.RPCGasCap = params.TxGas })),
	systest.Test(testEstimationWithRPCGasLimit(params.TxGas-1, true), systest.Named("GasEstimationWithRPCGasLimitTooLow"),
		systest.WithMultiNode(), systest.WithFollowerExecConfigOverride(func(cfg *gethexec.Config) { cfg.RPC.RPCGasCap = params.TxGas - 1 })),
}

func testRunDeploy(env *systest.Env) {
	auth := env.L2.TransactOpts("Owner")
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	_, deployTx, simple, err := localgen.DeploySimple(&auth, env.L2.Client)
	env.Require(err)
	env.L2.EnsureTxSucceeded(deployTx)

	tx, err := simple.Increment(&auth)
	env.Require(err, "failed to call Increment()")
	env.L2.EnsureTxSucceeded(tx)

	counter, err := simple.Counter(&bind.CallOpts{})
	env.Require(err, "failed to get counter")
	env.Equal(uint64(1), counter, "unexpected counter value")
}

func testRunBlobBasefeeReverts(env *systest.Env) {
	_, err := env.L2.Client.CallContract(env.Ctx, ethereum.CallMsg{
		Data: []byte{byte(vm.BLOBBASEFEE)},
	}, nil)
	env.NotNil(err, "expected BLOBBASEFEE to revert")
}

func testRunComponentEstimate(env *systest.Env) {
	l1BaseFee := new(big.Int).Set(arbostypes.DefaultInitialL1BaseFee)
	l2BaseFee := env.L2.HeaderByNumber(nil).BaseFee

	userBalance := big.NewInt(1e16)
	maxPriorityFeePerGas := big.NewInt(0)
	maxFeePerGas := arbmath.BigMulByUFrac(l2BaseFee, 3, 2)

	env.L2.Info.GenerateAccount("User")
	env.L2.TransferBalance("Owner", "User", userBalance)

	from := env.L2.Info.GetAddress("User")
	to := testhelpers.RandomAddress()
	gas := uint64(100000000)
	calldata := []byte{0x00, 0x12}
	value := big.NewInt(4096)

	nodeAbi, err := node_interfacegen.NodeInterfaceMetaData.GetAbi()
	env.Require(err)

	nodeMethod := nodeAbi.Methods["gasEstimateComponents"]
	estimateCalldata := append([]byte{}, nodeMethod.ID...)
	packed, err := nodeMethod.Inputs.Pack(to, false, calldata)
	env.Require(err)
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
	returnData, err := env.L2.Client.CallContract(env.Ctx, msg, nil)
	env.Require(err)

	outputs, err := nodeMethod.Outputs.Unpack(returnData)
	env.Require(err)
	env.Len(outputs, 4, "unexpected gasEstimateComponents output count")

	gasEstimate, _ := outputs[0].(uint64)
	gasEstimateForL1, _ := outputs[1].(uint64)
	baseFee, _ := outputs[2].(*big.Int)
	l1BaseFeeEstimate, _ := outputs[3].(*big.Int)

	tx := env.L2.Info.SignTxAs("User", &types.DynamicFeeTx{
		ChainID:   env.L2.ChainID(),
		Nonce:     0,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasEstimate,
		To:        &to,
		Value:     value,
		Data:      calldata,
	})

	l2Estimate := gasEstimate - gasEstimateForL1
	env.Logf("Est. %d - %d = %d", gasEstimate, gasEstimateForL1, l2Estimate)

	env.EqualBig(l1BaseFee, l1BaseFeeEstimate, "unexpected L1 basefee estimate")
	env.EqualBig(l2BaseFee, baseFee, "unexpected L2 basefee")

	env.L2.SendTx(tx)
	receipt := env.L2.EnsureTxSucceeded(tx)

	l2Used := receipt.GasUsed - receipt.GasUsedForL1
	env.Logf("True %d - %d = %d", receipt.GasUsed, receipt.GasUsedForL1, l2Used)

	env.True(float64(l2Estimate-l2Used) <= float64(gasEstimateForL1+l2Used)*gasestimator.EstimateGasErrorRatio,
		"estimate %d too far from used %d", l2Estimate, l2Used)
}

// testEstimationWithRPCGasLimit pins RPC.RPCGasCap on the follower: estimation
// against it fails when the cap is below intrinsic gas.
func testEstimationWithRPCGasLimit(gasCap uint64, wantErr bool) systest.Scenario {
	return func(env *systest.Env) {
		env.WaitForFollowersSync()
		addr := common.HexToAddress("0x12345678")
		estimateGas, err := env.Follower().Client.EstimateGas(env.Ctx, ethereum.CallMsg{To: &addr})
		if wantErr {
			env.NotNil(err, "EstimateGas passed with insufficient gas cap %d", gasCap)
			return
		}
		env.Require(err)
		env.True(estimateGas > params.TxGas, "incorrect gas estimate %d", estimateGas)

		_, err = env.Follower().Client.CallContract(env.Ctx, ethereum.CallMsg{To: &addr}, nil)
		env.Require(err)
	}
}

func testRunEstimate(env *systest.Env) {
	auth := env.L2.TransactOpts("Owner")
	auth.GasMargin = 0 // don't adjust, we want to see if the estimate alone is sufficient

	gasPrice := big.NewInt(params.GWei / 10)

	// set the gas price
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err, "could not deploy ArbOwner contract")
	tx, err := arbOwner.SetMinimumL2BaseFee(&auth, gasPrice)
	env.Require(err, "could not set L2 gas price")
	env.L2.EnsureTxSucceeded(tx)

	// connect to arbGasInfo precompile
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, env.L2.Client)
	env.Require(err, "could not deploy contract")

	// wait for price to come to equilibrium
	equilibrated := false
	numTriesLeft := 20
	for !equilibrated && numTriesLeft > 0 {
		// make an empty block to let the gas price update
		env.L2.Info.GasPrice = new(big.Int).Mul(env.L2.Info.GasPrice, big.NewInt(2))
		env.L2.TransferBalance("Owner", "Owner", common.Big0)

		// check if the price has equilibrated
		_, _, _, _, _, setPrice, err := arbGasInfo.GetPricesInWei(&bind.CallOpts{})
		env.Require(err, "could not get L2 gas price")
		if gasPrice.Cmp(setPrice) == 0 {
			equilibrated = true
		}
		numTriesLeft--
	}
	env.True(equilibrated, "L2 gas price did not converge on %v", gasPrice)

	initialBalance := env.L2.BalanceAt(auth.From)

	// deploy a test contract
	_, tx, simple, err := localgen.DeploySimple(&auth, env.L2.Client)
	env.Require(err, "could not deploy contract")
	receipt := env.L2.EnsureTxSucceeded(tx)

	header := env.L2.HeaderByNumber(receipt.BlockNumber)
	env.EqualBig(gasPrice, header.BaseFee, "header has wrong basefee")

	balance := env.L2.BalanceAt(auth.From)
	expectedCost := receipt.GasUsed * gasPrice.Uint64()
	observedCost := initialBalance.Uint64() - balance.Uint64()
	env.Equal(expectedCost, observedCost, "unexpected deployment cost")

	tx, err = simple.Increment(&auth)
	env.Require(err, "failed to call Increment()")
	env.L2.EnsureTxSucceeded(tx)

	counter, err := simple.Counter(&bind.CallOpts{})
	env.Require(err, "failed to get counter")
	env.Equal(uint64(1), counter, "unexpected counter value")
}

// testRunBlockDifficulty pins block difficulty to 1 at the matrix cell's ArbOS version.
func testRunBlockDifficulty(env *systest.Env) {
	auth := env.L2.TransactOpts("Owner")
	_, _, simple, err := localgen.DeploySimple(&auth, env.L2.Client)
	env.Require(err, "could not deploy contract")

	tx, err := simple.StoreDifficulty(&auth)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	difficulty, err := simple.GetBlockDifficulty(&bind.CallOpts{})
	env.Require(err)
	env.EqualBig(common.Big1, difficulty, "unexpected block difficulty")
}

func testRunDisableL1Charging(env *systest.Env) {
	addr := common.HexToAddress("0x12345678")

	gasWithL1Charging, err := env.L2.Client.EstimateGas(env.Ctx, ethereum.CallMsg{To: &addr})
	env.Require(err)

	gasWithoutL1Charging, err := env.L2.Client.EstimateGas(env.Ctx, ethereum.CallMsg{To: &addr, SkipL1Charging: true})
	env.Require(err)

	env.True(gasWithL1Charging > gasWithoutL1Charging, "SkipL1Charging didn't disable L1 charging")
	env.Equal(params.TxGas, gasWithoutL1Charging, "incorrect gas estimate with disabled L1 charging")

	_, err = env.L2.Client.CallContract(env.Ctx, ethereum.CallMsg{To: &addr, Gas: gasWithL1Charging}, nil)
	env.Require(err)

	_, err = env.L2.Client.CallContract(env.Ctx, ethereum.CallMsg{To: &addr, Gas: gasWithoutL1Charging}, nil)
	env.NotNil(err, "CallContract passed with insufficient gas")

	_, err = env.L2.Client.CallContract(env.Ctx, ethereum.CallMsg{To: &addr, Gas: gasWithoutL1Charging, SkipL1Charging: true}, nil)
	env.Require(err)
}
