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
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

var (
	fileSizeGauge       = metrics.NewRegisteredGauge("arb/addressfilter/file/size", nil)
	fileTooLargeCounter = metrics.NewRegisteredCounter("arb/addressfilter/file/toolarge_total", nil)
	syncFailureCounter  = metrics.NewRegisteredCounter("arb/addressfilter/sync/failure_total", nil)
)

// trimHexPrefix strips a leading "0x" or "0X" prefix from a hex string.
func trimHexPrefix(s string) string {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return s
}

// hashListPayload represents the JSON structure of the hash list file used for unmarshalling.
type hashListPayload struct {
	Id            string   `json:"id"`
	Salt          string   `json:"salt"`
	HashingScheme string   `json:"hashing_scheme,omitempty"`
	Hashes        []string `json:"hashes"`
}

type parsedPayload struct {
	Id     uuid.UUID
	Salt   uuid.UUID
	Scheme HashingScheme
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
		fileSizeGauge,
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

	s.hashStore.Store(parsedData.Id, parsedData.Salt, parsedData.Scheme, parsedData.Hashes, digest)
	log.Info("loaded restricted addr list", "hash_count", len(parsedData.Hashes), "etag", digest, "size_bytes", len(data), "scheme", parsedData.Scheme)
	return nil
}

// parseHashListJSON parses the JSON hash list file.
// Expected format: {"id":"uuid-string-representation", "salt": "uuid-string-representation", "hashing_scheme": "<sha256-stringinput|sha256-rawbytesinput>", "hashes": ["0xhex1", "0xhex2", ...]}
func parseHashListJSON(data []byte) (*parsedPayload, error) {
	var payload hashListPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	scheme := HashingScheme(payload.HashingScheme)
	switch scheme {
	case "":
		scheme = HashingSchemeStringInput
	case HashingSchemeStringInput, HashingSchemeRawBytesInput:
	default:
		return nil, fmt.Errorf("unknown hashing_scheme %q", payload.HashingScheme)
	}

	salt, err := uuid.Parse(payload.Salt)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(payload.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid filter set ID UUID: %w", err)
	}

	hashes := make([]common.Hash, len(payload.Hashes))
	for i, h := range payload.Hashes {
		hashBytes, err := hex.DecodeString(trimHexPrefix(h))
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
		Scheme: scheme,
		Hashes: hashes,
	}, nil
}
