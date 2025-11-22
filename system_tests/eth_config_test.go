// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
)

func TestEthConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	var result json.RawMessage

	l2rpc := builder.L2.Stack.Attach()
	err := l2rpc.CallContext(ctx, &result, "eth_config")
	Require(t, err)

	// Types are duplicated from go-ethereum/internal/ethapi/api.go
	type config struct {
		ActivationTime  uint64                    `json:"activationTime"`
		BlobSchedule    *params.BlobConfig        `json:"blobSchedule"`
		ChainId         *hexutil.Big              `json:"chainId"`
		ForkId          hexutil.Bytes             `json:"forkId"`
		Precompiles     map[string]common.Address `json:"precompiles"`
		SystemContracts map[string]common.Address `json:"systemContracts"`
	}

	type configResponse struct {
		Current *config `json:"current"`
		Next    *config `json:"next"`
		Last    *config `json:"last"`
	}

	want := &configResponse{
		Current: &config{
			ActivationTime: 0,
			BlobSchedule:   nil,
			ChainId:        (*hexutil.Big)(hexutil.MustDecodeBig("0x64aba")),
			ForkId:         (hexutil.Bytes)(hexutil.MustDecode("0x9aa9b1b0")),
			Precompiles: map[string]common.Address{
				"ArbAddressTable":       common.HexToAddress("0x0000000000000000000000000000000000000066"),
				"ArbAggregator":         common.HexToAddress("0x000000000000000000000000000000000000006d"),
				"ArbBLS":                common.HexToAddress("0x0000000000000000000000000000000000000067"),
				"ArbDebug":              common.HexToAddress("0x00000000000000000000000000000000000000ff"),
				"ArbFunctionTable":      common.HexToAddress("0x0000000000000000000000000000000000000068"),
				"ArbGasInfo":            common.HexToAddress("0x000000000000000000000000000000000000006c"),
				"ArbInfo":               common.HexToAddress("0x0000000000000000000000000000000000000065"),
				"ArbNativeTokenManager": common.HexToAddress("0x0000000000000000000000000000000000000073"),
				"ArbOwner":              common.HexToAddress("0x0000000000000000000000000000000000000070"),
				"ArbOwnerPublic":        common.HexToAddress("0x000000000000000000000000000000000000006b"),
				"ArbRetryableTx":        common.HexToAddress("0x000000000000000000000000000000000000006e"),
				"ArbStatistics":         common.HexToAddress("0x000000000000000000000000000000000000006f"),
				"ArbSys":                common.HexToAddress("0x0000000000000000000000000000000000000064"),
				"ArbWasm":               common.HexToAddress("0x0000000000000000000000000000000000000071"),
				"ArbWasmCache":          common.HexToAddress("0x0000000000000000000000000000000000000072"),
				"ArbosActs":             common.HexToAddress("0x00000000000000000000000000000000000a4b05"),
				"ArbosTest":             common.HexToAddress("0x0000000000000000000000000000000000000069"),
				"BLAKE2F":               common.HexToAddress("0x0000000000000000000000000000000000000009"),
				"BLS12_G1ADD":           common.HexToAddress("0x000000000000000000000000000000000000000b"),
				"BLS12_G1MSM":           common.HexToAddress("0x000000000000000000000000000000000000000c"),
				"BLS12_G2ADD":           common.HexToAddress("0x000000000000000000000000000000000000000d"),
				"BLS12_G2MSM":           common.HexToAddress("0x000000000000000000000000000000000000000e"),
				"BLS12_MAP_FP2_TO_G2":   common.HexToAddress("0x0000000000000000000000000000000000000011"),
				"BLS12_MAP_FP_TO_G1":    common.HexToAddress("0x0000000000000000000000000000000000000010"),
				"BLS12_PAIRING_CHECK":   common.HexToAddress("0x000000000000000000000000000000000000000f"),
				"BN254_ADD":             common.HexToAddress("0x0000000000000000000000000000000000000006"),
				"BN254_MUL":             common.HexToAddress("0x0000000000000000000000000000000000000007"),
				"BN254_PAIRING":         common.HexToAddress("0x0000000000000000000000000000000000000008"),
				"ECREC":                 common.HexToAddress("0x0000000000000000000000000000000000000001"),
				"ID":                    common.HexToAddress("0x0000000000000000000000000000000000000004"),
				"KZG_POINT_EVALUATION":  common.HexToAddress("0x000000000000000000000000000000000000000a"),
				"MODEXP":                common.HexToAddress("0x0000000000000000000000000000000000000005"),
				"P256VERIFY":            common.HexToAddress("0x0000000000000000000000000000000000000100"),
				"RIPEMD160":             common.HexToAddress("0x0000000000000000000000000000000000000003"),
				"SHA256":                common.HexToAddress("0x0000000000000000000000000000000000000002"),
			},
			SystemContracts: map[string]common.Address{
				"HISTORY_STORAGE_ADDRESS": common.HexToAddress("0x0000f90827f1c53a10cb7a02335b175320002935"),
			},
		},
		Next: nil,
		Last: nil,
	}

	got := &configResponse{}
	Require(t, json.Unmarshal(result, got))

	// Use go-cmp to show a readable structural diff on failure, and
	// also include pretty-printed JSON for both expected and actual.
	if diff := cmp.Diff(want, got, cmp.Transformer("hex", func(in *hexutil.Big) string {
		return in.String()
	})); diff != "" {
		t.Fatalf("config mismatch (-want +got):\n%s\n", diff)
	}
}
