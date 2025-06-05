package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/statetransfer"
)

func TestSimulateV1Push0(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	contractInfo := &statetransfer.AccountInitContractInfo{
		Code:            []byte{byte(vm.PUSH0)},
		ContractStorage: make(map[common.Hash]common.Hash),
	}
	accountInfo := statetransfer.AccountInitializationInfo{
		Addr:         common.HexToAddress("0x9930da85e75d753ca1b704ee53ebff948174384a"),
		EthBalance:   big.NewInt(0),
		Nonce:        1,
		ContractInfo: contractInfo,
	}
	builder.L2Info.ArbInitData.Accounts = append(builder.L2Info.ArbInitData.Accounts, accountInfo)
	cleanup := builder.Build(t)
	defer cleanup()

	code, err := builder.L2.Client.CodeAt(ctx, common.HexToAddress("0x9930da85e75d753ca1b704ee53ebff948174384a"), nil)
	Require(t, err)

	println(code)

	var callResponse interface{}
	err = builder.L2.Client.Client().CallContext(
		ctx,
		&callResponse,
		"eth_call",
		map[string]interface{}{
			"to": "0x9930da85e75d753ca1b704ee53ebff948174384a",
		},
	)
	Require(t, err)
	fmt.Println(callResponse)
	var simulateResponse interface{}
	err = builder.L2.Client.Client().CallContext(
		ctx,
		&simulateResponse,
		"eth_simulateV1",
		map[string]interface{}{
			"blockStateCalls": []map[string]interface{}{
				{
					"calls": []map[string]interface{}{
						{
							"to": "0x9930da85e75d753ca1b704ee53ebff948174384a",
						},
					},
				},
			},
		},
	)
	Require(t, err)
	fmt.Println(simulateResponse)
}
