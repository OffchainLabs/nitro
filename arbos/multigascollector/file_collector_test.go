// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package multigascollector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
)

func TestIdleCollector(t *testing.T) {
	tests := []struct {
		name      string
		config    CollectorConfig
		expectErr error
	}{
		{
			name: "valid config",
			config: CollectorConfig{
				OutputDir: t.TempDir(),
				BatchSize: 10,
			},
			expectErr: nil,
		},
		{
			name: "empty output directory",
			config: CollectorConfig{
				OutputDir: "",
				BatchSize: 10,
			},
			expectErr: ErrOutputDirRequired,
		},
		{
			name: "zero batch size",
			config: CollectorConfig{
				OutputDir: t.TempDir(),
				BatchSize: 0,
			},
			expectErr: ErrBatchSizeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			collector, err := NewFileCollector(tt.config)

			if tt.expectErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectErr, err)
				assert.Nil(t, collector)
			} else {
				require.NoError(t, err)
				require.NotNil(t, collector)
				assert.Equal(t, tt.config.OutputDir, collector.config.OutputDir)
				assert.Equal(t, tt.config.BatchSize, collector.config.BatchSize)

				collector.Start(ctx)
				collector.StopAndWait()
			}
		})
	}
}

func TestDataCollection(t *testing.T) {
	testCases := []struct {
		name        string
		batchSize   uint64
		inputData   []*Message
		expectFiles int
	}{
		{
			name:        "empty input",
			batchSize:   10,
			inputData:   nil,
			expectFiles: 0,
		},
		{
			name:      "empty block",
			batchSize: 10,
			inputData: []*Message{
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    20001,
						BlockHash:      []byte{0xaa, 0xbb, 0xcc},
						BlockTimestamp: 1111111111,
					},
				},
			},
			expectFiles: 1,
		},
		{
			name:      "discarded transaction",
			batchSize: 10,
			inputData: []*Message{
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 100},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 50},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 200},
							multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 1000},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindWasmComputation, Amount: 75},
							multigas.Pair{Kind: multigas.ResourceKindUnknown, Amount: 25},
						),
					},
				},
				{
					Type: MsgPrepareToCollectBlock,
				},
			},
			expectFiles: 0,
		},
		{
			name:      "single block - one transaction",
			batchSize: 1,
			inputData: []*Message{
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 100},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 50},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 200},
							multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 1000},
							multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 150},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindUnknown, Amount: 25},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
			},
			expectFiles: 1,
		},
		{
			name:      "start new block without finalising previous -> drop unfinalised txs",
			batchSize: 10,
			inputData: []*Message{
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:   []byte{0x01},
						TxIndex:  0,
						MultiGas: multigas.ComputationGas(1),
					},
				},
				// Start a new block before finalising the previous -> prior tx is dropped
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:   []byte{0x02},
						TxIndex:  0,
						MultiGas: multigas.ComputationGas(2),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    30001,
						BlockHash:      []byte{0x01, 0x02, 0x03},
						BlockTimestamp: 2222222222,
					},
				},
			},
			expectFiles: 1,
		},
		{
			name:      "multiple blocks - single batch",
			batchSize: 3,
			inputData: []*Message{
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 100},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 25},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 50},
							multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 150},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindWasmComputation, Amount: 200},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x45, 0x67, 0x89},
						TxIndex: 1,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 200},
							multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 500},
							multigas.Pair{Kind: multigas.ResourceKindUnknown, Amount: 15},
						),
					},
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x78, 0x9a, 0xbc},
						TxIndex: 2,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 75},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 30},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 150},
							multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 75},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12346,
						BlockHash:      []byte{0xde, 0xf1, 0x23},
						BlockTimestamp: 1234567891,
					},
				},
			},
			expectFiles: 1,
		},
		{
			name:      "multiple blocks - multiple batches",
			batchSize: 2,
			inputData: []*Message{
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 100},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 40},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 80},
							multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindWasmComputation, Amount: 60},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x78, 0x9a, 0xbc},
						TxIndex: 1,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 200},
							multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 60},
							multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 120},
							multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 75},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12346,
						BlockHash:      []byte{0xde, 0xf1, 0x23},
						BlockTimestamp: 1234567891,
					},
				},
				{
					Type: MsgPrepareToCollectBlock,
				},
				{
					Type: MsgTransactionMultiGas,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0xab, 0xcd, 0xef},
						TxIndex: 2,
						MultiGas: multigas.MultiGasFromPairs(
							multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 800},
							multigas.Pair{Kind: multigas.ResourceKindUnknown, Amount: 35},
							multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 150},
							multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 300},
							multigas.Pair{Kind: multigas.ResourceKindWasmComputation, Amount: 200},
						),
					},
				},
				{
					Type: MsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12347,
						BlockHash:      []byte{0x45, 0x67, 0x89},
						BlockTimestamp: 1234567892,
					},
				},
			},
			expectFiles: 2,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			config := CollectorConfig{
				OutputDir: tmpDir,
				BatchSize: int(tt.batchSize), //nolint:gosec
			}

			collector, err := NewFileCollector(config)
			require.NoError(t, err)
			collector.Start(context.Background())

			for _, msg := range tt.inputData {
				collector.Submit(msg)
			}
			collector.StopAndWait()

			// Open result files and decode data into 'got'
			files, err := filepath.Glob(filepath.Join(tmpDir, "multigas_batch_*.pb"))
			require.NoError(t, err)
			assert.Len(t, files, tt.expectFiles)

			var got []*proto.BlockMultiGasData
			for _, f := range files {
				raw, err := os.ReadFile(f)
				require.NoError(t, err)
				var batch proto.BlockMultiGasBatch
				require.NoError(t, protobuf.Unmarshal(raw, &batch))
				got = append(got, batch.Data...)
			}

			// Build expected protobufs from input messages
			var expected []*proto.BlockMultiGasData
			var curTxs []*proto.TransactionMultiGasData
			for _, m := range tt.inputData {
				switch m.Type {
				case MsgPrepareToCollectBlock:
					curTxs = nil // drop unfinalised txs
				case MsgTransactionMultiGas:
					curTxs = append(curTxs, m.Transaction.ToProto())
				case MsgFinaliseBlock:
					blk := m.Block.ToProto()
					if len(curTxs) > 0 {
						blk.Transactions = append(blk.Transactions, curTxs...)
					}
					expected = append(expected, blk)
					curTxs = nil
				}
			}

			// Proto-based comparison
			if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
				t.Fatalf("batch content mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
