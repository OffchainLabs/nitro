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
	testHashHex1    = "1111111111111111111111111111111111111111111111111111111111111111"
	testHashHex2    = "2222222222222222222222222222222222222222222222222222222222222222"
	testAddressHex1 = "d3cda913deb6f67967b99d67acdfa1712c293601"
	testAddressHex2 = "5aeda56215b167893e80b4fe645ba6d5bab767de"
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
	wantHash := common.HexToHash("0x" + testHashHex1)
	wantAddr := common.BytesToHash(common.HexToAddress("0x" + testAddressHex1).Bytes())
	cases := []struct {
		name          string
		input         string
		wantHash      common.Hash
		wantIsAddress bool
		wantErr       bool
	}{
		{"bare", testHashHex1, wantHash, false, false},
		{"0x prefix", "0x" + testHashHex1, wantHash, false, false},
		{"0X prefix", "0X" + testHashHex1, wantHash, false, false},
		{"address bare", testAddressHex1, wantAddr, true, false},
		{"address 0x prefix", "0x" + testAddressHex1, wantAddr, true, false},
		{"address 0X prefix", "0X" + testAddressHex1, wantAddr, true, false},
		{"address checksummed case", "0xD3CdA913deB6f67967B99D67aCDFa1712C293601", common.BytesToHash(common.HexToAddress("0xD3CdA913deB6f67967B99D67aCDFa1712C293601").Bytes()), true, false},
		{name: "too short", input: "1234", wantErr: true},
		{name: "39 hex", input: testAddressHex1[1:], wantErr: true},
		{name: "41 hex", input: testAddressHex1 + "a", wantErr: true},
		{name: "63 hex", input: testHashHex1[1:], wantErr: true},
		{name: "bad hex", input: "zz" + testHashHex1[2:], wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var h jsonHash
			err := h.UnmarshalText([]byte(c.input))
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.wantHash, h.hash)
			require.Equal(t, c.wantIsAddress, h.isAddress)
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
			require.Equal(t, uuid.MustParse(id), meta.ID)
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

func TestParseHashListStreamPlaintext(t *testing.T) {
	const id = "0fa6d8c0-0000-0000-0000-000000000002"
	wantAddrs := []common.Hash{
		common.BytesToHash(common.HexToAddress("0x" + testAddressHex1).Bytes()),
		common.BytesToHash(common.HexToAddress("0x" + testAddressHex2).Bytes()),
	}

	t.Run("no salt", func(t *testing.T) {
		jsonDoc := `{"id":"` + id + `","hashing_scheme":"plaintext","hashes":["0x` + testAddressHex1 + `","0x` + testAddressHex2 + `"]}`
		meta, hashes, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Equal(t, uuid.MustParse(id), meta.ID)
		require.Equal(t, uuid.Nil, meta.Salt)
		require.Equal(t, HashingSchemePlaintext, meta.Scheme)
		require.Equal(t, wantAddrs, hashes)
	})

	t.Run("salt present is ignored", func(t *testing.T) {
		jsonDoc := `{"id":"` + id + `","salt":"2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6","hashing_scheme":"plaintext","hashes":["0x` + testAddressHex1 + `"]}`
		meta, _, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, meta.Salt)
	})

	t.Run("invalid salt is ignored", func(t *testing.T) {
		jsonDoc := `{"id":"` + id + `","salt":"not-a-uuid","hashing_scheme":"plaintext","hashes":["0x` + testAddressHex1 + `"]}`
		meta, _, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, meta.Salt)
	})

	t.Run("hashes before scheme", func(t *testing.T) {
		// The load-bearing case for streaming: entries decode before the scheme is known.
		jsonDoc := `{"hashes":["0x` + testAddressHex1 + `","0x` + testAddressHex2 + `"],"id":"` + id + `","hashing_scheme":"plaintext"}`
		meta, hashes, err := parseHashListBytes([]byte(jsonDoc))
		require.NoError(t, err)
		require.Equal(t, HashingSchemePlaintext, meta.Scheme)
		require.Equal(t, wantAddrs, hashes)
	})

	t.Run("hash entry rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + id + `","hashing_scheme":"plaintext","hashes":["0x` + testHashHex1 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "requires 40-hex address entries")
	})

	t.Run("mixed entries rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + id + `","hashing_scheme":"plaintext","hashes":["0x` + testAddressHex1 + `","0x` + testHashHex1 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "requires 40-hex address entries")
	})

	t.Run("missing id rejected", func(t *testing.T) {
		jsonDoc := `{"hashing_scheme":"plaintext","hashes":["0x` + testAddressHex1 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "invalid filter set ID UUID")
	})
}

func TestParseHashListStreamAddressEntriesRejectedForSHA256Schemes(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	for _, scheme := range []string{"", "sha256-stringinput", "sha256-rawbytesinput"} {
		t.Run("scheme "+scheme, func(t *testing.T) {
			schemeField := ""
			if scheme != "" {
				schemeField = `"hashing_scheme":"` + scheme + `",`
			}
			jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `",` + schemeField + `"hashes":["0x` + testAddressHex1 + `"]}`
			_, _, err := parseHashListBytes([]byte(jsonDoc))
			require.ErrorContains(t, err, "requires 64-hex hash entries")
		})
	}

	t.Run("mixed entries rejected", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","salt":"` + salt + `","hashing_scheme":"sha256-stringinput","hashes":["0x` + testHashHex1 + `","0x` + testAddressHex1 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "requires 64-hex hash entries")
	})

	t.Run("missing salt still rejected for default scheme", func(t *testing.T) {
		jsonDoc := `{"id":"` + uuid.NewString() + `","hashes":["0x` + testHashHex1 + `"]}`
		_, _, err := parseHashListBytes([]byte(jsonDoc))
		require.ErrorContains(t, err, "invalid UUID")
	})
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
