// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbstate

import (
	"context"
	"encoding/binary"
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
	registry := daprovider.NewDAProviderRegistry()
	// No AnyTrust reader registered - simulates a non-AnyTrust chain

	tests := []struct {
		name    string
		payload []byte
	}{
		{"header only", []byte{daprovider.AnyTrustMessageHeaderFlag}},
		{"header plus 1 byte", []byte{daprovider.AnyTrustMessageHeaderFlag, 0x01}},
		{"header plus 31 bytes", append([]byte{daprovider.AnyTrustMessageHeaderFlag}, make([]byte, 31)...)},
		{"header with tree flag only", []byte{daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag}},
		{"header with tree flag plus 31 bytes", append([]byte{daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag}, make([]byte, 31)...)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := buildSequencerMsg(tc.payload)
			msg, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate, nil)
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
	registry := daprovider.NewDAProviderRegistry()

	// 33 bytes: 1 header + 32 keyset hash - this is long enough that it should
	// still error when no AnyTrust reader is configured
	headers := []struct {
		name   string
		header byte
	}{
		{"0x80", daprovider.AnyTrustMessageHeaderFlag},
		{"0x88", daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag},
	}
	for _, tc := range headers {
		t.Run(tc.name, func(t *testing.T) {
			payload := append([]byte{tc.header}, make([]byte, 32)...)
			data := buildSequencerMsg(payload)

			_, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate, nil)
			if err == nil {
				t.Fatal("expected error for AnyTrust message with no reader configured, got nil")
			}
		})
	}
}

func TestParseSequencerMessage_DACertWithFallbackReader(t *testing.T) {
	ctx := context.Background()
	registry := daprovider.NewDAProviderRegistry()
	if err := registry.SetupDACertificateReader(&daprovider.FallbackDACertReader{}, nil); err != nil {
		t.Fatalf("failed to register fallback DACert reader: %v", err)
	}

	payload := append([]byte{daprovider.DACertificateMessageHeaderFlag}, make([]byte, 64)...)
	data := buildSequencerMsg(payload)

	msg, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate, nil)
	if err != nil {
		t.Fatalf("expected no error with fallback DACert reader, got: %v", err)
	}
	if len(msg.Segments) != 0 {
		t.Fatalf("expected empty segments for rejected DACert batch, got %d segments", len(msg.Segments))
	}
}

func TestParseSequencerMessage_DACertNoReaderErrors(t *testing.T) {
	ctx := context.Background()
	registry := daprovider.NewDAProviderRegistry()

	payload := append([]byte{daprovider.DACertificateMessageHeaderFlag}, make([]byte, 64)...)
	data := buildSequencerMsg(payload)

	_, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, registry, daprovider.KeysetValidate, nil)
	if err == nil {
		t.Fatal("expected error for DACert message with no reader configured, got nil")
	}
}

func TestParseSequencerMessage_MinimalHeader(t *testing.T) {
	ctx := context.Background()
	// Ensure messages shorter than 40 bytes return an error
	_, err := ParseSequencerMessage(ctx, 0, common.Hash{}, make([]byte, 39), nil, daprovider.KeysetValidate, nil)
	if err == nil {
		t.Fatal("expected error for message shorter than 40 bytes")
	}
}

func TestParseSequencerMessage_EmptyPayload(t *testing.T) {
	ctx := context.Background()
	// Exactly 40 bytes = valid header with empty payload
	data := make([]byte, 40)
	msg, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, nil, daprovider.KeysetValidate, nil)
	if err != nil {
		t.Fatalf("expected no error for empty payload, got: %v", err)
	}
	if len(msg.Segments) != 0 {
		t.Fatalf("expected empty segments for empty payload, got %d segments", len(msg.Segments))
	}
}

func TestParseSequencerMessage_HeaderFieldParsing(t *testing.T) {
	ctx := context.Background()
	data := make([]byte, 40)
	binary.BigEndian.PutUint64(data[0:8], 100)  // MinTimestamp
	binary.BigEndian.PutUint64(data[8:16], 200) // MaxTimestamp
	binary.BigEndian.PutUint64(data[16:24], 10) // MinL1Block
	binary.BigEndian.PutUint64(data[24:32], 20) // MaxL1Block
	binary.BigEndian.PutUint64(data[32:40], 5)  // AfterDelayedMessages

	msg, err := ParseSequencerMessage(ctx, 0, common.Hash{}, data, nil, daprovider.KeysetValidate, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.MinTimestamp != 100 {
		t.Errorf("MinTimestamp = %d, want 100", msg.MinTimestamp)
	}
	if msg.MaxTimestamp != 200 {
		t.Errorf("MaxTimestamp = %d, want 200", msg.MaxTimestamp)
	}
	if msg.MinL1Block != 10 {
		t.Errorf("MinL1Block = %d, want 10", msg.MinL1Block)
	}
	if msg.MaxL1Block != 20 {
		t.Errorf("MaxL1Block = %d, want 20", msg.MaxL1Block)
	}
	if msg.AfterDelayedMessages != 5 {
		t.Errorf("AfterDelayedMessages = %d, want 5", msg.AfterDelayedMessages)
	}
}
