//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
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
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/statetransfer"
	"github.com/offchainlabs/arbstate/util"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

func main() {

	loglevel := flag.Int("loglevel", int(log.LvlInfo), "log level")

	l1role := flag.String("l1role", "none", "either sequencer, listener, or none")
	l1conn := flag.String("l1conn", "", "l1 connection (required if l1role != none)")
	l1keystore := flag.String("l1keystore", "", "l1 private key store (required if l1role == sequencer)")
	seqAccount := flag.String("l1SeqAccount", "", "l1 seq account to use (default is first account in keystore)")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase (1required if l1role == sequencer)")
	l1deploy := flag.Bool("l1deploy", false, "deploy L1 (if role == sequencer)")
	l1deployment := flag.String("l1deployment", "", "json file including the existing deployment information")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	forwardingtarget := flag.String("forwardingtarget", "", "transaction forwarding target URL (empty if sequencer)")
	batchpostermaxinterval := flag.Duration("batchpostermaxinterval", time.Minute, "maximum interval to post batches at (quicker if batches fill up)")

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

	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*loglevel))
	log.Root().SetHandler(glogger)

	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	nodeConf := arbnode.NodeConfigDefault
	nodeConf.ForwardingTarget = *forwardingtarget
	log.Info("Running Arbitrum node with", "role", *l1role)
	if *l1role == "none" {
		nodeConf.Sequencer = false
		nodeConf.L1Reader = false
		nodeConf.BatchPoster = false
	} else if *l1role == "listener" {
		nodeConf.Sequencer = false
		nodeConf.L1Reader = true
		nodeConf.BatchPoster = false
	} else if *l1role == "sequencer" {
		nodeConf.Sequencer = true
		nodeConf.L1Reader = true
		nodeConf.BatchPoster = true
	} else {
		flag.Usage()
		panic("l1role not recognized")
	}
	nodeConf.BatchPosterConfig.MaxBatchPostInterval = *batchpostermaxinterval

	if *l1validator {
		if !nodeConf.L1Reader {
			flag.Usage()
			panic("l1validator requires l1role other than \"none\"")
		}
		nodeConf.L1Validator = true
		nodeConf.L1ValidatorConfig.Strategy = *validatorstrategy
		nodeConf.L1ValidatorConfig.StakerInterval = *stakerinterval
		if !nodeConf.BlockValidator && !*l1validatorwithoutblockvalidator {
			flag.Usage()
			panic("L1 validator requires block validator to safely function")
		}
	}

	ctx := context.Background()

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
			l1TransactionOpts, err = util.GetTransactOptsFromKeystore(*l1keystore, *seqAccount, *l1passphrase, l1ChainId)
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
			var wasmModuleRoot common.Hash
			if nodeConf.BlockValidator {
				// TODO actually figure out the wasmModuleRoot
				panic("deploy as validator not yet supported")
			}
			var validators uint64
			if nodeConf.L1Validator {
				validators++
			}
			deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1Addr, validators, wasmModuleRoot, time.Minute*5)
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
	stackConf.WSHost = *wshost
	stackConf.WSPort = *wsport
	stackConf.WSOrigins = utils.SplitAndTrim(*wsorigins)
	stackConf.WSModules = append(stackConf.WSModules, "eth")
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
		l2BlockChain, err = arbnode.WriteOrTestBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack), initDataReader, blockNum, params.ArbitrumOneChainConfig())
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
		l2BlockChain, err = arbnode.GetBlockChain(chainDb, arbnode.DefaultCacheConfigFor(stack), params.ArbitrumOneChainConfig())
		if err != nil {
			panic(err)
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

	stack.Wait()
	node.StopAndWait()
}
