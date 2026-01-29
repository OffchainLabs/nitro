// Copyright 2026, Offchain Labs, Inc.
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
	HashingScheme string `json:"hashing_scheme,omitempty"`
	AddressHashes []struct {
		Hash string `json:"hash"`
	} `json:"address_hashes"`
}

type S3SyncManager struct {
	Syncer    *s3syncer.Syncer
	hashStore *HashStore
}

func NewS3SyncManager(ctx context.Context, config *Config, hashStore *HashStore) (*S3SyncManager, error) {
	s := &S3SyncManager{
		hashStore: hashStore,
	}
	syncer, err := s3syncer.NewSyncer(
		ctx,
		&config.S3,
		s.handleHashListData,
	)

	if err != nil {
		return nil, err
	}

	s.Syncer = syncer
	return s, nil
}

// handleHashListData parses the downloaded JSON data and loads it into the hashStore.
func (s *S3SyncManager) handleHashListData(data []byte, digest string) error {
	salt, hashes, err := parseHashListJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	s.hashStore.Store(salt, hashes, digest)
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

	// Validate hashing scheme - warn if not Sha256 but continue for forward compatibility
	if payload.HashingScheme != "" && payload.HashingScheme != "Sha256" {
		log.Warn("unknown hashing scheme in address list, continuing with Sha256 assumption",
			"scheme", payload.HashingScheme)
	}

	salt, err := hex.DecodeString(payload.Salt)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid salt hex: %w", err)
	}
	if len(salt) == 0 {
		return nil, nil, fmt.Errorf("salt cannot be empty")
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
