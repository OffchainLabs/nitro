//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func main() {

	loglevel := flag.Int("loglevel", int(log.LvlInfo), "log level")

	nol1Listener := flag.Bool("UNSAFEnol1listener", false, "DANGEROUS! disables listening to L1. To be used in test nodes only")
	l1conn := flag.String("l1conn", "", "l1 connection (required unless no l1 listener)")

	l1sequencer := flag.Bool("l1sequencer", false, "act and post to l1 as sequencer")
	l1keystore := flag.String("l1keystore", "", "l1 private key store (required if l1role == sequencer)")
	l1Account := flag.String("l1Account", "", "l1 seq account to use (default is first account in keystore)")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase (1required if l1role == sequencer)")
	l1deploy := flag.Bool("l1deploy", false, "deploy L1 (sequencer)")
	l1deployment := flag.String("l1deployment", "", "json file including the existing deployment information")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	l2ChainIdUint := flag.Uint64("l2chainid", params.ArbitrumTestnetChainConfig().ChainID.Uint64(), "L2 chain ID (determines Arbitrum network)")
	forwardingtarget := flag.String("forwardingtarget", "", "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	batchpostermaxinterval := flag.Duration("batchpostermaxinterval", time.Minute, "maximum interval to post batches at (quicker if batches fill up)")

	dataAvailabilityMode := flag.String("dataavailability.mode", "onchain", "where to read/write sequencer batches. Options: onchain, local (testing only)")
	dataAvailabilityLocalDiskDirectory := flag.String("dataavailability.localdisk.dir", "", "directory to store data availability files")

	datadir := flag.String("datadir", "", "directory to store chain state")
	importFile := flag.String("importfile", "", "path for json data to import")
	devInit := flag.Bool("dev", false, "init with dev data (1 account with balance) instead of file import")
	keystorepath := flag.String("keystore", "", "dir for keystore")
	keystorepassphrase := flag.String("passphrase", "passphrase", "passphrase for keystore")

	httphost := flag.String("httphost", "localhost", "http host")
	httpPort := flag.Int("httpport", 7545, "http port")
	httpvhosts := flag.String("httpvhosts", "localhost", "list of virtual hosts to accept requests from")
	wshost := flag.String("wshost", "localhost", "websocket host")
	wsport := flag.Int("wsport", 7546, "websocket port")
	wsorigins := flag.String("wsorigins", "localhost", "list of origins to accept requests from")
	wsexposeall := flag.Bool("wsexposeall", false, "expose private api via websocket")

	broadcasterEnabled := flag.Bool("feed.output.enabled", false, "enable the broadcaster")
	broadcasterAddr := flag.String("feed.output.addr", "0.0.0.0", "address to bind the relay feed output to")
	broadcasterIOTimeout := flag.Duration("feed.output.io-timeout", 5*time.Second, "duration to wait before timing out HTTP to WS upgrade")
	broadcasterPort := flag.Int("feed.output.port", 9642, "port to bind the relay feed output to")
	broadcasterPing := flag.Duration("feed.output.ping", 5*time.Second, "duration for ping interval")
	broadcasterClientTimeout := flag.Duration("feed.output.client-timeout", 15*time.Second, "duration to wait before timing out connections to client")
	broadcasterWorkers := flag.Int("feed.output.workers", 100, "Number of threads to reserve for HTTP to WS upgrade")

	feedInputUrl := flag.String("feed.input.url", "", "URL of sequence feed source")
	feedInputTimeout := flag.Duration("feed.input.timeout", 20*time.Second, "duration to wait before timing out conection to server")

	l1validator := flag.Bool("l1validator", false, "enable L1 validator and staker functionality")
	validatorstrategy := flag.String("validatorstrategy", "watchtower", "L1 validator strategy, either watchtower, defensive, stakeLatest, or makeNodes (requires l1role=validator)")
	l1validatorwithoutblockvalidator := flag.Bool("UNSAFEl1validatorwithoutblockvalidator", false, "DANGEROUS! allows running an L1 validator without a block validator")
	stakerinterval := flag.Duration("stakerinterval", time.Minute, "how often the L1 validator should check the status of the L1 rollup and maybe take action with its stake")
	wasmrootpath := flag.String("wasmrootpath", "", "path to wasm files (replay.wasm, wasi_stub.wasm, soft-float.wasm, go_stub.wasm, host_io.wasm, brotli.wasm)")
	wasmmoduleroot := flag.String("wasmmoduleroot", "", "wasm module root (if empty, read from <wasmrootpath>/module_root)")
	wasmcachepath := flag.String("wasmcachepath", "", "path for cache of wasm machines")

	flag.Parse()
	ctx := context.Background()

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*loglevel))
	log.Root().SetHandler(glogger)

	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)
	l2ChainId := new(big.Int).SetUint64(*l2ChainIdUint)

	nodeConf := arbnode.NodeConfigDefault
	if *nol1Listener {
		nodeConf.L1Reader = false
		nodeConf.Sequencer = true // we sequence messages, but not to l1
		nodeConf.BatchPoster = false
		if *l1sequencer || *l1validator {
			flag.Usage()
			panic("nol1listener cannot be used with l1sequencer or l1validator")
		}
	} else {
		nodeConf.L1Reader = true
	}

	if *l1sequencer {
		nodeConf.Sequencer = true
		nodeConf.BatchPoster = true
		nodeConf.BatchPosterConfig.MaxBatchPostInterval = *batchpostermaxinterval

		if *forwardingtarget != "" && *forwardingtarget != "null" {
			flag.Usage()
			panic("forwardingtarget set with l1sequencer")
		}
	} else {
		if *forwardingtarget == "" {
			flag.Usage()
			panic("forwardingtarget unset, and not l1sequencer (can set to \"null\" to disable forwarding)")
		}
		if *forwardingtarget == "null" {
			nodeConf.ForwardingTarget = ""
		} else {
			nodeConf.ForwardingTarget = *forwardingtarget
		}
	}

	if *dataAvailabilityMode == "onchain" {
		nodeConf.DataAvailabilityMode = das.OnchainDataAvailability
	} else if *dataAvailabilityMode == "local" {
		nodeConf.DataAvailabilityMode = das.LocalDataAvailability
		if *dataAvailabilityLocalDiskDirectory == "" {
			flag.Usage()
			panic("davaavailability.localdisk.dir must be specified if mode is set to local")
		}
	} else {
		flag.Usage()
		panic("dataavailability.mode not recognized")
	}

	if *wasmrootpath != "" {
		validator.StaticNitroMachineConfig.RootPath = *wasmrootpath
	} else {
		execfile, err := os.Executable()
		if err != nil {
			panic(err)
		}
		targetDir := filepath.Dir(filepath.Dir(execfile))
		validator.StaticNitroMachineConfig.RootPath = filepath.Join(targetDir, "machine")
	}

	wasmModuleRootString := *wasmmoduleroot
	if wasmModuleRootString == "" {
		fileToRead := path.Join(validator.StaticNitroMachineConfig.RootPath, "module_root")
		fileBytes, err := ioutil.ReadFile(fileToRead)
		if err != nil {
			if *l1deploy || (*l1validator && !*l1validatorwithoutblockvalidator) {
				panic(fmt.Errorf("failed reading wasmModuleRoot from file, err %w", err))
			}
		}
		wasmModuleRootString = strings.TrimSpace(string(fileBytes))
		if len(wasmModuleRootString) > 64 {
			wasmModuleRootString = wasmModuleRootString[0:64]
		}
	}
	wasmModuleRoot := common.HexToHash(wasmModuleRootString)

	if *l1validator {
		nodeConf.L1Validator = true
		nodeConf.L1ValidatorConfig.Strategy = *validatorstrategy
		nodeConf.L1ValidatorConfig.StakerInterval = *stakerinterval
		if !*l1validatorwithoutblockvalidator {
			nodeConf.BlockValidator = true
			if *wasmcachepath != "" {
				validator.StaticNitroMachineConfig.InitialMachineCachePath = *wasmcachepath
			}
			go func() {
				expectedRoot := wasmModuleRoot
				foundRoot, err := validator.GetInitialModuleRoot(ctx)
				if err != nil {
					panic(fmt.Errorf("failed reading wasmModuleRoot from machine: %w", err))
				}
				if foundRoot != expectedRoot {
					panic(fmt.Errorf("incompatible wasmModuleRoot expected: %v found %v", expectedRoot, foundRoot))
				} else {
					log.Info("loaded wasm machine", "wasmModuleRoot", foundRoot)
				}
			}()
		}
	}

	var l1client *ethclient.Client
	var deployInfo arbnode.RollupAddresses
	var l1TransactionOpts *bind.TransactOpts
	if nodeConf.L1Reader {
		var err error
		var l1Addr common.Address

		l1client, err = ethclient.Dial(*l1conn)
		if err != nil {
			flag.Usage()
			panic(err)
		}
		if nodeConf.BatchPoster || nodeConf.L1Validator {
			l1TransactionOpts, err = util.GetTransactOptsFromKeystore(*l1keystore, *l1Account, *l1passphrase, l1ChainId)
			if err != nil {
				panic(err)
			}
			l1Addr = l1TransactionOpts.From
		}
		if *l1deploy {
			if !nodeConf.BatchPoster {
				flag.Usage()
				panic("deploy but not sequencer")
			}
			if nodeConf.BlockValidator {
				// TODO actually figure out the wasmModuleRoot
				panic("deploy as validator not yet supported")
			}
			var validators uint64
			if nodeConf.L1Validator {
				validators++
			}
			deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1Addr, validators, wasmModuleRoot, l2ChainId, time.Minute*5)
			if err != nil {
				flag.Usage()
				panic(err)
			}
			deployInfo = *deployPtr
		} else {
			if *l1deployment == "" {
				flag.Usage()
				panic("not deploying, but no deployment specified")
			}
			rawDeployment, err := ioutil.ReadFile(*l1deployment)
			if err != nil {
				panic(err)
			}
			if err := json.Unmarshal(rawDeployment, &deployInfo); err != nil {
				panic(err)
			}
		}
	} else {
		*l1deploy = false
	}

	stackConf := node.DefaultConfig
	stackConf.DataDir = *datadir
	stackConf.HTTPHost = *httphost
	stackConf.HTTPPort = *httpPort
	stackConf.HTTPVirtualHosts = utils.SplitAndTrim(*httpvhosts)
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	// TODO: Add CLI option for this
	stackConf.HTTPModules = append(stackConf.HTTPModules, "debug")
	stackConf.WSHost = *wshost
	stackConf.WSPort = *wsport
	stackConf.WSOrigins = utils.SplitAndTrim(*wsorigins)
	stackConf.WSModules = append(stackConf.WSModules, "eth")
	// TODO: Add CLI option for this
	stackConf.WSModules = append(stackConf.WSModules, "debug")
	stackConf.WSExposeAll = *wsexposeall
	if *wsexposeall {
		stackConf.WSModules = append(stackConf.WSModules, "personal")
	}
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		panic(err)
	}

	devPrivKeyStr := "e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2"
	devPrivKey, err := crypto.HexToECDSA(devPrivKeyStr)
	if err != nil {
		panic(err)
	}
	devAddr := crypto.PubkeyToAddress(devPrivKey.PublicKey)
	log.Info("Dev node funded private key", "priv", devPrivKeyStr)
	log.Info("Funded public address", "addr", devAddr)

	if *keystorepath != "" {
		mykeystore := keystore.NewPlaintextKeyStore(*keystorepath)
		stack.AccountManager().AddBackend(mykeystore)
		var account accounts.Account
		if mykeystore.HasAddress(devAddr) {
			account.Address = devAddr
			account, err = mykeystore.Find(account)
		} else {
			account, err = mykeystore.ImportECDSA(devPrivKey, *keystorepassphrase)
		}
		if err != nil {
			panic(err)
		}
		err = mykeystore.Unlock(account, *keystorepassphrase)
		if err != nil {
			panic(err)
		}
	}

	nodeConf.Broadcaster = *broadcasterEnabled
	nodeConf.BroadcasterConfig = wsbroadcastserver.BroadcasterConfig{
		Addr:          *broadcasterAddr,
		IOTimeout:     *broadcasterIOTimeout,
		Port:          strconv.Itoa(*broadcasterPort),
		Ping:          *broadcasterPing,
		ClientTimeout: *broadcasterClientTimeout,
		Queue:         100,
		Workers:       *broadcasterWorkers,
	}

	nodeConf.BroadcastClient = *feedInputUrl != ""
	nodeConf.BroadcastClientConfig = broadcastclient.BroadcastClientConfig{
		Timeout: *feedInputTimeout,
		URL:     *feedInputUrl,
	}

	var initDataReader statetransfer.InitDataReader = nil

	chainDb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 0, 0, "", "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}

	if *importFile != "" {
		initDataReader, err = statetransfer.NewJsonInitDataReader(*importFile)
		if err != nil {
			panic(err)
		}
	} else if *devInit {
		initData := statetransfer.ArbosInitializationInfo{
			Accounts: []statetransfer.AccountInitializationInfo{
				{
					Addr:       devAddr,
					EthBalance: new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)),
					Nonce:      0,
				},
			},
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
		if err != nil {
			panic(err)
		}
	}

	chainConfig, err := arbos.GetChainConfig(new(big.Int).SetUint64(*l2ChainIdUint))
	if err != nil {
		panic(err)
	}

	var l2BlockChain *core.BlockChain
	if initDataReader != nil {
		blockReader, err := initDataReader.GetStoredBlockReader()
		if err != nil {
			panic(err)
		}
		blockNum, err := arbnode.ImportBlocksToChainDb(chainDb, blockReader)
		if err != nil {
			panic(err)
		}
		l2BlockChain, err = arbnode.WriteOrTestBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack), initDataReader, blockNum, chainConfig)
		if err != nil {
			panic(err)
		}

	} else {
		blocksInDb, err := chainDb.Ancients()
		if err != nil {
			panic(err)
		}
		if blocksInDb == 0 {
			panic("No initialization mode supplied, no blocks in Db")
		}
		l2BlockChain, err = arbnode.GetBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack), chainConfig)
		if err != nil {
			panic(err)
		}
	}

	// Check that this ArbOS state has the correct chain ID
	{
		statedb, err := l2BlockChain.State()
		if err != nil {
			panic(err)
		}
		arbosState, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			panic(err)
		}
		chainId, err := arbosState.ChainId()
		if err != nil {
			panic(err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			panic(fmt.Sprintf("attempted to launch node with chain ID %v on ArbOS state with chain ID %v", chainConfig.ChainID, chainId))
		}
	}

	node, err := arbnode.CreateNode(stack, chainDb, &nodeConf, l2BlockChain, l1client, &deployInfo, l1TransactionOpts, l1TransactionOpts, nil)
	if err != nil {
		panic(err)
	}
	if err := node.Start(ctx); err != nil {
		utils.Fatalf("Error starting node: %v\n", err)
	}

	if err := stack.Start(); err != nil {
		utils.Fatalf("Error starting protocol stack: %v\n", err)
	}
	<-signalChan
	log.Info("Shutting down node")
	node.StopAndWait()
	if err := stack.Close(); err != nil {
		utils.Fatalf("Error shutting down protocol stack: %v\n", err)
	}
	log.Info("Node shutdown complete")
}
