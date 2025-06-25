package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/statetransfer"
)

func TestSimulateV1Push0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	contractInfo := &statetransfer.AccountInitContractInfo{
		Code:            []byte{byte(vm.PUSH0)},
		ContractStorage: make(map[common.Hash]common.Hash),
	}
	contractAddr := "0x9930da85e75d753ca1b704ee53ebff948174384a"
	accountInfo := statetransfer.AccountInitializationInfo{
		Addr:         common.HexToAddress(contractAddr),
		EthBalance:   big.NewInt(0),
		Nonce:        1,
		ContractInfo: contractInfo,
	}
	builder.L2Info.ArbInitData.Accounts = append(builder.L2Info.ArbInitData.Accounts, accountInfo)
	cleanup := builder.Build(t)
	defer cleanup()

	// Make sure the same works for eth_call before testing eth_simulateV1.
	err := builder.L2.Client.Client().CallContext(
		ctx,
		nil,
		"eth_call",
		map[string]interface{}{
			"to": contractAddr,
		},
	)
	Require(t, err)

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
							"to": contractAddr,
						},
					},
				},
			},
		},
	)
	Require(t, err)
	simulateResponseByte, err := json.Marshal(simulateResponse)
	Require(t, err)
	if strings.Contains(string(simulateResponseByte), "error") {
		Fatal(t, "simulateV1 response contains error", simulateResponse)
	}
}
