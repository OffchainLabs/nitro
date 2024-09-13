package util

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/ipfshelper"
)

func GetL2ChainInfoIpfsFile(ctx context.Context, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) (string, error) {
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

func AggregateL2ChainInfoFiles(ctx context.Context, l2ChainInfoFiles []string, l2ChainInfoIpfsUrl string, l2ChainInfoIpfsDownloadPath string) []string {
	if l2ChainInfoIpfsUrl != "" {
		l2ChainInfoIpfsFile, err := GetL2ChainInfoIpfsFile(ctx, l2ChainInfoIpfsUrl, l2ChainInfoIpfsDownloadPath)
		if err != nil {
			log.Error("error getting l2 chain info file from ipfs", "err", err)
		}
		l2ChainInfoFiles = append(l2ChainInfoFiles, l2ChainInfoIpfsFile)
	}
	return l2ChainInfoFiles
}
