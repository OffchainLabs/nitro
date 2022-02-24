package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/util"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("deploying rollup")

	ctx := context.Background()

	l1conn := flag.String("l1conn", "", "l1 connection")
	l1keystore := flag.String("l1keystore", "", "l1 private key store")
	deployAccount := flag.String("l1DeployAccount", "", "l1 seq account to use (default is first account in keystore)")
	wasmmoduleroot := flag.String("wasmmoduleroot", "", "WASM module root hash")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	outfile := flag.String("l1deployment", "deploy.json", "deployment output json file")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	authorizevalidators := flag.Uint64("authorizevalidators", 0, "Number of validators to preemptively authorize")
	flag.Parse()
	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	l1TransactionOpts, err := util.GetTransactOptsFromKeystore(*l1keystore, *deployAccount, *l1passphrase, l1ChainId)
	if err != nil {
		flag.Usage()
		panic(err)
	}

	l1client, err := ethclient.Dial(*l1conn)
	if err != nil {
		flag.Usage()
		panic(err)
	}

	deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1TransactionOpts.From, *authorizevalidators, common.HexToHash(*wasmmoduleroot), time.Minute*5)
	if err != nil {
		flag.Usage()
		panic(err)
	}
	deployData, err := json.Marshal(deployPtr)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(*outfile, deployData, 0600); err != nil {
		panic(err)
	}
}
