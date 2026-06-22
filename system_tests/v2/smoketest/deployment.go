// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package smoketest

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var deploymentTests = func() []systest.Scenario {
	var tests []systest.Scenario
	for _, size := range []int{0, 1, 1000, 20000, params.DefaultMaxCodeSize} {
		tests = append(tests, systest.Test(testContractDeploy(size, nil), systest.Named(fmt.Sprintf("ContractDeploy%d", size))))
	}
	return append(tests,
		systest.Test(testContractDeploy(40000, vm.ErrMaxCodeSizeExceeded), systest.Named("ContractDeployExceedsCodeSize")),
		systest.Test(testContractDeploy(60000, core.ErrMaxInitCodeSizeExceeded), systest.Named("ContractDeployExceedsInitCodeSize")),
		systest.Test(testRunExtendedContractDeployment, systest.WithChainConfigOverride(func(c *params.ChainConfig) {
			c.ArbitrumChainParams.MaxCodeSize = params.DefaultMaxCodeSize * 3
			c.ArbitrumChainParams.MaxInitCodeSize = params.DefaultMaxInitCodeSize * 3
		})),
	)
}()

func testContractDeploy(size int, expectedErr error) systest.Scenario {
	return func(env *systest.Env) {
		deployContract(env, size, expectedErr)
	}
}

func testRunExtendedContractDeployment(env *systest.Env) {
	for _, size := range []int{0, 1, 1000, 20000, 30000, 40000, 60000, params.DefaultMaxCodeSize * 3} {
		deployContract(env, size, nil)
	}
	deployContract(env, 100000, vm.ErrMaxCodeSizeExceeded)
	deployContract(env, 200000, core.ErrMaxInitCodeSizeExceeded)
}

func deployContract(env *systest.Env, size int, expectedErr error) {
	contractCode := makeContractOfLength(size)
	deployCode := program.New().ReturnViaCodeCopy(contractCode).Bytes()

	deploymentGas, err := env.L2.Client.EstimateGas(env.Ctx, ethereum.CallMsg{Data: deployCode})
	if expectedErr != nil {
		env.ErrorContains(err, expectedErr.Error(), "expected contract deployment error %v", expectedErr)
		return
	}
	env.Require(err)

	tx := env.L2.Info.PrepareTxTo("Faucet", nil, deploymentGas, nil, deployCode)
	env.L2.SendTx(tx)
	receipt := env.L2.EnsureTxSucceeded(tx)

	deployedCode := env.L2.CodeAt(receipt.ContractAddress, receipt.BlockNumber)
	env.Zero(bytes.Compare(contractCode, deployedCode), "deployed code mismatch: want len %d, got len %d", len(contractCode), len(deployedCode))

	callResult := env.L2.CallContract(ethereum.CallMsg{To: &receipt.ContractAddress}, nil)
	env.Empty(callResult, "somehow got a non-empty result from contract")
}

// Alternating PC / POP — no-op padding of the requested length.
func makeContractOfLength(length int) []byte {
	ret := make([]byte, length)
	for i := range ret {
		if i%2 == 0 {
			ret[i] = byte(vm.PC)
		} else {
			ret[i] = byte(vm.POP)
		}
	}
	return ret
}
