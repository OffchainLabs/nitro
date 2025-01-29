package snapshotter

import (
	"errors"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/util/dbutil"
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
	ExportCanonicalHash(number uint64, hash common.Hash) error

	ExportTD(number uint64, hash common.Hash, tdRlp []byte) error
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
	Data:      "snapshot", // TODO
	DBEngine:  conf.PersistentConfigDefault.DBEngine,
	Ancient:   "", // TODO
	Handles:   conf.PersistentConfigDefault.Handles,
	Cache:     2048, // 2048 MB
	Namespace: "l2chaindata_export",
	Pebble:    conf.PebbleConfigDefault,
}

var GethDatabaseExporterConfigDefault = GethDatabaseExporterConfig{
	Output:         OutputConfigDefault,
	IdealBatchSize: 100 * 1024 * 1024, // 100 MB, TODO: figure out reasonable default, 100MB is used by dbconv, 100k is used by geth
}

func GethDatabaseExporterConfigAddOptions(prefix string, f *flag.FlagSet) {
	conf.DBConfigAddOptions("output", f, &GethDatabaseExporterConfigDefault.Output)
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
	if e.opened {
		return errors.New("already opened")
	}
	db, err := rawdb.Open(rawdb.OpenOptions{
		Type:               e.config.Output.DBEngine,
		Directory:          e.config.Output.Data,
		AncientsDirectory:  e.config.Output.Ancient,
		Namespace:          e.config.Output.Namespace,
		Cache:              e.config.Output.Cache,
		Handles:            e.config.Output.Handles,
		ReadOnly:           false,
		PebbleExtraOptions: e.config.Output.Pebble.ExtraOptions(e.config.Output.Namespace),
	})
	if err != nil {
		return err
	}
	if err := dbutil.UnfinishedConversionCheck(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return err
	}
	e.db = db
	e.opened = true // TODO: can we just check e.db == nil instead of !e.opened
	return nil
}

func (e *GethDatabaseExporter) Close() error {
	if err := e.db.Close(); err != nil {
		return err
	}
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
	rawdb.WriteHeadBlockHash(b.batch, hash)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportCanonicalHash(number uint64, hash common.Hash) error {
	rawdb.WriteCanonicalHash(b.batch, hash, number)
	return b.maybeFlush()
}

func (b *GethDatabaseExporterBatch) ExportTD(number uint64, hash common.Hash, tdRlp []byte) error {
	if err := b.batch.Put(rawdb.HeaderTDKey(number, hash), tdRlp); err != nil {
		return fmt.Errorf("failed to export block difficulty: %w", err)
	}
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
