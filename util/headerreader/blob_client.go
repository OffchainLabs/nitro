// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/jsonapi"
)

type BlobClient struct {
	ec                 *ethclient.Client
	beaconUrl          *url.URL
	secondaryBeaconUrl *url.URL
	httpClient         atomic.Pointer[http.Client]
	authorization      string

	// Filled in in Initialize()
	genesisTime    uint64
	secondsPerSlot uint64

	// Directory to save the fetched blobs
	blobDirectory string

	// Dangerous options
	skipBlobProofVerification bool
}

type BlobClientDangerousConfig struct {
	SkipBlobProofVerification bool `koanf:"skip-blob-proof-verification"`
}

type BlobClientConfig struct {
	BeaconUrl          string                    `koanf:"beacon-url"`
	SecondaryBeaconUrl string                    `koanf:"secondary-beacon-url"`
	BlobDirectory      string                    `koanf:"blob-directory"`
	Authorization      string                    `koanf:"authorization"`
	Dangerous          BlobClientDangerousConfig `koanf:"dangerous"`
}

var DefaultDangerousConfig = BlobClientDangerousConfig{
	SkipBlobProofVerification: false,
}

var DefaultBlobClientConfig = BlobClientConfig{
	BeaconUrl:          "",
	SecondaryBeaconUrl: "",
	BlobDirectory:      "",
	Authorization:      "",
	Dangerous:          DefaultDangerousConfig,
}

func BlobClientAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".beacon-url", DefaultBlobClientConfig.BeaconUrl, "Beacon Chain RPC URL to use for fetching blobs (normally on port 3500)")
	f.String(prefix+".secondary-beacon-url", DefaultBlobClientConfig.SecondaryBeaconUrl, "Backup beacon Chain RPC URL to use for fetching blobs (normally on port 3500) when unable to fetch from primary")
	f.String(prefix+".blob-directory", DefaultBlobClientConfig.BlobDirectory, "Full path of the directory to save fetched blobs")
	f.String(prefix+".authorization", DefaultBlobClientConfig.Authorization, "Value to send with the HTTP Authorization: header for Beacon REST requests, must include both scheme and scheme parameters")
	BlobClientDangerousAddOptions(prefix+".dangerous", f)
}

func BlobClientDangerousAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".skip-blob-proof-verification", DefaultDangerousConfig.SkipBlobProofVerification, "DANGEROUS! Skips verification of KZG proofs for blobs fetched from the beacon node.")
}

func NewBlobClient(config BlobClientConfig, ec *ethclient.Client) (*BlobClient, error) {
	beaconUrl, err := url.Parse(config.BeaconUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beacon chain URL: %w", err)
	}
	var secondaryBeaconUrl *url.URL
	if config.SecondaryBeaconUrl != "" {
		if secondaryBeaconUrl, err = url.Parse(config.SecondaryBeaconUrl); err != nil {
			return nil, fmt.Errorf("failed to parse secondary beacon chain URL: %w", err)
		}
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
	blobClient := &BlobClient{
		ec:                        ec,
		beaconUrl:                 beaconUrl,
		secondaryBeaconUrl:        secondaryBeaconUrl,
		authorization:             config.Authorization,
		blobDirectory:             config.BlobDirectory,
		skipBlobProofVerification: config.Dangerous.SkipBlobProofVerification,
	}
	blobClient.httpClient.Store(&http.Client{})
	return blobClient, nil
}

type fullResult[T any] struct {
	Data T `json:"data"`
}

func beaconRequest[T interface{}](b *BlobClient, ctx context.Context, beaconPath string, queryParams url.Values) (T, error) {
	var empty T

	fetchData := func(beaconUrl url.URL) (*http.Response, error) {
		beaconUrl.Path = path.Join(beaconUrl.Path, beaconPath)
		if queryParams != nil {
			beaconUrl.RawQuery = queryParams.Encode()
		}
		fullUrl := beaconUrl.String()
		req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, http.NoBody)
		if err != nil {
			return nil, err
		}
		if b.authorization != "" {
			req.Header.Set("Authorization", b.authorization)
		}
		resp, err := b.httpClient.Load().Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			log.Debug("beacon request returned response with non 200 OK status", "url", fullUrl, "status", resp.Status, "body", bodyStr)
			return nil, fmt.Errorf("response returned with status %s, want 200 OK. url: %s, body: %s", resp.Status, fullUrl, bodyStr)
		}
		return resp, nil
	}

	var resp *http.Response
	var err error
	if resp, err = fetchData(*b.beaconUrl); err != nil {
		if b.secondaryBeaconUrl != nil {
			log.Info("error fetching blob data from primary beacon URL, switching to secondary beacon URL", "err", err)
			if resp, err = fetchData(*b.secondaryBeaconUrl); err != nil {
				return empty, fmt.Errorf("error fetching blob data from secondary beacon URL: %w", err)
			}
		} else {
			return empty, err
		}
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

	return b.GetBlobsBySlot(ctx, slot, versionedHashes)
}

