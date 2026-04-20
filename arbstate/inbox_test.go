// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbstate

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider"
)

func buildSequencerMsg(payload []byte) []byte {
	// 40-byte L1 header (all zeros) + payload
	msg := make([]byte, 40+len(payload))
	// Leave the 40-byte header as zeros (valid timestamps/block numbers)
	copy(msg[40:], payload)
	return msg
}

func TestParseSequencerMessage_ShortAnyTrustNoReader(t *testing.T) {
	ctx := context.Background()
	registry := daprovider.NewReaderRegistry()
	// No AnyTrust reader registered - simulates a non-AnyTrust chain

	tests := []struct {
		name    string
		payload []byte
	}{
		{"header only", []byte{daprovider.DASMessageHeaderFlag}},
		{"header plus 1 byte", []byte{daprovider.DASMessageHeaderFlag, 0x01}},
		{"header plus 31 bytes", append([]byte{daprovider.DASMessageHeaderFlag}, make([]byte, 31)...)},
		{"header with tree flag only", []byte{daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag}},
		{"header with tree flag plus 31 bytes", append([]byte{daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag}, make([]byte, 31)...)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := buildSequencerMsg(tc.payload)
			msg, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate)
			if err != nil {
				t.Fatalf("expected no error for short AnyTrust message, got: %v", err)
			}
			if len(msg.Segments) != 0 {
				t.Fatalf("expected empty segments for short AnyTrust message, got %d segments", len(msg.Segments))
			}
		})
	}
}

func TestParseSequencerMessage_LongAnyTrustNoReaderErrors(t *testing.T) {
	ctx := context.Background()
	registry := daprovider.NewReaderRegistry()

	// 33 bytes: 1 header + 32 keyset hash - long enough that it should still
	// error when no AnyTrust reader is configured.
	headers := []struct {
		name   string
		header byte
	}{
		{"0x80", daprovider.DASMessageHeaderFlag},
		{"0x88", daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag},
	}
	for _, tc := range headers {
		t.Run(tc.name, func(t *testing.T) {
			payload := append([]byte{tc.header}, make([]byte, 32)...)
			data := buildSequencerMsg(payload)

			_, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate)
			if err == nil {
				t.Fatal("expected error for AnyTrust message with no reader configured, got nil")
			}
		})
	}
}
