//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("running node")
	stackConf := node.DefaultConfig
	stackConf.DataDir = "./data"
	stackConf.HTTPHost = "localhost"
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stack, err := node.New(&stackConf)
	if err != nil {
		if err != nil {
			utils.Fatalf("Error creating protocol stack: %v\n", err)
		}
	}
	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()
	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[common.HexToAddress("0xE3851AA36a7e4951015Dd4496E4B7237050b3CDd")] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether),
		Nonce:      0,
		PrivateKey: nil,
	}
	nodeConf.Genesis = &core.Genesis{
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
		BaseFee:    big.NewInt(0),
	}
	arbEngine := arbos.Engine{
		IsSequencer: true,
	}
	backend, err := eth.NewAdvanced(stack, &nodeConf, arbEngine)
	if err != nil {
		utils.Fatalf("Error creating backend: %v\n", err)
	}
	stack.RegisterAPIs(tracers.APIs(backend.APIBackend))
	backend.SetEtherbase(common.HexToAddress("0xbE8e5197Acd8597c282D29C066c08A03b657ED08"))
	log.Info("starting stack")
	if err := stack.Start(); err != nil {
		utils.Fatalf("Error starting protocol stack: %v\n", err)
	}

	if err := backend.StartMining(1); err != nil {
		utils.Fatalf("Error starting mining: %v\n", err)
	}
	stack.Wait()
}
