// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/warmbuffer"
)

var (
	fileSizeGauge       = metrics.NewRegisteredGauge("arb/addressfilter/file/size", nil)
	fileTooLargeCounter = metrics.NewRegisteredCounter("arb/addressfilter/file/toolarge_total", nil)
	syncFailureCounter  = metrics.NewRegisteredCounter("arb/addressfilter/sync/failure_total", nil)
)

// jsonHash decodes a single hex hash string in place, with no Go-string allocation, supporting an optional "0x"/"0X".
type jsonHash common.Hash

// UnmarshalJSON requires the element to be a JSON string and hands its contents to UnmarshalText. It implements
// json.Unmarshaler so non-string elements (null, numbers) are rejected; a bare TextUnmarshaler would instead leave
// them as the zero hash, since encoding/json skips it for null and other non-strings.
func (h *jsonHash) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("expected hash string, got %s", b)
	}
	return h.UnmarshalText(b[1 : len(b)-1])
}

func (h *jsonHash) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	text = bytes.TrimPrefix(text, []byte("0X"))
	if len(text) != 2*common.HashLength {
		return fmt.Errorf("invalid hash length: got %d hex chars, want %d", len(text), 2*common.HashLength)
	}
	_, err := hex.Decode(h[:], text)
	return err
}

// hashArray decodes a JSON array of hex hashes directly into its backing slice, reusing the capacity when preallocated
// and growing otherwise.
type hashArray []common.Hash

func (a *hashArray) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("hashes: expected JSON array")
	}
	dst := (*a)[:0]
	for dec.More() {
		i := len(dst)
		dst = append(dst, common.Hash{})
		if err := dec.Decode((*jsonHash)(&dst[i])); err != nil {
			return fmt.Errorf("hashes[%d]: %w", i, err)
		}
	}
	*a = dst
	return nil
}

// hashListPayload represents the JSON structure of the hash list file used for unmarshalling.
type hashListPayload struct {
	Id            string    `json:"id"`
	Salt          string    `json:"salt"`
	HashingScheme string    `json:"hashing_scheme,omitempty"`
	Hashes        hashArray `json:"hashes"`
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
	// hashesBacking is the preallocated slice the JSON hashes decode into, reused
	// across reloads; nil when preallocation is disabled.
	hashesBacking []common.Hash
}

func NewS3SyncManager(config *Config, hashStore *HashStore) *S3SyncManager {
	manager := &S3SyncManager{
		hashStore: hashStore,
	}
	if maxHashes := config.S3.NumPreallocatedHashes(); maxHashes > 0 {
		manager.hashesBacking = warmbuffer.MakeWarmArray[common.Hash](maxHashes)
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
// ctx is unused: the generic DataHandler provides it, but Store cannot block or fail.
func (s *S3SyncManager) handleHashListData(_ context.Context, data []byte, digest string) error {
	parsedData, err := parseHashListJSONInto(data, s.hashesBacking)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	s.hashStore.Store(parsedData.Id, parsedData.Salt, parsedData.Scheme, parsedData.Hashes, digest)
	log.Info("loaded restricted addr list", "filterSetID", parsedData.Id, "hash_count", len(parsedData.Hashes), "etag", digest, "size_bytes", len(data), "scheme", parsedData.Scheme)
	return nil
}

// parseHashListJSONInto parses the JSON hash list file. When backing is non-nil
// the hashes decode into it (reused across reloads); otherwise a new slice grows.
// Expected format: {"id":"uuid-string-representation", "salt": "uuid-string-representation", "hashing_scheme": "<sha256-stringinput|sha256-rawbytesinput>", "hashes": ["0xhex1", "0xhex2", ...]}
func parseHashListJSONInto(data []byte, backing []common.Hash) (*parsedPayload, error) {
	var payload hashListPayload
	if backing != nil {
		payload.Hashes = hashArray(backing[:0])
	}
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

	return &parsedPayload{
		Id:     id,
		Salt:   salt,
		Scheme: scheme,
		Hashes: []common.Hash(payload.Hashes),
	}, nil
}
