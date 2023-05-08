// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

//go:embed arbitrum_chain_info.json
var DefaultChainInfo []byte

type ChainInfo struct {
	ChainName             string                 `json:"chain-name"`
	ParentChainId         uint64                 `json:"parent-chain-id"`
	ChainParameters       *json.RawMessage       `json:"chain-parameters"`
	ChainConfig           *params.ChainConfig    `json:"chain-config"`
	RollupAddressesConfig *RollupAddressesConfig `json:"rollup"`
}

type RollupAddressesConfig struct {
	Bridge                 string `koanf:"bridge"`
	Inbox                  string `koanf:"inbox"`
	SequencerInbox         string `koanf:"sequencer-inbox"`
	Rollup                 string `koanf:"rollup"`
	ValidatorUtils         string `koanf:"validator-utils"`
	ValidatorWalletCreator string `koanf:"validator-wallet-creator"`
	DeployedAt             uint64 `koanf:"deployed-at"`
}

func (c *RollupAddressesConfig) ParseAddresses() (RollupAddresses, error) {
	a := RollupAddresses{
		DeployedAt: c.DeployedAt,
	}
	strs := []string{
		c.Bridge,
		c.Inbox,
		c.SequencerInbox,
		c.Rollup,
		c.ValidatorUtils,
		c.ValidatorWalletCreator,
	}
	addrs := []*common.Address{
		&a.Bridge,
		&a.Inbox,
		&a.SequencerInbox,
		&a.Rollup,
		&a.ValidatorUtils,
		&a.ValidatorWalletCreator,
	}
	names := []string{
		"Bridge",
		"Inbox",
		"SequencerInbox",
		"Rollup",
		"ValidatorUtils",
		"ValidatorWalletCreator",
	}
	if len(strs) != len(addrs) {
		return RollupAddresses{}, fmt.Errorf("internal error: attempting to parse %v strings into %v addresses", len(strs), len(addrs))
	}
	complete := true
	for i, s := range strs {
		if !common.IsHexAddress(s) {
			log.Error("invalid address", "name", names[i], "value", s)
			complete = false
		}
		*addrs[i] = common.HexToAddress(s)
	}
	if !complete {
		return RollupAddresses{}, fmt.Errorf("invalid addresses")
	}
	return a, nil
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
	if chainInfo != nil && chainInfo.ChainConfig != nil {
		chainInfo.ChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return chainInfo.ChainConfig, nil
	}
	staticChainConfig, err := getStaticChainConfig(chainId)
	if err != nil {
		return nil, err
	}
	staticChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
	return staticChainConfig, nil
}

func GetRollupAddressesConfig(chainId *big.Int, l2ChainInfoFiles []string) (*RollupAddressesConfig, error) {
	chainInfo, err := ProcessChainInfo(chainId.Uint64(), l2ChainInfoFiles)
	if err != nil {
		return nil, err
	}
	if chainInfo != nil {
		return chainInfo.RollupAddressesConfig, nil
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
	err := json.Unmarshal(DefaultChainInfo, &chainsInfo)
	if err != nil {
		return nil, err
	}
	if _, ok := chainsInfo[chainId]; !ok {
		return nil, nil
	}
	chainInfo := chainsInfo[chainId]
	return &chainInfo, nil
}

type RollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	DeployedAt             uint64         `json:"deployed-at"`
}
