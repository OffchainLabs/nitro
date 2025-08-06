package multigasCollector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"

	multigas "github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/multigasCollector/proto"
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

func TestNewCollector(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr error
	}{
		{
			name: "valid config",
			config: Config{
				OutputDir: t.TempDir(),
				BatchSize: 10,
			},
			expectErr: nil,
		},
		{
			name: "empty output directory",
			config: Config{
				OutputDir: "",
				BatchSize: 10,
			},
			expectErr: ErrOutputDirRequired,
		},
		{
			name: "zero batch size",
			config: Config{
				OutputDir: t.TempDir(),
				BatchSize: 0,
			},
			expectErr: ErrBatchSizeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make(chan *CollectorMessage)

			collector, err := NewCollector(tt.config, input)

			if tt.expectErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectErr, err)
				assert.Nil(t, collector)
			} else {
				require.NoError(t, err)
				require.NotNil(t, collector)
				assert.Equal(t, tt.config.OutputDir, collector.config.OutputDir)
				assert.Equal(t, tt.config.BatchSize, collector.config.BatchSize)

				close(input)
				collector.Wait()
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
			name:      "single block - one transaction",
			batchSize: 1,
			inputData: []*CollectorMessage{
				{
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
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
			},
			expectFiles: 1,
		},
		{
			name:      "multiple blocks - single batch",
			batchSize: 3,
			inputData: []*CollectorMessage{
				{
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
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
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12346,
						BlockHash:      []byte{0xde, 0xf1, 0x23},
						BlockTimestamp: 1234567891,
					},
				},
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
			},
			expectFiles: 1,
		},
		{
			name:      "multiple blocks - multiple batches",
			batchSize: 2,
			inputData: []*CollectorMessage{
				{
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12345,
						BlockHash:      []byte{0xab, 0xcd, 0xef},
						BlockTimestamp: 1234567890,
					},
				},
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
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12346,
						BlockHash:      []byte{0xde, 0xf1, 0x23},
						BlockTimestamp: 1234567891,
					},
				},
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
					Type: CollectorMsgBlock,
					Block: &BlockInfo{
						BlockNumber:    12347,
						BlockHash:      []byte{0x45, 0x67, 0x89},
						BlockTimestamp: 1234567892,
					},
				},
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
			},
			expectFiles: 2, // Two batches expected
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			input := make(chan *CollectorMessage, 10)

			config := Config{
				OutputDir: tmpDir,
				BatchSize: int(tt.batchSize), //nolint:gosec // safe: test batchSize is controlled
			}

			collector, err := NewCollector(config, input)
			require.NoError(t, err)

			// Send all input data and count blocks
			var blockCount int
			for _, msg := range tt.inputData {
				input <- msg
				if msg.Type == CollectorMsgBlock {
					blockCount++
				}
			}

			// Close input and wait for completion
			close(input)
			collector.Wait()

			// Verify file count
			files, err := filepath.Glob(filepath.Join(tmpDir, "multigas_batch_*.pb"))
			require.NoError(t, err)
			assert.Len(t, files, tt.expectFiles)

			// Read all batch data from files
			var allData []*proto.BlockMultiGasData
			for _, file := range files {
				data, err := os.ReadFile(file)
				require.NoError(t, err)

				var batch proto.BlockMultiGasBatch
				err = protobuf.Unmarshal(data, &batch)
				require.NoError(t, err)

				allData = append(allData, batch.Data...)
			}

			// Verify block count matches input
			assert.Len(t, allData, blockCount)

			// Iterate through input messages and verify against output data
			var blockIndex int
			var txIndexInBlock int
			for _, message := range tt.inputData {
				switch message.Type {
				case CollectorMsgBlock:
					assert.Equal(t, message.Block.BlockNumber, allData[blockIndex].BlockNumber)
					assert.Equal(t, message.Block.BlockHash, allData[blockIndex].BlockHash)
					assert.Equal(t, message.Block.BlockTimestamp, allData[blockIndex].BlockTimestamp)
					blockIndex++
					txIndexInBlock = 0 // Reset transaction counter for new block

				case CollectorMsgTransaction:
					txProto := allData[blockIndex-1].Transactions[txIndexInBlock]
					assert.Equal(t, message.Transaction.TxHash, txProto.TxHash)
					assert.Equal(t, message.Transaction.TxIndex, txProto.TxIndex)
					assert.Equal(t, message.Transaction.MultiGas.Get(multigas.ResourceKindComputation), txProto.MultiGas.Computation)
					assert.Equal(t, message.Transaction.MultiGas.Get(multigas.ResourceKindHistoryGrowth), txProto.MultiGas.HistoryGrowth)
					assert.Equal(t, message.Transaction.MultiGas.Get(multigas.ResourceKindStorageAccess), txProto.MultiGas.StorageAccess)
					assert.Equal(t, message.Transaction.MultiGas.Get(multigas.ResourceKindStorageGrowth), txProto.MultiGas.StorageGrowth)
					txIndexInBlock++ // Move to next transaction in this block
				}
			}
		})
	}
}

func TestCollectorChannelClosed(t *testing.T) {
	tmpDir := t.TempDir()
	input := make(chan *CollectorMessage, 10)

	config := Config{
		OutputDir: tmpDir,
		BatchSize: 10,
	}

	collector, err := NewCollector(config, input)
	require.NoError(t, err)

	// Add some data
	message := &CollectorMessage{
		Type: CollectorMsgBlock,
		Block: &BlockInfo{
			BlockNumber:    12345,
			BlockHash:      []byte{0xab, 0xcd, 0xef},
			BlockTimestamp: 1234567890,
		},
	}

	input <- message

	// Close input channel - should flush remaining data
	close(input)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Verify data was flushed
	files, err := filepath.Glob(filepath.Join(tmpDir, "multigas_batch_*.pb"))
	require.NoError(t, err)
	assert.Len(t, files, 1)

	collector.Wait()
}
