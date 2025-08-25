// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package multigascollector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
)

func TestTransactionMultiGasToProto(t *testing.T) {
	tests := []struct {
		name     string
		tx       *TransactionMultiGas
		expected func(*testing.T, *proto.TransactionMultiGasData)
	}{
		{
			name: "transaction with all gas dimensions and optional fields",
			tx: &TransactionMultiGas{
				TxHash:  []byte{0x12, 0x34, 0x56},
				TxIndex: 0,
				MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
					multigas.ResourceKindComputation:   100,
					multigas.ResourceKindHistoryGrowth: 50,
					multigas.ResourceKindStorageAccess: 200,
					multigas.ResourceKindStorageGrowth: 1000,
					multigas.ResourceKindL1Calldata:    150,
					multigas.ResourceKindL2Calldata:    300,
					multigas.ResourceKindUnknown:       10,
				}),
			},
			expected: func(t *testing.T, proto *proto.TransactionMultiGasData) {
				assert.Equal(t, []byte{0x12, 0x34, 0x56}, proto.TxHash)
				assert.Equal(t, uint32(0), proto.TxIndex)
				assert.Equal(t, uint64(100), proto.MultiGas.Computation)
				assert.Equal(t, uint64(50), proto.MultiGas.HistoryGrowth)
				assert.Equal(t, uint64(200), proto.MultiGas.StorageAccess)
				assert.Equal(t, uint64(1000), proto.MultiGas.StorageGrowth)
				assert.Equal(t, uint64(150), proto.MultiGas.L1Calldata)
				assert.Equal(t, uint64(300), proto.MultiGas.L2Calldata)
				assert.NotNil(t, proto.MultiGas.Unknown)
				assert.Equal(t, uint64(10), *proto.MultiGas.Unknown)
				assert.Nil(t, proto.MultiGas.Refund) // No refund in test data
			},
		},
		{
			name: "transaction with minimal gas dimensions (no optional fields)",
			tx: &TransactionMultiGas{
				TxHash:  []byte{0x78, 0x9a, 0xbc},
				TxIndex: 1,
				MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
					multigas.ResourceKindComputation: 150,
				}),
			},
			expected: func(t *testing.T, proto *proto.TransactionMultiGasData) {
				assert.Equal(t, []byte{0x78, 0x9a, 0xbc}, proto.TxHash)
				assert.Equal(t, uint32(1), proto.TxIndex)
				assert.Equal(t, uint64(150), proto.MultiGas.Computation)
				assert.Equal(t, uint64(0), proto.MultiGas.HistoryGrowth)
				assert.Equal(t, uint64(0), proto.MultiGas.StorageAccess)
				assert.Equal(t, uint64(0), proto.MultiGas.StorageGrowth)
				assert.Nil(t, proto.MultiGas.Unknown) // Should be nil since value was 0
				assert.Nil(t, proto.MultiGas.Refund)  // Should be nil since value was 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protoData := tt.tx.ToProto()
			tt.expected(t, protoData)
		})
	}
}

func TestBlockInfoToProto(t *testing.T) {
	blockInfo := &BlockInfo{
		BlockNumber:    12345,
		BlockHash:      []byte{0xab, 0xcd, 0xef},
		BlockTimestamp: 1234567890,
	}

	protoData := blockInfo.ToProto()

	// Verify block metadata
	assert.Equal(t, blockInfo.BlockNumber, protoData.BlockNumber)
	assert.Equal(t, blockInfo.BlockHash, protoData.BlockHash)
	assert.Equal(t, blockInfo.BlockTimestamp, protoData.BlockTimestamp)
	assert.Empty(t, protoData.Transactions) // No transactions initially
}
