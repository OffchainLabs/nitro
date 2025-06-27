package arbutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"testing"

	espressoTypes "github.com/EspressoSystems/espresso-network/sdks/go/types"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func mockMsgFetcher(index MessageIndex) ([]byte, error) {
	return []byte("message" + fmt.Sprint(index)), nil
}

func TestParsePayload(t *testing.T) {
	msgPositions := []MessageIndex{1, 2, 10, 24, 100}

	rawPayload, cnt := BuildRawHotShotPayload(msgPositions, mockMsgFetcher, 200*1024)
	if cnt != len(msgPositions) {
		t.Fatal("exceed transactions")
	}

	mockSignature := []byte("fake_signature")
	fakeSigner := func(payload []byte) ([]byte, error) {
		return mockSignature, nil
	}
	signedPayload, err := SignHotShotPayload(rawPayload, fakeSigner)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}

	// Parse the signed payload
	signature, userDataHash, indices, messages, err := ParseHotShotPayload(signedPayload)
	if err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	if !slices.Equal(userDataHash, crypto.Keccak256(rawPayload)) {
		t.Fatalf("User data hash is not for the correct payload")
	}

	// Validate parsed data
	if !bytes.Equal(signature, mockSignature) {
		t.Errorf("expected signature 'fake_signature', got %v", mockSignature)
	}

	for i, index := range indices {
		if MessageIndex(index) != msgPositions[i] {
			t.Errorf("expected index %d, got %d", msgPositions[i], index)
		}
	}

	expectedMessages := [][]byte{
		[]byte("message1"),
		[]byte("message2"),
		[]byte("message10"),
		[]byte("message24"),
		[]byte("message100"),
	}
	for i, message := range messages {
		if !bytes.Equal(message, expectedMessages[i]) {
			t.Errorf("expected message %s, got %s", expectedMessages[i], message)
		}
	}
}

func TestValidateIfPayloadIsInBlock(t *testing.T) {
	msgPositions := []MessageIndex{1, 2}

	rawPayload, _ := BuildRawHotShotPayload(msgPositions, mockMsgFetcher, 200*1024)
	fakeSigner := func(payload []byte) ([]byte, error) {
		return []byte("fake_signature"), nil
	}
	signedPayload, err := SignHotShotPayload(rawPayload, fakeSigner)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}

	// Validate payload in a block
	blockPayloads := []espressoTypes.Bytes{
		signedPayload,
		[]byte("other_payload"),
	}

	if !ValidateIfPayloadIsInBlock(signedPayload, blockPayloads) {
		t.Error("expected payload to be validated in block")
	}

	if ValidateIfPayloadIsInBlock([]byte("invalid_payload"), blockPayloads) {
		t.Error("did not expect invalid payload to be validated in block")
	}
}

func TestParsePayloadInvalidCases(t *testing.T) {
	invalidPayloads := []struct {
		description string
		payload     []byte
	}{
		{
			description: "Empty payload",
			payload:     []byte{},
		},
		{
			description: "Message size exceeds remaining payload",
			payload: func() []byte {
				var payload []byte
				sigSizeBuf := make([]byte, 8)
				binary.BigEndian.PutUint64(sigSizeBuf, 0)
				payload = append(payload, sigSizeBuf...)
				msgSizeBuf := make([]byte, 8)
				binary.BigEndian.PutUint64(msgSizeBuf, 100)
				payload = append(payload, msgSizeBuf...)
				return payload
			}(),
		},
	}

	for _, tc := range invalidPayloads {
		t.Run(tc.description, func(t *testing.T) {
			_, _, _, _, err := ParseHotShotPayload(tc.payload)
			if err == nil {
				t.Errorf("expected error for case '%s', but got none", tc.description)
			}
		})
	}
}

