// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

// Aggregate counters across all configured files; the per-file size gauge is
// registered dynamically in newFileSizeGauge.
var (
	fileTooLargeCounter = metrics.NewRegisteredCounter("arb/addressfilter/file/toolarge_total", nil)
	syncFailureCounter  = metrics.NewRegisteredCounter("arb/addressfilter/sync/failure_total", nil)
)

func newFileSizeGauge(fileConfig *FileConfig) *metrics.Gauge {
	name := fmt.Sprintf("arb/addressfilter/file/%s/%s/size",
		sanitizeMetricName(fileConfig.Bucket), sanitizeMetricName(fileConfig.ObjectKey))
	return metrics.GetOrRegisterGauge(name, nil)
}

// sanitizeMetricName replaces characters outside [a-zA-Z0-9_] with
// underscores, so bucket names and object keys can't inject metric-path
// separators or characters that metrics backends reject.
func sanitizeMetricName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}

// jsonHash is one decoded hash-list entry. It accepts two forms, each with an
// optional "0x"/"0X" prefix: a 64-hex-char 32-byte hash, or a 40-hex-char
// plain address (plaintext scheme). An address is placed in the low-order 20
// bytes of hash with the upper bytes zeroed — the same layout
// common.BytesToHash produces on the lookup side in hashAddress.
type jsonHash struct {
	hash      common.Hash
	isAddress bool
}

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
	switch len(text) {
	case 2 * common.HashLength:
		_, err := hex.Decode(h.hash[:], text)
		return err
	case 2 * common.AddressLength:
		h.isAddress = true
		_, err := hex.Decode(h.hash[common.HashLength-common.AddressLength:], text)
		return err
	default:
		return fmt.Errorf("invalid entry length: got %d hex chars, want %d (hash) or %d (address)",
			len(text), 2*common.HashLength, 2*common.AddressLength)
	}
}

type S3SyncManager struct {
	Syncer               *s3syncer.Syncer
	hashStore            *HashStore
	bucket               string
	objectKey            string
	minBytesPerHashEntry int
}

// NewS3SyncManager creates a sync manager for one hash-list file. fileConfig
// must have gone through withDefaults so that MinBytesPerHashEntry is positive.
func NewS3SyncManager(fileConfig *FileConfig, hashStore *HashStore, objectSizeGauge *metrics.Gauge) *S3SyncManager {
	manager := &S3SyncManager{
		hashStore:            hashStore,
		bucket:               fileConfig.Bucket,
		objectKey:            fileConfig.ObjectKey,
		minBytesPerHashEntry: fileConfig.MinBytesPerHashEntry,
	}
	syncer := s3syncer.NewSyncer(
		&fileConfig.Config,
		manager.handleHashListStream,
		objectSizeGauge,
	)

	manager.Syncer = syncer
	return manager
}

func (s *S3SyncManager) Initialize(ctx context.Context) error {
	return s.Syncer.Initialize(ctx)
}

// handleHashListStream parses the hash list JSON from the stream, loading the
// hashes into the hashStore as they decode; the new list is published only if
// the whole document parses and validates.
func (s *S3SyncManager) handleHashListStream(r io.Reader, size int64, digest string) error {
	var listMeta *ListMeta
	fill := func(addHash func(common.Hash)) (*ListMeta, error) {
		var err error
		listMeta, err = parseHashListStream(r, addHash)
		return listMeta, err
	}
	err := s.hashStore.Store(digest, estimateHashCount(size, s.minBytesPerHashEntry), fill)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	log.Info("loaded restricted addr list", "bucket", s.bucket, "key", s.objectKey, "filterSetID", listMeta.ID, "hash_count", s.hashStore.Size(), "etag", digest, "size_bytes", size, "scheme", listMeta.Scheme)
	return nil
}

// decodeStringToken decodes the next token, requiring a JSON string.
func decodeStringToken(dec *json.Decoder, key string) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	s, ok := tok.(string)
	if !ok {
		return "", fmt.Errorf("%s: expected JSON string, got %v", key, tok)
	}
	return s, nil
}

// entryKinds records which entry lengths appeared in the hashes array, so the
// scheme/entry-length consistency can be validated once the scheme is known
// (fields may appear in any order, so the scheme may arrive after the array).
type entryKinds struct {
	sawHash    bool
	sawAddress bool
}

