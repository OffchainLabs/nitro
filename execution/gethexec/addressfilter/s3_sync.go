// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

// trimHexPrefix strips a leading "0x" or "0X" prefix from a hex string.
func trimHexPrefix(s string) string {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return s
}

// hashListPayload represents the JSON structure of the hash list file used for unmarshalling.
type hashListPayload struct {
	Id            string `json:"id"`
	Salt          string `json:"salt"`
	HashingScheme string `json:"hashing_scheme,omitempty"`
	AddressHashes []struct {
		Hash string `json:"hash"`
	} `json:"address_hashes"`
}

type parsedPayload struct {
	Id     uuid.UUID
	Salt   uuid.UUID
	Hashes []common.Hash
}

type S3SyncManager struct {
	Syncer    *s3syncer.Syncer
	hashStore *HashStore
}

func NewS3SyncManager(config *Config, hashStore *HashStore) *S3SyncManager {
	manager := &S3SyncManager{
		hashStore: hashStore,
	}
	syncer := s3syncer.NewSyncer(
		&config.S3,
		manager.handleHashListData,
	)

	manager.Syncer = syncer
	return manager
}

func (s *S3SyncManager) Initialize(ctx context.Context) error {
	return s.Syncer.Initialize(ctx)
}

// handleHashListData parses the downloaded JSON data and loads it into the hashStore.
func (s *S3SyncManager) handleHashListData(data []byte, digest string) error {
	parsedData, err := parseHashListJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	s.hashStore.Store(parsedData.Id, parsedData.Salt, parsedData.Hashes, digest)
	log.Info("loaded restricted addr list", "hash_count", len(parsedData.Hashes), "etag", digest, "size_bytes", len(data))
	return nil
}

// parseHashListJSON parses the JSON hash list file.
// Expected format: {"salt": "uuid-string-representation", "address_hashes": [{"hash": "hex1"}, {"hash": "hex2"}, ...]}
func parseHashListJSON(data []byte) (*parsedPayload, error) {
	var payload hashListPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Validate hashing scheme - warn if not Sha256 but continue for forward compatibility
	if payload.HashingScheme != "" && payload.HashingScheme != "Sha256" {
		log.Warn("unknown hashing scheme in address list, continuing with Sha256 assumption",
			"scheme", payload.HashingScheme)
	}

	salt, err := uuid.Parse(payload.Salt)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(payload.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid filter set ID UUID: %w", err)
	}

	hashes := make([]common.Hash, len(payload.AddressHashes))
	for i, h := range payload.AddressHashes {
		hashBytes, err := hex.DecodeString(trimHexPrefix(h.Hash))
		if err != nil {
			return nil, fmt.Errorf("invalid hash hex at index %d: %w", i, err)
		}
		if len(hashBytes) != 32 {
			return nil, fmt.Errorf("invalid hash length at index %d: got %d, want 32", i, len(hashBytes))
		}
		copy(hashes[i][:], hashBytes)
	}
	return &parsedPayload{
		Id:     id,
		Salt:   salt,
		Hashes: hashes,
	}, nil
}
