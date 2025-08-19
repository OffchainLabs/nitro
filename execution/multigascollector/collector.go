package multigascollector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	flag "github.com/spf13/pflag"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/multigascollector/proto"
)

const (
	// batchFilenameFormat defines the naming pattern for batch files.
	// Format: multigas_batch_<start_block_number>_<end_block_number>.pb
	batchFilenameFormat = "multigas_batch_%010d_%010d.pb"

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

// CollectorConfig holds the configuration for the MultiGas collector.
type CollectorConfig struct {
	OutputDir      string `koanf:"output-dir"`
	BatchSize      int    `koanf:"batch-size"`
	ClearOutputDir bool   `koanf:"clear-output-dir"`
}

var DefaultCollectorConfig = CollectorConfig{
	OutputDir:      "",
	BatchSize:      2000,
	ClearOutputDir: false,
}

// Collector manages the asynchronous collection and batching of multi-dimensional
// gas data from blocks. It owns the message channel, buffers data in memory,
// and periodically writes batches to disk in protobuf format.
type Collector struct {
	config CollectorConfig
	in     chan *CollectorMessage

	blockBuffer       []*proto.BlockMultiGasData
	transactionBuffer []*proto.TransactionMultiGasData

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func MultigasCollectionAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".output-dir", DefaultCollectorConfig.OutputDir,
		"If set, enables Multigas collector and stores batches in this directory")
	f.Int(prefix+".batch-size", DefaultCollectorConfig.BatchSize,
		"Batch size (blocks per file) for Multigas collector. Ignored unless output-dir is set")
	f.Bool(prefix+".clear-output-dir", DefaultCollectorConfig.ClearOutputDir,
		"Whether to clear the output directory before starting the collector")
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
func NewCollector(config CollectorConfig) (*Collector, error) {
	if config.OutputDir == "" {
		return nil, ErrOutputDirRequired
	}
	if config.BatchSize <= 0 {
		return nil, ErrBatchSizeRequired
	}

	return &Collector{
		config:            config,
		in:                make(chan *CollectorMessage, CollectorMsgQueueSize),
		blockBuffer:       make([]*proto.BlockMultiGasData, 0, config.BatchSize),
		transactionBuffer: make([]*proto.TransactionMultiGasData, 0, defaultTxPrealloc),
	}, nil
}

// Start prepares the output directory (removes any previous contents)
// and begins background processing of incoming messages,
// should be called only once.
func (c *Collector) Start(parent context.Context) {
	// Reset the output directory, if enabled
	if c.config.ClearOutputDir {
		if err := os.RemoveAll(c.config.OutputDir); err != nil {
			log.Error("Multi-gas collector: failed to clear output dir", "dir", c.config.OutputDir, "err", err)
			return
		}
	}

	// Create the output directory
	if err := os.MkdirAll(c.config.OutputDir, 0o755); err != nil {
		log.Error("Multi-gas collector: failed to create output dir", "dir", c.config.OutputDir, "err", err)
		return
	}

	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel

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
	log.Info("Multi-gas collector stopped")
}

// Submit enqueues a message for collection (non-blocking). Drops on a full queue.
func (c *Collector) Submit(msg *CollectorMessage) {
	select {
	case c.in <- msg:
	default:
		log.Warn("Multi-gas collector dropping message",
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
// File naming pattern: multigas_batch_<start_block_number>_<end_block_number>.pb
func (c *Collector) flushBatch() error {
	batch := &proto.BlockMultiGasBatch{
		Data: make([]*proto.BlockMultiGasData, len(c.blockBuffer)),
	}
	copy(batch.Data, c.blockBuffer)

	data, err := protobuf.Marshal(batch)
	if err != nil {
		return err
	}

	start := c.blockBuffer[0].BlockNumber
	end := c.blockBuffer[len(c.blockBuffer)-1].BlockNumber
	filename := fmt.Sprintf(batchFilenameFormat, start, end)
	outPath := filepath.Join(c.config.OutputDir, filename)

	if err := os.WriteFile(outPath, data, 0600); err != nil {
		return err
	}

	log.Info("Multi-gas collector wrote multi-gas batch",
		"file", filename,
		"count", len(c.blockBuffer),
		"size_bytes", len(data))

	c.blockBuffer = c.blockBuffer[:0]

	return nil
}
