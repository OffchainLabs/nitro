// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// arbtest_AccountInfo is a convenience alias for arbtest.AccountInfo.
type arbtest_AccountInfo = arbtest.AccountInfo

func init() {
	RegisterTest("TestContractDeployment", testConfigContractDeployment, testRunContractDeployment)
}

func testConfigContractDeployment(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
	}}
}

func testRunContractDeployment(env *TestEnv) {
	account := env.L2Info.GetInfoWithPrivKey("Faucet")
	for _, size := range []int{0, 1, 1000, 20000, 24576} { // 24576 = params.DefaultMaxCodeSize
		testContractDeploy(env, makeContractOfLength(size), account, nil)
	}
	testContractDeploy(env, makeContractOfLength(40000), account, vm.ErrMaxCodeSizeExceeded)
	testContractDeploy(env, makeContractOfLength(60000), account, core.ErrMaxInitCodeSizeExceeded)
}

func testContractDeploy(env *TestEnv, contractCode []byte, account *arbtest_AccountInfo, expectedErr error) {
	env.T.Helper()
	ctx, client := env.Ctx, env.L2.Client

	// Build deploy code: PUSH32 <len>, DUP, PUSH1 <offset>, PUSH1 0, CODECOPY, PUSH1 0, RETURN
	deployCode := []byte{0x7F}
	deployCode = append(deployCode, arbmath.Uint64ToU256Bytes(uint64(len(contractCode)))...)
	var codeOffset byte = 42
	deployCode = append(deployCode, []byte{
		0x80,             // DUP
		0x60, codeOffset, // PUSH1 codeOffset
		0x60, 0x00, // PUSH1 0
		0x39,       // CODECOPY
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN
	}...)
	if len(deployCode) != int(codeOffset) {
		env.Fatal("computed codeOffset", codeOffset, "incorrectly, should be", len(deployCode))
	}
	deployCode = append(deployCode, contractCode...)

	deploymentGas, err := client.EstimateGas(ctx, ethereum.CallMsg{Data: deployCode})
	if expectedErr != nil {
		if err == nil {
			env.Fatal("missing expected contract deployment error", expectedErr)
		} else if strings.Contains(err.Error(), expectedErr.Error()) {
			return // expected error, success
		}
	}
	env.Require(err)

	chainId, err := client.ChainID(ctx)
	env.Require(err)
	latestHeader, err := client.HeaderByNumber(ctx, nil)
	env.Require(err)
	nonce, err := client.PendingNonceAt(ctx, account.Address)
	env.Require(err)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: common.Big0,
		GasFeeCap: latestHeader.BaseFee,
		Gas:       deploymentGas,
		To:        nil,
		Value:     common.Big0,
		Data:      deployCode,
	})
	tx, err = types.SignTx(tx, types.LatestSignerForChainID(chainId), account.PrivateKey)
	env.Require(err)

	err = client.SendTransaction(ctx, tx)
	env.Require(err)
	receipt := env.EnsureTxSucceeded(tx)

	deployedCode, err := client.CodeAt(ctx, receipt.ContractAddress, receipt.BlockNumber)
	env.Require(err)
	if !bytes.Equal(deployedCode, contractCode) {
		env.Fatal("expected deployed code length", len(contractCode), "got", len(deployedCode))
	}

	callResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &receipt.ContractAddress}, nil)
	env.Require(err)
	if len(callResult) > 0 {
		env.Fatal("somehow got a non-empty result from contract:", callResult)
	}
}

// makeContractOfLength creates a contract of the given length that does nothing.
func makeContractOfLength(length int) []byte {
	ret := make([]byte, length)
	for i := range ret {
		if i%2 == 0 {
			ret[i] = 0x58 // PC
		} else {
			ret[i] = 0x50 // POP
		}
	}
	return ret
}
