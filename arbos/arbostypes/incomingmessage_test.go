// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbostypes

import (
	"bytes"
	"errors"
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

// buildBatchPostingReportL2msg constructs an L2msg payload for a
// BatchPostingReport that references batchData with the given batchNum.
func buildBatchPostingReportL2msg(t *testing.T, batchData []byte, batchNum uint64) []byte {
	t.Helper()
	dataHash := crypto.Keccak256Hash(batchData)
	// L2msg: 32 (timestamp) + 20 (address) + 32 (dataHash) + 32 (batchNum) + 32 (baseFee) = 148 bytes
	var l2msg []byte
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1000)).Bytes()...)                  // timestamp
	l2msg = append(l2msg, common.Address{}.Bytes()...)                                    // poster address
	l2msg = append(l2msg, dataHash.Bytes()...)                                            // data hash
	l2msg = append(l2msg, common.BigToHash(big.NewInt(0).SetUint64(batchNum)).Bytes()...) // batch number
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1)).Bytes()...)                     // base fee
	return l2msg
}

// buildBatchPostingReportMsg constructs a serialized BatchPostingReport message
// whose L2msg references batchData with the given batchNum.
func buildBatchPostingReportMsg(t *testing.T, batchData []byte, batchNum uint64) []byte {
	t.Helper()
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			Poster:      common.Address{},
			BlockNumber: 1,
			Timestamp:   1000,
			RequestId:   &common.Hash{},
			L1BaseFee:   big.NewInt(1),
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
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
	// Verify computed values match what GetDataStats and LegacyCostForStats
	// would produce for the batch data.
	expectedStats := GetDataStats(batchData)
	if msg.BatchDataStats.Length != expectedStats.Length {
		t.Fatalf("BatchDataStats.Length = %d, want %d", msg.BatchDataStats.Length, expectedStats.Length)
	}
	if msg.BatchDataStats.NonZeros != expectedStats.NonZeros {
		t.Fatalf("BatchDataStats.NonZeros = %d, want %d", msg.BatchDataStats.NonZeros, expectedStats.NonZeros)
	}
	expectedCost := LegacyCostForStats(expectedStats)
	if *msg.LegacyBatchGasCost != expectedCost {
		t.Fatalf("LegacyBatchGasCost = %d, want %d", *msg.LegacyBatchGasCost, expectedCost)
	}
}

func TestFillInBatchGasFieldsSkipsWhenAlreadyPopulated(t *testing.T) {
	// When both BatchDataStats and LegacyBatchGasCost are already set,
	// the fetcher must not be called.
	legacyCost := uint64(42)
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg:              make([]byte, 148),
		BatchDataStats:     &BatchDataStats{Length: 10, NonZeros: 5},
		LegacyBatchGasCost: &legacyCost,
	}
	fetcher := func(batchNum uint64) ([]byte, error) {
		t.Fatal("fetcher should not be called when fields are already populated")
		return nil, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fields must remain unchanged.
	if msg.BatchDataStats.Length != 10 || msg.BatchDataStats.NonZeros != 5 {
		t.Fatal("BatchDataStats was modified")
	}
	if *msg.LegacyBatchGasCost != 42 {
		t.Fatalf("LegacyBatchGasCost was modified: got %d", *msg.LegacyBatchGasCost)
	}
}

func TestFillInBatchGasFieldsHashMismatch(t *testing.T) {
	// If the fetcher returns data whose hash doesn't match the batch
	// posting report, an error must be returned.
	batchData := []byte("correct batch data")
	var batchNum uint64 = 3
	serialized := buildBatchPostingReportMsg(t, batchData, batchNum)

	msg, err := ParseIncomingL1Message(bytes.NewReader(serialized), func(num uint64) ([]byte, error) {
		return []byte("wrong batch data"), nil
	})
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if msg != nil {
		t.Fatal("expected nil message on error")
	}
}

