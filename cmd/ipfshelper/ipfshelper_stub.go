//go:build !ipfs
// +build !ipfs

package ipfshelper

import (
	"context"
	"errors"
)

type IpfsHelper struct{}

var ErrIpfsNotSupported = errors.New("ipfs not supported")

var DefaultIpfsProfiles = "default ipfs profiles stub"

func CanBeIpfsPath(pathString string) bool {
	return false
}

func CreateIpfsHelper(ctx context.Context, downloadPath string, clientOnly bool, peerList []string, profiles string) (*IpfsHelper, error) {
	return nil, ErrIpfsNotSupported
}

func (h *IpfsHelper) DownloadFile(ctx context.Context, cidString string, destinationDir string) (string, error) {
	return "", ErrIpfsNotSupported
}

func (h *IpfsHelper) Close() error {
	return ErrIpfsNotSupported
}