// decodeHashesArray streams the elements of the "hashes" array (or a JSON null)
// into addHash, one at a time, never holding more than one element in memory.
func decodeHashesArray(dec *json.Decoder, addHash func(common.Hash), kinds *entryKinds) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("hashes: %w", err)
	}
	if tok == nil {
		return nil // JSON null: same as an empty array
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return errors.New("hashes: expected JSON array")
	}
	for i := 0; dec.More(); i++ {
		var h jsonHash
		if err := dec.Decode(&h); err != nil {
			return fmt.Errorf("hashes[%d]: %w", i, err)
		}
		if h.isAddress {
			kinds.sawAddress = true
		} else {
			kinds.sawHash = true
		}
		addHash(h.hash)
	}
	if _, err := dec.Token(); err != nil { // consume the closing ']'
		return fmt.Errorf("hashes: %w", err)
	}
	return nil
}

// skipJSONValue consumes the next JSON value without buffering it: scalars are
// a single token, and composites are skipped by tracking delimiter depth (the
// decoder itself validates delimiter matching).
func skipJSONValue(dec *json.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		if depth == 0 {
			return nil
		}
	}
}

// parseHashListStream parses the hash list JSON document from r, streaming each
// decoded hash to addHash, so the document is never buffered in memory. Fields may
// appear in any order; the metadata is validated and returned once the whole
// document has been consumed. Unknown fields are ignored.
// Expected format: {"id":"uuid-string-representation", "salt": "uuid-string-representation", "hashing_scheme": "<sha256-stringinput|sha256-rawbytesinput|plaintext>", "hashes": ["0xhex1", "0xhex2", ...]}
// For the sha256 schemes each entry is a 64-hex hash and "salt" is required.
// For "plaintext" each entry is a plain 40-hex address (stored left-padded to
// 32 bytes) and "salt" is ignored.
func parseHashListStream(r io.Reader, addHash func(common.Hash)) (*ListMeta, error) {
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("JSON parse failed: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("expected JSON object, got %v", tok)
	}

	var idStr, saltStr, schemeStr string
	var kinds entryKinds
	seenHashes := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("JSON parse failed: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected object key, got %v", keyTok)
		}
		switch key {
		case "id":
			idStr, err = decodeStringToken(dec, key)
		case "salt":
			saltStr, err = decodeStringToken(dec, key)
		case "hashing_scheme":
			schemeStr, err = decodeStringToken(dec, key)
		case "hashes":
			// A duplicate would stream both arrays into addHash; reject it instead of
			// silently unioning them.
			if seenHashes {
				return nil, errors.New("duplicate hashes field")
			}
			seenHashes = true
			err = decodeHashesArray(dec, addHash, &kinds)
		default:
			err = skipJSONValue(dec)
		}
		if err != nil {
			return nil, err
		}
	}
	if _, err := dec.Token(); err != nil { // consume the closing '}'
		return nil, fmt.Errorf("JSON parse failed: %w", err)
	}
	switch tok, err := dec.Token(); {
	case errors.Is(err, io.EOF): // expected: the document consumed the whole stream
	case err != nil:
		return nil, fmt.Errorf("unexpected content after JSON document: %w", err)
	default:
		return nil, fmt.Errorf("unexpected content after JSON document: token %v", tok)
	}

	scheme := HashingScheme(schemeStr)
	switch scheme {
	case "":
		scheme = HashingSchemeStringInput
	case HashingSchemeStringInput, HashingSchemeRawBytesInput, HashingSchemePlaintext:
	default:
		return nil, fmt.Errorf("unknown hashing_scheme %q", schemeStr)
	}

	if scheme == HashingSchemePlaintext {
		if kinds.sawHash {
			return nil, fmt.Errorf("hashing_scheme %q requires %d-hex address entries, found %d-hex hash entries",
				scheme, 2*common.AddressLength, 2*common.HashLength)
		}
	} else if kinds.sawAddress {
		return nil, fmt.Errorf("hashing_scheme %q requires %d-hex hash entries, found %d-hex address entries",
			scheme, 2*common.HashLength, 2*common.AddressLength)
	}

	salt := uuid.Nil
	if scheme != HashingSchemePlaintext {
		var err error
		salt, err = uuid.Parse(saltStr)
		if err != nil {
			return nil, err
		}
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter set ID UUID: %w", err)
	}

	return &ListMeta{
		ID:     id,
		Salt:   salt,
		Scheme: scheme,
	}, nil
}
