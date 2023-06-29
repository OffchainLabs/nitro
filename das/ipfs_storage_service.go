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
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ipfs/go-cid"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
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
	Peers       []string      `koanf:"peers"`

	// Pinning options
	PinAfterGet   bool    `koanf:"pin-after-get"`
	PinPercentage float64 `koanf:"pin-percentage"`
}

var DefaultIpfsStorageServiceConfig = IpfsStorageServiceConfig{
	Enable:      false,
	RepoDir:     "",
	ReadTimeout: time.Minute,
	Profiles:    "",
	Peers:       []string{},

	PinAfterGet:   true,
	PinPercentage: 100.0,
}

func IpfsStorageServiceConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultIpfsStorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from IPFS")
	f.String(prefix+".repo-dir", DefaultIpfsStorageServiceConfig.RepoDir, "directory to use to store the local IPFS repo")
	f.Duration(prefix+".read-timeout", DefaultIpfsStorageServiceConfig.ReadTimeout, "timeout for IPFS reads, since by default it will wait forever. Treat timeout as not found")
	f.String(prefix+".profiles", DefaultIpfsStorageServiceConfig.Profiles, "comma separated list of IPFS profiles to use, see https://docs.ipfs.tech/how-to/default-profile")
	f.StringSlice(prefix+".peers", DefaultIpfsStorageServiceConfig.Peers, "list of IPFS peers to connect to, eg /ip4/1.2.3.4/tcp/12345/p2p/abc...xyz")
	f.Bool(prefix+".pin-after-get", DefaultIpfsStorageServiceConfig.PinAfterGet, "pin sequencer batch data in IPFS")
	f.Float64(prefix+".pin-percentage", DefaultIpfsStorageServiceConfig.PinPercentage, "percent of sequencer batch data to pin, as a floating point number in the range 0.0 to 100.0")
}

type IpfsStorageService struct {
	config     IpfsStorageServiceConfig
	ipfsHelper *ipfshelper.IpfsHelper
	ipfsApi    coreiface.CoreAPI
}

func NewIpfsStorageService(ctx context.Context, config IpfsStorageServiceConfig) (*IpfsStorageService, error) {
	ipfsHelper, err := ipfshelper.CreateIpfsHelper(ctx, config.RepoDir, false, config.Peers, config.Profiles)
	if err != nil {
		return nil, err
	}
	addrs, err := ipfsHelper.GetPeerHostAddresses()
	if err != nil {
		return nil, err
	}
	log.Info("IPFS node started up", "hostAddresses", addrs)

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

	doPin := false // If true, pin every block related to this batch
	if s.config.PinAfterGet {
		if s.config.PinPercentage == 100.0 {
			doPin = true
		} else if (rand.Float64() * 100.0) <= s.config.PinPercentage {
			doPin = true
		}

	}

	oracle := func(h common.Hash) ([]byte, error) {
		thisCid, err := hashToCid(h)
		if err != nil {
			return nil, err
		}

		ipfsPath := path.IpfsPath(thisCid)
		log.Trace("Retrieving IPFS path", "path", ipfsPath.String())

		parentCtx := ctx
		if doPin {
			// If we want to pin this batch, then detach from the parent context so
			// we are not canceled before s.config.ReadTimeout.
			parentCtx = context.Background()
		}

		timeoutCtx, cancel := context.WithTimeout(parentCtx, s.config.ReadTimeout)
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

		if doPin {
			go func() {
				pinCtx, pinCancel := context.WithTimeout(context.Background(), s.config.ReadTimeout)
				defer pinCancel()
				err := s.ipfsApi.Pin().Add(pinCtx, ipfsPath)
				// Recursive pinning not needed, each dastree preimage fits in a single
				// IPFS block.
				if err != nil {
					// Pinning is best-effort.
					log.Warn("Failed to pin in IPFS", "hash", pretty.PrettyHash(hash), "path", ipfsPath.String())
				} else {
					log.Trace("Pin in IPFS successful", "hash", pretty.PrettyHash(hash), "path", ipfsPath.String())
				}
			}()
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

	var chunks [][]byte

	record := func(_ common.Hash, value []byte) {
		chunks = append(chunks, value)
	}

	_ = dastree.RecordHash(record, data)

	numChunks := len(chunks)
	resultChan := make(chan error, numChunks)
	for _, chunk := range chunks {
		_chunk := chunk
		go func() {
			blockStat, err := s.ipfsApi.Block().Put(
				ctx,
				bytes.NewReader(_chunk),
				options.Block.CidCodec("raw"), // Store the data in raw form since the hash in the CID must be the hash
				// of the preimage for our lookup scheme to work.
				options.Block.Hash(multihash.KECCAK_256, -1), // Use keccak256 to calculate the hash to put in the block's
				// CID, since it is the same algo used by dastree.
				options.Block.Pin(true)) // Keep the data in the local IPFS repo, don't GC it.
			if err == nil {
				log.Trace("Wrote IPFS path", "path", blockStat.Path().String())
			}
			resultChan <- err
		}()
	}

	successfullyWrittenChunks := 0
	for err := range resultChan {
		if err != nil {
			return err
		}
		successfullyWrittenChunks++
		if successfullyWrittenChunks == numChunks {
			return nil
		}
	}
	panic("unreachable")
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
