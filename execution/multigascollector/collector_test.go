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

	"github.com/offchainlabs/nitro/execution/multigascollector/proto"
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

			collector, err := NewCollector(tt.config)

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
		inputData   []*CollectorMessage
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
			inputData: []*CollectorMessage{
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgFinaliseBlock,
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
			inputData: []*CollectorMessage{
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   100,
							multigas.ResourceKindHistoryGrowth: 50,
							multigas.ResourceKindStorageAccess: 200,
							multigas.ResourceKindStorageGrowth: 1000,
							multigas.ResourceKindUnknown:       25,
						}),
					},
				},
				{Type: CollectorMsgStartBlock},
			},
			expectFiles: 0,
		},
		{
			name:      "single block - one transaction",
			batchSize: 1,
			inputData: []*CollectorMessage{
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   100,
							multigas.ResourceKindHistoryGrowth: 50,
							multigas.ResourceKindStorageAccess: 200,
							multigas.ResourceKindStorageGrowth: 1000,
							multigas.ResourceKindUnknown:       25,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
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
			inputData: []*CollectorMessage{
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x01},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation: 1,
						}),
					},
				},
				// Start a new block before finalising the previous -> prior tx is dropped
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x02},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation: 2,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
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
			inputData: []*CollectorMessage{
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   100,
							multigas.ResourceKindHistoryGrowth: 25,
							multigas.ResourceKindStorageAccess: 50,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x45, 0x67, 0x89},
						TxIndex: 1,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   200,
							multigas.ResourceKindStorageGrowth: 500,
							multigas.ResourceKindUnknown:       15,
						}),
					},
				},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x78, 0x9a, 0xbc},
						TxIndex: 2,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   75,
							multigas.ResourceKindHistoryGrowth: 30,
							multigas.ResourceKindStorageAccess: 150,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
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
			inputData: []*CollectorMessage{
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x12, 0x34, 0x56},
						TxIndex: 0,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   100,
							multigas.ResourceKindHistoryGrowth: 40,
							multigas.ResourceKindStorageAccess: 80,
							multigas.ResourceKindStorageGrowth: 300,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0x78, 0x9a, 0xbc},
						TxIndex: 1,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   200,
							multigas.ResourceKindHistoryGrowth: 60,
							multigas.ResourceKindStorageAccess: 120,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
					Block: &BlockInfo{
						BlockNumber:    12346,
						BlockHash:      []byte{0xde, 0xf1, 0x23},
						BlockTimestamp: 1234567891,
					},
				},
				{Type: CollectorMsgStartBlock},
				{
					Type: CollectorMsgTransaction,
					Transaction: &TransactionMultiGas{
						TxHash:  []byte{0xab, 0xcd, 0xef},
						TxIndex: 2,
						MultiGas: *multigas.MultiGasFromMap(map[multigas.ResourceKind]uint64{
							multigas.ResourceKindComputation:   300,
							multigas.ResourceKindStorageGrowth: 800,
							multigas.ResourceKindUnknown:       35,
						}),
					},
				},
				{
					Type: CollectorMsgFinaliseBlock,
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

			collector, err := NewCollector(config)
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
				case CollectorMsgStartBlock:
					curTxs = nil // drop unfinalised txs
				case CollectorMsgTransaction:
					curTxs = append(curTxs, m.Transaction.ToProto())
				case CollectorMsgFinaliseBlock:
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
