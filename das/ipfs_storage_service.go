// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// IPFS DAS backend.
// It takes advantage of IPFS' content addressing scheme to be able to directly retrieve
// the batches from IPFS using their root hash from the L1 sequencer inbox contract.

package das

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

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
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"
)

type IpfsStorageServiceConfig struct {
	Enable      bool          `koanf:"enable"`
	RepoDir     string        `koanf:"repo-dir"`
	ReadTimeout time.Duration `koanf:"read-timeout"`
	Profiles    string        `koanf:"profiles"`
}

var DefaultIpfsStorageServiceConfig = IpfsStorageServiceConfig{
	Enable:      false,
	RepoDir:     "",
	ReadTimeout: time.Minute,
	Profiles:    "test", // Default to test, see profiles here https://github.com/ipfs/kubo/blob/e550d9e4761ea394357c413c02ade142c0dea88c/config/profile.go
}

func IpfsStorageServiceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultIpfsStorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from IPFS")
	f.String(prefix+".repo-dir", DefaultIpfsStorageServiceConfig.RepoDir, "directory to use to store the local IPFS repo")
	f.Duration(prefix+".read-timeout", DefaultIpfsStorageServiceConfig.ReadTimeout, "timeout for IPFS reads, since by default it will wait forever. Treat timeout as not found")
	f.String(prefix+".profiles", DefaultIpfsStorageServiceConfig.Profiles, "comma separated list of IPFS profiles to use")
}

type IpfsStorageService struct {
	config     IpfsStorageServiceConfig
	ipfsHelper *ipfshelper.IpfsHelper
	ipfsApi    icore.CoreAPI
}

func NewIpfsStorageService(ctx context.Context, config IpfsStorageServiceConfig) (*IpfsStorageService, error) {
	ipfsHelper, err := ipfshelper.CreateIpfsHelper(ctx, config.RepoDir, false, config.Profiles)
	if err != nil {
		return nil, err
	}
	return &IpfsStorageService{
		config:     config,
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

// GetByHash retrieves and reconstructs one batch's data, using IPFS to retrieve the preimages
// for each chunk of data and the dastree nodes.
func (s *IpfsStorageService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	log.Trace("das.IpfsStorageService.GetByHash", "hash", pretty.PrettyHash(hash))

	oracle := func(h common.Hash) ([]byte, error) {
		thisCid, err := hashToCid(h)
		if err != nil {
			return nil, err
		}

		ipfsPath := path.IpfsPath(thisCid)
		log.Trace("Retrieving IPFS path", "path", ipfsPath.String())

		timeoutCtx, cancel := context.WithTimeout(ctx, s.config.ReadTimeout)
		defer cancel()
		rdr, err := s.ipfsApi.Block().Get(timeoutCtx, ipfsPath)
		if err != nil {
			if timeoutCtx.Err() != nil {
				return nil, ErrNotFound
			}
			return nil, err
		}

		data, err := io.ReadAll(rdr)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	return dastree.Content(hash, oracle)
}

// Put stores all the preimages required to reconstruct the dastree for single batch,
// ie the hashed data chunks and dastree nodes.
// This takes advantage of IPFS supporting keccak256 on raw data blocks for calculating
// its CIDs, and the fact that the dastree structure uses keccak256 for addressing its
// nodes, to directly store the dastree structure in IPFS.
// IPFS default block size is 256KB and dastree max block size is 64KB so each dastree
// node and data chunk easily fits within an IPFS block.
func (s *IpfsStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	logPut("das.IpfsStorageService.Put", data, timeout, s)
	record := func(_ common.Hash, value []byte) error {
		blockStat, err := s.ipfsApi.Block().Put(
			ctx,
			bytes.NewReader(value),
			options.Block.CidCodec("raw"), // Store the data in raw form since the hash in the CID must be the hash
			// of the preimage for our lookup scheme to work.
			options.Block.Hash(multihash.KECCAK_256, -1), // Use keccak256 to calculate the hash to put in the block's
			// CID, since it is the same algo used by dastree.
			options.Block.Pin(true)) // Keep the data in the local IPFS repo, don't GC it.
		if err != nil {
			return err
		}
		log.Trace("Wrote IPFS path", "path", blockStat.Path().String())
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
