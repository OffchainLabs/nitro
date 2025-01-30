package snapshotter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type DatabaseSnapshotterConfig struct {
	Enable         bool                       `koanf:"enable"`
	TrieCleanLimit int                        `koanf:"trie-clean-limit"`
	Threads        int                        `koanf:"threads"`
	GethExporter   GethDatabaseExporterConfig `koanf:"geth-exporter"`
}

var DatabaseSnapshotterConfigDefault = DatabaseSnapshotterConfig{
	Enable:       false,
	Threads:      runtime.NumCPU(),
	GethExporter: GethDatabaseExporterConfigDefault,
}

func DatabaseSnapshotterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DatabaseSnapshotterConfigDefault.Enable, "enables database snapshotter")
	f.Int(prefix+".threads", DatabaseSnapshotterConfigDefault.Threads, "the number of threads to use when traversing the tires")
	GethDatabaseExporterConfigAddOptions(prefix+".geth-exporter", f)
}

type DatabaseSnapshotter struct {
	stopwaiter.StopWaiter

	config *DatabaseSnapshotterConfig

	db       ethdb.Database
	bc       *core.BlockChain
	exporter BlockChainExporter

	triggerChan chan common.Hash
	resultChan  chan error
}

func NewDatabaseSnapshotter(db ethdb.Database, bc *core.BlockChain, config *DatabaseSnapshotterConfig, triggerChan chan common.Hash, resultChan chan error) *DatabaseSnapshotter {
	return &DatabaseSnapshotter{
		config:      config,
		db:          db,
		bc:          bc,
		exporter:    NewGethDatabaseExporter(&config.GethExporter),
		triggerChan: triggerChan,
		resultChan:  resultChan,
	}
}

func (s *DatabaseSnapshotter) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)

	sigusr2 := make(chan os.Signal, 1)
	signal.Notify(sigusr2, syscall.SIGUSR2)

	s.LaunchThread(func(ctx context.Context) {
		for {
			var blockHash common.Hash
			select {
			case <-ctx.Done():
				return
			case <-sigusr2:
				if !s.config.Enable {
					continue
				}
				log.Info("Database snapshot triggered by SIGUSR2")
				blockHash = common.Hash{}
			case blockHash = <-s.triggerChan:
				if !s.config.Enable {
					log.Warn("Ignoring database snapshot trigger, snapshotter disabled", "blockHash", blockHash)
					continue
				}
				log.Info("Database snapshot tiggered", "blockHash", blockHash)
			}
			if blockHash == (common.Hash{}) {
				header := s.bc.CurrentHeader()
				if header == nil {
					log.Error("Aborting snapshot: failed to get current head header.")
					continue
				}
				blockHash = header.Hash()
			}
			log.Info("Creating database snapshot", "blockHash", blockHash)
			err := s.CreateSnapshot(ctx, blockHash)
			if err != nil {
				log.Error("Database snapshot failed", "err", err)
			}
			if s.resultChan != nil {
				s.resultChan <- err
			}
		}
	})
}

func (s *DatabaseSnapshotter) findLastAvailableState(ctx context.Context, triedb *triedb.Database, blockHash common.Hash) (*types.Header, error) {
	header := s.bc.GetHeaderByHash(blockHash)
	if header == nil {
		return nil, fmt.Errorf("header not found for block hash: %v", blockHash)
	}
	stateDatabase := state.NewDatabaseWithNodeDB(s.db, triedb)
	stateFor := func(header *types.Header) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
		statedb, err := state.New(header.Root, stateDatabase, nil)
		if err != nil {
			return nil, nil, err
		}
		// we don't need to reference the state root, as we opened triedb from disk and the triedb dirties cache is empty
		return statedb, arbitrum.NoopStateRelease, nil
	}
	_, lastHeader, _, err := arbitrum.FindLastAvailableState(ctx, s.bc, stateFor, header, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to find latest available state not newer then for block %v (hash %v): %w", header.Number.Uint64(), header.Hash(), err)
	}
	return lastHeader, nil
}

