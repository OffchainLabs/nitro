package snapshotter

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

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

	"github.com/offchainlabs/nitro/util/containers"
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

type SnapshotResult struct {
	GenesisHash   common.Hash `json:"genesisHash"`
	GenesisNumber uint64      `json:"genesisNumber"`
	HeadHash      common.Hash `json:"headHash"`
	HeadNumber    uint64      `json:"headNumber"`
}

type snapshotTrigger struct {
	blockHash common.Hash
	promise   *containers.Promise[SnapshotResult]
}

type DatabaseSnapshotter struct {
	stopwaiter.StopWaiter

	config *DatabaseSnapshotterConfig

	db       ethdb.Database
	bc       *core.BlockChain
	exporter BlockChainExporter

	triggerChan chan snapshotTrigger
}

func NewDatabaseSnapshotter(db ethdb.Database, bc *core.BlockChain, config *DatabaseSnapshotterConfig) *DatabaseSnapshotter {
	return &DatabaseSnapshotter{
		config:      config,
		db:          db,
		bc:          bc,
		exporter:    NewGethDatabaseExporter(&config.GethExporter),
		triggerChan: make(chan snapshotTrigger, 1),
	}
}

func (s *DatabaseSnapshotter) Trigger(blockHash common.Hash) containers.PromiseInterface[SnapshotResult] {
	if !s.Started() {
		return containers.NewReadyPromise(SnapshotResult{}, errors.New("not started"))
	}
	promise := containers.NewPromise[SnapshotResult](nil)
	select {
	case s.triggerChan <- snapshotTrigger{blockHash: blockHash, promise: &promise}:
	default:
		promise.ProduceError(errors.New("already scheduled"))
	}
	return &promise
}

func (s *DatabaseSnapshotter) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)

	sigusr2 := make(chan os.Signal, 1)
	signal.Notify(sigusr2, syscall.SIGUSR2)

	s.LaunchThread(func(ctx context.Context) {
		for {
			var trigger snapshotTrigger
			select {
			case <-ctx.Done():
				return
			case <-sigusr2:
				if !s.config.Enable {
					continue
				}
				log.Info("Database snapshot triggered by SIGUSR2")
				trigger = snapshotTrigger{
					blockHash: common.Hash{},
					promise:   nil,
				}
			case trigger = <-s.triggerChan:
				if !s.config.Enable {
					log.Warn("Ignoring database snapshot trigger, snapshotter disabled", "blockHash", trigger.blockHash)
					if trigger.promise != nil {
						trigger.promise.ProduceError(errors.New("database snapshotter is disabled"))
					}
					continue
				}
				log.Info("Database snapshot tiggered", "blockHash", trigger.blockHash)
			}
			// mostly needed for SIGUSR2 case
			if trigger.blockHash == (common.Hash{}) {
				header := s.bc.CurrentHeader()
				if header == nil {
					err := errors.New("aborting snapshot: failed to get current head header")
					log.Error(err.Error())
					if trigger.promise != nil {
						trigger.promise.ProduceError(err)
					}
					continue
				}
				trigger.blockHash = header.Hash()
			}
			log.Info("Creating database snapshot", "blockHash", trigger.blockHash)
			result, err := s.CreateSnapshot(ctx, trigger.blockHash)
			if err != nil {
				log.Error("Database snapshot failed", "err", err)
				if trigger.promise != nil {
					trigger.promise.ProduceError(err)
				}
			} else {
				if trigger.promise != nil {
					trigger.promise.Produce(*result)
				}
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
	log.Info("exporting blocks", "blockFrom", number, "blockTo", lastNumber)
	startedAt := time.Now()
	lastLog := time.Now()
	var hash common.Hash
	for number <= lastNumber {
		hash = rawdb.ReadCanonicalHash(s.db, number)
		if hash == (common.Hash{}) {
			return fmt.Errorf("canonical hash for block %v not found", number)
		}
		if err := batch.ExportCanonicalHash(number, hash); err != nil {
			return fmt.Errorf("failed to export canonical hash: %w", err)
		}
		tdRlp := rawdb.ReadTdRLP(s.db, hash, number)
		if len(tdRlp) == 0 {
			return fmt.Errorf("total difficulty for block %v (hash %v) not found", number, hash)
		}
		if err := batch.ExportTD(number, hash, tdRlp); err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) total difficulty: %w", number, hash, err)
		}
		headerRlp := rawdb.ReadHeaderRLP(s.db, hash, number)
		if len(headerRlp) == 0 {
			return fmt.Errorf("header for block %v (hash %v) not found", number, hash)
		}
		if err := batch.ExportBlockHeader(number, hash, headerRlp); err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) header: %w", number, hash, err)
		}
		bodyRlp := rawdb.ReadBodyRLP(s.db, hash, number)
		if len(bodyRlp) == 0 {
			return fmt.Errorf("body for block %v (hash %v) not found", number, hash)
		}
		if err := batch.ExportBlockBody(number, hash, bodyRlp); err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) body: %w", number, hash, err)
		}
		receiptsRlp := rawdb.ReadReceiptsRLP(s.db, hash, number)
		if len(receiptsRlp) == 0 {
			return fmt.Errorf("receipts for block %v (hash %v) not found", number, hash)
		}
		if err := batch.ExportBlockReceipts(number, hash, receiptsRlp); err != nil {
			return fmt.Errorf("failed to export block %v (hash %v) receipts: %w", number, hash, err)
		}
		if time.Since(lastLog) > time.Minute && number != genesisNumber {
			elapsed := time.Since(startedAt)
			log.Info("exporting blocks", "currentBlock", number, "blockTo", lastNumber, "elapsed", elapsed, "eta", time.Duration(float32(elapsed)*float32(lastNumber-number)/float32(number-genesisNumber)))
			lastLog = time.Now()
		}
		number++
	}
	if err := batch.ExportHead(number, hash); err != nil {
		return fmt.Errorf("failed to export head number %v (hash %v): %w", number, hash, err)
	}
	block0Hash := rawdb.ReadCanonicalHash(s.db, 0)
	if block0Hash == (common.Hash{}) {
		return fmt.Errorf("block 0 canonical hash not found")
	}
	if err := batch.ExportCanonicalHash(0, block0Hash); err != nil {
		return fmt.Errorf("failed to export canonical hash for block 0: %w", err)
	}
	chainConfigJson, err := s.db.Get(rawdb.ConfigKey(block0Hash))
	if err != nil {
		return fmt.Errorf("failed to read stored chain config, block 0 hash %v, err: %w", block0Hash, err)
	}
	if len(chainConfigJson) == 0 {
		return fmt.Errorf("failed to read stored chain config, block 0 hash: %v", block0Hash)
	}
	if err := batch.ExportChainConfig(block0Hash, chainConfigJson); err != nil {
		return fmt.Errorf("failed to export chain config: %w", err)
	}
	log.Info("exported blocks", "blocks", lastNumber-genesisNumber+1, "elapsed", time.Since(startedAt))
	return nil
}