// Get blobs for a specific beacon chain slot.
func (b *BlobClient) GetBlobsBySlot(ctx context.Context, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	if b.secondsPerSlot == 0 {
		return nil, errors.New("BlobClient hasn't been initialized")
	}

	blobs, err := b.getBlobs(ctx, slot, versionedHashes)
	if err != nil {
		// Create a new HTTP client with a dedicated transport that disables connection reuse.
		// With the default client (nil Transport), Go reuses the global DefaultTransport and its connection pool,
		// which may keep using the same problematic backend connection. Disabling keep-alives forces a fresh TCP
		// connection on the next request, increasing the chance of hitting a healthy backend behind a load balancer.
		b.httpClient.Store(&http.Client{Transport: &http.Transport{DisableKeepAlives: true}})
		return nil, fmt.Errorf("error fetching blobs for slot %d: %w", slot, err)
	}
	return blobs, nil
}

func (b *BlobClient) getBlobs(ctx context.Context, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	beaconPath := fmt.Sprintf("/eth/v1/beacon/blobs/%d", slot)
	queryParams := url.Values{}
	for _, hash := range versionedHashes {
		queryParams.Add("versioned_hashes", hash.Hex())
	}

	// Construct the full URL for error reporting
	fullUrl := *b.beaconUrl
	fullUrl.Path = path.Join(fullUrl.Path, beaconPath)
	fullUrl.RawQuery = queryParams.Encode()

	response, err := beaconRequest[[]hexutil.Bytes](b, ctx, beaconPath, queryParams)
	if err != nil {
		// #nosec G115
		roughAgeOfSlot := uint64(time.Now().Unix()) - (b.genesisTime + slot*b.secondsPerSlot)
		if roughAgeOfSlot > b.secondsPerSlot*32*4096 {
			return nil, fmt.Errorf("beacon client in getBlobs got error fetching older blobs in slot: %d, url: %s, an archive endpoint is required, please refer to https://docs.arbitrum.io/run-arbitrum-node/l1-ethereum-beacon-chain-rpc-providers, err: %w", slot, fullUrl.String(), err)
		} else {
			return nil, fmt.Errorf("beacon client in getBlobs got error fetching non-expired blobs in slot: %d, url: %s, err: %w", slot, fullUrl.String(), err)
		}
	}

	if len(versionedHashes) > 0 && len(response) != len(versionedHashes) {
		return nil, fmt.Errorf("expected %d blobs for slot %d but got %d", len(versionedHashes), slot, len(response))
	}

	output := make([]kzg4844.Blob, len(response))
	computedHashes := make([]common.Hash, len(response))

	for i, blobData := range response {
		if len(blobData) != len(output[i]) {
			return nil, fmt.Errorf("blob at index %d has incorrect length %d, expected %d", i, len(blobData), len(output[i]))
		}
		copy(output[i][:], blobData)

		// Compute commitment and versioned hash for validation and storage
		commitment, err := kzg4844.BlobToCommitment(&output[i])
		if err != nil {
			return nil, fmt.Errorf("failed to compute commitment for blob %d: %w", i, err)
		}
		computedHashes[i] = blobs.CommitmentToVersionedHash(commitment)

		// Validate against provided hashes if present
		if len(versionedHashes) > 0 {
			if computedHashes[i] != versionedHashes[i] {
				return nil, fmt.Errorf("blob %d versioned hash mismatch: expected %s, got %s", i, versionedHashes[i].Hex(), computedHashes[i].Hex())
			}
		}
	}

	// Save blobs to disk in version 1 format if blobDirectory is configured
	if b.blobDirectory != "" {
		if err := saveBlobsV1ToDisk(output, computedHashes, slot, b.blobDirectory); err != nil {
			return nil, err
		}
	}

	return output, nil
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

// blobStorageV1 represents the version 1 blob storage format.
// Stores blobs as a map from versioned hash to blob data for fast lookups.
type blobStorageV1 struct {
	Version int                      `json:"version"`
	Data    map[string]hexutil.Bytes `json:"data"`
}

// saveBlobsV0ToDisk saves blobs in version 0 format (legacy blob_sidecars format).
// This is kept for test purposes to create V0 format files for testing backwards compatibility.
func saveBlobsV0ToDisk(rawData json.RawMessage, slot uint64, blobDirectory string) error {
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

// saveBlobsV1ToDisk saves blobs in version 1 format (versioned hash -> blob map)
func saveBlobsV1ToDisk(blobs []kzg4844.Blob, versionedHashes []common.Hash, slot uint64, blobDirectory string) error {
	if len(blobs) != len(versionedHashes) {
		return fmt.Errorf("mismatch between number of blobs (%d) and versioned hashes (%d)", len(blobs), len(versionedHashes))
	}

	// Build map from versioned hash to blob data
	blobMap := make(map[string]hexutil.Bytes, len(blobs))
	for i := range blobs {
		hashStr := versionedHashes[i].Hex()
		blobMap[hashStr] = blobs[i][:]
	}

	storage := blobStorageV1{
		Version: 1,
		Data:    blobMap,
	}

	jsonData, err := json.Marshal(storage)
	if err != nil {
		return fmt.Errorf("unable to marshal blobs into JSON: %w", err)
	}

	filePath := path.Join(blobDirectory, fmt.Sprint(slot))
	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write blob data to disk: %w", err)
	}

	return nil
}

// ReadBlobsFromDisk reads blobs from disk storage and returns them in the order of the requested versioned hashes.
// Supports both version 0 (blob_sidecars) and version 1 (hash map) formats.
// Returns error if any requested blob is not found.
func ReadBlobsFromDisk(blobDirectory string, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	if len(versionedHashes) == 0 {
		return nil, fmt.Errorf("versionedHashes cannot be empty")
	}

	filePath := path.Join(blobDirectory, fmt.Sprint(slot))
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob file not found for slot %d: %w", slot, err)
		}
		return nil, fmt.Errorf("failed to read blob file for slot %d: %w", slot, err)
	}

	version, err := detectBlobFileFormat(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect blob file format for slot %d: %w", slot, err)
	}

	switch version {
	case 0:
		return readBlobsV0(data, versionedHashes)
	case 1:
		return readBlobsV1(data, versionedHashes)
	default:
		return nil, fmt.Errorf("unsupported blob storage version: %d", version)
	}
}

