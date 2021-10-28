//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("running node")

	devPrivKeyStr := "e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2"
	devPrivKey, err := crypto.HexToECDSA(devPrivKeyStr)
	if err != nil {
		panic(err)
	}
	devAddr := crypto.PubkeyToAddress(devPrivKey.PublicKey)
	log.Info("Dev node funded private key", "priv", devPrivKeyStr)
	log.Info("Funded public address", "addr", devAddr)

	genesisAlloc := make(core.GenesisAlloc)
	genesisAlloc[devAddr] = core.GenesisAccount{
		Balance:    new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(10)),
		Nonce:      0,
		PrivateKey: nil,
	}

	stack, err := arbnode.CreateStack()
	if err != nil {
		panic(err)
	}
	_, err = arbnode.CreateArbBackend(stack, genesisAlloc)
	if err != nil {
		panic(err)
	}

	if err := stack.Start(); err != nil {
		utils.Fatalf("Error starting protocol stack: %v\n", err)
	}

	stack.Wait()
}
