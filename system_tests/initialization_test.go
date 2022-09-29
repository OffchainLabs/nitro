// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// Each contract gets a set of storage cells with values, and code that returns a sum of their cell
// Getting expectedsum proves both code and storage cells are correct
func InitOneContract(prand *testhelpers.PseudoRandomDataSource) (*statetransfer.AccountInitContractInfo, *big.Int) {
	storageMap := make(map[common.Hash]common.Hash)
	code := []byte{0x60, 0x0} // PUSH1 0
	sum := big.NewInt(0)
	numCells := int(prand.GetUint64() % 1000)
	for i := 0; i < numCells; i++ {
		storageAddr := prand.GetHash()
		storageVal := prand.GetAddress().Hash() // 20 bytes so sum won't overflow
		code = append(code, 0x7f)               // PUSH32
		code = append(code, storageAddr[:]...)  // storageAdr
		code = append(code, 0x54)               // SLOAD
		code = append(code, 0x01)               // ADD
		storageMap[storageAddr] = storageVal
		sum.Add(sum, storageVal.Big())
	}
	code = append(code, 0x60, 0x00) // PUSH1 0
	code = append(code, 0x52)       // MSTORE
	code = append(code, 0x60, 0x20) // PUSH1 32
	code = append(code, 0x60, 0x00) // PUSH1 0
	code = append(code, 0xf3)       // RETURN
	return &statetransfer.AccountInitContractInfo{
		ContractStorage: storageMap,
		Code:            code,
	}, sum
}

func TestInitContract(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	expectedSums := make(map[common.Address]*big.Int)
	prand := testhelpers.NewPseudoRandomDataSource(t, 1)
	l2info := NewArbTestInfo(t, params.ArbitrumDevTestChainConfig().ChainID)
	for i := 0; i < 50; i++ {
		contractData, sum := InitOneContract(prand)
		accountAddress := prand.GetAddress()
		accountInfo := statetransfer.AccountInitializationInfo{
			Addr:         accountAddress,
			EthBalance:   big.NewInt(0),
			Nonce:        1,
			ContractInfo: contractData,
		}
		l2info.ArbInitData.Accounts = append(l2info.ArbInitData.Accounts, accountInfo)
		expectedSums[accountAddress] = sum
	}
	_, node, client := CreateTestL2WithConfig(t, ctx, l2info, arbnode.ConfigDefaultL2Test(), true)
	defer node.StopAndWait()

	for accountAddress, sum := range expectedSums {
		msg := ethereum.CallMsg{
			To: &accountAddress,
		}
		res, err := client.CallContract(ctx, msg, big.NewInt(0))
		Require(t, err)
		resBig := new(big.Int).SetBytes(res)
		if resBig.Cmp(sum) != 0 {
			t.Fatal("address {} exp {} got {}", accountAddress, sum, resBig)
		}
	}
}
