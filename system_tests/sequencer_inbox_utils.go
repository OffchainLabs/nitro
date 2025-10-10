// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/daprovider/referenceda"
)

// SequencerInboxHeader represents the decoded header of a sequencer inbox message
type SequencerInboxHeader struct {
	MinTimestamp             uint64
	MaxTimestamp             uint64
	MinBlockNumber           uint64
	MaxBlockNumber           uint64
	AfterDelayedMessagesRead uint64
}

// DecodeSequencerInboxHeader decodes the 40-byte header from a sequencer inbox message
func DecodeSequencerInboxHeader(message []byte) (*SequencerInboxHeader, error) {
	if len(message) < 40 {
		return nil, fmt.Errorf("message too short for header: %d bytes", len(message))
	}

	header := &SequencerInboxHeader{
		MinTimestamp:             binary.BigEndian.Uint64(message[0:8]),
		MaxTimestamp:             binary.BigEndian.Uint64(message[8:16]),
		MinBlockNumber:           binary.BigEndian.Uint64(message[16:24]),
		MaxBlockNumber:           binary.BigEndian.Uint64(message[24:32]),
		AfterDelayedMessagesRead: binary.BigEndian.Uint64(message[32:40]),
	}

	return header, nil
}

// PrintSequencerInboxMessage prints a formatted view of a sequencer inbox message
func PrintSequencerInboxMessage(t *testing.T, label string, message []byte) {
	t.Logf("\n=== %s ===", label)
	t.Logf("Total message length: %d bytes", len(message))

	if len(message) < 40 {
		t.Logf("Message too short to have a header")
		t.Logf("Raw message: %s", hex.EncodeToString(message))
		return
	}

	header, err := DecodeSequencerInboxHeader(message)
	if err != nil {
		t.Logf("Error decoding header: %v", err)
		return
	}

	// Print header details
	t.Logf("Header (40 bytes):")
	t.Logf("  MinTimestamp:             %d (%s)", header.MinTimestamp, time.Unix(int64(header.MinTimestamp), 0).UTC())
	t.Logf("  MaxTimestamp:             %d (%s)", header.MaxTimestamp, time.Unix(int64(header.MaxTimestamp), 0).UTC())
	t.Logf("  MinBlockNumber:           %d", header.MinBlockNumber)
	t.Logf("  MaxBlockNumber:           %d", header.MaxBlockNumber)
	t.Logf("  AfterDelayedMessagesRead: %d", header.AfterDelayedMessagesRead)
	t.Logf("  Header (hex):             %s", hex.EncodeToString(message[:40]))

	// Print data after header
	dataAfterHeader := message[40:]
	t.Logf("Data after header (%d bytes):", len(dataAfterHeader))
	if len(dataAfterHeader) > 0 {
		// Check if it's a CustomDA certificate
		if len(dataAfterHeader) > 0 && dataAfterHeader[0] == 0x01 {
			t.Logf("  Type: CustomDA certificate (0x01)")
			if len(dataAfterHeader) >= 33 {
				sha256Hash := common.BytesToHash(dataAfterHeader[1:33])
				t.Logf("  SHA256 hash: %s", sha256Hash.Hex())
			}
			// Try to deserialize and extract signer
			cert, err := referenceda.Deserialize(dataAfterHeader)
			if err != nil {
				t.Logf("  Certificate: Failed to deserialize (%v)", err)
			} else {
				signer, err := cert.RecoverSigner()
				if err != nil {
					t.Logf("  Signer: Failed to recover (%v)", err)
				} else {
					t.Logf("  Signer: %s", signer.Hex())
				}
			}
		} else {
			t.Logf("  Type: Other (first byte: 0x%02x)", dataAfterHeader[0])
		}

		// Show first 100 bytes of data or all if less
		displayLen := len(dataAfterHeader)
		if displayLen > 100 {
			displayLen = 100
		}
		t.Logf("  Data (first %d bytes): %s", displayLen, hex.EncodeToString(dataAfterHeader[:displayLen]))
		if len(dataAfterHeader) > 100 {
			t.Logf("  ... (%d more bytes)", len(dataAfterHeader)-100)
		}
	}

	// Calculate and print the full message hash
	// Note: The actual hash stored in sequencerInboxAccs is keccak256(header + data)
	messageHash := crypto.Keccak256Hash(message)
	t.Logf("Full message keccak256: %s", messageHash.Hex())
}

// CompareSequencerInboxMessages compares two sequencer inbox messages and highlights differences
func CompareSequencerInboxMessages(t *testing.T, msgA, msgB []byte) {
	t.Logf("\n=== Comparing Sequencer Inbox Messages ===")

	if len(msgA) != len(msgB) {
		t.Logf("❌ Message lengths differ: A=%d bytes, B=%d bytes", len(msgA), len(msgB))
	} else {
		t.Logf("✓ Message lengths match: %d bytes", len(msgA))
	}

	headerA, errA := DecodeSequencerInboxHeader(msgA)
	headerB, errB := DecodeSequencerInboxHeader(msgB)

	if errA != nil || errB != nil {
		t.Logf("Cannot compare headers due to errors")
		return
	}

	// Compare headers field by field
	t.Logf("\nHeader comparison:")
	compareField := func(name string, a, b uint64) {
		if a == b {
			t.Logf("  ✓ %s: %d (match)", name, a)
		} else {
			t.Logf("  ❌ %s: A=%d, B=%d (differ by %d)", name, a, b, int64(a)-int64(b))
		}
	}

	compareField("MinTimestamp", headerA.MinTimestamp, headerB.MinTimestamp)
	compareField("MaxTimestamp", headerA.MaxTimestamp, headerB.MaxTimestamp)
	compareField("MinBlockNumber", headerA.MinBlockNumber, headerB.MinBlockNumber)
	compareField("MaxBlockNumber", headerA.MaxBlockNumber, headerB.MaxBlockNumber)
	compareField("AfterDelayedMessagesRead", headerA.AfterDelayedMessagesRead, headerB.AfterDelayedMessagesRead)

	// Compare data after header
	if len(msgA) >= 40 && len(msgB) >= 40 {
		dataA := msgA[40:]
		dataB := msgB[40:]

		if len(dataA) == len(dataB) && hex.EncodeToString(dataA) == hex.EncodeToString(dataB) {
			t.Logf("\n✓ Data after header matches (%d bytes)", len(dataA))
		} else {
			t.Logf("\n❌ Data after header differs")
			t.Logf("  A: %s", hex.EncodeToString(dataA))
			t.Logf("  B: %s", hex.EncodeToString(dataB))
		}
	}
}
