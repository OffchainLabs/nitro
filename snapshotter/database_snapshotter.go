package snapshotter

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type DatabaseSnapshotterConfig struct {
	TrieCleanLimit int `koanf:"trie-clean-limit"`
	Threads        int `koanf:"threads"`
}

type DatabaseSnapshotter struct {
	stopwaiter.StopWaiter

	config *DatabaseSnapshotterConfig

	db       ethdb.Database
	bc       *core.BlockChain
	exporter BlockChainExporter

	snapshotTrigger chan struct{}
}

func CreateDatabaseSnapshotter(db ethdb.Database, bc *core.BlockChain, config *DatabaseSnapshotterConfig) *DatabaseSnapshotter {
	return &DatabaseSnapshotter{
		config: config,
		db:     db,
		bc:     bc,
	}
}

func (s *DatabaseSnapshotter) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)
	// TODO
}

func (s *DatabaseSnapshotter) findLastAvailableStateRoot(ctx context.Context, triedb *triedb.Database, header *types.Header) (common.Hash, error) {
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
		return common.Hash{}, fmt.Errorf("failed to find latest available state not newer then for block %v: %w", header.Number.Uint64(), err)
	}
	return lastHeader.Root, nil
}

func (s *DatabaseSnapshotter) CreateSnapshot(ctx context.Context, header *types.Header) error {
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
	startWorker := func(work func(batch BlockChainExporterBatch) error) {
		batch := <-batchPool
		workersRunning.Add(1)
		go func() {
			defer func() {
				batchPool <- batch
				workersRunning.Add(-1)
			}()
			results <- work(batch)
		}()
	}
	hashConfig := *hashdb.Defaults
	hashConfig.CleanCacheSize = s.config.TrieCleanLimit * 1024 * 1024
	trieConfig := &triedb.Config{
		Preimages: false,
		HashDB:    &hashConfig,
	}
	triedb := triedb.NewDatabase(s.db, trieConfig)
	root, err := s.findLastAvailableStateRoot(ctx, triedb, header)
	if err != nil {
		return err
	}
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
			err := <-results
			if err != nil {
				return err
			}
			hash := accountTrieHash
			startWorker(func(batch BlockChainExporterBatch) error {
				// get trie node directly from triedb
				blob, err := triedb.Node(hash)
				if err != nil {
					return err
				}
				return batch.ExportAccountTrieNode(hash, blob)
			})
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
				err = <-results
				if err != nil {
					return err
				}
				codeHash := common.BytesToHash(data.CodeHash)
				startWorker(func(batch BlockChainExporterBatch) error {
					code := rawdb.ReadCode(s.db, codeHash)
					if len(code) == 0 {
						return fmt.Errorf("code not found, hash: %v", codeHash)
					}
					return batch.ExportCode(codeHash, code)
				})
			}
			if data.Root != (common.Hash{}) {
				trieID := trie.StorageTrieID(data.Root, key, data.Root)
				storageTr, err := trie.NewStateTrie(trieID, triedb)
				if err != nil {
					return err
				}
				for i := int64(0); i < int64(32) && ctx.Err() == nil; i++ {
					err = <-results
					if err != nil {
						return err
					}
					storageIt, err := storageTr.NodeIterator(big.NewInt(i << 3).Bytes())
					if err != nil {
						return err
					}
					endKey := trie.KeybytesToHex(big.NewInt((i + 1) << 3).Bytes())
					isLastKeyRange := i == 32
					startWorker(func(batch BlockChainExporterBatch) error {
						for storageIt.Next(true) && ctx.Err() == nil {
							if !isLastKeyRange && bytes.Compare(storageIt.Path(), endKey) >= 0 {
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
				}
			}
		}
		if accountIt.Error() != nil {
			return accountIt.Error()
		}
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
