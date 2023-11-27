package espresso

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func removeWhitespace(s string) string {
	// Split the string on whitespace then concatenate the segments
	return strings.Join(strings.Fields(s), "")
}

// Reference data taken from the reference sequencer implementation
// (https://github.com/EspressoSystems/espresso-sequencer/blob/main/data)

var ReferenceNmtRoot NmtRoot = NmtRoot{
	Root: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

var ReferenceL1BLockInfo L1BlockInfo = L1BlockInfo{
	Number:    123,
	Timestamp: *NewU256().SetUint64(0x456),
	Hash:      common.Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
}

var ReferenceHeader Header = Header{
	TransactionsRoot: ReferenceNmtRoot,
	Metadata: Metadata{
		Timestamp:   789,
		L1Head:      124,
		L1Finalized: &ReferenceL1BLockInfo,
	},
}

func TestEspressoTypesNmtRootJson(t *testing.T) {
	data := []byte(removeWhitespace(`{
		"root": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
	}`))

	// Check encoding.
	encoded, err := json.Marshal(ReferenceNmtRoot)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	require.Equal(t, encoded, data)

	// Check decoding
	var decoded NmtRoot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	require.Equal(t, decoded, ReferenceNmtRoot)

	CheckJsonRequiredFields[NmtRoot](t, data, "root")
}

func TestEspressoTypesL1BLockInfoJson(t *testing.T) {
	data := []byte(removeWhitespace(`{
		"number": 123,
		"timestamp": "0x456",
		"hash": "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}`))

	// Check encoding.
	encoded, err := json.Marshal(ReferenceL1BLockInfo)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	require.Equal(t, encoded, data)

	// Check decoding
	var decoded L1BlockInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	require.Equal(t, decoded, ReferenceL1BLockInfo)

	CheckJsonRequiredFields[L1BlockInfo](t, data, "number", "timestamp", "hash")
}

func TestEspressoTypesHeaderJson(t *testing.T) {
	data := []byte(removeWhitespace(`{
		"transactions_root": {
			"root": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
		},
		"metadata": {
			"timestamp": 789,
			"l1_head": 124,
			"l1_finalized": {
				"number": 123,
				"timestamp": "0x456",
				"hash": "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			}
		}
	}`))

	// Check encoding.
	encoded, err := json.Marshal(ReferenceHeader)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	require.Equal(t, encoded, data)

	// Check decoding
	var decoded Header
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	require.Equal(t, decoded, ReferenceHeader)

	CheckJsonRequiredFields[Header](t, data, "transactions_root", "metadata")
}

func TestEspressoMetadataJson(t *testing.T) {
	data := []byte(removeWhitespace(`{
			"timestamp": 789,
			"l1_head": 124,
			"l1_finalized": {
				"number": 123,
				"timestamp": "0x456",
				"hash": "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			}
		}`))
	m := ReferenceHeader.Metadata

	// Check encoding.
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	require.Equal(t, encoded, data)

	// Check decoding
	var decoded Metadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	require.Equal(t, decoded, m)

	CheckJsonRequiredFields[Metadata](t, data, "timestamp", "l1_head")
}

func TestEspressoTransactionJson(t *testing.T) {
	data := []byte(removeWhitespace(`{
		"vm": 0,
		"payload": [1,2,3,4,5]
	}`))
	tx := Transaction{
		Vm:      0,
		Payload: []byte{1, 2, 3, 4, 5},
	}

	// Check encoding.
	encoded, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	require.Equal(t, encoded, data)

	// Check decoding
	var decoded Transaction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	require.Equal(t, decoded, tx)

	CheckJsonRequiredFields[Transaction](t, data, "vm", "payload")
}

// Commitment tests ported from the reference sequencer implementation
// (https://github.com/EspressoSystems/espresso-sequencer/blob/main/sequencer/src/block.rs)

func TestEspressoTypesNmtRootCommit(t *testing.T) {
	require.Equal(t, ReferenceNmtRoot.Commit(), Commitment{251, 80, 232, 195, 91, 2, 138, 18, 240, 231, 31, 172, 54, 204, 90, 42, 215, 42, 72, 187, 15, 28, 128, 67, 149, 117, 26, 114, 232, 57, 190, 10})
}

func TestEspressoTypesL1BlockInfoCommit(t *testing.T) {
	require.Equal(t, ReferenceL1BLockInfo.Commit(), Commitment{224, 122, 115, 150, 226, 202, 216, 139, 51, 221, 23, 79, 54, 243, 107, 12, 12, 144, 113, 99, 133, 217, 207, 73, 120, 182, 115, 84, 210, 230, 126, 148})
}

func TestEspressoTypesHeaderCommit(t *testing.T) {
	require.Equal(t, ReferenceHeader.Commit(), Commitment{26, 77, 186, 162, 251, 241, 135, 23, 132, 5, 196, 207, 131, 64, 207, 215, 141, 144, 146, 65, 158, 30, 169, 102, 251, 183, 101, 149, 168, 173, 161, 149})
}

func TestEspressoCommitmentFromU256TrailingZero(t *testing.T) {
	comm := Commitment{209, 146, 197, 195, 145, 148, 17, 211, 52, 72, 28, 120, 88, 182, 204, 206, 77, 36, 56, 35, 3, 143, 77, 186, 69, 233, 104, 30, 90, 105, 48, 0}
	roundTrip, err := CommitmentFromUint256(comm.Uint256())
	require.Nil(t, err)
	require.Equal(t, comm, roundTrip)
}

func CheckJsonRequiredFields[T any](t *testing.T, data []byte, fields ...string) {
	// Parse the JSON object into a map so we can selectively delete fields.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	for _, field := range fields {
		data, err := json.Marshal(withoutKey(obj, field))
		require.Nil(t, err, "failed to marshal JSON")

		var dec T
		err = json.Unmarshal(data, &dec)
		require.NotNil(t, err, "unmarshalling without required field %s should fail", field)
	}
}

func withoutKey[K comparable, V any](m map[K]V, key K) map[K]V {
	copied := make(map[K]V)
	for k, v := range m {
		if k != key {
			copied[k] = v
		}
	}
	return copied
}
