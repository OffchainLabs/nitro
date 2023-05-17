// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package chaininfo

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/ipfshelper"
)

//go:embed arbitrum_chain_info.json
var DefaultChainInfo []byte

type ChainInfo struct {
	ChainId         uint64              `json:"chain-id"`
	ChainName       string              `json:"chain-name"`
	ParentChainId   uint64              `json:"parent-chain-id"`
	ChainParameters *json.RawMessage    `json:"chain-parameters"`
	ChainConfig     *params.ChainConfig `json:"chain-config"`
	RollupAddresses *RollupAddresses    `json:"rollup"`
}

func GetChainConfig(ctx context.Context, chainId *big.Int, chainName string, genesisBlockNum uint64, l2ChainInfoFiles []string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (*params.ChainConfig, error) {
	chainInfo, err := ProcessChainInfo(ctx, chainId.Uint64(), chainName, l2ChainInfoFiles, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
	if err != nil {
		return nil, err
	}
	if chainInfo.ChainConfig != nil {
		chainInfo.ChainConfig.ArbitrumChainParams.GenesisBlockNum = genesisBlockNum
		return chainInfo.ChainConfig, nil
	}
	if chainId.Uint64() != 0 {
		return nil, fmt.Errorf("missing chain config for L2 chain ID %v", chainId)
	} else {
		return nil, fmt.Errorf("missing chain config for L2 chain name %v", chainName)
	}
}

func GetRollupAddressesConfig(ctx context.Context, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (RollupAddresses, error) {
	chainInfo, err := ProcessChainInfo(ctx, chainId, chainName, l2ChainInfoFiles, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
	if err != nil {
		return RollupAddresses{}, err
	}
	if chainInfo.RollupAddresses != nil {
		return *chainInfo.RollupAddresses, nil
	}
	if chainId != 0 {
		return RollupAddresses{}, fmt.Errorf("missing rollup addresses for L2 chain ID %v", chainId)
	} else {
		return RollupAddresses{}, fmt.Errorf("missing rollup addresses for L2 chain name %v", chainName)
	}
}

func ProcessChainInfo(ctx context.Context, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (*ChainInfo, error) {
	combinedL2ChainInfoFile := l2ChainInfoFiles
	if l2ChainInfoIpfsUrl != "" {
		l2ChainInfoIpfsFile, err := getL2ChainInfoIpfsFile(ctx, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
		if err != nil {
			log.Error("error getting l2 chain info file from ipfs", "err", err)
		}
		combinedL2ChainInfoFile = append(combinedL2ChainInfoFile, l2ChainInfoIpfsFile)
	}
	for _, l2ChainInfoFile := range combinedL2ChainInfoFile {
		chainsInfoBytes, err := os.ReadFile(l2ChainInfoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s err %w", l2ChainInfoFile, err)
		}
		var chainsInfo []ChainInfo
		err = json.Unmarshal(chainsInfoBytes, &chainsInfo)
		if err != nil {
			decodedChainsInfoBytes, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(chainsInfoBytes)))
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(decodedChainsInfoBytes, &chainsInfo)
			if err != nil {
				return nil, err
			}
		}
		for _, chainInfo := range chainsInfo {
			if chainInfo.ChainId == chainId || chainInfo.ChainName == chainName {
				return &chainInfo, nil
			}
		}
	}

	var chainsInfo []ChainInfo
	err := json.Unmarshal(DefaultChainInfo, &chainsInfo)
	if err != nil {
		return nil, err
	}
	for _, chainInfo := range chainsInfo {
		if chainInfo.ChainId == chainId || chainInfo.ChainName == chainName {
			return &chainInfo, nil
		}
	}
	if chainId != 0 {
		return nil, fmt.Errorf("unsupported L2 chain ID %v", chainId)
	} else {
		return nil, fmt.Errorf("unsupported L2 chain chain %v", chainName)
	}
}

func getL2ChainInfoIpfsFile(ctx context.Context, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (string, error) {
	ipfsNode, err := ipfshelper.CreateIpfsHelper(ctx, l2ChainInfoIpfsDownloadPath, false, []string{}, ipfshelper.DefaultIpfsProfiles)
	if err != nil {
		return "", err
	}
	log.Info("Downloading l2 info file via IPFS", "url", l2ChainInfoIpfsDownloadPath)
	l2ChainInfoFile, downloadErr := ipfsNode.DownloadFile(ctx, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
	closeErr := ipfsNode.Close()
	if downloadErr != nil {
		if closeErr != nil {
			log.Error("Failed to close IPFS node after download error", "err", closeErr)
		}
		return "", fmt.Errorf("failed to download file from IPFS: %w", downloadErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("failed to close IPFS node: %w", err)
	}
	return l2ChainInfoFile, nil
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
