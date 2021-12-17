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
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("running node")

	l1role := flag.String("l1role", "none", "either sequencer, listener, or none")
	l1conn := flag.String("l1conn", "", "l1 connection (required if l1role != none)")
	l1keyfile := flag.String("l1keyfile", "", "l1 private key file (required if l1role == sequencer)")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase (1required if l1role == sequencer)")
	l1deploy := flag.Bool("l1deploy", false, "deploy L1 (if role == sequencer)")
	l1deployment := flag.String("l1deployment", "", "json file including the existing deployment information")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	forwardingtarget := flag.String("forwardingtarget", "", "transaction forwarding target URL (empty if sequencer)")

	datadir := flag.String("datadir", "", "directory to store chain state")
	keystorepath := flag.String("keystore", "", "dir for keystore")
	keystorepassphrase := flag.String("passphrase", "passphrase", "passphrase for keystore")
	httphost := flag.String("httphost", "localhost", "http host")
	httpPort := flag.Int("httpport", 7545, "http port")
	httpvhosts := flag.String("httpvhosts", "localhost", "list of virtual hosts to accept requests from")
	wshost := flag.String("wshost", "localhost", "websocket host")
	wsport := flag.Int("wsport", 7546, "websocket port")
	wsorigins := flag.String("wsorigins", "localhost", "list of origins to accept requests from")
	wsexposeall := flag.Bool("wsexposeall", false, "expose private api via websocket")

	broadcasterEnabled := flag.Bool("broadcaster", false, "enable the broadcaster")
	broadcasterAddr := flag.String("broadcaster.addr", "0.0.0.0", "address to bind the relay feed output to")
	broadcasterIOTimeout := flag.Duration("broadcaster.io-timeout", 5*time.Second, "duration to wait before timing out HTTP to WS upgrade")
	broadcasterPort := flag.Int("broadcaster.port", 9642, "port to bind the relay feed output to")
	broadcasterPing := flag.Duration("broadcaster.ping", 5*time.Second, "duration for ping interval")
	broadcasterClientTimeout := flag.Duration("broadcaster.client-timeout", 15*time.Second, "duraction to wait before timing out connections to client")
	broadcasterWorkers := flag.Int("broadcaster.workers", 100, "Number of threads to reserve for HTTP to WS upgrade")

	flag.Parse()

	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	nodeConf := arbnode.NodeConfigDefault
	nodeConf.ForwardingTarget = *forwardingtarget
	log.Info("Running with", "role", *l1role)
	if *l1role == "none" {
		nodeConf.L1Reader = false
		nodeConf.BatchPoster = false
	} else if *l1role == "listener" {
		nodeConf.L1Reader = true
		nodeConf.BatchPoster = false
	} else if *l1role == "sequencer" {
		nodeConf.L1Reader = true
		nodeConf.BatchPoster = true
	} else {
		flag.Usage()
		panic("l1role not recognized")
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
		if nodeConf.BatchPoster {
			if *l1keyfile == "" {
				flag.Usage()
				panic("sequencer requires l1 keyfile")
			}
			fileReader, err := os.Open(*l1keyfile)
			if err != nil {
				flag.Usage()
				panic("sequencer without valid l1priv key")
			}
			l1TransactionOpts, err = bind.NewTransactorWithChainID(fileReader, *l1passphrase, l1ChainId)
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
			deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1Addr, time.Minute*5)
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

	genesisAlloc := make(core.GenesisAlloc)
	genesisAlloc[devAddr] = core.GenesisAccount{
		Balance:    new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)),
		Nonce:      0,
		PrivateKey: nil,
	}
	l2Genesys := &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(params.InitialBaseFee / 100),
	}

	chainDb, l2blockchain, err := arbnode.CreateDefaultBlockChain(stack, l2Genesys)
	if err != nil {
		panic(err)
	}
	node, err := arbnode.CreateNode(stack, chainDb, &nodeConf, l2blockchain, l1client, &deployInfo, l1TransactionOpts)
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
}