func TestSerdeSubmittedEspressoTx(t *testing.T) {
	submiitedTx := SubmittedEspressoTx{
		Hash:    "0x1234",
		Pos:     []MessageIndex{MessageIndex(10)},
		Payload: []byte{0, 1, 2, 3},
	}

	b, err := rlp.EncodeToBytes(&submiitedTx)
	if err != nil {
		t.Error("failed to encode")
	}

	var expected SubmittedEspressoTx
	err = rlp.DecodeBytes(b, &expected)
	if err != nil {
		t.Error("failed to encode")
	}

	if submiitedTx.Hash != expected.Hash {
		t.Error("failed to check hash")
	}

	if submiitedTx.Pos[0] != expected.Pos[0] {
		t.Error("failed to check pos")
	}

	if !bytes.Equal(submiitedTx.Payload, expected.Payload) {
		t.Error("failed to check payload")
	}
}

func TestSerdeSubmittedEspressoTxBackwardCompatibility(t *testing.T) {
	// This test ensures that we can deserialize the old SubmittedEspressoTx, which did not have
	// the `SubmittedAt` field, into the new struct. It uses a static RLP-encoded artifact
	// generated from the old struct definition. See testdata/old_submitted_espresso_tx.md for details.
	type OldSubmittedEspressoTx struct {
		Hash    string
		Pos     []MessageIndex
		Payload []byte
	}

	// This represents the original data that was used to create the RLP artifact.
	oldSubmittedTx := OldSubmittedEspressoTx{
		Hash:    "0x1234",
		Pos:     []MessageIndex{MessageIndex(10)},
		Payload: []byte{0, 1, 2, 3},
	}

	b, err := os.ReadFile("testdata/old_submitted_espresso_tx.rlp")
	if err != nil {
		t.Fatalf("Failed to read RLP artifact: %v", err)
	}

	// First, validate that the artifact correctly decodes to the old struct format.
	var decodedOldTx OldSubmittedEspressoTx
	if err := rlp.DecodeBytes(b, &decodedOldTx); err != nil {
		t.Fatalf("Failed to decode artifact into OldSubmittedEspressoTx: %v", err)
	}
	if decodedOldTx.Hash != oldSubmittedTx.Hash {
		t.Errorf("Hash mismatch in old struct: got %v, want %v", decodedOldTx.Hash, oldSubmittedTx.Hash)
	}
	if len(decodedOldTx.Pos) != 1 || decodedOldTx.Pos[0] != oldSubmittedTx.Pos[0] {
		t.Errorf("Pos mismatch in old struct: got %v, want %v", decodedOldTx.Pos, oldSubmittedTx.Pos)
	}
	if !bytes.Equal(decodedOldTx.Payload, oldSubmittedTx.Payload) {
		t.Errorf("Payload mismatch in old struct: got %x, want %x", decodedOldTx.Payload, oldSubmittedTx.Payload)
	}

	// Now, decode the same artifact into the new struct to test backward compatibility.
	var expected SubmittedEspressoTx
	if err := rlp.DecodeBytes(b, &expected); err != nil {
		t.Fatalf("Failed to decode artifact into new SubmittedEspressoTx: %v", err)
	}

	if oldSubmittedTx.Hash != expected.Hash {
		t.Errorf("Failed to check hash after decoding, got %v, want %v", expected.Hash, oldSubmittedTx.Hash)
	}

	if len(expected.Pos) != 1 || oldSubmittedTx.Pos[0] != expected.Pos[0] {
		t.Errorf("Pos mismatch: got %v, want %v", expected.Pos, oldSubmittedTx.Pos)
	}

	if !bytes.Equal(oldSubmittedTx.Payload, expected.Payload) {
		t.Errorf("Payload mismatch: got %x, want %x", expected.Payload, oldSubmittedTx.Payload)
	}

	if !expected.SubmittedAt.IsZero() {
		t.Errorf("Expected SubmittedAt to be zero after decoding old data, but got %v", expected.SubmittedAt)
	}
}