func (s *DatabaseSnapshotter) exportBlocks(ctx context.Context, batch BlockChainExporterBatch, lastHeader *types.Header) error {
	genesisNumber := s.bc.Config().ArbitrumChainParams.GenesisBlockNum
	lastNumber := lastHeader.Number.Uint64()
	if lastNumber < genesisNumber {
		return fmt.Errorf("failed to export blocks: last block (number %v) older then genesis (number %v)", lastNumber, genesisNumber)
	}

	number := genesisNumber
	var hash common.Hash
	for number <= lastNumber {
		hash = rawdb.ReadCanonicalHash(s.db, number)
		if hash == (common.Hash{}) {
			return fmt.Errorf("canonical hash for block %v not found", number)
		}
		tdRlp := rawdb.ReadTdRLP(s.db, hash, number)
		if len(tdRlp) == 0 {
			return fmt.Errorf("total difficulty for block %v (hash %v) not found", number, hash)
		}
		err := batch.ExportTD(number, hash, tdRlp)
		if err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) total difficulty: %w", number, hash, err)
		}
		err = batch.ExportCanonicalHash(number, hash)
		if err != nil {
			return fmt.Errorf("failed to export canonical hash: %w", err)
		}
		headerRlp := rawdb.ReadHeaderRLP(s.db, hash, number)
		if len(headerRlp) == 0 {
			return fmt.Errorf("header for block %v (hash %v) not found", number, hash)
		}
		err = batch.ExportBlockHeader(number, hash, headerRlp)
		if err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) header: %w", number, hash, err)
		}
		bodyRlp := rawdb.ReadBodyRLP(s.db, hash, number)
		if len(bodyRlp) == 0 {
			return fmt.Errorf("body for block %v (hash %v) not found", number, hash)
		}
		err = batch.ExportBlockBody(number, hash, bodyRlp)
		if err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) body: %w", number, hash, err)
		}
		receiptsRlp := rawdb.ReadReceiptsRLP(s.db, hash, number)
		if len(receiptsRlp) == 0 {
			return fmt.Errorf("receipts for block %v (hash %v) not found", number, hash)
		}
		err = batch.ExportBlockReceipts(number, hash, receiptsRlp)
		if err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) receipts: %w", number, hash, err)
		}
		number++
	}
	err := batch.ExportHead(number, hash)
	if err != nil {
		return fmt.Errorf("failed to export head number %v (hash %v): %w", number, hash, err)
	}
	return nil
}

func (s *DatabaseSnapshotter) CreateSnapshot(ctx context.Context, blockHash common.Hash) error {
	if err := s.exporter.Open(); err != nil {
		return fmt.Errorf("failed to open blockchain exporter: %w", err)
	}
	threads := s.config.Threads
	results := make(chan error, threads)
	for i := 0; i < threads; i++ {
		results <- nil
	}
	batchPool := make(chan BlockChainExporterBatch, threads)
	for i := 0; i < threads; i++ {
		batch, err := s.exporter.NewBatch()
		if err != nil {
			return fmt.Errorf("Failed to create new blockchain exporter batch: %w", err)
		}
		batchPool <- batch
	}
	var workersRunning atomic.Int32
	startWorker := func(work func(batch BlockChainExporterBatch) error) error {
		err := <-results
		if err != nil {
			return err
		}
		batch := <-batchPool
		workersRunning.Add(1)
		go func() {
			defer func() {
				batchPool <- batch
				workersRunning.Add(-1)
			}()
			results <- work(batch)
		}()
		return nil
	}
	hashConfig := *hashdb.Defaults
	hashConfig.CleanCacheSize = s.config.TrieCleanLimit * 1024 * 1024
	trieConfig := &triedb.Config{
		Preimages: false,
		HashDB:    &hashConfig,
	}
	triedb := triedb.NewDatabase(s.db, trieConfig)
	defer triedb.Close()
	lastHeader, err := s.findLastAvailableState(ctx, triedb, blockHash)
	if err != nil {
		return err
	}
	workersCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	header := lastHeader
	log.Info("Starting blocks export worker", "blockNumber", header.Number.Uint64(), "blockHash", header.Hash())
	err = startWorker(func(batch BlockChainExporterBatch) error {
		return s.exportBlocks(workersCtx, batch, header)
	})
	if err != nil {
		return err
	}
	genesisNumber := s.bc.Config().ArbitrumChainParams.GenesisBlockNum
	genesisHeader := s.bc.GetHeaderByNumber(genesisNumber)
	if genesisHeader == nil {
		return errors.New("genesis header not found")
	}
	log.Info("Exporting genesis state", "genesisNumber", genesisHeader.Number.Uint64(), "genesisHash", genesisHeader.Hash())
	if err = s.exportState(workersCtx, startWorker, triedb, genesisHeader.Root); err != nil {
		return err
	}
	log.Info("Exporting last state", "blockNumber", header.Number.Uint64(), "blockHash", header.Hash())
	if err = s.exportState(workersCtx, startWorker, triedb, lastHeader.Root); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for i := 0; i < threads; i++ {
		err = <-results
		if err != nil {
			return err
		}
	}
	// flush all batches from batchPool
	for i := 0; i < threads; i++ {
		batch := <-batchPool
		if err = batch.Flush(); err != nil {
			return err
		}
	}
	return s.exporter.Close()
}

