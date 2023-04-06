// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	extract "github.com/codeclysm/extract/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/cmd/ipfshelper"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type InitConfig struct {
	Force           bool          `koanf:"force"`
	Url             string        `koanf:"url"`
	DownloadPath    string        `koanf:"download-path"`
	DownloadPoll    time.Duration `koanf:"download-poll"`
	DevInit         bool          `koanf:"dev-init"`
	DevInitAddr     string        `koanf:"dev-init-address"`
	DevInitBlockNum uint64        `koanf:"dev-init-blocknum"`
	Empty           bool          `koanf:"empty"`
	AccountsPerSync uint          `koanf:"accounts-per-sync"`
	ImportFile      string        `koanf:"import-file"`
	ThenQuit        bool          `koanf:"then-quit"`
}

var InitConfigDefault = InitConfig{
	Force:           false,
	Url:             "",
	DownloadPath:    "/tmp/",
	DownloadPoll:    time.Minute,
	DevInit:         false,
	DevInitAddr:     "",
	DevInitBlockNum: 0,
	ImportFile:      "",
	AccountsPerSync: 100000,
	ThenQuit:        false,
}

func InitConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".force", InitConfigDefault.Force, "if true: in case database exists init code will be reexecuted and genesis block compared to database")
	f.String(prefix+".url", InitConfigDefault.Url, "url to download initializtion data - will poll if download fails")
	f.String(prefix+".download-path", InitConfigDefault.DownloadPath, "path to save temp downloaded file")
	f.Duration(prefix+".download-poll", InitConfigDefault.DownloadPoll, "how long to wait between polling attempts")
	f.Bool(prefix+".dev-init", InitConfigDefault.DevInit, "init with dev data (1 account with balance) instead of file import")
	f.String(prefix+".dev-init-address", InitConfigDefault.DevInitAddr, "Address of dev-account. Leave empty to use the dev-wallet.")
	f.Uint64(prefix+".dev-init-blocknum", InitConfigDefault.DevInitBlockNum, "Number of preinit blocks. Must exist in ancient database.")
	f.Bool(prefix+".empty", InitConfigDefault.DevInit, "init with empty state")
	f.Bool(prefix+".then-quit", InitConfigDefault.ThenQuit, "quit after init is done")
	f.String(prefix+".import-file", InitConfigDefault.ImportFile, "path for json data to import")
	f.Uint(prefix+".accounts-per-sync", InitConfigDefault.AccountsPerSync, "during init - sync database every X accounts. Lower value for low-memory systems. 0 disables.")
}

