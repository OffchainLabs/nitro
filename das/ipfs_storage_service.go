// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/offchainlabs/nitro/das/dastree"
	flag "github.com/spf13/pflag"
)

type IpfsStorageServiceConfig struct {
	Enable   bool   `koanf:"enable"`
	RepoDir  string `koanf:"repo-dir"`
	Profiles string `koanf:"profiles"`
}

var DefaultIpfsStorageServiceConfig = IpfsStorageServiceConfig{
	Enable:   false,
	RepoDir:  "",
	Profiles: "test",
}

func IpfsStorageServiceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultIpfsStorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from IPFS")
	f.String(prefix+".repo-dir", DefaultIpfsStorageServiceConfig.RepoDir, "directory to use to store the local IPFS repo")
	f.String(prefix+".profiles", DefaultIpfsStorageServiceConfig.Profiles, "comma separated list of IPFS profiles to use")
}

type IpfsStorageService struct {
	ipfsHelper *ipfshelper.IpfsHelper
	ipfsApi    icore.CoreAPI
}

func NewIpfsStorageService(ctx context.Context, config IpfsStorageServiceConfig) (*IpfsStorageService, error) {
	ipfsHelper, err := ipfshelper.CreateIpfsHelper(ctx, config.RepoDir, false, config.Profiles)
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
	oracle := func(h common.Hash) ([]byte, error) {
		thisCid, err := hashToCid(h)
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

	return dastree.Content(hash, oracle)
}

func (s *IpfsStorageService) Put(ctx context.Context, data []byte, _ uint64) error {
	record := func(_ common.Hash, value []byte) error {
		blockStat, err := s.ipfsApi.Block().Put(ctx, bytes.NewReader(value), options.Block.CidCodec("raw"), options.Block.Hash(multihash.KECCAK_256, -1), options.Block.Pin(true))
		if err != nil {
			return err
		}
		log.Info("Written path", "path", blockStat.Path().String())
		return nil
	}

	_, err := dastree.RecordHash(record, data)
	return err
}

func (s *IpfsStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.KeepForever, nil
}

func (s *IpfsStorageService) Sync(ctx context.Context) error {
	return nil
}
func (s *IpfsStorageService) Close(ctx context.Context) error {
	return s.ipfsHelper.Close()
}
func (s *IpfsStorageService) String() string {
	return "IpfsStorageService"
}
func (s *IpfsStorageService) HealthCheck(ctx context.Context) error {
	testData := []byte("Test-Data")
	err := s.Put(ctx, testData, 0)
	if err != nil {
		return err
	}
	res, err := s.GetByHash(ctx, dastree.Hash(testData))
	if err != nil {
		return err
	}
	if !bytes.Equal(res, testData) {
		return errors.New("invalid GetByHash result")
	}
	return nil
}
