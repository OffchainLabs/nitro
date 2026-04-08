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

func TestNilFetcherBehavior(t *testing.T) {
	tests := []struct {
		name           string
		kind           uint8
		useParentBlock bool
		wantErr        error
	}{
		{"FillInBatchGasFields/BatchPostingReport", L1MessageType_BatchPostingReport, false, ErrNilBatchFetcher},
		{"FillInBatchGasFields/L2Message", L1MessageType_L2Message, false, nil},
		{"WithParentBlock/BatchPostingReport", L1MessageType_BatchPostingReport, true, ErrNilBatchFetcher},
		{"WithParentBlock/L2Message", L1MessageType_L2Message, true, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l2msgLen := 32
			if tc.kind == L1MessageType_BatchPostingReport {
				l2msgLen = 148
			}
			msg := &L1IncomingMessage{
				Header: &L1IncomingMessageHeader{Kind: tc.kind},
				L2msg:  make([]byte, l2msgLen),
			}
			var err error
			if tc.useParentBlock {
				err = msg.FillInBatchGasFieldsWithParentBlock(nil, 42)
			} else {
				err = msg.FillInBatchGasFields(nil)
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got: %v", tc.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func assertBatchGasFieldsPopulated(t *testing.T, msg *L1IncomingMessage, batchData []byte) {
	t.Helper()
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
	expectedCost := LegacyCostForStats(expectedStats)
	if *msg.LegacyBatchGasCost != expectedCost {
		t.Fatalf("LegacyBatchGasCost = %d, want %d", *msg.LegacyBatchGasCost, expectedCost)
	}
}

// buildBatchPostingReportL2msg constructs an L2msg payload for a
// BatchPostingReport that embeds the Keccak256 hash of batchData along with
// the given batchNum, matching the format expected by
// ParseBatchPostingReportMessageFields.
func buildBatchPostingReportL2msg(t *testing.T, batchData []byte, batchNum uint64) []byte {
	t.Helper()
	dataHash := crypto.Keccak256Hash(batchData)
	// L2msg: 32 (timestamp) + 20 (address) + 32 (dataHash) + 32 (batchNum) + 32 (baseFee) = 148 bytes
	// (excludes optional 8-byte extraGas field, which defaults to 0 on EOF)
	var l2msg []byte
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1000)).Bytes()...)                  // timestamp
	l2msg = append(l2msg, common.Address{}.Bytes()...)                                    // poster address
	l2msg = append(l2msg, dataHash.Bytes()...)                                            // data hash
	l2msg = append(l2msg, common.BigToHash(big.NewInt(0).SetUint64(batchNum)).Bytes()...) // batch number
	l2msg = append(l2msg, common.BigToHash(big.NewInt(1)).Bytes()...)                     // base fee
	return l2msg
}