// detectBlobFileFormat detects the storage format version.
// Returns 0 for old format (blob_sidecars), 1 for new format (hash map).
func detectBlobFileFormat(data []byte) (int, error) {
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		return 0, fmt.Errorf("failed to parse blob file: %w", err)
	}
	return versionCheck.Version, nil
}

// readBlobsV1 reads blobs from version 1 format (versioned hash -> blob map)
func readBlobsV1(data []byte, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	var storage blobStorageV1
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version 1 blob storage: %w", err)
	}

	// Lookup each requested hash and maintain order
	result := make([]kzg4844.Blob, len(versionedHashes))
	for i, hash := range versionedHashes {
		hashStr := hash.Hex()
		blobData, found := storage.Data[hashStr]
		if !found {
			return nil, fmt.Errorf("blob not found for versioned hash %s", hashStr)
		}
		if len(blobData) != len(result[i]) {
			return nil, fmt.Errorf("blob has incorrect length %d, expected %d for hash %s", len(blobData), len(result[i]), hashStr)
		}
		copy(result[i][:], blobData)

		// Validate blob matches its versioned hash
		commitment, err := kzg4844.BlobToCommitment(&result[i])
		if err != nil {
			return nil, fmt.Errorf("failed to compute commitment for blob at hash %s: %w", hashStr, err)
		}
		computedHash := blobs.CommitmentToVersionedHash(commitment)
		if computedHash != hash {
			return nil, fmt.Errorf("blob validation failed: computed hash %s does not match requested hash %s", computedHash.Hex(), hashStr)
		}
	}

	return result, nil
}

