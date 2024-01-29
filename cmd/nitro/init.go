// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/ipfshelper"
	"github.com/offchainlabs/nitro/cmd/pruning"
	"github.com/offchainlabs/nitro/cmd/staterecovery"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func downloadInit(ctx context.Context, initConfig *conf.InitConfig) (string, error) {
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

func validateBlockChain(blockChain *core.BlockChain, chainConfig *params.ChainConfig) error {
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
	if chainId.Cmp(chainConfig.ChainID) != 0 {
		return fmt.Errorf("attempted to launch node with chain ID %v on ArbOS state with chain ID %v", chainConfig.ChainID, chainId)
	}
	oldSerializedConfig, err := currentArbosState.ChainConfig()
	if err != nil {
		return fmt.Errorf("failed to get old chain config from ArbOS state: %w", err)
	}
	if len(oldSerializedConfig) != 0 {
		var oldConfig params.ChainConfig
		err = json.Unmarshal(oldSerializedConfig, &oldConfig)
		if err != nil {
			return fmt.Errorf("failed to deserialize old chain config: %w", err)
		}
		currentBlock := blockChain.CurrentBlock()
		if currentBlock == nil {
			return errors.New("failed to get current block")
		}
		if err := oldConfig.CheckCompatible(chainConfig, currentBlock.Number.Uint64(), currentBlock.Time); err != nil {
			return fmt.Errorf("invalid chain config, not compatible with previous: %w", err)
		}
	}

	return nil
}

func openInitializeChainDb(ctx context.Context, stack *node.Node, config *NodeConfig, chainId *big.Int, cacheConfig *core.CacheConfig, l1Client arbutil.L1Interface, rollupAddrs chaininfo.RollupAddresses) (ethdb.Database, *core.BlockChain, error) {
	if !config.Init.Force {
		if readOnlyDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", true); err == nil {
			if chainConfig := gethexec.TryReadStoredChainConfig(readOnlyDb); chainConfig != nil {
				readOnlyDb.Close()
				if !arbmath.BigEquals(chainConfig.ChainID, chainId) {
					return nil, nil, fmt.Errorf("database has chain ID %v but config has chain ID %v (are you sure this database is for the right chain?)", chainConfig.ChainID, chainId)
				}
				chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", config.Execution.Caching.DatabaseCache, config.Persistent.Handles, config.Persistent.Ancient, "", false)
				if err != nil {
					return chainDb, nil, err
				}
				err = pruning.PruneChainDb(ctx, chainDb, stack, &config.Init, cacheConfig, l1Client, rollupAddrs, config.Node.ValidatorRequired())
				if err != nil {
					return chainDb, nil, fmt.Errorf("error pruning: %w", err)
				}
				l2BlockChain, err := gethexec.GetBlockChain(chainDb, cacheConfig, chainConfig, config.Execution.TxLookupLimit)
				if err != nil {
					return chainDb, nil, err
				}
				err = validateBlockChain(l2BlockChain, chainConfig)
				if err != nil {
					return chainDb, l2BlockChain, err
				}
				if config.Init.RecreateMissingState {
					err = staterecovery.RecreateMissingStates(chainDb, l2BlockChain, cacheConfig)
					if err != nil {
						return chainDb, l2BlockChain, err
					}
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

	chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", config.Execution.Caching.DatabaseCache, config.Persistent.Handles, config.Persistent.Ancient, "", false)
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
					Addr:       common.HexToAddress(config.Init.DevInitAddress),
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
		chainConfig = gethexec.TryReadStoredChainConfig(chainDb)
		if chainConfig == nil {
			return chainDb, nil, errors.New("no --init.* mode supplied and chain data not in expected directory")
		}
		l2BlockChain, err = gethexec.GetBlockChain(chainDb, cacheConfig, chainConfig, config.Execution.TxLookupLimit)
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
		combinedL2ChainInfoFiles := config.Chain.InfoFiles
		if config.Chain.InfoIpfsUrl != "" {
			l2ChainInfoIpfsFile, err := util.GetL2ChainInfoIpfsFile(ctx, config.Chain.InfoIpfsUrl, config.Chain.InfoIpfsDownloadPath)
			if err != nil {
				log.Error("error getting l2 chain info file from ipfs", "err", err)
			}
			combinedL2ChainInfoFiles = append(combinedL2ChainInfoFiles, l2ChainInfoIpfsFile)
		}
		chainConfig, err = chaininfo.GetChainConfig(new(big.Int).SetUint64(config.Chain.ID), config.Chain.Name, genesisBlockNr, combinedL2ChainInfoFiles, config.Chain.InfoJson)
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
		var parsedInitMessage *arbostypes.ParsedInitMessage
		if config.Node.ParentChainReader.Enable {
			delayedBridge, err := arbnode.NewDelayedBridge(l1Client, rollupAddrs.Bridge, rollupAddrs.DeployedAt)
			if err != nil {
				return chainDb, nil, fmt.Errorf("failed creating delayed bridge while attempting to get serialized chain config from init message: %w", err)
			}
			deployedAt := new(big.Int).SetUint64(rollupAddrs.DeployedAt)
			delayedMessages, err := delayedBridge.LookupMessagesInRange(ctx, deployedAt, deployedAt, nil)
			if err != nil {
				return chainDb, nil, fmt.Errorf("failed getting delayed messages while attempting to get serialized chain config from init message: %w", err)
			}
			var initMessage *arbostypes.L1IncomingMessage
			for _, msg := range delayedMessages {
				if msg.Message.Header.Kind == arbostypes.L1MessageType_Initialize {
					initMessage = msg.Message
					break
				}
			}
			if initMessage == nil {
				return chainDb, nil, fmt.Errorf("failed to get init message while attempting to get serialized chain config")
			}
			parsedInitMessage, err = initMessage.ParseInitMessage()
			if err != nil {
				return chainDb, nil, err
			}
			if parsedInitMessage.ChainId.Cmp(chainId) != 0 {
				return chainDb, nil, fmt.Errorf("expected L2 chain ID %v but read L2 chain ID %v from init message in L1 inbox", chainId, parsedInitMessage.ChainId)
			}
			if parsedInitMessage.ChainConfig != nil {
				if err := parsedInitMessage.ChainConfig.CheckCompatible(chainConfig, chainConfig.ArbitrumChainParams.GenesisBlockNum, 0); err != nil {
					return chainDb, nil, fmt.Errorf("incompatible chain config read from init message in L1 inbox: %w", err)
				}
			}
			log.Info("Read serialized chain config from init message", "json", string(parsedInitMessage.SerializedChainConfig))
		} else {
			serializedChainConfig, err := json.Marshal(chainConfig)
			if err != nil {
				return chainDb, nil, err
			}
			parsedInitMessage = &arbostypes.ParsedInitMessage{
				ChainId:               chainConfig.ChainID,
				InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
				ChainConfig:           chainConfig,
				SerializedChainConfig: serializedChainConfig,
			}
			log.Warn("Created fake init message as L1Reader is disabled and serialized chain config from init message is not available", "json", string(serializedChainConfig))
		}

		l2BlockChain, err = gethexec.WriteOrTestBlockChain(chainDb, cacheConfig, initDataReader, chainConfig, parsedInitMessage, config.Execution.TxLookupLimit, config.Init.AccountsPerSync)
		if err != nil {
			return chainDb, nil, err
		}
	}
	txIndexWg.Wait()
	err = chainDb.Sync()
	if err != nil {
		return chainDb, l2BlockChain, err
	}

	err = pruning.PruneChainDb(ctx, chainDb, stack, &config.Init, cacheConfig, l1Client, rollupAddrs, config.Node.ValidatorRequired())
	if err != nil {
		return chainDb, nil, fmt.Errorf("error pruning: %w", err)
	}

	err = validateBlockChain(l2BlockChain, chainConfig)
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
