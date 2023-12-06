// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

//go:embed arbitrum_chain_info.json
var DefaultChainInfo []byte

type ChainInfo struct {
	ChainName             string `json:"chain-name"`
	ParentChainId         uint64 `json:"parent-chain-id"`
	ParentChainIsArbitrum *bool  `json:"parent-chain-is-arbitrum"`
	// This is the forwarding target to submit transactions to, called the sequencer URL for clarity
	SequencerUrl              string              `json:"sequencer-url"`
	SecondaryForwardingTarget string              `json:"secondary-forwarding-target"`
	FeedUrl                   string              `json:"feed-url"`
	SecondaryFeedUrl          string              `json:"secondary-feed-url"`
	DasIndexUrl               string              `json:"das-index-url"`
	HasGenesisState           bool                `json:"has-genesis-state"`
	ChainConfig               *params.ChainConfig `json:"chain-config"`
	RollupAddresses           *RollupAddresses    `json:"rollup"`
}

func GetChainConfig(chainId *big.Int, chainName string, genesisBlockNum uint64, l2ChainInfoFiles []string, l2ChainInfoJson string) (*params.ChainConfig, error) {
	chainInfo, err := ProcessChainInfo(chainId.Uint64(), chainName, l2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return nil, err
	}
	if chainInfo.ChainConfig != nil {
		chainInfo.ChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return chainInfo.ChainConfig, nil
	}
	if chainId.Uint64() != 0 {
		return nil, fmt.Errorf("missing chain config for L2 chain ID %v", chainId)
	}
	return nil, fmt.Errorf("missing chain config for L2 chain name %v", chainName)
}

func GetRollupAddressesConfig(chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string) (RollupAddresses, error) {
	chainInfo, err := ProcessChainInfo(chainId, chainName, l2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return RollupAddresses{}, err
	}
	if chainInfo.RollupAddresses != nil {
		return *chainInfo.RollupAddresses, nil
	}
	if chainId != 0 {
		return RollupAddresses{}, fmt.Errorf("missing rollup addresses for L2 chain ID %v", chainId)
	}
	return RollupAddresses{}, fmt.Errorf("missing rollup addresses for L2 chain name %v", chainName)
}

func ProcessChainInfo(chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string) (*ChainInfo, error) {
	if l2ChainInfoJson != "" {
		chainInfo, err := findChainInfo(chainId, chainName, []byte(l2ChainInfoJson))
		if err != nil || chainInfo != nil {
			return chainInfo, err
		}
	}
	for _, l2ChainInfoFile := range l2ChainInfoFiles {
		chainsInfoBytes, err := os.ReadFile(l2ChainInfoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s err %w", l2ChainInfoFile, err)
		}
		chainInfo, err := findChainInfo(chainId, chainName, chainsInfoBytes)
		if err != nil || chainInfo != nil {
			return chainInfo, err
		}
	}

	chainInfo, err := findChainInfo(chainId, chainName, DefaultChainInfo)
	if err != nil || chainInfo != nil {
		return chainInfo, err
	}
	if chainId != 0 {
		return nil, fmt.Errorf("unsupported chain ID %v", chainId)
	}
	if chainName != "" {
		return nil, fmt.Errorf("unsupported chain name %v", chainName)
	}
	return nil, errors.New("must specify --chain.id or --chain.name to choose rollup")
}

func findChainInfo(chainId uint64, chainName string, chainsInfoBytes []byte) (*ChainInfo, error) {
	var chainsInfo []ChainInfo
	err := json.Unmarshal(chainsInfoBytes, &chainsInfo)
	if err != nil {
		return nil, err
	}
	if chainId == 0 && chainName == "" && len(chainsInfo) == 1 {
		// If single chain info and no chain id/name given, default to single chain info
		return &chainsInfo[0], nil
	}
	for _, chainInfo := range chainsInfo {
		if (chainId == 0 || chainInfo.ChainConfig.ChainID.Uint64() == chainId) && (chainName == "" || chainInfo.ChainName == chainName) {
			return &chainInfo, nil
		}
	}
	return nil, nil
}

type RollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	NativeToken            common.Address `json:"native-token"`
	UpgradeExecutor        common.Address `json:"upgrade-executor"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	StakeToken             common.Address `json:"stake-token"`
	DeployedAt             uint64         `json:"deployed-at"`
}
