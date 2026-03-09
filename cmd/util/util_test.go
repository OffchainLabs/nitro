// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

func TestReadChainConfig(t *testing.T) {
	expectedConfig := &params.ChainConfig{
		ChainID:        big.NewInt(42161),
		HomesteadBlock: big.NewInt(0),
	}
	serializedChainConfig, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatalf("failed to marshal chain config: %v", err)
	}

	genesis := &core.Genesis{
		SerializedChainConfig: string(serializedChainConfig),
	}

	chainConfig, returnedSerializedConfig, err := ReadChainConfig(genesis)
	if err != nil {
		t.Fatalf("ReadChainConfig() unexpected error: %v", err)
	}
	if chainConfig.ChainID.Cmp(expectedConfig.ChainID) != 0 {
		t.Fatalf("unexpected chain id: got %v want %v", chainConfig.ChainID, expectedConfig.ChainID)
	}
	if string(returnedSerializedConfig) != string(serializedChainConfig) {
		t.Fatalf("unexpected serialized chain config: got %s want %s", string(returnedSerializedConfig), string(serializedChainConfig))
	}
}

func TestReadChainConfigRejectsDeprecatedConfigField(t *testing.T) {
	genesis := &core.Genesis{
		Config: &params.ChainConfig{ChainID: big.NewInt(42161)},
	}

	if _, _, err := ReadChainConfig(genesis); err == nil {
		t.Fatal("ReadChainConfig() succeeded with deprecated config field")
	}
}
