// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

// hashListPayload represents the JSON structure of the hash list file used for unmarshalling.
type hashListPayload struct {
	Salt          string `json:"salt"`
	AddressHashes []struct {
		Hash string `json:"hash"`
	} `json:"address_hashes"`
}

type S3SyncManager struct {
	Syncer *s3syncer.Syncer
	store  *HashStore
}

func NewS3SyncManager(ctx context.Context, config *Config, store *HashStore) (*S3SyncManager, error) {
	s := &S3SyncManager{
		store: store,
	}
	syncer, err := s3syncer.NewSyncer(
		ctx,
		&config.S3,
		s.handleHashListData,
		// These are initial settings that can be tuned as needed.
		s3syncer.WithDownloadConfig(s3syncer.DownloadConfig{
			PartSizeMB:         100,
			Concurrency:        10,
			PartBodyMaxRetries: 5,
		}))

	if err != nil {
		return nil, err
	}

	s.Syncer = syncer
	return s, nil
}

// handleHashListData parses the downloaded JSON data and loads it into the store.
func (s *S3SyncManager) handleHashListData(data []byte, digest string) error {
	salt, hashes, err := parseHashListJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	s.store.Load(salt, hashes, digest)
	log.Info("loaded restricted addr list", "hash_count", len(hashes), "etag", digest, "size_bytes", len(data))
	return nil
}

// parseHashListJSON parses the JSON hash list file.
// Expected format: {"salt": "hex...", "address_hashes": [{"hash": "hex1"}, {"hash": "hex2"}, ...]}
func parseHashListJSON(data []byte) ([]byte, []common.Hash, error) {
	var payload hashListPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	salt, err := hex.DecodeString(payload.Salt)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid salt hex: %w", err)
	}

	hashes := make([]common.Hash, len(payload.AddressHashes))
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
