package multigasCollector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	protobuf "google.golang.org/protobuf/proto"

	multigas "github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/multigasCollector/proto"
	"github.com/offchainlabs/nitro/util/stopwaiter"
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
	stopwaiter.StopWaiter

	config Config
	input  <-chan *CollectorMessage

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

// NewCollector returns an initialized collector.
//
// It validates the configuration and ensures the output directory exists.
//
// Parameters:
//   - config: Configuration specifying output directory and batch size
//   - input: Channel supplying CollectorMessage values
//
// Returns:
//   - *Collector: The initialized collector
//   - error: Configuration validation or initialization error
//
// Usage:
//
//	c, _ := NewCollector(cfg, input)
//	c.Start(ctx)
//	// ... send messages ...
//	c.StopAndWait() // flushes and stops regardless of channel state
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

	return &Collector{
		config:       config,
		input:        input,
		batchNum:     0,
		blockBuffer:  make([]*proto.BlockMultiGasData, 0, config.BatchSize),
		currentBlock: nil,
	}, nil
}

// Start begins background processing using StopWaiter.
// Processing continues until StopAndWait() is called or ctx is canceled.
// Channel close is handled (remaining data is flushed), but shutdown does not
// depend on it.
func (c *Collector) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
	c.LaunchThread(func(ctx context.Context) {
		c.processData(ctx)
	})
}

// StopAndWait stops background processing and waits for it to finish.
func (c *Collector) StopAndWait() {
	c.StopWaiter.StopAndWait()
	log.Info("Multi-gas collector stopped")
}

// processData consumes input and batches writes.
// Stop is driven by StopWaiter (ctx). Channel close is tolerated:
// we flush once, ignore the channel thereafter, and wait for Stop.
func (c *Collector) processData(ctx context.Context) {
	in := c.input
	for {
		select {
		case <-ctx.Done():
			// Drain any ready items to avoid dropping work on Stop
		drain:
			for {
				select {
				case msg, ok := <-in:
					if !ok {
						break drain
					}
					c.handleMessage(msg)
					continue
				default:
				}
				break
			}
			c.finalize()
			return

		case msg, ok := <-in:
			if !ok {
				// Channel closed: flush and keep running until Stop (no reliance on close)
				c.finalize()
				in = nil
				continue
			}
			c.handleMessage(msg)
		}
	}
}

// --- internal ---

// handleMessage processes a single CollectorMessage and updates the in-memory batch.
// If a batch reaches config.BatchSize, it is flushed immediately.
func (c *Collector) handleMessage(msg *CollectorMessage) {
	switch msg.Type {
	case CollectorMsgBlock:
		if c.currentBlock != nil {
			c.blockBuffer = append(c.blockBuffer, c.currentBlock)
			if len(c.blockBuffer) >= c.config.BatchSize {
				if err := c.flushBatch(); err != nil {
					log.Error("Failed to flush batch", "error", err)
				}
			}
		}
		c.currentBlock = msg.Block.ToProto()

	case CollectorMsgTransaction:
		if c.currentBlock == nil {
			log.Error("Received transaction before block message")
			return
		}
		c.currentBlock.Transactions = append(c.currentBlock.Transactions, msg.Transaction.ToProto())

	default:
		log.Error("Unknown message type")
	}
}

func (c *Collector) finalize() {
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
