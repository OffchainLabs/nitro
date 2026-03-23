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

func TestFillInBatchGasFieldsNilFetcherBatchPostingReport(t *testing.T) {
	// FillInBatchGasFields must return an error when batchFetcher is nil and
	// the message kind is BatchPostingReport, to avoid silently producing
	// messages with missing gas fields.
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: make([]byte, 148),
	}
	if err := msg.FillInBatchGasFields(nil); err == nil {
		t.Fatal("expected error when batchFetcher is nil for BatchPostingReport")
	}
}

func TestFillInBatchGasFieldsNilFetcherNonBatchPostingReport(t *testing.T) {
	// FillInBatchGasFields must not error when batchFetcher is nil and the
	// message kind is not BatchPostingReport (no fields need to be filled).
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_L2Message,
		},
		L2msg: make([]byte, 32),
	}
	if err := msg.FillInBatchGasFields(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFillInBatchGasFieldsWithParentBlockNilFetcherBatchPostingReport(t *testing.T) {
	// FillInBatchGasFieldsWithParentBlock must return an error when
	// batchFetcher is nil and the message kind is BatchPostingReport.
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: make([]byte, 148),
	}
	if err := msg.FillInBatchGasFieldsWithParentBlock(nil, 0); err == nil {
		t.Fatal("expected error when batchFetcher is nil for BatchPostingReport")
	}
}

func TestFillInBatchGasFieldsWithParentBlockNilFetcherNonBatchPostingReport(t *testing.T) {
	// FillInBatchGasFieldsWithParentBlock must not error when batchFetcher
	// is nil and the message kind is not BatchPostingReport.
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_L2Message,
		},
		L2msg: make([]byte, 32),
	}
	if err := msg.FillInBatchGasFieldsWithParentBlock(nil, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseIncomingL1MessageNilFetcherNonBatchPostingReport(t *testing.T) {
	// ParseIncomingL1Message with nil fetcher should succeed for non-
	// BatchPostingReport messages since no gas fields need to be filled.
	requestId := common.Hash{}
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_L2Message,
			Poster:      common.Address{},
			BlockNumber: 1,
			Timestamp:   1000,
			RequestId:   &requestId,
			L1BaseFee:   big.NewInt(1),
		},
		L2msg: []byte{0x01},
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Header.Kind != L1MessageType_L2Message {
		t.Fatalf("expected L2Message kind, got %d", parsed.Header.Kind)
	}
}

// buildBatchPostingReportMsg constructs a serialized BatchPostingReport message
// whose L2msg references batchData with the given batchNum.
func buildBatchPostingReportMsg(t *testing.T, batchData []byte, batchNum uint64) []byte {
	t.Helper()
	dataHash := crypto.Keccak256Hash(batchData)
	// L2msg: 32 (timestamp) + 20 (address) + 32 (dataHash) + 32 (batchNum) + 32 (baseFee) = 148 bytes
	var l2msg []byte
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1000)).Bytes()...)                  // timestamp
	l2msg = append(l2msg, common.Address{}.Bytes()...)                                    // poster address
	l2msg = append(l2msg, dataHash.Bytes()...)                                            // data hash
	l2msg = append(l2msg, common.BigToHash(big.NewInt(0).SetUint64(batchNum)).Bytes()...) // batch number
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1)).Bytes()...)                     // base fee

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
	// ParseIncomingL1Message must return an error (not panic) when batchFetcher
	// is nil and the message kind is BatchPostingReport, because the batch gas
	// fields cannot be filled without a fetcher.
	serialized := buildBatchPostingReportMsg(t, []byte("batch data"), 1)
	_, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err == nil {
		t.Fatal("expected error when batchFetcher is nil for BatchPostingReport")
	}
}

func TestParseAndFillGasFieldsWithFetcher(t *testing.T) {
	// Exercises parsing a BatchPostingReport with a real fetcher that
	// fills in the batch gas fields during parsing.
	batchData := []byte("test batch data")
	var batchNum uint64 = 7
	serialized := buildBatchPostingReportMsg(t, batchData, batchNum)

	fetcher := func(num uint64) ([]byte, error) {
		if num != batchNum {
			t.Fatalf("fetcher called with unexpected batch number %d, want %d", num, batchNum)
		}
		return batchData, nil
	}
	msg, err := ParseIncomingL1Message(bytes.NewReader(serialized), fetcher)
	if err != nil {
		t.Fatalf("ParseIncomingL1Message: %v", err)
	}
	if msg.BatchDataStats == nil {
		t.Fatal("expected BatchDataStats to be populated")
	}
	if msg.LegacyBatchGasCost == nil {
		t.Fatal("expected LegacyBatchGasCost to be populated")
	}
}
