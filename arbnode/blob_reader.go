// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/spf13/pflag"
)

type BlobClient struct {
	ec         arbutil.L1Interface
	beaconUrl  *url.URL
	httpClient *http.Client

	// Filled in in Initialize()
	genesisTime    uint64
	secondsPerSlot uint64
}

type BlobClientConfig struct {
	BeaconChainUrl string `koanf:"beacon-chain-url"`
}

var DefaultBlobClientConfig = BlobClientConfig{
	BeaconChainUrl: "",
}

func BlobClientAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".beacon-chain-url", DefaultBlobClientConfig.BeaconChainUrl, "Beacon Chain url to use for fetching blobs (normally on port 3500)")
}

func NewBlobClient(config BlobClientConfig, ec arbutil.L1Interface) (*BlobClient, error) {
	beaconUrl, err := url.Parse(config.BeaconChainUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beacon chain URL: %w", err)
	}
	return &BlobClient{
		ec:         ec,
		beaconUrl:  beaconUrl,
		httpClient: &http.Client{},
	}, nil
}

type fullResult[T any] struct {
	Data T `json:"data"`
}

func beaconRequest[T interface{}](b *BlobClient, ctx context.Context, beaconPath string) (T, error) {
	// Unfortunately, methods on a struct can't be generic.

	var empty T

	// not really a deep copy, but copies the Path part we care about
	url := *b.beaconUrl
	url.Path = path.Join(url.Path, beaconPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), http.NoBody)
	if err != nil {
		return empty, err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}

	var full fullResult[T]
	if err := json.Unmarshal(body, &full); err != nil {
		return empty, err
	}

	return full.Data, nil
}

// Get all the blobs associated with a particular block.
func (b *BlobClient) GetBlobs(ctx context.Context, blockHash common.Hash, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	header, err := b.ec.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if b.secondsPerSlot == 0 {
		return nil, errors.New("BlobClient hasn't been initialized")
	}
	slot := (header.Time - b.genesisTime) / b.secondsPerSlot
	return b.blobSidecars(ctx, slot, versionedHashes)
}

type blobResponseItem struct {
	BlockRoot       string               `json:"block_root"`
	Index           jsonapi.Uint64String `json:"index"`
	Slot            jsonapi.Uint64String `json:"slot"`
	BlockParentRoot string               `json:"block_parent_root"`
	ProposerIndex   jsonapi.Uint64String `json:"proposer_index"`
	Blob            hexutil.Bytes        `json:"blob"`
	KzgCommitment   hexutil.Bytes        `json:"kzg_commitment"`
	KzgProof        hexutil.Bytes        `json:"kzg_proof"`
}

func (b *BlobClient) blobSidecars(ctx context.Context, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	response, err := beaconRequest[[]blobResponseItem](b, ctx, fmt.Sprintf("/eth/v1/beacon/blob_sidecars/%d", slot))
	if err != nil {
		return nil, fmt.Errorf("error calling beacon client in blobSidecars: %w", err)
	}

	if len(response) < len(versionedHashes) {
		return nil, fmt.Errorf("expected at least %d blobs for slot %d but only got %d", len(versionedHashes), slot, len(response))
	}

	output := make([]kzg4844.Blob, len(versionedHashes))
	outputsFound := make([]bool, len(versionedHashes))

	for _, blobItem := range response {
		var commitment kzg4844.Commitment
		copy(commitment[:], blobItem.KzgCommitment)
		versionedHash := blobs.CommitmentToVersionedHash(commitment)

		// The versioned hashes of the blob commitments are produced in the by HASH_OPCODE_BYTE,
		// presumably in the order they were added to the tx. The spec is unclear if the blobs
		// need to be returned in any particular order from the beacon API, so we put them back in
		// the order from the tx.
		var outputIdx int
		var found bool
		for outputIdx = range versionedHashes {
			if versionedHashes[outputIdx] == versionedHash {
				found = true
				if outputsFound[outputIdx] {
					return nil, fmt.Errorf("found blob with versioned hash %v twice", versionedHash)
				}
				outputsFound[outputIdx] = true
				break
			}
		}
		if !found {
			continue
		}

		copy(output[outputIdx][:], blobItem.Blob)

		var proof kzg4844.Proof
		copy(proof[:], blobItem.KzgProof)

		err = kzg4844.VerifyBlobProof(output[outputIdx], commitment, proof)
		if err != nil {
			return nil, fmt.Errorf("failed to verify blob proof for blob at slot(%d) at index(%d), blob(%s)", slot, blobItem.Index, pretty.FirstFewChars(blobItem.Blob.String()))
		}
	}

	for i, found := range outputsFound {
		if !found {
			return nil, fmt.Errorf("missing blob %v in slot %v, can't reconstruct batch payload", versionedHashes[i], slot)
		}
	}

	return output, nil
}

type genesisResponse struct {
	GenesisTime jsonapi.Uint64String `json:"genesis_time"`
	// don't currently care about other fields, add if needed
}

type getSpecResponse struct {
	SecondsPerSlot jsonapi.Uint64String `json:"SECONDS_PER_SLOT"`
}

func (b *BlobClient) Initialize(ctx context.Context) error {
	genesis, err := beaconRequest[genesisResponse](b, ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return fmt.Errorf("error calling beacon client in genesisTime: %w", err)
	}
	b.genesisTime = uint64(genesis.GenesisTime)

	spec, err := beaconRequest[getSpecResponse](b, ctx, "/eth/v1/config/spec")
	if err != nil {
		return fmt.Errorf("error calling beacon client in secondsPerSlot: %w", err)
	}
	if spec.SecondsPerSlot == 0 {
		return errors.New("Got SECONDS_PER_SLOT of zero from beacon client")
	}
	b.secondsPerSlot = uint64(spec.SecondsPerSlot)

	return nil
}
