package staterecovery

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
)

func RecreateMissingStates(chainDb ethdb.Database, bc *core.BlockChain, cacheConfig *core.CacheConfig) error {
	log.Info("Recreating missing states...")
	start := time.Now()
	current := bc.Genesis().NumberU64() + 1
	last := bc.CurrentBlock().Number.Uint64()

	previousBlock := bc.GetBlockByNumber(current - 1)
	if previousBlock == nil {
		return fmt.Errorf("genesis block is missing")
	}
	hashConfig := *hashdb.Defaults
	hashConfig.CleanCacheSize = cacheConfig.TrieCleanLimit
	trieConfig := &trie.Config{
		Preimages: false,
		HashDB:    &hashConfig,
	}
	database := state.NewDatabaseWithConfig(chainDb, trieConfig)
	defer database.TrieDB().Close()
	previousState, err := state.New(previousBlock.Root(), database, nil)
	if err != nil {
		return fmt.Errorf("genesis state is missing: %w", err)
	}
	database.TrieDB().Reference(previousBlock.Root(), common.Hash{})
	logged := time.Now()
	recreated := 0
	for current <= last {
		if time.Since(logged) > 1*time.Minute {
			log.Info("Recreating missing states", "block", current, "target", last, "remaining", last-current, "elapsed", time.Since(start), "recreated", recreated)
			logged = time.Now()
		}
		currentBlock := bc.GetBlockByNumber(current)
		if currentBlock == nil {
			return fmt.Errorf("missing block %d", current)
		}
		currentState, err := state.New(currentBlock.Root(), database, nil)
		if err != nil {
			_, _, _, err := bc.Processor().Process(currentBlock, previousState, vm.Config{})
			if err != nil {
				return fmt.Errorf("processing block %d failed: %v", current, err)
			}
			root, err := previousState.Commit(current, bc.Config().IsEIP158(currentBlock.Number()))
			if err != nil {
				return fmt.Errorf("StateDB commit failed, number %d root %v: %w", current, currentBlock.Root().Hex(), err)
			}
			if root.Cmp(currentBlock.Root()) != 0 {
				return fmt.Errorf("reached different state root after processing block %d, want %v, have %v", current, currentBlock.Root(), root)
			}
			// commit to disk
			err = database.TrieDB().Commit(root, false) // TODO report = true, do we want this many logs?
			if err != nil {
				return fmt.Errorf("TrieDB commit failed, number %d root %v: %w", current, root, err)
			}
			currentState, err = state.New(currentBlock.Root(), database, nil)
			if err != nil {
				return fmt.Errorf("state reset after block %d failed: %v", current, err)
			}
			database.TrieDB().Reference(currentBlock.Root(), common.Hash{})
			database.TrieDB().Dereference(previousBlock.Root())
			recreated++
		}
		current++
		previousState = currentState
		previousBlock = currentBlock
	}
	log.Info("Finished recreating missing states", "elapsed", time.Since(start), "recreated", recreated)
	return nil
}
