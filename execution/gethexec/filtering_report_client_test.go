// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

func makeLoggedTestBytes(n int) hexutil.Bytes {
	b := make(hexutil.Bytes, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func makeFilteredAddressRecord(eventRuleMatch *filter.EventRuleMatch) filter.FilteredAddressRecord {
	return filter.FilteredAddressRecord{
		FilterSetID: "filter-set",
		FilteredAddressWithReason: filter.FilteredAddressWithReason{
			Address: common.HexToAddress("0x0102"),
			FilterReason: filter.FilterReason{
				Reason:         filter.ReasonEventRule,
				EventRuleMatch: eventRuleMatch,
			},
		},
	}
}

func TestReportForLog(t *testing.T) {
	longData := makeLoggedTestBytes(maxLoggedReportBytesLen + 100)
	shortData := makeLoggedTestBytes(10)
	report := addressfilter.FilteredTxReport{
		ID:     "report-id",
		TxHash: common.HexToHash("0x0304"),
		TxRLP:  makeLoggedTestBytes(maxLoggedReportBytesLen + 50),
		FilteredAddresses: []filter.FilteredAddressRecord{
			makeFilteredAddressRecord(nil),
			makeFilteredAddressRecord(&filter.EventRuleMatch{
				MatchedEvent:      "NoRawLog",
				MatchedTopicIndex: 0,
				RawLog:            nil,
			}),
			makeFilteredAddressRecord(&filter.EventRuleMatch{
				MatchedEvent:      "LongData",
				MatchedTopicIndex: 1,
				RawLog: &filter.RawLog{
					Address: common.HexToAddress("0x0506"),
					Topics:  []common.Hash{common.HexToHash("0x0708")},
					Data:    longData,
				},
			}),
			makeFilteredAddressRecord(&filter.EventRuleMatch{
				MatchedEvent:      "ShortData",
				MatchedTopicIndex: 2,
				RawLog: &filter.RawLog{
					Address: common.HexToAddress("0x090a"),
					Topics:  []common.Hash{common.HexToHash("0x0b0c")},
					Data:    shortData,
				},
			}),
		},
		ChainID:           42,
		BlockNumber:       7,
		ParentBlockHash:   common.HexToHash("0x0d0e"),
		PositionInBlock:   3,
		FilteredAt:        time.Unix(1700000000, 0),
		IsDelayed:         false,
		DelayedReportData: nil,
	}

	logged := reportForLog(&report)

	if len(logged.TxRLP) != maxLoggedReportBytesLen {
		t.Errorf("expected logged TxRLP length %d, got %d", maxLoggedReportBytesLen, len(logged.TxRLP))
	}
	if !bytes.Equal(logged.TxRLP, report.TxRLP[:maxLoggedReportBytesLen]) {
		t.Error("logged TxRLP is not a prefix of the original TxRLP")
	}
	if len(logged.FilteredAddresses) != len(report.FilteredAddresses) {
		t.Fatalf("expected %d logged filtered addresses, got %d", len(report.FilteredAddresses), len(logged.FilteredAddresses))
	}

	loggedLongData := logged.FilteredAddresses[2].EventRuleMatch.RawLog.Data
	if len(loggedLongData) != maxLoggedReportBytesLen {
		t.Errorf("expected logged RawLog.Data length %d, got %d", maxLoggedReportBytesLen, len(loggedLongData))
	}
	if !bytes.Equal(loggedLongData, longData[:maxLoggedReportBytesLen]) {
		t.Error("logged RawLog.Data is not a prefix of the original data")
	}
	if !bytes.Equal(logged.FilteredAddresses[3].EventRuleMatch.RawLog.Data, shortData) {
		t.Error("short RawLog.Data should be unchanged")
	}
	if logged.FilteredAddresses[0].EventRuleMatch != nil {
		t.Error("record without EventRuleMatch should stay nil")
	}
	if logged.FilteredAddresses[1].EventRuleMatch.RawLog != nil {
		t.Error("record without RawLog should keep a nil RawLog")
	}

	// The original report is sent over RPC after logging, so it must not be
	// mutated: truncation has to happen on copies.
	if len(report.TxRLP) != maxLoggedReportBytesLen+50 {
		t.Errorf("original TxRLP was mutated, length %d", len(report.TxRLP))
	}
	if len(report.FilteredAddresses[2].EventRuleMatch.RawLog.Data) != maxLoggedReportBytesLen+100 {
		t.Errorf("original RawLog.Data was mutated, length %d", len(report.FilteredAddresses[2].EventRuleMatch.RawLog.Data))
	}
	if reflect.ValueOf(logged.FilteredAddresses[2].EventRuleMatch).Pointer() == reflect.ValueOf(report.FilteredAddresses[2].EventRuleMatch).Pointer() {
		t.Error("truncated record should not share EventRuleMatch with the original")
	}
	if reflect.ValueOf(logged.FilteredAddresses[2].EventRuleMatch.RawLog).Pointer() == reflect.ValueOf(report.FilteredAddresses[2].EventRuleMatch.RawLog).Pointer() {
		t.Error("truncated record should not share RawLog with the original")
	}

	loggedJSON, err := json.Marshal(logged)
	if err != nil {
		t.Fatalf("failed to marshal logged report: %v", err)
	}
	if !strings.Contains(string(loggedJSON), `"matchedEvent":"LongData"`) {
		t.Error("logged report JSON should contain nested EventRuleMatch contents")
	}
}

func TestReportForLogAtLimit(t *testing.T) {
	report := addressfilter.FilteredTxReport{
		ID:                "report-id",
		TxHash:            common.HexToHash("0x0304"),
		TxRLP:             makeLoggedTestBytes(maxLoggedReportBytesLen),
		FilteredAddresses: nil,
		ChainID:           42,
		BlockNumber:       7,
		ParentBlockHash:   common.HexToHash("0x0d0e"),
		PositionInBlock:   3,
		FilteredAt:        time.Unix(1700000000, 0),
		IsDelayed:         false,
		DelayedReportData: nil,
	}
	logged := reportForLog(&report)
	if !bytes.Equal(logged.TxRLP, report.TxRLP) {
		t.Error("TxRLP at exactly the limit should not be truncated")
	}
}
