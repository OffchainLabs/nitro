// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package rpc

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var simulateTests = []systest.Scenario{
	systest.Test(testRunSimulateV1Push0,
		systest.WithInitDataOverride(func(d *statetransfer.ArbosInitializationInfo) {
			d.Accounts = append(d.Accounts, statetransfer.AccountInitializationInfo{
				Addr:       common.HexToAddress(push0ContractAddr),
				EthBalance: big.NewInt(0),
				Nonce:      1,
				ContractInfo: &statetransfer.AccountInitContractInfo{
					Code:            []byte{byte(vm.PUSH0)},
					ContractStorage: make(map[common.Hash]common.Hash),
				},
			})
		})),
}

const push0ContractAddr = "0x9930da85e75d753ca1b704ee53ebff948174384a"

func testRunSimulateV1Push0(env *systest.Env) {
	l2rpc := env.L2.Stack.Attach()

	// Make sure the same works for eth_call before testing eth_simulateV1.
	env.Require(l2rpc.CallContext(env.Ctx, nil, "eth_call", map[string]any{
		"to": push0ContractAddr,
	}))

	var simulateResponse any
	env.Require(l2rpc.CallContext(env.Ctx, &simulateResponse, "eth_simulateV1", map[string]any{
		"blockStateCalls": []map[string]any{
			{
				"calls": []map[string]any{
					{"to": push0ContractAddr},
				},
			},
		},
	}))
	simulateResponseByte, err := json.Marshal(simulateResponse)
	env.Require(err)
	env.True(!strings.Contains(string(simulateResponseByte), "error"), "simulateV1 response contains error: %v", simulateResponse)
}
