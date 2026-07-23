// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

const (
	testHashHex1 = "1111111111111111111111111111111111111111111111111111111111111111"
	testHashHex2 = "2222222222222222222222222222222222222222222222222222222222222222"
)

func makeHashesJSON(t *testing.T, salt string, hashes []string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"id": uuid.NewString(), "salt": salt, "hashes": hashes})
	require.NoError(t, err)
	return b
}

// parseHashListBytes runs the streaming parser over an in-memory document,
// collecting the streamed hashes. Test convenience wrapper.
func parseHashListBytes(data []byte) (*ListMeta, []common.Hash, error) {
	var hashes []common.Hash
	meta, err := parseHashListStream(bytes.NewReader(data), func(h common.Hash) { hashes = append(hashes, h) })
	return meta, hashes, err
}

func TestJSONHashUnmarshalText(t *testing.T) {
	want := common.HexToHash("0x" + testHashHex1)
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"bare", testHashHex1, false},
		{"0x prefix", "0x" + testHashHex1, false},
		{"0X prefix", "0X" + testHashHex1, false},
		{"too short", "1234", true},
		{"bad hex", "zz" + testHashHex1[2:], true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var h jsonHash
			err := h.UnmarshalText([]byte(c.in))
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, common.Hash(h))
		})
	}
}

func TestParseHashListStreamFieldOrder(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	const id = "0fa6d8c0-0000-0000-0000-000000000001"
	h1 := common.HexToHash("0x" + testHashHex1)
	h2 := common.HexToHash("0x" + testHashHex2)

	cases := []struct {
		name    string
		jsonDoc string
	}{
		{
			"metadata before hashes",
			`{"id":"` + id + `","salt":"` + salt + `","hashing_scheme":"sha256-stringinput","hashes":["` + testHashHex1 + `","` + testHashHex2 + `"]}`,
		},
		{
			// The load-bearing case for streaming: hashes decode before the salt is known.
			"metadata after hashes",
			`{"hashes":["` + testHashHex1 + `","` + testHashHex2 + `"],"id":"` + id + `","salt":"` + salt + `","hashing_scheme":"sha256-stringinput"}`,
		},
		{
			"metadata interleaved with hashes",
			`{"id":"` + id + `","hashes":["` + testHashHex1 + `","` + testHashHex2 + `"],"salt":"` + salt + `","hashing_scheme":"sha256-stringinput"}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			meta, hashes, err := parseHashListBytes([]byte(c.jsonDoc))
			require.NoError(t, err)
			require.Equal(t, uuid.MustParse(id), meta.Id)
			require.Equal(t, uuid.MustParse(salt), meta.Salt)
			require.Equal(t, HashingSchemeStringInput, meta.Scheme)
			require.Equal(t, []common.Hash{h1, h2}, hashes)
		})
	}
}

func TestParseHashListStreamHashesVariants(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"

	t.Run("empty array", func(t *testing.T) {
		_, hashes, err := parseHashListBytes(makeHashesJSON(t, salt, []string{}))
		require.NoError(t, err)
		require.Empty(t, hashes)
	})

	t.Run("null hashes", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":null}`
		_, hashes, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Empty(t, hashes)
	})

	t.Run("missing hashes", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `"}`
		_, hashes, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Empty(t, hashes)
	})

	t.Run("whitespace tolerated", func(t *testing.T) {
		jsonDoc := " {\n\"id\": \"" + uuid.NewString() + "\" ,\n\"salt\": \"" + salt + "\" ,\n\"hashes\": [ \"0x" + testHashHex1 + "\" ]\n} "
		_, hashes, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Len(t, hashes, 1)
	})

	t.Run("non-array hashes", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":"x"}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "expected JSON array")
	})

	t.Run("null element rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":["` + testHashHex1 + `",null]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "hashes[1]", "null element should be rejected, not decoded to the zero hash")
	})

	t.Run("non-string element rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":[123]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.Error(t, err)
	})

	t.Run("duplicate hashes field rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":["` + testHashHex1 + `"],"hashes":["` + testHashHex2 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "duplicate hashes")
	})

	t.Run("error mid-array streams the prefix", func(t *testing.T) {
		jsonDoc := `{"salt":"` + salt + `","hashes":["` + testHashHex1 + `","bad","` + testHashHex2 + `"]}`
		var streamed []common.Hash
		_, err := parseHashListStream(bytes.NewReader([]byte(jsonDoc)), func(h common.Hash) { streamed = append(streamed, h) })
		require.ErrorContains(t, err, "hashes[1]")
		require.Equal(t, []common.Hash{common.HexToHash("0x" + testHashHex1)}, streamed,
			"elements before the bad one stream to add; the caller must discard on error")
	})
}

func TestParseHashListStreamUnknownFields(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	jsonDoc := `{
		"issued_at": "2026-05-13T12:05:45Z",
		"count": 1,
		"nested": {"a": [1, {"b": null}], "c": "d"},
		"list": [[]],
		"salt": "` + salt + `",
		"id": "` + uuid.NewString() + `",
		"hashes": ["` + testHashHex1 + `"],
		"trailer": true
	}`
	meta, hashes, err := parseHashListBytes([]byte(jsonDoc))
	require.NoError(t, err)
	require.Equal(t, uuid.MustParse(salt), meta.Salt)
	require.Len(t, hashes, 1)
}

func TestParseHashListStreamMalformed(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	valid := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashes":[]}`

	cases := []struct {
		name    string
		jsonDoc string
	}{
		{"not json", "not json"},
		{"top-level array", `[]`},
		{"truncated object", `{"salt":"` + salt + `"`},
		{"truncated array", `{"salt":"` + salt + `","hashes":["` + testHashHex1 + `"`},
		{"trailing garbage", valid + "garbage"},
		{"second document", valid + valid},
		{"non-string metadata", `{"id":1,"salt":"` + salt + `","hashes":[]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := parseHashListBytes([]byte(c.jsonDoc))
			require.Error(t, err)
		})
	}
}
