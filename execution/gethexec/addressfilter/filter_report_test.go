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
				Address: common.HexToAddress("0xdead"),
				FilterReason: filter.FilterReason{
					Reason:         filter.ReasonFrom,
					EventRuleMatch: nil,
				},
			},
		},
		ChainID:           42161,
		BlockNumber:       1042,
		ParentBlockHash:   common.HexToHash("0x1234"),
		PositionInBlock:   3,
		FilteredAt:        time.Date(2026, 2, 27, 14, 30, 0, 0, time.UTC),
		IsDelayed:         false,
		DelayedReportData: nil,
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	expectedJSON := `{
		"id": "019539b4-6b30-7e5a-8000-1a2b3c4d5e6f",
		"txHash": "0x0000000000000000000000000000000000000000000000000000000000abc123",
		"txRLP": "0xf86c",
		"filteredAddresses": [
			{
				"address": "0x000000000000000000000000000000000000dead",
				"reason": "from"
			}
		],
		"chainId": 42161,
		"blockNumber": 1042,
		"parentBlockHash": "0x0000000000000000000000000000000000000000000000000000000000001234",
		"positionInBlock": 3,
		"filteredAt": "2026-02-27T14:30:00Z",
		"isDelayed": false
	}`
	assert.JSONEq(t, expectedJSON, string(data))

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
				Address: common.HexToAddress("0xdead"),
				FilterReason: filter.FilterReason{
					Reason:         filter.ReasonDealiasedFrom,
					EventRuleMatch: nil,
				},
			},
		},
		ChainID:         421614,
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

	expectedJSON := `{
		"id": "019539b4-9a10-7f3b-8000-5f6e7d8c9b0a",
		"txHash": "0x0000000000000000000000000000000000000000000000000000000000def789",
		"txRLP": "0xf86c",
		"filteredAddresses": [
			{
				"address": "0x000000000000000000000000000000000000dead",
				"reason": "dealiased_from"
			}
		],
		"chainId": 421614,
		"blockNumber": 1043,
		"parentBlockHash": "0x0000000000000000000000000000000000000000000000000000000000abcdef",
		"positionInBlock": 0,
		"filteredAt": "2026-02-27T14:31:00Z",
		"isDelayed": true,
		"delayedInboxRequestId": "0x0000000000000000000000000000000000000000000000000000000000000001"
	}`
	assert.JSONEq(t, expectedJSON, string(data))

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
				Address: common.HexToAddress("0xbeef"),
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
		ChainID:           42161,
		BlockNumber:       1044,
		ParentBlockHash:   common.HexToHash("0x5678"),
		PositionInBlock:   1,
		FilteredAt:        time.Date(2026, 2, 27, 14, 32, 0, 0, time.UTC),
		IsDelayed:         false,
		DelayedReportData: nil,
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	expectedJSON := `{
		"id": "019539b4-6b30-7e5a-8000-aabbccddeeff",
		"txHash": "0x0000000000000000000000000000000000000000000000000000000000aaa111",
		"txRLP": "0xf8",
		"filteredAddresses": [
			{
				"address": "0x000000000000000000000000000000000000beef",
				"reason": "event_rule",
				"matchedEvent": "Transfer(address,address,uint256)",
				"matchedTopicIndex": 2,
				"rawLog": {
					"address": "0x000000000000000000000000000000000000dead",
					"topics": [
						"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
						"0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						"0x000000000000000000000000beefbeefbeefbeefbeefbeefbeefbeefbeefbeef"
					],
					"data": "0x0000"
				}
			}
		],
		"chainId": 42161,
		"blockNumber": 1044,
		"parentBlockHash": "0x0000000000000000000000000000000000000000000000000000000000005678",
		"positionInBlock": 1,
		"filteredAt": "2026-02-27T14:32:00Z",
		"isDelayed": false
	}`
	assert.JSONEq(t, expectedJSON, string(data))

	// Round-trip
	var decoded FilteredTxReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, report, decoded)
}
