package multigasCollector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	protobuf "google.golang.org/protobuf/proto"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/multigasCollector/proto"
)

const (
	// batchFilenameFormat defines the naming pattern for batch files.
	// Format: multigas_batch_<batch_number>_<timestamp>.pb
	batchFilenameFormat = "multigas_batch_%010d_%d.pb"

	// Preallocate for 2000 transactions per block
	defaultTxPrealloc = 2000

	// CollectorMsgQueueSize defines the size of the collector message queue.
	CollectorMsgQueueSize = 1024
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
	// CollectorMsgStartBlock indicates a message for starting a new block without metadata.
	CollectorMsgStartBlock CollectorMessageType = iota
	// CollectorMsgTransaction indicates a message for multi-gas data of a transaction.
	CollectorMsgTransaction
	// CollectorMsgFinaliseBlock indicates a message finalising a block with metadata.
	CollectorMsgFinaliseBlock
)

// CollectorMessage represents a message sent to the collector.
type CollectorMessage struct {
	Type CollectorMessageType

	Block       *BlockInfo
	Transaction *TransactionMultiGas
}

// Config holds the configuration for the MultiGas collector.
type Config struct {
	OutputDir string `koanf:"output-dir"`
	BatchSize int    `koanf:"batch-size"`
}

// Collector manages the asynchronous collection and batching of multi-dimensional
// gas data from blocks. It owns the message channel, buffers data in memory,
// and periodically writes batches to disk in protobuf format.
type Collector struct {
	config Config
	in     chan *CollectorMessage

	batchNum          int64
	blockBuffer       []*proto.BlockMultiGasData
	transactionBuffer []*proto.TransactionMultiGasData

	cancel context.CancelFunc
	wg     sync.WaitGroup
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
// Validates the configuration and ensures the output directory exists.
func NewCollector(config Config) (*Collector, error) {
	if config.OutputDir == "" {
		return nil, ErrOutputDirRequired
	}
	if config.BatchSize <= 0 {
		return nil, ErrBatchSizeRequired
	}
	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateOutputDir, err)
	}

	return &Collector{
		config:            config,
		in:                nil,
		batchNum:          0,
		blockBuffer:       make([]*proto.BlockMultiGasData, 0, config.BatchSize),
		transactionBuffer: make([]*proto.TransactionMultiGasData, 0, defaultTxPrealloc),
	}, nil
}

// Start begins background processing of incoming messages.
func (c *Collector) Start(parent context.Context) {
	if c.in != nil {
		log.Warn("Multi-gas collector already started, ignoring Start call")
		return
	}

	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel
	c.in = make(chan *CollectorMessage, CollectorMsgQueueSize)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.processData(ctx)
	}()
}

// StopAndWait cancels processing, drains ready items, flushes once, and waits for shutdown.
func (c *Collector) StopAndWait() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	c.in = nil // Clear the input channel to prevent further submissions
	log.Info("Multi-gas collector stopped")
}

// Submit enqueues a message for collection (non-blocking). Drops on a full queue.
func (c *Collector) Submit(msg *CollectorMessage) {
	if c.in == nil {
		log.Debug("Multi-gas collector disabled; dropping message", "type", msg.Type)
		return
	}

	select {
	case c.in <- msg:
	default:
		log.Debug("Multi-gas collector dropping message",
			"queueCapacity", cap(c.in), "queueLen", len(c.in), "type", msg.Type)
	}
}

// processData consumes input and batches writes.
// Shutdown is driven by context cancellation. On cancel, finalise() drains
// queued items, flushes once, and exits.
func (c *Collector) processData(ctx context.Context) {
	for {
		select {
		case msg := <-c.in:
			c.handleMessage(msg)
		case <-ctx.Done():
			c.finalise()
			return
		}
	}
}

// handleMessage processes a single CollectorMessage and updates in-memory buffers.
// TXs are accumulated in transactionBuffer. When a Block message arrives, all
// buffered TXs are wrapped into that block, appended to blockBuffer
func (c *Collector) handleMessage(msg *CollectorMessage) {
	switch msg.Type {
	case CollectorMsgStartBlock:
		// If c.transactionBuffer contains unflushed transactions, block was not finalised (stay silent)
		if len(c.transactionBuffer) > 0 {
			c.transactionBuffer = c.transactionBuffer[:0]
		}

	case CollectorMsgTransaction:
		if msg.Transaction == nil {
			log.Error("Multi-gas collector transaction message missing payload")
			return
		}
		c.transactionBuffer = append(c.transactionBuffer, msg.Transaction.ToProto())

	case CollectorMsgFinaliseBlock:
		if msg.Block == nil {
			log.Error("Multi-gas collector block message missing payload")
			return
		}
		block := msg.Block.ToProto()
		if len(c.transactionBuffer) > 0 {
			block.Transactions = append(block.Transactions, c.transactionBuffer...)
		}

		c.blockBuffer = append(c.blockBuffer, block)
		if len(c.blockBuffer) >= c.config.BatchSize {
			if err := c.flushBatch(); err != nil {
				log.Error("Multi-gas collector failed to flush batch", "error", err)
			}
		}
		c.transactionBuffer = c.transactionBuffer[:0]

	default:
		log.Error("Multi-gas collector, unknown message type", "type", msg.Type)
	}
}

func (c *Collector) finalise() {
	// Drain channel on shutdown
drainLoop:
	for {
		select {
		case msg := <-c.in:
			c.handleMessage(msg)
		default:
			break drainLoop
		}
	}

	if len(c.transactionBuffer) > 0 {
		log.Warn("Multi-gas collector finalising with unassociated transactions", "count", len(c.transactionBuffer))
		c.transactionBuffer = c.transactionBuffer[:0]
	}
	if len(c.blockBuffer) > 0 {
		if err := c.flushBatch(); err != nil {
			log.Error("Multi-gas collector failed to flush final batch", "error", err)
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
	outPath := filepath.Join(c.config.OutputDir, filename)

	if err := os.WriteFile(outPath, data, 0600); err != nil {
		return err
	}

	log.Info("Multi-gas collector wrote multi-gas batch",
		"file", filename,
		"count", len(c.blockBuffer),
		"size_bytes", len(data))

	c.blockBuffer = c.blockBuffer[:0]
	c.batchNum++

	return nil
}