func downloadInit(ctx context.Context, initConfig *InitConfig) (string, error) {
	if initConfig.Url == "" {
		return "", nil
	}
	if strings.HasPrefix(initConfig.Url, "file:") {
		return initConfig.Url[5:], nil
	}
	if ipfshelper.CanBeIpfsPath(initConfig.Url) {
		ipfsNode, err := ipfshelper.CreateIpfsHelper(ctx, initConfig.DownloadPath, false, []string{}, ipfshelper.DefaultIpfsProfiles)
		if err != nil {
			return "", err
		}
		log.Info("Downloading initial database via IPFS", "url", initConfig.Url)
		initFile, downloadErr := ipfsNode.DownloadFile(ctx, initConfig.Url, initConfig.DownloadPath)
		closeErr := ipfsNode.Close()
		if downloadErr != nil {
			if closeErr != nil {
				log.Error("Failed to close IPFS node after download error", "err", closeErr)
			}
			return "", fmt.Errorf("Failed to download file from IPFS: %w", downloadErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("Failed to close IPFS node: %w", err)
		}
		return initFile, nil
	}
	grabclient := grab.NewClient()
	log.Info("Downloading initial database", "url", initConfig.Url)
	fmt.Println()
	printTicker := time.NewTicker(time.Second)
	defer printTicker.Stop()
	attempt := 0
	for {
		attempt++
		req, err := grab.NewRequest(initConfig.DownloadPath, initConfig.Url)
		if err != nil {
			panic(err)
		}
		resp := grabclient.Do(req.WithContext(ctx))
		firstPrintTime := time.Now().Add(time.Second * 2)
	updateLoop:
		for {
			select {
			case <-printTicker.C:
				if time.Now().After(firstPrintTime) {
					bps := resp.BytesPerSecond()
					if bps == 0 {
						bps = 1 // avoid division by zero
					}
					done := resp.BytesComplete()
					total := resp.Size()
					timeRemaining := time.Second * (time.Duration(total-done) / time.Duration(bps))
					timeRemaining = timeRemaining.Truncate(time.Millisecond * 10)
					fmt.Printf("\033[2K\r  transferred %v / %v bytes (%.2f%%) [%.2fMbps, %s remaining]",
						done,
						total,
						resp.Progress()*100,
						bps*8/1000000,
						timeRemaining.String())
				}
			case <-resp.Done:
				if err := resp.Err(); err != nil {
					fmt.Printf("\n  attempt %d failed: %v\n", attempt, err)
					break updateLoop
				}
				fmt.Printf("\n")
				log.Info("Download done", "filename", resp.Filename, "duration", resp.Duration())
				fmt.Println()
				return resp.Filename, nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(initConfig.DownloadPoll):
		}
	}
}

func validateBlockChain(blockChain *core.BlockChain, expectedChainId *big.Int) error {
	statedb, err := blockChain.State()
	if err != nil {
		return err
	}
	currentArbosState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	chainId, err := currentArbosState.ChainId()
	if err != nil {
		return err
	}
	if chainId.Cmp(expectedChainId) != 0 {
		return fmt.Errorf("attempted to launch node with chain ID %v on ArbOS state with chain ID %v", expectedChainId, chainId)
	}
	return nil
}

func openInitializeChainDb(ctx context.Context, stack *node.Node, config *NodeConfig, chainId *big.Int, cacheConfig *core.CacheConfig) (ethdb.Database, *core.BlockChain, error) {
	if !config.Init.Force {
		if readOnlyDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, config.Persistent.Ancients, "", true); err == nil {
			if chainConfig := arbnode.TryReadStoredChainConfig(readOnlyDb); chainConfig != nil {
				readOnlyDb.Close()
				chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", config.Node.Caching.DatabaseCache, config.Persistent.Handles, config.Persistent.Ancients, "", false)
				if err != nil {
					return chainDb, nil, err
				}
				l2BlockChain, err := arbnode.GetBlockChain(chainDb, cacheConfig, chainConfig, &config.Node)
				if err != nil {
					return chainDb, nil, err
				}
				err = validateBlockChain(l2BlockChain, chainConfig.ChainID)
				if err != nil {
					return chainDb, l2BlockChain, err
				}
				return chainDb, l2BlockChain, nil
			}
			readOnlyDb.Close()
		}
	}

	initFile, err := downloadInit(ctx, &config.Init)
	if err != nil {
		return nil, nil, err
	}

	if initFile != "" {
		reader, err := os.Open(initFile)
		if err != nil {
			return nil, nil, fmt.Errorf("couln't open init '%v' archive: %w", initFile, err)
		}
		stat, err := reader.Stat()
		if err != nil {
			return nil, nil, err
		}
		log.Info("extracting downloaded init archive", "size", fmt.Sprintf("%dMB", stat.Size()/1024/1024))
		err = extract.Archive(context.Background(), reader, stack.InstanceDir(), nil)
		if err != nil {
			return nil, nil, fmt.Errorf("couln't extract init archive '%v' err:%w", initFile, err)
		}
	}

	var initDataReader statetransfer.InitDataReader = nil

	chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", config.Node.Caching.DatabaseCache, config.Persistent.Handles, config.Persistent.Ancients, "", false)
	if err != nil {
		return chainDb, nil, err
	}

	if config.Init.ImportFile != "" {
		initDataReader, err = statetransfer.NewJsonInitDataReader(config.Init.ImportFile)
		if err != nil {
			return chainDb, nil, fmt.Errorf("error reading import file: %w", err)
		}
	}
	if config.Init.Empty {
		if initDataReader != nil {
			return chainDb, nil, errors.New("multiple init methods supplied")
		}
		initData := statetransfer.ArbosInitializationInfo{
			NextBlockNumber: 0,
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}
	if config.Init.DevInit {
		if initDataReader != nil {
			return chainDb, nil, errors.New("multiple init methods supplied")
		}
		initData := statetransfer.ArbosInitializationInfo{
			NextBlockNumber: config.Init.DevInitBlockNum,
			Accounts: []statetransfer.AccountInitializationInfo{
				{
					Addr:       common.HexToAddress(config.Init.DevInitAddr),
					EthBalance: new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)),
					Nonce:      0,
				},
			},
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}

	var chainConfig *params.ChainConfig

	var l2BlockChain *core.BlockChain
	txIndexWg := sync.WaitGroup{}
	if initDataReader == nil {
		chainConfig = arbnode.TryReadStoredChainConfig(chainDb)
		if chainConfig == nil {
			return chainDb, nil, errors.New("no --init.* mode supplied and chain data not in expected directory")
		}
		l2BlockChain, err = arbnode.GetBlockChain(chainDb, cacheConfig, chainConfig, &config.Node)
		if err != nil {
			return chainDb, nil, err
		}
		genesisBlockNr := chainConfig.ArbitrumChainParams.GenesisBlockNum
		genesisBlock := l2BlockChain.GetBlockByNumber(genesisBlockNr)
		if genesisBlock != nil {
			log.Info("loaded genesis block from database", "number", genesisBlockNr, "hash", genesisBlock.Hash())
		} else {
			// The node will probably die later, but might as well not kill it here?
			log.Error("database missing genesis block", "number", genesisBlockNr)
		}
		testUpdateTxIndex(chainDb, chainConfig, &txIndexWg)
	} else {
		genesisBlockNr, err := initDataReader.GetNextBlockNumber()
		if err != nil {
			return chainDb, nil, err
		}
		chainConfig, err = arbos.GetChainConfig(chainId, genesisBlockNr)
		if err != nil {
			return chainDb, nil, err
		}
		testUpdateTxIndex(chainDb, chainConfig, &txIndexWg)
		ancients, err := chainDb.Ancients()
		if err != nil {
			return chainDb, nil, err
		}
		if ancients < genesisBlockNr {
			return chainDb, nil, fmt.Errorf("%v pre-init blocks required, but only %v found", genesisBlockNr, ancients)
		}
		if ancients > genesisBlockNr {
			storedGenHash := rawdb.ReadCanonicalHash(chainDb, genesisBlockNr)
			storedGenBlock := rawdb.ReadBlock(chainDb, storedGenHash, genesisBlockNr)
			if storedGenBlock.Header().Root == (common.Hash{}) {
				return chainDb, nil, fmt.Errorf("attempting to init genesis block %x, but this block is in database with no state root", genesisBlockNr)
			}
			log.Warn("Re-creating genesis though it seems to exist in database", "blockNr", genesisBlockNr)
		}
		log.Info("Initializing", "ancients", ancients, "genesisBlockNr", genesisBlockNr)
		if config.Init.ThenQuit {
			cacheConfig.SnapshotWait = true
		}
		l2BlockChain, err = arbnode.WriteOrTestBlockChain(chainDb, cacheConfig, initDataReader, chainConfig, &config.Node, config.Init.AccountsPerSync)
		if err != nil {
			return chainDb, nil, err
		}
	}
	txIndexWg.Wait()
	err = chainDb.Sync()
	if err != nil {
		return chainDb, l2BlockChain, err
	}

	err = validateBlockChain(l2BlockChain, chainConfig.ChainID)
	if err != nil {
		return chainDb, l2BlockChain, err
	}

	return chainDb, l2BlockChain, nil
}

func testTxIndexUpdated(chainDb ethdb.Database, lastBlock uint64) bool {
	var transactions types.Transactions
	blockHash := rawdb.ReadCanonicalHash(chainDb, lastBlock)
	reReadNumber := rawdb.ReadHeaderNumber(chainDb, blockHash)
	if reReadNumber == nil {
		return false
	}
	for ; ; lastBlock-- {
		blockHash := rawdb.ReadCanonicalHash(chainDb, lastBlock)
		block := rawdb.ReadBlock(chainDb, blockHash, lastBlock)
		transactions = block.Transactions()
		if len(transactions) == 0 {
			if lastBlock == 0 {
				return true
			}
			continue
		}
		entry := rawdb.ReadTxLookupEntry(chainDb, transactions[len(transactions)-1].Hash())
		return entry != nil
	}
}

func testUpdateTxIndex(chainDb ethdb.Database, chainConfig *params.ChainConfig, globalWg *sync.WaitGroup) {
	lastBlock := chainConfig.ArbitrumChainParams.GenesisBlockNum
	if lastBlock == 0 {
		// no Tx, no need to update index
		return
	}

	lastBlock -= 1
	if testTxIndexUpdated(chainDb, lastBlock) {
		return
	}

	var localWg sync.WaitGroup
	threads := runtime.NumCPU()
	var failedTxIndiciesMutex sync.Mutex
	failedTxIndicies := make(map[common.Hash]uint64)
	for thread := 0; thread < threads; thread++ {
		thread := thread
		localWg.Add(1)
		go func() {
			batch := chainDb.NewBatch()
			for blockNum := uint64(thread); blockNum <= lastBlock; blockNum += uint64(threads) {
				blockHash := rawdb.ReadCanonicalHash(chainDb, blockNum)
				block := rawdb.ReadBlock(chainDb, blockHash, blockNum)
				receipts := rawdb.ReadRawReceipts(chainDb, blockHash, blockNum)
				for i, receipt := range receipts {
					// receipt.TxHash isn't populated as we used ReadRawReceipts
					txHash := block.Transactions()[i].Hash()
					if receipt.Status != 0 || receipt.GasUsed != 0 {
						rawdb.WriteTxLookupEntries(batch, blockNum, []common.Hash{txHash})
					} else {
						failedTxIndiciesMutex.Lock()
						prev, exists := failedTxIndicies[txHash]
						if !exists || prev < blockNum {
							failedTxIndicies[txHash] = blockNum
						}
						failedTxIndiciesMutex.Unlock()
					}
				}
				rawdb.WriteHeaderNumber(batch, block.Header().Hash(), blockNum)
				if blockNum%1_000_000 == 0 {
					log.Info("writing tx lookup entries", "block", blockNum)
				}
				if batch.ValueSize() >= ethdb.IdealBatchSize {
					err := batch.Write()
					if err != nil {
						panic(err)
					}
					batch.Reset()
				}
			}
			err := batch.Write()
			if err != nil {
				panic(err)
			}
			localWg.Done()
		}()
	}

	globalWg.Add(1)
	go func() {
		localWg.Wait()
		batch := chainDb.NewBatch()
		for txHash, blockNum := range failedTxIndicies {
			if rawdb.ReadTxLookupEntry(chainDb, txHash) == nil {
				rawdb.WriteTxLookupEntries(batch, blockNum, []common.Hash{txHash})
			}
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				err := batch.Write()
				if err != nil {
					panic(err)
				}
				batch.Reset()
			}
		}
		err := batch.Write()
		if err != nil {
			panic(err)
		}
		log.Info("Tx lookup entries written")
		globalWg.Done()
	}()
}
