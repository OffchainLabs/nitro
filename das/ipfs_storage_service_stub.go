// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// IPFS DAS backend stub
// a stub. we don't currently support ipfs

//go:build !ipfs
// +build !ipfs

package das

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbstate"
	flag "github.com/spf13/pflag"
)

var ErrIpfsNotSupported = errors.New("ipfs not supported")

type IpfsStorageServiceConfig struct {
	Enable bool
}

var DefaultIpfsStorageServiceConfig = IpfsStorageServiceConfig{
	Enable: false,
}

func IpfsStorageServiceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultIpfsStorageServiceConfig.Enable, "legacy option - not supported")
}

type IpfsStorageService struct {
}

func NewIpfsStorageService(ctx context.Context, config IpfsStorageServiceConfig) (*IpfsStorageService, error) {
	return nil, ErrIpfsNotSupported
}

func (s *IpfsStorageService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	return nil, ErrIpfsNotSupported
}

func (s *IpfsStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	return ErrIpfsNotSupported
}

func (s *IpfsStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.KeepForever, ErrIpfsNotSupported
}

func (s *IpfsStorageService) Sync(ctx context.Context) error {
	return ErrIpfsNotSupported
}

func (s *IpfsStorageService) Close(ctx context.Context) error {
	return ErrIpfsNotSupported
}

func (s *IpfsStorageService) String() string {
	return "IpfsStorageService-not supported"
}

func (s *IpfsStorageService) HealthCheck(ctx context.Context) error {
	return ErrIpfsNotSupported
}
