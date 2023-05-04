// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/chain_info"
)

type ChainInfo struct {
	ChainName       string              `json:"chain-name"`
	ParentChainId   uint64              `json:"parent-chain-id"`
	ChainParameters *json.RawMessage    `json:"chain-parameters"`
	ChainConfig     *params.ChainConfig `json:"chain-config"`
}

func getStaticChainConfig(chainId *big.Int) (*params.ChainConfig, error) {
	for _, potentialChainConfig := range params.ArbitrumSupportedChainConfigs {
		if potentialChainConfig.ChainID.Cmp(chainId) == 0 {
			return potentialChainConfig, nil
		}
	}
	return nil, fmt.Errorf("unsupported L2 chain ID %v", chainId)
}

func GetChainConfig(chainId *big.Int, genesisBlockNum uint64, l2ChainInfoFiles []string) (*params.ChainConfig, error) {
	chainInfo, err := ProcessChainInfo(chainId.Uint64(), l2ChainInfoFiles)
	if err != nil {
		return nil, err
	}
	if chainInfo != nil {
		chainInfo.ChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return chainInfo.ChainConfig, nil
	}
	if len(l2ChainInfoFiles) == 0 {
		staticChainConfig, err := getStaticChainConfig(chainId)
		if err != nil {
			return nil, err
		}
		staticChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return staticChainConfig, nil
	}
	return nil, fmt.Errorf("unsupported L2 chain ID %v", chainId)
}

func ProcessChainInfo(chainId uint64, l2ChainInfoFiles []string) (*ChainInfo, error) {
	for _, l2ChainInfoFile := range l2ChainInfoFiles {
		chainsInfoBytes, err := os.ReadFile(l2ChainInfoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s err %w", l2ChainInfoFile, err)
		}
		var chainsInfo map[uint64]ChainInfo
		err = json.Unmarshal(chainsInfoBytes, &chainsInfo)
		if err != nil {
			return nil, err
		}
		if _, ok := chainsInfo[chainId]; !ok {
			continue
		}
		chainInfo := chainsInfo[chainId]
		return &chainInfo, nil
	}

	var chainsInfo map[uint64]ChainInfo
	err := json.Unmarshal(chain_info.DefaultChainInfo, &chainsInfo)
	if err != nil {
		return nil, err
	}
	if _, ok := chainsInfo[chainId]; !ok {
		return nil, nil
	}
	chainInfo := chainsInfo[chainId]
	return &chainInfo, nil
}
