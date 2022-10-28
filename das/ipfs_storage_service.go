// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ipfs/go-cid"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/multiformats/go-multihash"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/ipfshelper"
)

type IpfsStorageServiceConfig struct {
	Enable  bool   `koanf:"enable"`
	RepoDir string `koanf:"repo-dir"`
	// Something about expiry
}

var DefaultIpfsStorageServiceConfig = IpfsStorageServiceConfig{
	Enable:  false,
	RepoDir: "",
}

type IpfsStorageService struct {
	ipfsHelper *ipfshelper.IpfsHelper
	ipfsApi    icore.CoreAPI
}

func NewIpfsStorageService(ctx context.Context, repoDirectory string, profiles string) (*IpfsStorageService, error) {
	ipfsHelper, err := ipfshelper.CreateIpfsHelper(ctx, repoDirectory, false, profiles)
	if err != nil {
		return nil, err
	}
	return &IpfsStorageService{
		ipfsHelper: ipfsHelper,
		ipfsApi:    ipfsHelper.GetAPI(),
	}, nil
}

func hashToCid(hash common.Hash) (cid.Cid, error) {
	multiEncodedHashBytes, err := multihash.Encode(hash[:], multihash.KECCAK_256)
	if err != nil {
		return cid.Cid{}, err
	}

	_, multiHash, err := multihash.MHFromBytes(multiEncodedHashBytes)
	if err != nil {
		return cid.Cid{}, err
	}

	return cid.NewCidV1(cid.Raw, multiHash), nil

}

func (s *IpfsStorageService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	thisCid, err := hashToCid(hash)
	if err != nil {
		return nil, err
	}

	ipfsPath := path.IpfsPath(thisCid)
	log.Info("Retrieving path", "path", ipfsPath.String())

	rdr, err := s.ipfsApi.Block().Get(ctx, ipfsPath)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *IpfsStorageService) Put(ctx context.Context, data []byte, expirationTime uint64) error {
	_ = expirationTime // TODO do something with this

	blockStat, err := s.ipfsApi.Block().Put(ctx, bytes.NewReader(data), options.Block.CidCodec("raw"), options.Block.Hash(multihash.KECCAK_256, -1), options.Block.Pin(true))
	if err != nil {
		return err
	}
	log.Info("Written path", "path", blockStat.Path().String())
	return nil
}

func (s *IpfsStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.KeepForever, nil
}

func (s *IpfsStorageService) Sync(ctx context.Context) error {
	return nil
}
func (s *IpfsStorageService) Close(ctx context.Context) error {
	return nil
}
func (s *IpfsStorageService) String() string {
	return "IpfsStorageService"
}
func (s *IpfsStorageService) HealthCheck(ctx context.Context) error {
	return nil
}
