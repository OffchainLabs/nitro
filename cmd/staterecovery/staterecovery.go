package staterecovery

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
	"github.com/ethereum/go-ethereum/trie/triedb/pathdb"
)

func RecreateMissingStates(chainDb ethdb.Database, bc *core.BlockChain, cacheConfig *core.CacheConfig, startBlock uint64) error {
	start := time.Now()
	currentHeader := bc.CurrentBlock()
	if currentHeader == nil {
		return fmt.Errorf("current header is nil")
	}
	target := currentHeader.Number.Uint64()
	current := startBlock
	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum
	if current < genesis+1 {
		current = genesis + 1
		log.Warn("recreate-missing-states-from before genesis+1, starting from genesis+1", "configured", startBlock, "override", current)
	}
	previousBlock := bc.GetBlockByNumber(current - 1)
	if previousBlock == nil {
		return fmt.Errorf("start block parent is missing, parent block number: %d", current-1)
	}

	trieConfig := &trie.Config{
		Preimages: false,
	}
	if cacheConfig.StateScheme == rawdb.PathScheme {
		trieConfig.PathDB = pathdb.Defaults
	} else {
		hashConfig := *hashdb.Defaults
		hashConfig.CleanCacheSize = cacheConfig.TrieCleanLimit * 1024 * 1024
		trieConfig.HashDB = &hashConfig
	}

	database := state.NewDatabaseWithConfig(chainDb, trieConfig)
	defer database.TrieDB().Close()
	previousState, err := state.New(previousBlock.Root(), database, nil)
	if err != nil {
		return fmt.Errorf("state of start block parent is missing: %w", err)
	}
	// we don't need to reference states with `trie.Database.Reference` here, because:
	// * either the state nodes will be read from disk and then cached in cleans cache
	// * or they will be recreated, saved to disk and then also cached in cleans cache
	logged := time.Unix(0, 0)
	recreated := 0
	for {
		currentBlock := bc.GetBlockByNumber(current)
		if currentBlock == nil {
			break
		}
		if time.Since(logged) > 1*time.Minute {
			log.Info("Recreating missing states", "block", current, "target", target, "remaining", int64(target)-int64(current), "elapsed", time.Since(start), "recreated", recreated)
			logged = time.Now()
		}
		currentState, err := state.New(currentBlock.Root(), database, nil)
		if err != nil {
			_, _, _, err := bc.Processor().Process(currentBlock, previousState, vm.Config{})
			if err != nil {
				return fmt.Errorf("processing block %d failed: %w", current, err)
			}
			root, err := previousState.Commit(current, bc.Config().IsEIP158(currentBlock.Number()))
			if err != nil {
				return fmt.Errorf("StateDB commit failed, number %d root %v: %w", current, currentBlock.Root(), err)
			}
			if root.Cmp(currentBlock.Root()) != 0 {
				return fmt.Errorf("reached different state root after processing block %d, have %v, want %v", current, root, currentBlock.Root())
			}
			// commit to disk
			err = database.TrieDB().Commit(root, false)
			if err != nil {
				return fmt.Errorf("TrieDB commit failed, number %d root %v: %w", current, root, err)
			}
			currentState, err = state.New(currentBlock.Root(), database, nil)
			if err != nil {
				return fmt.Errorf("state reset after block %d failed: %w", current, err)
			}
			recreated++
		}
		current++
		previousState = currentState
	}
	log.Info("Finished recreating missing states", "elapsed", time.Since(start), "recreated", recreated)
	return nil
}
