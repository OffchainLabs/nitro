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
		manager.handleHashListStream,
		fileSizeGauge,
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
	var hashCount int
	var listMeta *ListMeta
	fill := func(addHash func(common.Hash)) (*ListMeta, error) {
		var err error
		listMeta, err = parseHashListStream(r, func(h common.Hash) {
			hashCount++
			addHash(h)
		})
		return listMeta, err
	}
	err := s.hashStore.Store(digest, s3syncer.EstimateHashCount(size), fill)
	if err != nil {
		return fmt.Errorf("failed to parse hash list: %w", err)
	}

	log.Info("loaded restricted addr list", "filterSetID", listMeta.Id, "hash_count", hashCount, "etag", digest, "size_bytes", size, "scheme", listMeta.Scheme)
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

// decodeHashesArray streams the elements of the "hashes" array (or a JSON null)
// into addHash, one at a time, never holding more than one element in memory.
func decodeHashesArray(dec *json.Decoder, addHash func(common.Hash)) error {
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
		addHash(common.Hash(h))
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
// Expected format: {"id":"uuid-string-representation", "salt": "uuid-string-representation", "hashing_scheme": "<sha256-stringinput|sha256-rawbytesinput>", "hashes": ["0xhex1", "0xhex2", ...]}
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
			err = decodeHashesArray(dec, addHash)
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
	case HashingSchemeStringInput, HashingSchemeRawBytesInput:
	default:
		return nil, fmt.Errorf("unknown hashing_scheme %q", schemeStr)
	}

	salt, err := uuid.Parse(saltStr)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter set ID UUID: %w", err)
	}

	return &ListMeta{
		Id:     id,
		Salt:   salt,
		Scheme: scheme,
	}, nil
}
