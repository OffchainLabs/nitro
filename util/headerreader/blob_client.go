// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package headerreader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/spf13/pflag"
)

type BlobClient struct {
	ec            arbutil.L1Interface
	beaconUrl     *url.URL
	httpClient    *http.Client
	authorization string

	// Filled in in Initialize()
	genesisTime    uint64
	secondsPerSlot uint64

	// Directory to save the fetched blobs
	blobDirectory string
}

type BlobClientConfig struct {
	BeaconUrl     string `koanf:"beacon-url"`
	BlobDirectory string `koanf:"blob-directory"`
	Authorization string `koanf:"authorization"`
}

var DefaultBlobClientConfig = BlobClientConfig{
	BeaconUrl:     "",
	BlobDirectory: "",
	Authorization: "",
}

func BlobClientAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".beacon-url", DefaultBlobClientConfig.BeaconUrl, "Beacon Chain RPC URL to use for fetching blobs (normally on port 3500)")
	f.String(prefix+".blob-directory", DefaultBlobClientConfig.BlobDirectory, "Full path of the directory to save fetched blobs")
	f.String(prefix+".authorization", DefaultBlobClientConfig.Authorization, "Value to send with the HTTP Authorization: header for Beacon REST requests, must include both scheme and scheme parameters")
}

func NewBlobClient(config BlobClientConfig, ec arbutil.L1Interface) (*BlobClient, error) {
	beaconUrl, err := url.Parse(config.BeaconUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beacon chain URL: %w", err)
	}
	if config.BlobDirectory != "" {
		if _, err = os.Stat(config.BlobDirectory); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(config.BlobDirectory, os.ModePerm); err != nil {
					return nil, fmt.Errorf("error creating blob directory: %w", err)
				}
			} else {
				return nil, fmt.Errorf("invalid blob directory path: %w", err)
			}
		}
	}
	return &BlobClient{
		ec:            ec,
		beaconUrl:     beaconUrl,
		authorization: config.Authorization,
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

	if b.authorization != "" {
		req.Header.Set("Authorization", b.authorization)
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

const trailingCharsOfResponse = 25

func (b *BlobClient) blobSidecars(ctx context.Context, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	rawData, err := beaconRequest[json.RawMessage](b, ctx, fmt.Sprintf("/eth/v1/beacon/blob_sidecars/%d", slot))
	if err != nil {
		return nil, fmt.Errorf("error calling beacon client in blobSidecars: %w", err)
	}
	var response []blobResponseItem
	if err := json.Unmarshal(rawData, &response); err != nil {
		rawDataStr := string(rawData)
		log.Debug("response from beacon URL cannot be unmarshalled into array of blobResponseItem in blobSidecars", "slot", slot, "responseLength", len(rawDataStr), "response", rawDataStr)
		if len(rawDataStr) > 100 {
			return nil, fmt.Errorf("error unmarshalling response from beacon URL into array of blobResponseItem in blobSidecars: %w. Trailing %d characters of the response: %s", err, trailingCharsOfResponse, rawDataStr[len(rawDataStr)-trailingCharsOfResponse:])
		} else {
			return nil, fmt.Errorf("error unmarshalling response from beacon URL into array of blobResponseItem in blobSidecars: %w. Response: %s", err, rawDataStr)
		}
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
		if err := saveBlobDataToDisk(rawData, slot, b.blobDirectory); err != nil {
			return nil, err
		}
	}

	return output, nil
}

func saveBlobDataToDisk(rawData json.RawMessage, slot uint64, blobDirectory string) error {
	filePath := path.Join(blobDirectory, fmt.Sprint(slot))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file to store fetched blobs")
	}
	full := fullResult[json.RawMessage]{Data: rawData}
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

type getSpecResponse struct {
	SecondsPerSlot jsonapi.Uint64String `json:"SECONDS_PER_SLOT"`
}

func (b *BlobClient) Initialize(ctx context.Context) error {
	genesis, err := beaconRequest[genesisResponse](b, ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return fmt.Errorf("error calling beacon client to get genesisTime: %w", err)
	}
	b.genesisTime = uint64(genesis.GenesisTime)

	spec, err := beaconRequest[getSpecResponse](b, ctx, "/eth/v1/config/spec")
	if err != nil {
		return fmt.Errorf("error calling beacon client to get secondsPerSlot: %w", err)
	}
	if spec.SecondsPerSlot == 0 {
		return errors.New("got SECONDS_PER_SLOT of zero from beacon client")
	}
	b.secondsPerSlot = uint64(spec.SecondsPerSlot)

	return nil
}
