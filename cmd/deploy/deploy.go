// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/offchainlabs/nitro/validator"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/util"
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
	wasmrootpath := flag.String("wasmrootpath", "", "path to machine folders")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	outfile := flag.String("l1deployment", "deploy.json", "deployment output json file")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	l2ChainIdUint := flag.Uint64("l2chainid", params.ArbitrumTestnetChainConfig().ChainID.Uint64(), "L2 chain ID")
	authorizevalidators := flag.Uint64("authorizevalidators", 0, "Number of validators to preemptively authorize")
	flag.Parse()
	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)
	l2ChainId := new(big.Int).SetUint64(*l2ChainIdUint)

	wallet := conf.WalletConfig{
		Pathname:     *l1keystore,
		Account:      *deployAccount,
		PasswordImpl: *l1passphrase,
	}
	l1TransactionOpts, err := util.GetTransactOptsFromWallet(&wallet, l1ChainId)
	if err != nil {
		flag.Usage()
		log.Error("error reading keystore")
		panic(err)
	}

	l1client, err := ethclient.Dial(*l1conn)
	if err != nil {
		flag.Usage()
		log.Error("error creating l1client")
		panic(err)
	}

	machineConfig := validator.DefaultNitroMachineConfig
	machineConfig.RootPath = *wasmrootpath

	deployPtr, err := arbnode.DeployOnL1(ctx, l1client, l1TransactionOpts, l1TransactionOpts.From, *authorizevalidators, common.HexToHash(*wasmmoduleroot), l2ChainId, arbnode.DefaultL1ReaderConfig, machineConfig)
	if err != nil {
		flag.Usage()
		log.Error("error deploying on l1")
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
