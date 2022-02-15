package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbnode"
)

func main() {
	ctx := context.Background()

	l1conn := flag.String("l1conn", "", "l1 connection")
	l1keyfile := flag.String("l1keyfile", "", "l1 private key file")
	wasmmoduleroot := flag.String("wasmmoduleroot", "", "WASM module root hash")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	flag.Parse()
	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	fileReader, err := os.Open(*l1keyfile)
	if err != nil {
		flag.Usage()
		log.Fatalln("sequencer without valid l1priv key", "err", err)
	}

	l1client, err := ethclient.Dial(*l1conn)
	if err != nil {
		flag.Usage()
		panic(err)
	}
	l1TransactionOpts, err := bind.NewTransactorWithChainID(fileReader, *l1passphrase, l1ChainId)
	if err != nil {
		panic(err)
	}
	deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1TransactionOpts.From, common.HexToHash(*wasmmoduleroot), time.Minute*5)
	if err != nil {
		flag.Usage()
		panic(err)
	}
	deployData, err := json.Marshal(deployPtr)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("deploy.json", deployData, 0600); err != nil {
		panic(err)
	}
}