func (s *DatabaseSnapshotter) CreateSnapshot(ctx context.Context, blockHash common.Hash) (*SnapshotResult, error) {
	if s.bc.StateCache().TrieDB().Scheme() != rawdb.HashScheme {
		return nil, errors.New("unsupported state scheme, database snapshotter supports only hash state scheme")
	}
	if err := s.exporter.Open(false); err != nil {
		return nil, fmt.Errorf("failed to open blockchain exporter: %w", err)
	}
	defer func() {
		if s.exporter.IsOpened() {
			if err := s.exporter.Close(false); err != nil {
				log.Error("failed to close blockchain exporter", "err", err)
			}
		}
	}()

	threads := s.config.Threads
	results := make(chan error, threads)
	for i := 0; i < threads; i++ {
		results <- nil
	}
	batchPool := make(chan BlockChainExporterBatch, threads)
	for i := 0; i < threads; i++ {
		batch, err := s.exporter.NewBatch()
		if err != nil {
			return nil, fmt.Errorf("failed to create new blockchain exporter batch: %w", err)
		}
		batchPool <- batch
	}
	startWorker := func(work func(batch BlockChainExporterBatch) error) error {
		err := <-results
		if err != nil {
			return err
		}
		batch := <-batchPool
		go func() {
			defer func() {
				batchPool <- batch
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
		return nil, err
	}
	workersCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	header := lastHeader
	log.Info("starting blocks export worker", "blockNumber", header.Number.Uint64(), "blockHash", header.Hash())
	err = startWorker(func(batch BlockChainExporterBatch) error {
		return s.exportBlocks(workersCtx, batch, header)
	})
	if err != nil {
		return nil, err
	}
	genesisNumber := s.bc.Config().ArbitrumChainParams.GenesisBlockNum
	genesisHeader := s.bc.GetHeaderByNumber(genesisNumber)
	if genesisHeader == nil {
		return nil, errors.New("genesis header not found")
	}
	genesisHash := genesisHeader.Hash()
	log.Info("exporting genesis state", "genesisNumber", genesisNumber, "genesisHash", genesisHash)
	if err = s.exportState(workersCtx, startWorker, triedb, genesisHeader.Root); err != nil {
		return nil, err
	}
	log.Info("exporting last state", "blockNumber", lastHeader.Number.Uint64(), "blockHash", lastHeader.Hash())
	if err = s.exportState(workersCtx, startWorker, triedb, lastHeader.Root); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for i := 0; i < threads; i++ {
		err = <-results
		if err != nil {
			return nil, err
		}
	}
	// flush all batches from batchPool
	for i := 0; i < threads; i++ {
		batch := <-batchPool
		if err = batch.Flush(); err != nil {
			return nil, err
		}
	}
	if err := s.exporter.Close(true); err != nil {
		return nil, err
	}
	return &SnapshotResult{
		GenesisNumber: genesisNumber,
		GenesisHash:   genesisHash,
		HeadHash:      lastHeader.Hash(),
		HeadNumber:    lastHeader.Number.Uint64(),
	}, nil
}

func (s *DatabaseSnapshotter) exportState(ctx context.Context, startWorker func(work func(batch BlockChainExporterBatch) error) error, triedb *triedb.Database, root common.Hash) error {
	var threadsRunning atomic.Int32
	startedAt := time.Now()
	lastLog := time.Now()
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
				threadsRunning.Add(1)
				defer threadsRunning.Add(-1)
				// get trie node directly from triedb
				blob, err := triedb.Node(hash)
				if err != nil {
					return err
				}
				if err := batch.ExportAccountTrieNode(hash, blob); err != nil {
					return err
				}
				return nil
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
			if time.Since(lastLog) >= time.Second*30 {
				lastLog = time.Now()
				progress := 256 * 256 / float32(binary.BigEndian.Uint16(key.Bytes()[:2]))
				elapsed := time.Since(startedAt)
				log.Info("exporting trie database", "accountKey", key, "elapsed", elapsed, "eta", time.Duration(float32(elapsed)*progress)-elapsed)
			}
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
					threadsRunning.Add(1)
					defer threadsRunning.Add(-1)
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
				for i := int64(0); i < int64(32) && ctx.Err() == nil; i++ {
					// note: we are passing data.Root as stateRoot here, to skip the check for stateRoot existence in trie.newTrieReader,
					// we already check that when opening state trie and reading the account node
					trieID := trie.StorageTrieID(data.Root, key, data.Root)
					// StateTrie is not safe for concurrent use, so we open new one for each thread
					storageTr, err := trie.NewStateTrie(trieID, triedb)
					if err != nil {
						return err
					}
					startKey := big.NewInt(i << 3).Bytes()
					storageIt, err := storageTr.NodeIterator(startKey)
					if err != nil {
						return err
					}
					startPath := trie.KeybytesToHex(startKey)
					startPath = startPath[:len(startPath)-1] // remove terminator byte
					endKey := big.NewInt((i + 1) << 3).Bytes()
					endPath := trie.KeybytesToHex(endKey) // by including terminator byte we make sure that we exhaust all nodes with the endKey prefix
					isLastKeyRange := i == 31
					err = startWorker(func(batch BlockChainExporterBatch) error {
						threadsRunning.Add(1)
						defer threadsRunning.Add(-1)
						threadStartedAt := time.Now()
						threadLastLog := time.Now()
						var threadExportedNodes uint64
						var threadExportedNodeBlobBytes uint64
						var firstPath, lastPath []byte
						for storageIt.Next(true) && ctx.Err() == nil {
							if !isLastKeyRange && bytes.Compare(storageIt.Path(), endPath) > 0 {
								break
							}
							if firstPath == nil {
								firstPath = storageIt.Path()
							}
							lastPath = storageIt.Path()
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
								threadExportedNodeBlobBytes += uint64(len(blob))
								threadExportedNodes++
							}
							if storageIt.Leaf() {
								if time.Since(threadLastLog) > 5*time.Minute {
									elapsedTotal := time.Since(startedAt)
									elapsedThread := time.Since(threadStartedAt)
									start := binary.BigEndian.Uint16(common.BytesToHash(startKey).Bytes()[:2])
									current := binary.BigEndian.Uint16(common.BytesToHash(storageIt.LeafKey()).Bytes()[:2])
									threadProgress := float32(current-start) / float32(1<<16)
									var threadEta time.Duration
									if threadProgress == 0 {
										threadEta = time.Duration(-1)
									} else {
										threadEta = time.Duration(float32(elapsedThread)/threadProgress - float32(elapsedThread))
									}
									log.Info("exporting trie database - exporting storage trie taking long", "key", key, "elapsedTotal", elapsedTotal, "elapsedThread", elapsedThread, "threadExportedNodes", threadExportedNodes, "threadExportedNodeBlobBytes", threadExportedNodeBlobBytes, "threads", threadsRunning.Load(), "threadProgress", threadProgress, "threadEta", threadEta)
									threadLastLog = time.Now()
								}
							}
						}

						if err := storageIt.Error(); err != nil {
							return err
						}
						log.Trace("Exporting database - storage traversal worker finished", "owner", key, "startPath", startPath, "endPath", endPath, "firstPath", firstPath, "lastPath", lastPath)
						return nil
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
	if err := ctx.Err(); err != nil {
		return err
	}
	// TODO: calculate and log exported size / number of nodes
	log.Info("exported state", "root", root, "elapsed", time.Since(startedAt))
	return nil
}
