// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
	"time"
)

func TestEncodeFrameHeaderAndCRC(t *testing.T) {
	payload := []byte("robinhood")
	data, err := encodeFrame(frame{kind: FrameTransaction, flags: 3, sequence: 42, payload: payload}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != HeaderSize+len(payload) || string(data[:4]) != Magic {
		t.Fatalf("unexpected frame size or magic: %d", len(data))
	}
	if got := binary.BigEndian.Uint16(data[4:6]); got != ProtocolVersion {
		t.Fatalf("version = %d", got)
	}
	if got := binary.BigEndian.Uint64(data[8:16]); got != 42 {
		t.Fatalf("sequence = %d", got)
	}
	if got := binary.BigEndian.Uint32(data[20:24]); got != crc32.Checksum(payload, crcTable) {
		t.Fatalf("crc = %#x", got)
	}
}

func TestEncodeFrameBounds(t *testing.T) {
	if _, err := encodeFrame(frame{payload: []byte{1}}, HeaderSize); err == nil {
		t.Fatal("expected max-frame bounds error")
	}
	if _, err := encodeFrame(frame{payload: []byte{1}}, HeaderSize-1); err == nil {
		t.Fatal("expected invalid max-frame error")
	}
}

func TestDecodeFrameRejectsCorruption(t *testing.T) {
	data, err := EncodeFrame(Frame{Kind: FrameHeartbeat, Sequence: 7, Payload: []byte("x")}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 1
	if _, err := DecodeFrame(data, 1024); err == nil {
		t.Fatal("expected CRC error")
	}
}

func TestConfigValidation(t *testing.T) {
	c := DefaultConfig
	c.Enable = true
	c.SocketPath = t.TempDir() + "/feed.sock"
	c.QueueSize = 16
	c.MaxFrameBytes = 1024
	c.WriteTimeout = time.Millisecond
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.SocketPath = "relative.sock"
	if err := c.Validate(); err == nil {
		t.Fatal("expected absolute path validation error")
	}
}
