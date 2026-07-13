// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
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

func TestHashArrayUnmarshalJSON(t *testing.T) {
	h1 := common.HexToHash("0x" + testHashHex1)
	h2 := common.HexToHash("0x" + testHashHex2)

	var normal hashArray
	require.NoError(t, json.Unmarshal([]byte(`["0x`+testHashHex1+`","`+testHashHex2+`"]`), &normal))
	require.Equal(t, []common.Hash{h1, h2}, []common.Hash(normal))

	var empty hashArray
	require.NoError(t, json.Unmarshal([]byte(`[]`), &empty))
	require.Len(t, empty, 0)

	var null hashArray
	require.NoError(t, json.Unmarshal([]byte(`null`), &null))
	require.Len(t, null, 0)

	var whitespace hashArray
	require.NoError(t, json.Unmarshal([]byte(" [ \"0x"+testHashHex1+"\" ] "), &whitespace))
	require.Len(t, whitespace, 1)

	var notArray hashArray
	require.Error(t, json.Unmarshal([]byte(`"x"`), &notArray), "non-array should error")

	var nullElem hashArray
	require.Error(t, json.Unmarshal([]byte(`["`+testHashHex1+`",null]`), &nullElem), "null element should be rejected, not decoded to the zero hash")

	var numberElem hashArray
	require.Error(t, json.Unmarshal([]byte(`[123]`), &numberElem), "non-string element should be rejected")
}

func TestParseHashListJSONReusesBacking(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	backing := make([]common.Hash, 8)

	p1, err := parseHashListJSONInto(makeHashesJSON(t, salt, []string{testHashHex1}), backing)
	require.NoError(t, err)
	require.Len(t, p1.Hashes, 1)
	require.Same(t, &backing[0], &p1.Hashes[0], "first parse should decode into the backing")

	p2, err := parseHashListJSONInto(makeHashesJSON(t, salt, []string{testHashHex1, testHashHex2}), backing)
	require.NoError(t, err)
	require.Len(t, p2.Hashes, 2)
	require.Same(t, &backing[0], &p2.Hashes[0], "second parse should reuse the backing, no realloc")
}

func TestParseHashListJSONNilBackingGrows(t *testing.T) {
	const salt = "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6"
	p, err := parseHashListJSONInto(makeHashesJSON(t, salt, []string{testHashHex1, testHashHex2}), nil)
	require.NoError(t, err)
	require.Len(t, p.Hashes, 2)
}
