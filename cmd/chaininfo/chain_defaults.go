// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
		MaxUncompressedBatchSize:  arbChainParams.MaxUncompressedBatchSize,
	}
}

func CopyBlobScheduleConfig(blobSchedule *params.BlobScheduleConfig) *params.BlobScheduleConfig {
	blobScheduleCopy := &params.BlobScheduleConfig{}
	if blobSchedule.Cancun != nil {
		blobScheduleCopy.Cancun = &params.BlobConfig{
			Target:         blobSchedule.Cancun.Target,
			Max:            blobSchedule.Cancun.Max,
			UpdateFraction: blobSchedule.Cancun.UpdateFraction,
		}
	}
	if blobSchedule.Prague != nil {
		blobScheduleCopy.Prague = &params.BlobConfig{
			Target:         blobSchedule.Prague.Target,
			Max:            blobSchedule.Prague.Max,
			UpdateFraction: blobSchedule.Prague.UpdateFraction,
		}
	}
	if blobSchedule.Osaka != nil {
		blobScheduleCopy.Osaka = &params.BlobConfig{
			Target:         blobSchedule.Osaka.Target,
			Max:            blobSchedule.Osaka.Max,
			UpdateFraction: blobSchedule.Osaka.UpdateFraction,
		}
	}
	if blobSchedule.BPO1 != nil {
		blobScheduleCopy.BPO1 = &params.BlobConfig{
			Target:         blobSchedule.BPO1.Target,
			Max:            blobSchedule.BPO1.Max,
			UpdateFraction: blobSchedule.BPO1.UpdateFraction,
		}
	}
	if blobSchedule.BPO2 != nil {
		blobScheduleCopy.BPO2 = &params.BlobConfig{
			Target:         blobSchedule.BPO2.Target,
			Max:            blobSchedule.BPO2.Max,
			UpdateFraction: blobSchedule.BPO2.UpdateFraction,
		}
	}
	if blobSchedule.BPO3 != nil {
		blobScheduleCopy.BPO3 = &params.BlobConfig{
			Target:         blobSchedule.BPO3.Target,
			Max:            blobSchedule.BPO3.Max,
			UpdateFraction: blobSchedule.BPO3.UpdateFraction,
		}
	}
	if blobSchedule.BPO4 != nil {
		blobScheduleCopy.BPO4 = &params.BlobConfig{
			Target:         blobSchedule.BPO4.Target,
			Max:            blobSchedule.BPO4.Max,
			UpdateFraction: blobSchedule.BPO4.UpdateFraction,
		}
	}
	if blobSchedule.BPO5 != nil {
		blobScheduleCopy.BPO5 = &params.BlobConfig{
			Target:         blobSchedule.BPO5.Target,
			Max:            blobSchedule.BPO5.Max,
			UpdateFraction: blobSchedule.BPO5.UpdateFraction,
		}
	}
	if blobSchedule.Amsterdam != nil {
		blobScheduleCopy.Amsterdam = &params.BlobConfig{
			Target:         blobSchedule.Amsterdam.Target,
			Max:            blobSchedule.Amsterdam.Max,
			UpdateFraction: blobSchedule.Amsterdam.UpdateFraction,
		}
	}
	if blobSchedule.UBT != nil {
		blobScheduleCopy.UBT = &params.BlobConfig{
			Target:         blobSchedule.UBT.Target,
			Max:            blobSchedule.UBT.Max,
			UpdateFraction: blobSchedule.UBT.UpdateFraction,
		}
	}
	return blobScheduleCopy
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
	if chainConfig.BlobScheduleConfig != nil {
		copy.BlobScheduleConfig = CopyBlobScheduleConfig(chainConfig.BlobScheduleConfig)
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
func ArbitrumDevTestAnyTrustParams() params.ArbitrumChainParams {
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
func ArbitrumDevTestAnyTrustChainConfig() *params.ChainConfig {
	return fetchChainConfig("anytrust-dev-test")
}
