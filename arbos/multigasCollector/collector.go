package multigasCollector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	protobuf "google.golang.org/protobuf/proto"

	multigas "github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/multigasCollector/proto"
)

const (
	// batchFilenameFormat defines the naming pattern for batch files.
	// Format: multigas_batch_<batch_number>_<timestamp>.pb
	batchFilenameFormat = "multigas_batch_%010d_%d.pb"
)

var (
	ErrOutputDirRequired = errors.New("output directory is required")
	ErrBatchSizeRequired = errors.New("batch size must be greater than zero")
	ErrCreateOutputDir   = errors.New("failed to create output directory")
)

// TransactionMultiGas represents gas data for a single transaction
type TransactionMultiGas struct {
	TxHash   []byte
	TxIndex  uint32
	MultiGas multigas.MultiGas
}

// BlockInfo represents information about a block
type BlockInfo struct {
	BlockNumber    uint64
	BlockHash      []byte
	BlockTimestamp uint64
}

// CollectorMessageType defines the type of message being processed by the collector.
type CollectorMessageType int

const (
	// CollectorMsgBlock indicates a message starting a new block.
	CollectorMsgBlock CollectorMessageType = iota
	// CollectorMsgTransaction indicates a message for multi-gas data of a transaction.
	CollectorMsgTransaction
)

// CollectorMessage represents a message sent to the collector.
type CollectorMessage struct {
	Type CollectorMessageType

	Block       *BlockInfo
	Transaction *TransactionMultiGas
}

// Config holds the configuration for the MultiGas collector.
type Config struct {
	OutputDir string
	BatchSize int
}

// Collector manages the asynchronous collection and batching of multi-dimensional
// gas data from blocks. It receives BlockTransactionMultiGas data through a channel, buffers
// it in memory, and periodically writes batches to disk in protobuf format.
type Collector struct {
	config       Config
	input        <-chan *CollectorMessage
	wg           sync.WaitGroup
	batchNum     int64
	blockBuffer  []*proto.BlockMultiGasData
	currentBlock *proto.BlockMultiGasData
}

// ToProto converts the TransactionMultiGas to its protobuf representation.
func (tx *TransactionMultiGas) ToProto() *proto.TransactionMultiGasData {
	multiGasData := &proto.MultiGasData{
		Computation:   tx.MultiGas.Get(multigas.ResourceKindComputation),
		StorageAccess: tx.MultiGas.Get(multigas.ResourceKindStorageAccess),
		StorageGrowth: tx.MultiGas.Get(multigas.ResourceKindStorageGrowth),
		HistoryGrowth: tx.MultiGas.Get(multigas.ResourceKindHistoryGrowth),
	}

	if unknown := tx.MultiGas.Get(multigas.ResourceKindUnknown); unknown > 0 {
		multiGasData.Unknown = &unknown
	}

	if refund := tx.MultiGas.GetRefund(); refund > 0 {
		multiGasData.Refund = &refund
	}

	return &proto.TransactionMultiGasData{
		TxHash:   tx.TxHash,
		TxIndex:  tx.TxIndex,
		MultiGas: multiGasData,
	}
}

// ToProto converts the BlockInfo to its protobuf representation.
func (btmg *BlockInfo) ToProto() *proto.BlockMultiGasData {
	return &proto.BlockMultiGasData{
		BlockNumber:    btmg.BlockNumber,
		BlockHash:      btmg.BlockHash,
		BlockTimestamp: btmg.BlockTimestamp,
	}
}