// readBlobsV0 reads blobs from version 0 format (blob_sidecars array)
func readBlobsV0(data []byte, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	// Parse the old format wrapped in fullResult
	var full fullResult[[]blobResponseItem]
	if err := json.Unmarshal(data, &full); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version 0 blob storage: %w", err)
	}

	// Build a map from versioned hash to blob by computing hashes from commitments.
	// We're doing this because the old format (directly serializing the "data" field
	// from response from the old blob_sidecars endpoint) doesn't have the versioned hash.
	blobMap := make(map[common.Hash]kzg4844.Blob)
	for _, item := range full.Data {
		// Compute versioned hash from stored KZG commitment
		if len(item.KzgCommitment) != len(kzg4844.Commitment{}) {
			return nil, fmt.Errorf("invalid KZG commitment length: %d, expected %d", len(item.KzgCommitment), len(kzg4844.Commitment{}))
		}
		var storedCommitment kzg4844.Commitment
		copy(storedCommitment[:], item.KzgCommitment)
		versionedHash := blobs.CommitmentToVersionedHash(storedCommitment)

		// Copy blob data
		if len(item.Blob) != len(kzg4844.Blob{}) {
			return nil, fmt.Errorf("invalid blob length: %d, expected %d", len(item.Blob), len(kzg4844.Blob{}))
		}
		var blob kzg4844.Blob
		copy(blob[:], item.Blob)

		// Validate blob matches the stored commitment
		computedCommitment, err := kzg4844.BlobToCommitment(&blob)
		if err != nil {
			return nil, fmt.Errorf("failed to compute commitment for blob: %w", err)
		}
		if computedCommitment != storedCommitment {
			return nil, fmt.Errorf("blob validation failed: computed commitment does not match stored commitment for versioned hash %s", versionedHash.Hex())
		}

		blobMap[versionedHash] = blob
	}

	// Lookup each requested hash and maintain order
	result := make([]kzg4844.Blob, len(versionedHashes))
	for i, hash := range versionedHashes {
		blob, found := blobMap[hash]
		if !found {
			return nil, fmt.Errorf("blob not found for versioned hash %s", hash.Hex())
		}
		result[i] = blob
	}

	return result, nil
}

type genesisResponse struct {
	GenesisTime jsonapi.Uint64String `json:"genesis_time"`
	// don't currently care about other fields, add if needed
}

type getSpecResponse struct {
	SecondsPerSlot jsonapi.Uint64String `json:"SECONDS_PER_SLOT"`
}

func (b *BlobClient) Initialize(ctx context.Context) error {
	genesis, err := beaconRequest[genesisResponse](b, ctx, "/eth/v1/beacon/genesis", nil)
	if err != nil {
		return fmt.Errorf("error calling beacon client to get genesisTime: %w", err)
	}
	b.genesisTime = uint64(genesis.GenesisTime)

	spec, err := beaconRequest[getSpecResponse](b, ctx, "/eth/v1/config/spec", nil)
	if err != nil {
		return fmt.Errorf("error calling beacon client to get secondsPerSlot: %w", err)
	}
	if spec.SecondsPerSlot == 0 {
		return errors.New("got SECONDS_PER_SLOT of zero from beacon client")
	}
	b.secondsPerSlot = uint64(spec.SecondsPerSlot)

	return nil
}
