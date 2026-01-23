// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package eventfilter

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestExtractAddresses_EdgeCases(t *testing.T) {
	rule := EventRule{
		Signature:      crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
		TopicAddresses: []int{1, 2},
		Bypass:         &BypassRule{TopicIndex: 2, EqualsTo: common.Address{}},
	}
	filter, err := NewEventFilter([]EventRule{rule})
	if err != nil {
		t.Fatal(err)
	}

	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")

	tests := []struct {
		name     string
		topics   []common.Hash
		data     []byte
		expected int // expected number of addresses, -1 means nil
	}{
		{
			name:     "empty topics",
			topics:   []common.Hash{},
			expected: -1,
		},
		{
			name:     "unknown signature",
			topics:   []common.Hash{crypto.Keccak256Hash([]byte("Unknown(address)"))},
			expected: -1,
		},
		{
			name:     "signature only, no indexed params",
			topics:   []common.Hash{rule.Signature},
			expected: 0,
		},
		{
			name:     "one indexed param",
			topics:   []common.Hash{rule.Signature, common.BytesToHash(addr1.Bytes())},
			expected: 1,
		},
		{
			name:     "two indexed params",
			topics:   []common.Hash{rule.Signature, common.BytesToHash(addr1.Bytes()), common.BytesToHash(addr2.Bytes())},
			expected: 2,
		},
		{
			name:     "bypass triggered (to == 0x0)",
			topics:   []common.Hash{rule.Signature, common.BytesToHash(addr1.Bytes()), common.BytesToHash(common.Address{}.Bytes())},
			expected: -1,
		},
		{
			name:     "duplicate addresses",
			topics:   []common.Hash{rule.Signature, common.BytesToHash(addr1.Bytes()), common.BytesToHash(addr1.Bytes())},
			expected: 1, // deduped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ExtractAddresses(tt.topics, tt.data, common.Address{}, common.Address{})
			if tt.expected == -1 {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if len(result) != tt.expected {
					t.Errorf("expected %d addresses, got %d", tt.expected, len(result))
				}
			}
		})
	}
}

func TestExtractAddresses_TransferRules(t *testing.T) {
	rulesJSON := `{
		"rules": [
			{
				"event": "Transfer(address,address,uint256)",
				"topicAddresses": [1, 2],
				"bypass": {"topicIndex": 2, "equalsTo": "0x0000000000000000000000000000000000000000"}
			},
			{
				"event": "TransferSingle(address,address,address,uint256,uint256)",
				"topicAddresses": [2, 3],
				"bypass": {"topicIndex": 3, "equalsTo": "0x0000000000000000000000000000000000000000"}
			},
			{
				"event": "TransferBatch(address,address,address,uint256[],uint256[])",
				"topicAddresses": [2, 3],
				"bypass": {"topicIndex": 3, "equalsTo": "0x0000000000000000000000000000000000000000"}
			}
		]
	}`

	filter, err := NewEventFilterFromJSON([]byte(rulesJSON))
	if err != nil {
		t.Fatal(err)
	}

	operator := common.HexToAddress("0x1111111111111111111111111111111111111111")
	from := common.HexToAddress("0x2222222222222222222222222222222222222222")
	to := common.HexToAddress("0x3333333333333333333333333333333333333333")
	zero := common.Address{}

	transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	transferSingleSig := crypto.Keccak256Hash([]byte("TransferSingle(address,address,address,uint256,uint256)"))
	transferBatchSig := crypto.Keccak256Hash([]byte("TransferBatch(address,address,address,uint256[],uint256[])"))

	tests := []struct {
		name      string
		topics    []common.Hash
		wantAddrs []common.Address
		wantNil   bool
	}{
		{
			name: "ERC20 transfer",
			topics: []common.Hash{
				transferSig,
				common.BytesToHash(from.Bytes()),
				common.BytesToHash(to.Bytes()),
			},
			wantAddrs: []common.Address{from, to},
		},
		{
			name: "ERC20 burn (bypass)",
			topics: []common.Hash{
				transferSig,
				common.BytesToHash(from.Bytes()),
				common.BytesToHash(zero.Bytes()),
			},
			wantNil: true,
		},
		{
			name: "ERC1155 TransferSingle",
			topics: []common.Hash{
				transferSingleSig,
				common.BytesToHash(operator.Bytes()),
				common.BytesToHash(from.Bytes()),
				common.BytesToHash(to.Bytes()),
			},
			wantAddrs: []common.Address{from, to},
		},
		{
			name: "ERC1155 TransferSingle burn (bypass)",
			topics: []common.Hash{
				transferSingleSig,
				common.BytesToHash(operator.Bytes()),
				common.BytesToHash(from.Bytes()),
				common.BytesToHash(zero.Bytes()),
			},
			wantNil: true,
		},
		{
			name: "ERC1155 TransferBatch",
			topics: []common.Hash{
				transferBatchSig,
				common.BytesToHash(operator.Bytes()),
				common.BytesToHash(from.Bytes()),
				common.BytesToHash(to.Bytes()),
			},
			wantAddrs: []common.Address{from, to},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ExtractAddresses(tt.topics, nil, common.Address{}, common.Address{})

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.wantAddrs) {
				t.Errorf("expected %d addresses, got %d", len(tt.wantAddrs), len(result))
				return
			}

			resultSet := make(map[common.Address]struct{})
			for _, a := range result {
				resultSet[a] = struct{}{}
			}
			for _, want := range tt.wantAddrs {
				if _, ok := resultSet[want]; !ok {
					t.Errorf("missing expected address %s", want.Hex())
				}
			}
		})
	}
}
