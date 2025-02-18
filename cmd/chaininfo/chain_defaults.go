// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

var DefaultChainConfigs map[string]*params.ChainConfig

func init() {
	var chainsInfo []ChainInfo
	err := json.Unmarshal(DefaultChainsInfoBytes, &chainsInfo)
	if err != nil {
		panic(fmt.Errorf("error initializing default chainsInfo: %w", err))
	}
	if len(chainsInfo) == 0 {
		panic("Default chainsInfo is empty")
	}
	DefaultChainConfigs = make(map[string]*params.ChainConfig)
	for _, chainInfo := range chainsInfo {
		DefaultChainConfigs[chainInfo.ChainName] = chainInfo.ChainConfig
	}
}

func CopyArbitrumChainParams(arbChainParams params.ArbitrumChainParams) params.ArbitrumChainParams {
	return params.ArbitrumChainParams{
		EnableArbOS:               arbChainParams.EnableArbOS,
		AllowDebugPrecompiles:     arbChainParams.AllowDebugPrecompiles,
		DataAvailabilityCommittee: arbChainParams.DataAvailabilityCommittee,
		InitialArbOSVersion:       arbChainParams.InitialArbOSVersion,
		InitialChainOwner:         arbChainParams.InitialChainOwner,
		GenesisBlockNum:           arbChainParams.GenesisBlockNum,
		MaxCodeSize:               arbChainParams.MaxCodeSize,
		MaxInitCodeSize:           arbChainParams.MaxInitCodeSize,
	}
}

func CopyChainConfig(chainConfig *params.ChainConfig) *params.ChainConfig {
	copy := &params.ChainConfig{
		DAOForkSupport:      chainConfig.DAOForkSupport,
		ArbitrumChainParams: CopyArbitrumChainParams(chainConfig.ArbitrumChainParams),
		Clique: &params.CliqueConfig{
			Period: chainConfig.Clique.Period,
			Epoch:  chainConfig.Clique.Epoch,
		},
	}
	if chainConfig.ChainID != nil {
		copy.ChainID = new(big.Int).Set(chainConfig.ChainID)
	}
	if chainConfig.HomesteadBlock != nil {
		copy.HomesteadBlock = new(big.Int).Set(chainConfig.HomesteadBlock)
	}
	if chainConfig.DAOForkBlock != nil {
		copy.DAOForkBlock = new(big.Int).Set(chainConfig.DAOForkBlock)
	}
	if chainConfig.EIP150Block != nil {
		copy.EIP150Block = new(big.Int).Set(chainConfig.EIP150Block)
	}
	if chainConfig.EIP155Block != nil {
		copy.EIP155Block = new(big.Int).Set(chainConfig.EIP155Block)
	}
	if chainConfig.EIP158Block != nil {
		copy.EIP158Block = new(big.Int).Set(chainConfig.EIP158Block)
	}
	if chainConfig.ByzantiumBlock != nil {
		copy.ByzantiumBlock = new(big.Int).Set(chainConfig.ByzantiumBlock)
	}
	if chainConfig.ConstantinopleBlock != nil {
		copy.ConstantinopleBlock = new(big.Int).Set(chainConfig.ConstantinopleBlock)
	}
	if chainConfig.PetersburgBlock != nil {
		copy.PetersburgBlock = new(big.Int).Set(chainConfig.PetersburgBlock)
	}
	if chainConfig.IstanbulBlock != nil {
		copy.IstanbulBlock = new(big.Int).Set(chainConfig.IstanbulBlock)
	}
	if chainConfig.MuirGlacierBlock != nil {
		copy.MuirGlacierBlock = new(big.Int).Set(chainConfig.MuirGlacierBlock)
	}
	if chainConfig.BerlinBlock != nil {
		copy.BerlinBlock = new(big.Int).Set(chainConfig.BerlinBlock)
	}
	if chainConfig.LondonBlock != nil {
		copy.LondonBlock = new(big.Int).Set(chainConfig.LondonBlock)
	}
	return copy
}

func fetchArbitrumChainParams(chainName string) params.ArbitrumChainParams {
	originalConfig, ok := DefaultChainConfigs[chainName]
	if !ok {
		panic(fmt.Sprintf("%s chain config not found in DefaultChainConfigs", chainName))
	}
	return CopyArbitrumChainParams(originalConfig.ArbitrumChainParams)
}

func ArbitrumOneParams() params.ArbitrumChainParams {
	return fetchArbitrumChainParams("arb1")
}
func ArbitrumNovaParams() params.ArbitrumChainParams {
	return fetchArbitrumChainParams("nova")
}
func ArbitrumRollupGoerliTestnetParams() params.ArbitrumChainParams {
	return fetchArbitrumChainParams("goerli-rollup")
}
func ArbitrumDevTestParams() params.ArbitrumChainParams {
	return fetchArbitrumChainParams("arb-dev-test")
}
func ArbitrumDevTestDASParams() params.ArbitrumChainParams {
	return fetchArbitrumChainParams("anytrust-dev-test")
}

func fetchChainConfig(chainName string) *params.ChainConfig {
	originalConfig, ok := DefaultChainConfigs[chainName]
	if !ok {
		panic(fmt.Sprintf("%s chain config not found in DefaultChainConfigs", chainName))
	}
	return CopyChainConfig(originalConfig)
}

func ArbitrumOneChainConfig() *params.ChainConfig {
	return fetchChainConfig("arb1")
}
func ArbitrumNovaChainConfig() *params.ChainConfig {
	return fetchChainConfig("nova")
}
func ArbitrumRollupGoerliTestnetChainConfig() *params.ChainConfig {
	return fetchChainConfig("goerli-rollup")
}
func ArbitrumDevTestChainConfig() *params.ChainConfig {
	return fetchChainConfig("arb-dev-test")
}
func ArbitrumDevTestDASChainConfig() *params.ChainConfig {
	return fetchChainConfig("anytrust-dev-test")
}
