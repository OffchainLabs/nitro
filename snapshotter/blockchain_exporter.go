package snapshotter

import (
	"errors"
	"flag"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/cmd/conf"
)

type BlockChainExporter interface {
	Open() error
	NewBatch() (BlockChainExporterBatch, error)
	Close() error
}

// the batch automatically writes internal batches
// remember to call Flush before discarding the batch
type BlockChainExporterBatch interface {
	// exports head block number and hash
	ExportHead(number uint64, hash common.Hash) error

	ExportBlockHeader(number uint64, hash common.Hash, headerRlp []byte) error
	ExportBlockBody(number uint64, hash common.Hash, bodyRlp []byte) error
	ExportBlockReceipts(number uint64, hash common.Hash, receiptsRlp []byte) error

	ExportAccountTrieNode(hash common.Hash, nodeBlob []byte) error
	ExportStorageTrieNode(hash common.Hash, nodeBlob []byte) error
	ExportCode(hash common.Hash, code []byte) error

	// flushes any remaining data not yet flushed automatically
	Flush() error
}

type GethDatabaseExporterConfig struct {
	Output         conf.DBConfig `koanf:"output"`
	IdealBatchSize int           `koanf:"ideal-batch-size"`
}

var OutputConfigDefault = conf.DBConfig{
	DBEngine:  "pebble",
	Ancient:   "",
	Handles:   conf.PersistentConfigDefault.Handles,
	Cache:     2048, // 2048 MB
	Namespace: "l2chaindata_export",
	Pebble:    conf.PebbleConfigDefault,
}

var GethDatabaseExporterConfigDefault = conf.GethDatabaseExporter{
	Output:         OutputConfigDefault,
	IdealBatchSize: 100 * 1024 * 1024, // 100 MB, TODO: figure out reasonable default, 100MB is used by dbconv, 100k is used by geth
}

func GethDatabaseExporterConfigAddOptions(f *flag.FlagSet) {
	conf.DBConfigAddOptions("output", f, &DefaultGethDatabaseExporterConfig.Output)
}

// GethDatabaseExporter is not thread safe
type GethDatabaseExporter struct {
	config *GethDatabaseExporterConfig

	opened  bool
	db      ethdb.Database
	batches []ethdb.Batch
}

func NewGethDatabaseExporter(config *GethDatabaseExporterConfig) *GethDatabaseExporter {
	return &GethDatabaseExporter{
		config: config,
	}
}

func (e *GethDatabaseExporter) Open() error {
	// TODO open e.db
	return nil
}

func (e *GethDatabaseExporter) Close() error {
	// TODO close e.db
	e.opened = false
	return nil
}

func (e *GethDatabaseExporter) NewBatch() (BlockChainExporterBatch, error) {
	if !e.opened {
		return nil, errors.New("not opened")
	}
	batch := e.db.NewBatch()
	e.batches = append(e.batches, batch)
	return &GethDatabaseExporterBatch{
		batch:          batch,
		idealBatchSize: e.config.IdealBatchSize,
	}, nil
}

type GethDatabaseExporterBatch struct {
	batch          ethdb.Batch
	idealBatchSize int
}

func (b *GethDatabaseExporterBatch) ExportHead(number uint64, hash common.Hash) error {
	rawdb.WriteHeadHeaderHash(b.batch, hash)
	rawdb.WriteHeadFastBlockHash(b.batch, hash)
	rawdb.WriteCanonicalHash(b.batch, hash, number)
	rawdb.WriteHeadBlockHash(b.batch, hash)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportBlockHeader(number uint64, hash common.Hash, headerRlp []byte) error {
	rawdb.WriteHeaderNumber(b.batch, hash, number)
	if err := b.batch.Put(rawdb.HeaderKey(number, hash), headerRlp); err != nil {
		return fmt.Errorf("failed to export block header: %w", err)
	}
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportBlockBody(number uint64, hash common.Hash, bodyRlp []byte) error {
	rawdb.WriteBodyRLP(b.batch, hash, number, bodyRlp)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportBlockReceipts(number uint64, hash common.Hash, receiptsRlp []byte) error {
	if err := b.batch.Put(rawdb.BlockReceiptsKey(number, hash), receiptsRlp); err != nil {
		return fmt.Errorf("failed to export block header: %w", err)
	}
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportAccountTrieNode(hash common.Hash, nodeBlob []byte) error {
	return b.exportTrieNode(hash, nodeBlob)
}

func (b *GethDatabaseExporterBatch) ExportStorageTrieNode(hash common.Hash, nodeBlob []byte) error {
	return b.exportTrieNode(hash, nodeBlob)
}

func (b *GethDatabaseExporterBatch) exportTrieNode(hash common.Hash, nodeBlob []byte) error {
	rawdb.WriteLegacyTrieNode(b.batch, hash, nodeBlob)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportCode(hash common.Hash, code []byte) error {
	rawdb.WriteCode(b.batch, hash, code)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) maybeFlush() error {
	if b.batch.ValueSize() >= b.idealBatchSize {
		if err := b.batch.Write(); err != nil {
			return fmt.Errorf("failed to auto-flush geth db export batch: %w", err)
		}
		b.batch.Reset()
	}
	return nil
}

func (b *GethDatabaseExporterBatch) Flush() error {
	if b.batch.ValueSize() > 0 {
		if err := b.batch.Write(); err != nil {
			return fmt.Errorf("failed to flush geth db export batch: %w", err)
		}
		b.batch.Reset()
	}
	return nil
}
