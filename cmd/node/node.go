//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"flag"
	"math/big"
	"os"

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
	l1bridge := flag.String("l1bridge", "", "l1 bridge address (required if using l1 and !l1deploy)")
	l1inbox := flag.String("l1inbox", "", "l1 inbox address (required if using l1 and !l1deploy)")
	l1seqinbox := flag.String("l1seqinbox", "", "l1 sequencer inbox address (required if using l1 and !l1deploy)")
	l1deployedAt := flag.Uint64("l1deployedat", 0, "l1 deployed at (required if using l1 and !l1deploy)")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	forwardingtarget := flag.String("forwardingtarget", "", "transaction forwarding target URL (empty if sequencer)")

	keystorepath := flag.String("keystore", "", "dir for keystore")
	keystorepassphrase := flag.String("passphrase", "passphrase", "passphrase for keystore")
	httphost := flag.String("httphost", "localhost", "http host")
	httpPort := flag.Int("httpport", 7545, "http port")
	wshost := flag.String("wshost", "localhost", "websocket host")
	wsport := flag.Int("wsport", 7546, "websocket port")
	wsexposeall := flag.Bool("wsexposeall", false, "expose private api via websocket")
	flag.Parse()

	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	nodeConf := arbnode.NodeConfigDefault
	nodeConf.ForwardingTarget = *forwardingtarget
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
			deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1Addr)
			if err != nil {
				flag.Usage()
				panic(err)
			}
			deployInfo = *deployPtr
		} else {
			paramsOk := common.IsHexAddress(*l1inbox) && common.IsHexAddress(*l1bridge) && common.IsHexAddress(*l1seqinbox) && (*l1deployedAt != 0)
			if !paramsOk {
				flag.Usage()
				panic("not deploying, and missing required deploy info")
			}
			deployInfo.Bridge = common.HexToAddress(*l1bridge)
			deployInfo.Inbox = common.HexToAddress(*l1inbox)
			deployInfo.SequencerInbox = common.HexToAddress(*l1seqinbox)
			deployInfo.DeployedAt = *l1deployedAt
		}
	} else {
		*l1deploy = false
	}

	stackConf := node.DefaultConfig
	stackConf.DataDir = "" // TODO: parametrise. Support resuming after shutdown..
	stackConf.HTTPHost = *httphost
	stackConf.HTTPPort = *httpPort
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stackConf.WSHost = *wshost
	stackConf.WSPort = *wsport
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

	genesisAlloc := make(core.GenesisAlloc)
	genesisAlloc[devAddr] = core.GenesisAccount{
		Balance:    new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(10)),
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
