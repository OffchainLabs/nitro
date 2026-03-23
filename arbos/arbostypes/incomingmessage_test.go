// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbostypes

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestFillInBatchGasFieldsNilFetcher(t *testing.T) {
	// Must not panic when batchFetcher is nil, even for BatchPostingReport messages.
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: make([]byte, 148),
	}
	if err := msg.FillInBatchGasFields(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.BatchDataStats != nil {
		t.Error("expected BatchDataStats to remain nil with nil fetcher")
	}
	if msg.LegacyBatchGasCost != nil {
		t.Error("expected LegacyBatchGasCost to remain nil with nil fetcher")
	}
}

// buildBatchPostingReportMsg constructs a serialized BatchPostingReport message
// whose L2msg references batchData with the given batchNum.
func buildBatchPostingReportMsg(t *testing.T, batchData []byte, batchNum uint64) []byte {
	t.Helper()
	dataHash := crypto.Keccak256Hash(batchData)
	// L2msg: 32 (timestamp) + 20 (address) + 32 (dataHash) + 32 (batchNum) + 32 (baseFee) = 148 bytes
	var l2msg []byte
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1000)).Bytes()...)  // timestamp
	l2msg = append(l2msg, common.Address{}.Bytes()...)                    // poster address
	l2msg = append(l2msg, dataHash.Bytes()...)                            // data hash
	l2msg = append(l2msg, common.BigToHash(big.NewInt(0).SetUint64(batchNum)).Bytes()...) // batch number
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1)).Bytes()...)     // base fee

	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			Poster:      common.Address{},
			BlockNumber: 1,
			Timestamp:   1000,
			RequestId:   &common.Hash{},
			L1BaseFee:   big.NewInt(1),
		},
		L2msg: l2msg,
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return serialized
}

func TestParseIncomingL1MessageNilFetcherBatchPostingReport(t *testing.T) {
	// ParseIncomingL1Message must not panic when batchFetcher is nil and the
	// message kind is BatchPostingReport. This is the end-to-end path that
	// triggered the original nil pointer dereference.
	serialized := buildBatchPostingReportMsg(t, []byte("batch data"), 1)
	msg, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Header.Kind != L1MessageType_BatchPostingReport {
		t.Fatalf("expected BatchPostingReport kind, got %d", msg.Header.Kind)
	}
	if msg.BatchDataStats != nil {
		t.Error("expected BatchDataStats to remain nil with nil fetcher")
	}
	if msg.LegacyBatchGasCost != nil {
		t.Error("expected LegacyBatchGasCost to remain nil with nil fetcher")
	}
}

func TestTwoStepParseAndFillGasFields(t *testing.T) {
	// Exercises the inbox_tracker.go pattern: parse with nil fetcher first,
	// then fill gas fields with a real fetcher in a separate call.
	batchData := []byte("test batch data")
	var batchNum uint64 = 7
	serialized := buildBatchPostingReportMsg(t, batchData, batchNum)

	// Step 1: Parse with nil fetcher (should not panic or error).
	msg, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Fatalf("parse with nil fetcher: %v", err)
	}
	if msg.BatchDataStats != nil || msg.LegacyBatchGasCost != nil {
		t.Fatal("gas fields should be nil after parsing with nil fetcher")
	}

	// Step 2: Fill gas fields with a real fetcher.
	fetcher := func(num uint64) ([]byte, error) {
		if num != batchNum {
			t.Fatalf("fetcher called with unexpected batch number %d, want %d", num, batchNum)
		}
		return batchData, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("FillInBatchGasFields: %v", err)
	}
	if msg.BatchDataStats == nil {
		t.Fatal("expected BatchDataStats to be populated after FillInBatchGasFields")
	}
	if msg.LegacyBatchGasCost == nil {
		t.Fatal("expected LegacyBatchGasCost to be populated after FillInBatchGasFields")
	}
}
