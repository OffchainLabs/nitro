// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestFilteredTxReportJSON_NotDelayed(t *testing.T) {
	report := FilteredTxReport{
		ID:     "019539b4-6b30-7e5a-8000-1a2b3c4d5e6f",
		TxHash: common.HexToHash("0xabc123"),
		TxRLP:  hexutil.Bytes{0xf8, 0x6c},
		FilteredAddresses: []filter.FilteredAddressRecord{
			{
				Address:     common.HexToAddress("0xdead"),
				FilterSetId: "filter-set-1",
				FilterReason: filter.FilterReason{
					Reason: filter.ReasonFrom,
				},
			},
		},
		BlockNumber:       1042,
		ParentBlockHash:   common.HexToHash("0x1234"),
		PositionInBlock:   3,
		FilteredAt:        time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC),
		IsDelayed:         false,
		DelayedReportData: nil,
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	// isDelayed should be false
	assert.JSONEq(t, `false`, string(raw["isDelayed"]))
	// delayedInboxRequestId should not be present
	_, hasDelayedField := raw["delayedInboxRequestId"]
	assert.False(t, hasDelayedField, "delayedInboxRequestId should be absent when not delayed")

	// Filtered address should have reason "from" and no event rule fields
	var addrRecords []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["filteredAddresses"], &addrRecords))
	require.Len(t, addrRecords, 1)
	assert.JSONEq(t, `"from"`, string(addrRecords[0]["reason"]))
	_, hasMatchedEvent := addrRecords[0]["matchedEvent"]
	assert.False(t, hasMatchedEvent, "matchedEvent should be absent for non-event_rule reason")
	_, hasMatchedTopicIndex := addrRecords[0]["matchedTopicIndex"]
	assert.False(t, hasMatchedTopicIndex, "matchedTopicIndex should be absent for non-event_rule reason")
	_, hasRawLog := addrRecords[0]["rawLog"]
	assert.False(t, hasRawLog, "rawLog should be absent for non-event_rule reason")

	// Round-trip
	var decoded FilteredTxReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, report, decoded)
}

func TestFilteredTxReportJSON_Delayed(t *testing.T) {
	requestId := common.HexToHash("0x01")
	report := FilteredTxReport{
		ID:     "019539b4-9a10-7f3b-8000-5f6e7d8c9b0a",
		TxHash: common.HexToHash("0xdef789"),
		TxRLP:  hexutil.Bytes{0xf8, 0x6c},
		FilteredAddresses: []filter.FilteredAddressRecord{
			{
				Address:     common.HexToAddress("0xdead"),
				FilterSetId: "filter-set-1",
				FilterReason: filter.FilterReason{
					Reason: filter.ReasonDealiasedFrom,
				},
			},
		},
		BlockNumber:     1043,
		ParentBlockHash: common.HexToHash("0xabcdef"),
		PositionInBlock: 0,
		FilteredAt:      time.Date(2026, 2, 27, 14, 31, 0, 0, time.UTC),
		IsDelayed:       true,
		DelayedReportData: &DelayedReportData{
			InboxRequestId: requestId,
		},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.JSONEq(t, `true`, string(raw["isDelayed"]))
	_, hasDelayedField := raw["delayedInboxRequestId"]
	assert.True(t, hasDelayedField, "delayedInboxRequestId should be present when delayed")

	// Round-trip
	var decoded FilteredTxReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, report, decoded)
}

func TestFilteredTxReportJSON_EventRule(t *testing.T) {
	report := FilteredTxReport{
		ID:     "019539b4-6b30-7e5a-8000-aabbccddeeff",
		TxHash: common.HexToHash("0xaaa111"),
		TxRLP:  hexutil.Bytes{0xf8},
		FilteredAddresses: []filter.FilteredAddressRecord{
			{
				Address:     common.HexToAddress("0xbeef"),
				FilterSetId: "filter-set-2",
				FilterReason: filter.FilterReason{
					Reason: filter.ReasonEventRule,
					EventRuleMatch: &filter.EventRuleMatch{
						MatchedEvent:      "Transfer(address,address,uint256)",
						MatchedTopicIndex: 2,
						RawLog: &filter.RawLog{
							Address: common.HexToAddress("0xdead"),
							Topics: []common.Hash{
								common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
								common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
								common.HexToHash("0x000000000000000000000000beefbeefbeefbeefbeefbeefbeefbeefbeefbeef"),
							},
							Data: hexutil.Bytes{0x00, 0x00},
						},
					},
				},
			},
		},
		BlockNumber:       1044,
		ParentBlockHash:   common.HexToHash("0x5678"),
		PositionInBlock:   1,
		FilteredAt:        time.Date(2026, 2, 27, 14, 32, 0, 0, time.UTC),
		IsDelayed:         false,
		DelayedReportData: nil,
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	var addrRecords []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["filteredAddresses"], &addrRecords))
	require.Len(t, addrRecords, 1)

	assert.JSONEq(t, `"event_rule"`, string(addrRecords[0]["reason"]))
	assert.JSONEq(t, `"Transfer(address,address,uint256)"`, string(addrRecords[0]["matchedEvent"]))
	assert.JSONEq(t, `2`, string(addrRecords[0]["matchedTopicIndex"]))
	_, hasRawLog := addrRecords[0]["rawLog"]
	assert.True(t, hasRawLog, "rawLog should be present for event_rule reason")

	// Round-trip
	var decoded FilteredTxReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, report, decoded)
}