func (s *DatabaseSnapshotter) exportState(ctx context.Context, startWorker func(work func(batch BlockChainExporterBatch) error) error, triedb *triedb.Database, root common.Hash) error {
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), triedb)
	if err != nil {
		return fmt.Errorf("failed to open state trie, root: %v, err: %w", root, err)
	}
	accountIt, err := tr.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to create account iterator: %w", err)
	}
	for accountIt.Next(true) && ctx.Err() == nil {
		accountTrieHash := accountIt.Hash()
		// If the iterator hash is the empty hash, this is an embedded node
		if accountTrieHash != (common.Hash{}) {
			hash := accountTrieHash
			err := startWorker(func(batch BlockChainExporterBatch) error {
				// get trie node directly from triedb
				blob, err := triedb.Node(hash)
				if err != nil {
					return err
				}
				return batch.ExportAccountTrieNode(hash, blob)
			})
			if err != nil {
				return err
			}
		}
		if accountIt.Leaf() {
			keyBytes := accountIt.LeafKey()
			if len(keyBytes) != len(common.Hash{}) {
				return fmt.Errorf("unexpected account trie leaf key length: %v", len(keyBytes))
			}
			key := common.BytesToHash(keyBytes)
			var data types.StateAccount
			if err := rlp.DecodeBytes(accountIt.LeafBlob(), &data); err != nil {
				return fmt.Errorf("failed to decode account data: %w", err)
			}
			if !bytes.Equal(data.CodeHash, types.EmptyCodeHash[:]) {
				if len(data.CodeHash) != len(common.Hash{}) {
					return fmt.Errorf("unexpected code hash length: %v", len(keyBytes))
				}
				codeHash := common.BytesToHash(data.CodeHash)
				err := startWorker(func(batch BlockChainExporterBatch) error {
					code := rawdb.ReadCode(s.db, codeHash)
					if len(code) == 0 {
						return fmt.Errorf("code not found, hash: %v", codeHash)
					}
					return batch.ExportCode(codeHash, code)
				})
				if err != nil {
					return err
				}
			}
			if data.Root != (common.Hash{}) {
				trieID := trie.StorageTrieID(data.Root, key, data.Root)
				storageTr, err := trie.NewStateTrie(trieID, triedb)
				if err != nil {
					return err
				}
				for i := int64(0); i < int64(32) && ctx.Err() == nil; i++ {
					storageIt, err := storageTr.NodeIterator(big.NewInt(i << 3).Bytes())
					if err != nil {
						return err
					}
					endKey := trie.KeybytesToHex(big.NewInt((i + 1) << 3).Bytes())
					isLastKeyRange := i == 31
					err = startWorker(func(batch BlockChainExporterBatch) error {
						for storageIt.Next(true) && ctx.Err() == nil {
							if !isLastKeyRange && bytes.Compare(storageIt.Path(), endKey) > 0 {
								return nil
							}
							storageTrieHash := storageIt.Hash()
							if storageTrieHash != (common.Hash{}) {
								// get trie node directly from triedb
								blob, err := triedb.Node(storageTrieHash)
								if err != nil {
									return err
								}
								if err := batch.ExportStorageTrieNode(storageTrieHash, blob); err != nil {
									return err
								}
							}
						}
						return storageIt.Error()
					})
					if err != nil {
						return err
					}
				}
			}
		}
		if accountIt.Error() != nil {
			return accountIt.Error()
		}
	}
	return ctx.Err()
}
