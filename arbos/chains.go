// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	encoding_json "encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/params"
)

type ChainInfo struct {
	ChainName       string                    `json:"chain-name"`
	ParentChainId   uint64                    `json:"parent-chain-id"`
	ChainParameters *encoding_json.RawMessage `json:"chain-parameters"`
	ChainConfig     *params.ChainConfig       `json:"chain-config"`
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
	for _, l2ChainInfoFile := range l2ChainInfoFiles {
		chainsInfoBytes, err := os.ReadFile(l2ChainInfoFile)
		if err != nil {
			return nil, err
		}
		var chainsInfo map[uint64]ChainInfo
		err = encoding_json.Unmarshal(chainsInfoBytes, &chainsInfo)
		if err != nil {
			return nil, err
		}
		if _, ok := chainsInfo[chainId.Uint64()]; !ok {
			continue
		}
		chainConfig := chainsInfo[chainId.Uint64()].ChainConfig
		chainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return chainConfig, nil
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
