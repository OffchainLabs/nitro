// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package restrictedaddr

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3client"
)

const (
	S3DownloadPartSizeMB         = 32
	S3DownloadPartBodyMaxRetries = 5
	S3DownloadConcurrency        = 10
)

// hashListPayload represents the JSON structure of the hash list file used for unmarshalling.
type hashListPayload struct {
	Salt          string `json:"salt"`
	AddressHashes []struct {
		Hash string `json:"hash"`
	} `json:"address_hashes"`
}

type S3Syncer struct {
	client s3client.FullClient
	config *Config
	store  *HashStore
}

func NewS3Syncer(ctx context.Context, config *Config, store *HashStore) (*S3Syncer, error) {
	client, err := s3client.NewS3FullClient(ctx, config.S3AccessKey, config.S3SecretKey, config.S3Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	return &S3Syncer{
		client: client,
		config: config,
		store:  store,
	}, nil
}

// CheckAndSync checks if the S3 object has changed (via ETag) and downloads it if so.
func (s *S3Syncer) CheckAndSync(ctx context.Context) error {
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(s.config.S3ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.S3Bucket, s.config.S3ObjectKey, err)
	}

	currentETagDigest := aws.ToString(headOutput.ETag)

	// Compare with stored ETag digest
	if currentETagDigest == s.store.Digest() {
		log.Debug("restricted addr list unchanged", "etag digest", currentETagDigest)
		return nil
	}

	log.Info("restricted addr list changed, downloading", "old_etag", s.store.Digest(), "new_etag_digest", currentETagDigest, "size_bytes")
	return s.downloadAndLoad(ctx, currentETagDigest)
}

// DownloadAndLoad downloads the hash list from S3 and loads it into the store.
// This is used for initial load where we need to fetch metadata first.
func (s *S3Syncer) DownloadAndLoad(ctx context.Context) error {
	// Get metadata first to know content length for buffer pre-allocation
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(s.config.S3ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.S3Bucket, s.config.S3ObjectKey, err)
	}

	etagDigest := aws.ToString(headOutput.ETag)
	return s.downloadAndLoad(ctx, etagDigest)
}

func (s *S3Syncer) downloadAndLoad(ctx context.Context, etagDigest string) error {
	downloader := manager.NewDownloader(s.client.Client(), func(d *manager.Downloader) {
		d.PartSize = S3DownloadPartSizeMB * 1024 * 1024 // 32 MB parts
		d.PartBodyMaxRetries = S3DownloadPartBodyMaxRetries
		d.Concurrency = S3DownloadConcurrency
	})

	tempFile, err := os.CreateTemp("", "restricted-addr-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	// Download - SDK handles chunking, concurrency, and retry
	_, err = downloader.Download(ctx, tempFile, &s3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(s.config.S3ObjectKey),
	})

	if err != nil {
		return fmt.Errorf("download failed for s3://%s/%s: %w", s.config.S3Bucket, s.config.S3ObjectKey, err)
	}

	data, err := os.ReadFile(tempPath)
	if err != nil {
		return fmt.Errorf("failed to read temp file: %w", err)
	}

	salt, hashes, err := parseHashListJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	s.store.Load(salt, hashes, etagDigest)
	log.Info("loaded restricted addr list", "hash_count", len(hashes), "etag", etagDigest, "size_bytes", len(data))
	return nil
}

// parseHashListJSON parses the JSON hash list file.
// Expected format: {"salt": "hex...", "hashes": ["hex1", "hex2", ...]}
func parseHashListJSON(data []byte) ([]byte, [][32]byte, error) {
	var payload hashListPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	salt, err := hex.DecodeString(payload.Salt)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid salt hex: %w", err)
	}

	hashes := make([][32]byte, len(payload.AddressHashes))
	for i, h := range payload.AddressHashes {
		hashBytes, err := hex.DecodeString(h.Hash)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid hash hex at index %d: %w", i, err)
		}
		if len(hashBytes) != 32 {
			return nil, nil, fmt.Errorf("invalid hash length at index %d: got %d, want 32", i, len(hashBytes))
		}
		copy(hashes[i][:], hashBytes)
	}

	return salt, hashes, nil
}
