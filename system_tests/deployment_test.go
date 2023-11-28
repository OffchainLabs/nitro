// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

func testContractDeployment(t *testing.T, ctx context.Context, client *ethclient.Client, contractCode []byte, accountInfo *AccountInfo, expectedEstimateGasError error) {
	// First, we need to make the "deploy code" which returns the contractCode to be deployed
	deployCode := []byte{
		0x7F, // PUSH32
	}
	// len(contractCode)
	deployCode = append(deployCode, math.U256Bytes(big.NewInt(int64(len(contractCode))))...)
	var codeOffset byte = 42
	deployCode = append(deployCode, []byte{
		0x80,             // DUP
		0x60, codeOffset, // PUSH1 [codeOffset]
		0x60, 0x00, // PUSH1 0
		0x39,       // CODECOPY
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN
	}...)
	if len(deployCode) != int(codeOffset) {
		Fatal(t, "computed codeOffset", codeOffset, "incorrectly, should be", len(deployCode))
	}
	deployCode = append(deployCode, contractCode...)

	deploymentGas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		Data: deployCode,
	})
	if expectedEstimateGasError != nil {
		if err == nil {
			Fatal(t, "missing expected contract deployment error", expectedEstimateGasError)
		} else if strings.Contains(err.Error(), expectedEstimateGasError.Error()) {
			// success
			return
		}
		// else, fall through to Require, as this error is unexpected
	}
	Require(t, err)

	chainId, err := client.ChainID(ctx)
	Require(t, err)
	latestHeader, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	nonce, err := client.PendingNonceAt(ctx, accountInfo.Address)
	Require(t, err)

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
	tx, err = types.SignTx(tx, types.LatestSignerForChainID(chainId), accountInfo.PrivateKey)
	Require(t, err)

	err = client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	deployedCode, err := client.CodeAt(ctx, receipt.ContractAddress, receipt.BlockNumber)
	Require(t, err)
	if !bytes.Equal(deployedCode, contractCode) {
		Fatal(t, "expected to deploy code of length", len(contractCode), "but got code of length", len(deployedCode))
	}

	callResult, err := client.CallContract(ctx, ethereum.CallMsg{To: &receipt.ContractAddress}, nil)
	Require(t, err)
	if len(callResult) > 0 {
		Fatal(t, "somehow got a non-empty result from contract of", callResult)
	}
}

// Makes a contract which does nothing but takes up a given length
func makeContractOfLength(length int) []byte {
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		if i%2 == 0 {
			ret[i] = 0x58 // PC
		} else {
			ret[i] = 0x50 // POP
		}
	}
	return ret
}

func TestContractDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	account := builder.L2Info.GetInfoWithPrivKey("Faucet")
	for _, size := range []int{0, 1, 1000, 20000, params.MaxCodeSize} {
		testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(size), account, nil)
	}

	testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(40000), account, vm.ErrMaxCodeSizeExceeded)
	testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(60000), account, core.ErrMaxInitCodeSizeExceeded)
}

func TestExtendedContractDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.chainConfig.ArbitrumChainParams.MaxCodeSize = params.MaxCodeSize * 3
	builder.chainConfig.ArbitrumChainParams.MaxInitCodeSize = params.MaxInitCodeSize * 3
	cleanup := builder.Build(t)
	defer cleanup()

	account := builder.L2Info.GetInfoWithPrivKey("Faucet")
	for _, size := range []int{0, 1, 1000, 20000, 30000, 40000, 60000, params.MaxCodeSize * 3} {
		testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(size), account, nil)
	}

	testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(100000), account, vm.ErrMaxCodeSizeExceeded)
	testContractDeployment(t, ctx, builder.L2.Client, makeContractOfLength(200000), account, core.ErrMaxInitCodeSizeExceeded)
}
