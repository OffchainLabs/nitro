// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

func TestGetGenesisFileNameFromDirectoryWithCorrectFile(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	genesisFileName := fmt.Sprintf("%d.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	genesis := core.Genesis{
		Config: &params.ChainConfig{
			ChainID: big.NewInt(int64(chainId)),
		},
		GasLimit:   0,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	result, err := GetGenesisFileNameFromDirectory(tempDir, chainId)
	Require(t, err)
	if result != genesisFilePath {
		t.Fatalf("expected %s, got %s", genesisFilePath, result)
	}
}

func TestGetGenesisFileNameFromDirectoryWithWrongFileName(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	genesisFileName := fmt.Sprintf("%d_wrong.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	genesis := core.Genesis{
		Config: &params.ChainConfig{
			ChainID: big.NewInt(int64(chainId)),
		},
		GasLimit:   0,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	_, err = GetGenesisFileNameFromDirectory(tempDir, chainId)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetGenesisFileNameFromDirectoryWithWrongChainId(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	wrongChainId := uint64(42162)
	genesisFileName := fmt.Sprintf("%d.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	genesis := core.Genesis{
		Config: &params.ChainConfig{
			ChainID: big.NewInt(int64(wrongChainId)),
		},
		GasLimit:   0,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	_, err = GetGenesisFileNameFromDirectory(tempDir, chainId)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