// NewCollector creates and starts a new multi-gas data collector.
//
// The collector will:
// 1. Validate the provided configuration
// 2. Create the output directory if it doesn't exist
// 3. Start a background goroutine to process incoming data
// 4. Return immediately, ready to receive data on the input channel
//
// Parameters:
//   - config: Configuration specifying output directory and batch size
//   - input: Channel for receiving BlockTransactionMultiGas data (collector takes ownership)
//
// Returns:
//   - *Collector: The initialized collector ready to receive data
//   - error: Configuration validation or initialization error
//
// The caller should close the input channel when done sending data, then call
// Wait() to ensure all data has been written to disk.
func NewCollector(config Config, input <-chan *CollectorMessage) (*Collector, error) {
	if config.OutputDir == "" {
		return nil, ErrOutputDirRequired
	}

	if config.BatchSize == 0 {
		return nil, ErrBatchSizeRequired
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, ErrCreateOutputDir
	}

	c := &Collector{
		config:       config,
		input:        input,
		wg:           sync.WaitGroup{},
		batchNum:     0,
		blockBuffer:  make([]*proto.BlockMultiGasData, 0, config.BatchSize),
		currentBlock: nil,
	}

	// Start processing data in a separate goroutine
	c.wg.Add(1)
	go c.processData()

	return c, nil
}

// processData is the main processing loop that runs in a background goroutine.
// It continuously reads BlockTransactionMultiGas data from the input channel, converts it
// to protobuf format, buffers it, and writes batches to disk when the buffer
// fills up or when the channel is closed.
func (c *Collector) processData() {
	defer c.wg.Done()

	for msg := range c.input {
		switch msg.Type {
		case CollectorMsgBlock:
			if c.currentBlock != nil {
				// Add the current block to the buffer
				c.blockBuffer = append(c.blockBuffer, c.currentBlock)

				// Save buffered blocks to disk if batch size is reached
				if len(c.blockBuffer) >= c.config.BatchSize {
					if err := c.flushBatch(); err != nil {
						log.Error("Failed to flush batch", "error", err)
					}
				}
			}
			// Reset cache to new block
			c.currentBlock = msg.Block.ToProto()

		case CollectorMsgTransaction:
			if c.currentBlock == nil {
				log.Error("Received transaction before block message")
				continue
			}

			// Convert transaction to protobuf and add to current block
			c.currentBlock.Transactions = append(c.currentBlock.Transactions, msg.Transaction.ToProto())

		default:
			log.Error("Unknown message type")
		}
	}

	// Channel closed, flush remaining data
	if c.currentBlock != nil {
		c.blockBuffer = append(c.blockBuffer, c.currentBlock)
		c.currentBlock = nil
	}

	if len(c.blockBuffer) > 0 {
		if err := c.flushBatch(); err != nil {
			log.Error("Failed to flush final batch", "error", err)
		}
	}
}

// flushBatch writes the current buffer contents to disk as a protobuf batch file.
// This method:
// 1. Creates a BlockMultiGasBatch protobuf message with current buffer data
// 2. Serializes the batch to binary protobuf format
// 3. Writes the data to a uniquely named file
// 4. Clears the buffer and increments the batch counter
//
// File naming pattern: multigas_batch_<batch_number>_<timestamp>.pb
func (c *Collector) flushBatch() error {
	batch := &proto.BlockMultiGasBatch{
		Data: make([]*proto.BlockMultiGasData, len(c.blockBuffer)),
	}
	copy(batch.Data, c.blockBuffer)

	data, err := protobuf.Marshal(batch)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf(batchFilenameFormat, c.batchNum, time.Now().Unix())
	filepath := filepath.Join(c.config.OutputDir, filename)

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return err
	}

	log.Info("Wrote multi-gas batch",
		"file", filename,
		"count", len(c.blockBuffer),
		"size_bytes", len(data))

	c.blockBuffer = c.blockBuffer[:0]
	c.batchNum++

	return nil
}

// Wait blocks until the collector has finished processing all data and shut down.
// This method should be called after closing the input channel to ensure all
// data has been written to disk before the program exits.
//
// Usage pattern:
//
//	close(input)       // Signal no more data
//	collector.Wait()   // Wait for shutdown
//
// This method is safe to call multiple times and will return immediately if
// the collector has already stopped.
func (c *Collector) Wait() {
	c.wg.Wait()
	log.Info("Multi-gas collector stopped")
}
