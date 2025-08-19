package multigascollector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	protobuf "google.golang.org/protobuf/proto"

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

// MessageType defines the message type processed by the file collector.
type MessageType int

const (
	// MsgStartBlock indicates the start of a new block without metadata.
	MsgStartBlock MessageType = iota
	// MsgTransaction indicates a multi-gas data record for a transaction.
	MsgTransaction
	// MsgFinaliseBlock indicates finalisation of a block with metadata.
	MsgFinaliseBlock
)

// Message is passed internally on the FileCollector's channel.
type Message struct {
	Type        MessageType
	Block       *BlockInfo
	Transaction *TransactionMultiGas
}

// FileCollector manages the asynchronous collection and batching of multi-dimensional
// gas data from blocks. It owns the message channel, buffers data in memory,
// and periodically writes batches to disk in protobuf format.
type FileCollector struct {
	config CollectorConfig
	in     chan *Message

	currentBlockNum   uint64
	blockBuffer       []*proto.BlockMultiGasData
	transactionBuffer []*proto.TransactionMultiGasData

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewFileCollector validates the configuration and returns an initialized collector.
func NewFileCollector(config CollectorConfig) (*FileCollector, error) {
	if config.OutputDir == "" {
		return nil, ErrOutputDirRequired
	}
	if config.BatchSize <= 0 {
		return nil, ErrBatchSizeRequired
	}

	return &FileCollector{
		config:            config,
		in:                make(chan *Message, CollectorMsgQueueSize),
		blockBuffer:       make([]*proto.BlockMultiGasData, 0, config.BatchSize),
		transactionBuffer: make([]*proto.TransactionMultiGasData, 0, defaultTxPrealloc),
	}, nil
}

// Start prepares the output directory (removes any previous contents)
// and begins background processing of incoming messages,
// should be called only once.
func (c *FileCollector) Start(parent context.Context) {
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
func (c *FileCollector) StopAndWait() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	log.Info("Multi-gas collector stopped")
}

// StartBlock signals the beginning of a new block.
func (c *FileCollector) StartBlock(blockNum uint64) {
	c.Submit(&Message{
		Type: MsgStartBlock,
		Block: &BlockInfo{
			BlockNumber: blockNum,
		},
	})
}

// AddTransaction records multi-gas data for a transaction.
func (c *FileCollector) AddTransaction(tx TransactionMultiGas) {
	c.Submit(&Message{
		Type:        MsgTransaction,
		Transaction: &tx,
	})
}

// FinaliseBlock finalises the current block with metadata and flushes buffered txs.
func (c *FileCollector) FinaliseBlock(info BlockInfo) {
	c.Submit(&Message{
		Type:  MsgFinaliseBlock,
		Block: &info,
	})
}

// Submit enqueues a message for collection (non-blocking). Drops on a full queue.
func (c *FileCollector) Submit(msg *Message) {
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
func (c *FileCollector) processData(ctx context.Context) {
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
func (c *FileCollector) handleMessage(msg *Message) {
	switch msg.Type {
	case MsgStartBlock:
		if msg.Block == nil {
			log.Error("Multi-gas collector start block message missing payload")
			return
		}
		c.currentBlockNum = msg.Block.BlockNumber

		// If c.transactionBuffer contains unflushed transactions, block was not finalised (stay silent)
		if len(c.transactionBuffer) > 0 {
			c.transactionBuffer = c.transactionBuffer[:0]
		}

	case MsgTransaction:
		if msg.Transaction == nil {
			log.Error("Multi-gas collector transaction message missing payload")
			return
		}
		c.transactionBuffer = append(c.transactionBuffer, msg.Transaction.ToProto())

	case MsgFinaliseBlock:
		if msg.Block == nil {
			log.Error("Multi-gas collector finalise block message missing payload")
			return
		}

		if c.currentBlockNum > 0 && c.currentBlockNum != msg.Block.BlockNumber {
			log.Error("Multi-gas collector: finalising block does not match current block",
				"expected", c.currentBlockNum, "got", msg.Block.BlockNumber)
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
		c.currentBlockNum = 0

	default:
		log.Error("Multi-gas collector, unknown message type", "type", msg.Type)
	}
}

// finalise drains channel, warns and cleans transaction buffer and finally flushes the block buffer
func (c *FileCollector) finalise() {
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
func (c *FileCollector) flushBatch() error {
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
