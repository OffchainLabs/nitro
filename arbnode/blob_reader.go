// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	// The genesis time time and seconds per slot won't change so only request them once.
	cachedGenesisTime    uint64
	cachedSecondsPerSlot uint64

	// Directory to save the fetcehd blobs
	blobDirectory string
}

type BlobClientConfig struct {
	BeaconChainUrl string `koanf:"beacon-chain-url"`
	BlobDirectory  string `koanf:"blob-directory"`
}

var DefaultBlobClientConfig = BlobClientConfig{
	BeaconChainUrl: "",
	BlobDirectory:  "",
}

func BlobClientAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".beacon-chain-url", DefaultBlobClientConfig.BeaconChainUrl, "Beacon Chain url to use for fetching blobs")
	f.String(prefix+".blob-directory", DefaultBlobClientConfig.BlobDirectory, "Full path of the directory to save fetched blobs")
}

func NewBlobClient(config BlobClientConfig, ec arbutil.L1Interface) (*BlobClient, error) {
	beaconUrl, err := url.Parse(config.BeaconChainUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beacon chain URL: %w", err)
	}
	if _, err = os.Stat(config.BlobDirectory); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(config.BlobDirectory, os.ModePerm); err != nil {
				return nil, fmt.Errorf("error creating blob directory: %v", err)
			}
		} else {
			return nil, fmt.Errorf("invalid blob directory path: %v", err)
		}
	}
	return &BlobClient{
		ec:            ec,
		beaconUrl:     beaconUrl,
		httpClient:    &http.Client{},
		blobDirectory: config.BlobDirectory,
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
	genesisTime, err := b.genesisTime(ctx)
	if err != nil {
		return nil, err
	}
	secondsPerSlot, err := b.secondsPerSlot(ctx)
	if err != nil {
		return nil, err
	}
	slot := (header.Time - genesisTime) / secondsPerSlot
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

	if b.blobDirectory != "" {
		if err := saveBlobDataToDisk(response, slot, b.blobDirectory); err != nil {
			return nil, err
		}
	}

	return output, nil
}

func saveBlobDataToDisk(response []blobResponseItem, slot uint64, blobDirectory string) error {
	filePath := path.Join(blobDirectory, fmt.Sprint(slot))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file to store fetched blobs")
	}
	full := fullResult[[]blobResponseItem]{Data: response}
	fullbytes, err := json.Marshal(full)
	if err != nil {
		return fmt.Errorf("unable to marshal data into bytes while attempting to store fetched blobs")
	}
	if _, err := file.Write(fullbytes); err != nil {
		return fmt.Errorf("failed to write blob data to disk")
	}
	file.Close()
	return nil
}

type genesisResponse struct {
	GenesisTime jsonapi.Uint64String `json:"genesis_time"`
	// don't currently care about other fields, add if needed
}

func (b *BlobClient) genesisTime(ctx context.Context) (uint64, error) {
	if b.cachedGenesisTime > 0 {
		return b.cachedGenesisTime, nil
	}
	gr, err := beaconRequest[genesisResponse](b, ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return 0, fmt.Errorf("error calling beacon client in genesisTime: %w", err)
	}
	b.cachedGenesisTime = uint64(gr.GenesisTime)
	return b.cachedGenesisTime, nil
}

type getSpecResponse struct {
	SecondsPerSlot jsonapi.Uint64String `json:"SECONDS_PER_SLOT"`
}

func (b *BlobClient) secondsPerSlot(ctx context.Context) (uint64, error) {
	if b.cachedSecondsPerSlot > 0 {
		return b.cachedSecondsPerSlot, nil
	}
	gr, err := beaconRequest[getSpecResponse](b, ctx, "/eth/v1/config/spec")
	if err != nil {
		return 0, fmt.Errorf("error calling beacon client in secondsPerSlot: %w", err)
	}
	b.cachedSecondsPerSlot = uint64(gr.SecondsPerSlot)
	return b.cachedSecondsPerSlot, nil

}