func TestFillInBatchGasFieldsWithParentBlockPopulatesFields(t *testing.T) {
	// FillInBatchGasFieldsWithParentBlock with a real fetcher must
	// populate both BatchDataStats and LegacyBatchGasCost.
	batchData := []byte("test batch data for parent block")
	var batchNum uint64 = 5

	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			BlockNumber: 100,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	var parentBlockSeen uint64
	fetcher := func(num uint64, parentBlock uint64) ([]byte, error) {
		if num != batchNum {
			t.Fatalf("fetcher called with unexpected batch number %d, want %d", num, batchNum)
		}
		parentBlockSeen = parentBlock
		return batchData, nil
	}
	if err := msg.FillInBatchGasFieldsWithParentBlock(fetcher, 99); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parentBlockSeen != 99 {
		t.Fatalf("fetcher received parentChainBlockNumber %d, want 99", parentBlockSeen)
	}
	if msg.BatchDataStats == nil {
		t.Fatal("expected BatchDataStats to be populated")
	}
	if msg.LegacyBatchGasCost == nil {
		t.Fatal("expected LegacyBatchGasCost to be populated")
	}
	expectedStats := GetDataStats(batchData)
	if msg.BatchDataStats.Length != expectedStats.Length || msg.BatchDataStats.NonZeros != expectedStats.NonZeros {
		t.Fatalf("BatchDataStats mismatch: got %+v, want %+v", msg.BatchDataStats, expectedStats)
	}
}

func TestFillInBatchGasFieldsFetcherErrorWithLegacyCost(t *testing.T) {
	// When LegacyBatchGasCost is already set but BatchDataStats is nil, and
	// the fetcher returns an error, the function should succeed (pre-arbos50
	// fallback) and leave BatchDataStats nil.
	batchData := []byte("some batch")
	var batchNum uint64 = 2
	legacyCost := uint64(999)
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg:              buildBatchPostingReportL2msg(t, batchData, batchNum),
		LegacyBatchGasCost: &legacyCost,
	}
	fetcher := func(num uint64) ([]byte, error) {
		return nil, errors.New("batch not available")
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.BatchDataStats != nil {
		t.Fatal("expected BatchDataStats to remain nil on fetcher error with LegacyBatchGasCost set")
	}
	if *msg.LegacyBatchGasCost != legacyCost {
		t.Fatalf("LegacyBatchGasCost changed: got %d, want %d", *msg.LegacyBatchGasCost, legacyCost)
	}
}

func TestFillInBatchGasFieldsFetcherErrorWithoutLegacyCost(t *testing.T) {
	// When neither LegacyBatchGasCost nor BatchDataStats is set and the
	// fetcher returns an error, the function must propagate that error.
	batchData := []byte("some batch")
	var batchNum uint64 = 2
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	fetchErr := errors.New("batch not available")
	fetcher := func(num uint64) ([]byte, error) {
		return nil, fetchErr
	}
	err := msg.FillInBatchGasFields(fetcher)
	if err == nil {
		t.Fatal("expected error when fetcher fails and LegacyBatchGasCost is nil")
	}
	if !errors.Is(err, fetchErr) {
		t.Fatalf("expected wrapped fetch error, got: %v", err)
	}
}

func TestFillInBatchGasFieldsOnlyBatchDataStatsSet(t *testing.T) {
	// When BatchDataStats is already set but LegacyBatchGasCost is nil,
	// the fetcher must not be called and LegacyBatchGasCost must be
	// recomputed from the existing stats.
	stats := &BatchDataStats{Length: 100, NonZeros: 40}
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg:          make([]byte, 148),
		BatchDataStats: stats,
	}
	fetcher := func(batchNum uint64) ([]byte, error) {
		t.Fatal("fetcher should not be called when BatchDataStats is already set")
		return nil, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.LegacyBatchGasCost == nil {
		t.Fatal("expected LegacyBatchGasCost to be computed")
	}
	expectedCost := LegacyCostForStats(stats)
	if *msg.LegacyBatchGasCost != expectedCost {
		t.Fatalf("LegacyBatchGasCost = %d, want %d", *msg.LegacyBatchGasCost, expectedCost)
	}
	if msg.BatchDataStats != stats {
		t.Fatal("BatchDataStats pointer changed unexpectedly")
	}
}

func TestFillInBatchGasFieldsTruncatedL2msg(t *testing.T) {
	// A BatchPostingReport with a truncated L2msg (too short to parse)
	// must return a parse error, not panic.
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: []byte("too short"),
	}
	fetcher := func(batchNum uint64) ([]byte, error) {
		t.Fatal("fetcher should not be called when L2msg is unparseable")
		return nil, nil
	}
	err := msg.FillInBatchGasFields(fetcher)
	if err == nil {
		t.Fatal("expected error for truncated L2msg")
	}
}