// buildBatchPostingReportMsg constructs a serialized BatchPostingReport message
// whose L2msg embeds the Keccak256 hash of batchData along with the given batchNum.
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
	if !errors.Is(err, ErrNilBatchFetcher) {
		t.Fatalf("expected ErrNilBatchFetcher, got: %v", err)
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
	assertBatchGasFieldsPopulated(t, msg, batchData)
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

func TestParseIncomingL1MessageHashMismatch(t *testing.T) {
	// ParseIncomingL1Message must return ErrBatchHashMismatch when the
	// fetcher returns data whose Keccak256 hash doesn't match the hash
	// embedded in the BatchPostingReport's L2msg.
	batchData := []byte("correct batch data")
	var batchNum uint64 = 3
	serialized := buildBatchPostingReportMsg(t, batchData, batchNum)

	msg, err := ParseIncomingL1Message(bytes.NewReader(serialized), func(num uint64) ([]byte, error) {
		return []byte("wrong batch data"), nil
	})
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !errors.Is(err, ErrBatchHashMismatch) {
		t.Fatalf("expected ErrBatchHashMismatch, got: %v", err)
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
	assertBatchGasFieldsPopulated(t, msg, batchData)
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

func TestFillInBatchGasFieldsFetcherSuccessWithLegacyCost(t *testing.T) {
	// When LegacyBatchGasCost is already set but BatchDataStats is nil, and
	// the fetcher succeeds, both fields should be populated from the fetched
	// data (overwriting the pre-existing LegacyBatchGasCost).
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
		return batchData, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBatchGasFieldsPopulated(t, msg, batchData)
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
	// computed from the existing stats.
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
}

func TestFillInBatchGasFieldsPopulatesFields(t *testing.T) {
	// FillInBatchGasFields with a working fetcher must populate both
	// BatchDataStats and LegacyBatchGasCost.
	batchData := []byte("test batch data direct")
	var batchNum uint64 = 3
	var blockNumber uint64 = 42

	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			BlockNumber: blockNumber,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	fetcher := func(num uint64) ([]byte, error) {
		if num != batchNum {
			t.Fatalf("fetcher called with unexpected batch number %d, want %d", num, batchNum)
		}
		return batchData, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBatchGasFieldsPopulated(t, msg, batchData)
}

func TestFillInBatchGasFieldsWithParentBlockPassesBlockNumber(t *testing.T) {
	// FillInBatchGasFieldsWithParentBlock must forward the
	// parentChainBlockNumber argument to the fetcher.
	batchData := []byte("block number test data")
	var batchNum uint64 = 1
	var blockNumber uint64 = 777

	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			BlockNumber: blockNumber,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	var parentBlockSeen uint64
	wrappedFetcher := func(num uint64, parentBlock uint64) ([]byte, error) {
		parentBlockSeen = parentBlock
		return batchData, nil
	}
	if err := msg.FillInBatchGasFieldsWithParentBlock(wrappedFetcher, blockNumber); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parentBlockSeen != blockNumber {
		t.Fatalf("fetcher received parentChainBlockNumber %d, want %d", parentBlockSeen, blockNumber)
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
	if !errors.Is(err, ErrParseBatchPostingReport) {
		t.Fatalf("expected ErrParseBatchPostingReport, got: %v", err)
	}
}

func TestFillInBatchGasFieldsFetcherReturnsNilData(t *testing.T) {
	// When the fetcher returns (nil, nil) — no error but nil data — the
	// Keccak256 hash of an empty input won't match the expected hash, so
	// the function must return ErrBatchHashMismatch.
	batchData := []byte("real batch data")
	var batchNum uint64 = 4
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind: L1MessageType_BatchPostingReport,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	fetcher := func(num uint64) ([]byte, error) {
		return nil, nil
	}
	err := msg.FillInBatchGasFields(fetcher)
	if err == nil {
		t.Fatal("expected error when fetcher returns nil data")
	}
	if !errors.Is(err, ErrBatchHashMismatch) {
		t.Fatalf("expected ErrBatchHashMismatch, got: %v", err)
	}
}

func TestFillInBatchGasFieldsEmptyBatchData(t *testing.T) {
	// When the fetcher returns empty (non-nil) batch data whose hash
	// matches the L2msg, both fields should be populated correctly.
	batchData := []byte{}
	var batchNum uint64 = 8
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			BlockNumber: 50,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	fetcher := func(num uint64) ([]byte, error) {
		if num != batchNum {
			t.Fatalf("fetcher called with unexpected batch number %d, want %d", num, batchNum)
		}
		return batchData, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertBatchGasFieldsPopulated(t, msg, batchData)
}

func TestFillInBatchGasFieldsIdempotent(t *testing.T) {
	// Calling FillInBatchGasFields twice must be idempotent — the second
	// call should be a no-op since both fields are already populated.
	batchData := []byte("idempotent batch data")
	var batchNum uint64 = 6
	msg := &L1IncomingMessage{
		Header: &L1IncomingMessageHeader{
			Kind:        L1MessageType_BatchPostingReport,
			BlockNumber: 20,
		},
		L2msg: buildBatchPostingReportL2msg(t, batchData, batchNum),
	}
	callCount := 0
	fetcher := func(num uint64) ([]byte, error) {
		callCount++
		return batchData, nil
	}
	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected fetcher to be called once, got %d", callCount)
	}
	firstStats := *msg.BatchDataStats
	firstCost := *msg.LegacyBatchGasCost

	if err := msg.FillInBatchGasFields(fetcher); err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected fetcher not to be called again, got %d calls", callCount)
	}
	if msg.BatchDataStats.Length != firstStats.Length || msg.BatchDataStats.NonZeros != firstStats.NonZeros {
		t.Fatal("BatchDataStats changed on second call")
	}
	if *msg.LegacyBatchGasCost != firstCost {
		t.Fatalf("LegacyBatchGasCost changed on second call: got %d, want %d", *msg.LegacyBatchGasCost, firstCost)
	}
}

func TestFillInBatchGasFieldsAllMessageKindsWithNilFetcher(t *testing.T) {
	// All non-BatchPostingReport message kinds must succeed with a nil
	// fetcher, since there are no batch gas fields to fill.
	kinds := []uint8{
		L1MessageType_L2Message,
		L1MessageType_EndOfBlock,
		L1MessageType_L2FundedByL1,
		L1MessageType_RollupEvent,
		L1MessageType_SubmitRetryable,
		L1MessageType_BatchForGasEstimation,
		L1MessageType_Initialize,
		L1MessageType_EthDeposit,
		L1MessageType_Invalid,
	}
	for _, kind := range kinds {
		msg := &L1IncomingMessage{
			Header: &L1IncomingMessageHeader{
				Kind: kind,
			},
			L2msg: make([]byte, 32),
		}
		if err := msg.FillInBatchGasFields(nil); err != nil {
			t.Fatalf("kind %d: unexpected error with nil fetcher: %v", kind, err)
		}
	}
}
