package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
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
					timeRemaining := (time.Second * time.Duration(total-done)) / time.Duration(bps)
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
					fmt.Printf("\033[2K\r  attempt %d failed: %v", attempt, err)
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
		if readOnlyDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", true); err == nil {
			if chainConfig := arbnode.TryReadStoredChainConfig(readOnlyDb); chainConfig != nil {
				readOnlyDb.Close()
				chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", false)
				if err != nil {
					return nil, nil, err
				}
				l2BlockChain, err := arbnode.GetBlockChain(chainDb, cacheConfig, chainConfig, &config.Node)
				if err != nil {
					return nil, nil, err
				}
				err = validateBlockChain(l2BlockChain, chainConfig.ChainID)
				if err != nil {
					return nil, nil, err
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

	chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", false)
	if err != nil {
		return nil, nil, err
	}

	if config.Init.ImportFile != "" {
		initDataReader, err = statetransfer.NewJsonInitDataReader(config.Init.ImportFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading import file: %w", err)
		}
	}
	if config.Init.Empty {
		if initDataReader != nil {
			return nil, nil, errors.New("multiple init methods supplied")
		}
		initData := statetransfer.ArbosInitializationInfo{
			NextBlockNumber: 0,
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}
	if config.Init.DevInit {
		if initDataReader != nil {
			return nil, nil, errors.New("multiple init methods supplied")
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
			return nil, nil, errors.New("no --init.* mode supplied and chain data not in expected directory")
		}
		l2BlockChain, err = arbnode.GetBlockChain(chainDb, cacheConfig, chainConfig, &config.Node)
		if err != nil {
			panic(err)
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
			panic(err)
		}
		chainConfig, err = arbos.GetChainConfig(chainId, genesisBlockNr)
		if err != nil {
			panic(err)
		}
		testUpdateTxIndex(chainDb, chainConfig, &txIndexWg)
		ancients, err := chainDb.Ancients()
		if err != nil {
			panic(err)
		}
		if ancients < genesisBlockNr {
			panic(fmt.Sprint(genesisBlockNr, " pre-init blocks required, but only ", ancients, " found"))
		}
		if ancients > genesisBlockNr {
			storedGenHash := rawdb.ReadCanonicalHash(chainDb, genesisBlockNr)
			storedGenBlock := rawdb.ReadBlock(chainDb, storedGenHash, genesisBlockNr)
			if storedGenBlock.Header().Root == (common.Hash{}) {
				panic(fmt.Errorf("Attempting to init genesis block %x, but this block is in database with no state root", genesisBlockNr))
			}
			log.Warn("Re-creating genesis though it seems to exist in database", "blockNr", genesisBlockNr)
		}
		log.Info("Initializing", "ancients", ancients, "genesisBlockNr", genesisBlockNr)
		l2BlockChain, err = arbnode.WriteOrTestBlockChain(chainDb, cacheConfig, initDataReader, chainConfig, &config.Node, config.Init.AccountsPerSync)
		if err != nil {
			panic(err)
		}
	}
	txIndexWg.Wait()
	err = chainDb.Sync()
	if err != nil {
		panic(err)
	}

	err = validateBlockChain(l2BlockChain, chainConfig.ChainID)
	if err != nil {
		return nil, nil, err
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
		return (entry != nil)
	}
}

func testUpdateTxIndex(chainDb ethdb.Database, chainConfig *params.ChainConfig, txIndexWg *sync.WaitGroup) {
	lastBlock := chainConfig.ArbitrumChainParams.GenesisBlockNum
	if lastBlock == 0 {
		// no Tx, no need to update index
		return
	}

	lastBlock -= 1
	if testTxIndexUpdated(chainDb, lastBlock) {
		return
	}

	txIndexWg.Add(1)
	log.Info("writing Tx lookup entries")

	go func() {
		batch := chainDb.NewBatch()
		for blockNum := uint64(0); blockNum <= lastBlock; blockNum++ {
			blockHash := rawdb.ReadCanonicalHash(chainDb, blockNum)
			block := rawdb.ReadBlock(chainDb, blockHash, blockNum)
			txs := block.Transactions()
			txHashes := make([]common.Hash, 0, len(txs))
			txHashMap := make(map[common.Hash]int, len(txs))
			for _, tx := range txs {
				txHash := tx.Hash()
				txHashes = append(txHashes, txHash)
				txHashMap[txHash]++
			}
			for txHash := range txHashMap {
				if entry := rawdb.ReadTxLookupEntry(chainDb, txHash); entry != nil {
					txHashMap[txHash]++
				}
			}
			var receipts types.Receipts = nil
			for i, txHash := range txHashes {
				if txHashMap[txHash] > 1 {
					if receipts == nil {
						receipts = rawdb.ReadReceipts(chainDb, blockHash, blockNum, chainConfig)
					}
					if receipts[i].Status == 0 && receipts[i].GasUsed == 0 {
						log.Info("Not indexing failed duplicate transaction", "block", blockNum, "txHash", txHash, "index", i)
						txHashes[i] = common.Hash{}
						txHashMap[txHash]--
					}
				}
			}
			rawdb.WriteTxLookupEntries(batch, blockNum, txHashes)
			rawdb.WriteHeaderNumber(batch, block.Header().Hash(), blockNum)
			if (batch.ValueSize() >= ethdb.IdealBatchSize) || blockNum == lastBlock {
				err := batch.Write()
				if err != nil {
					panic(err)
				}
				batch.Reset()
			}
		}
		txIndexWg.Done()
		log.Info("Tx lookup entries written")
	}()
}
