//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/statetransfer"
)

func DirNameFor(dirPath string, blockNum uint64) string {
	return filepath.Join(dirPath, fmt.Sprintf("state.%09d", blockNum))
}

func main() {
	nodeUrl := flag.String("nodeurl", "https://arb1-graph.arbitrum.io/rpc", "URL of classic chain node")
	dataPath := flag.String("dir", "target/data", "path to data directory")
	blockNumParam := flag.Int64("blocknum", -1, "negative for current")
	prevBlockNumInt := flag.Int64("prevblocknum", -1, "-1 for no previous data")
	newAPI := flag.Bool("newapi", false, "use new api")
	blocksOnly := flag.Bool("blocksonly", false, "only query blocks")
	assumePrevValid := flag.Bool("assumeprevvalid", false, "assume previous data is valid")

	flag.Parse()
	ctx := context.Background()

	rpcClient, err := rpc.DialContext(ctx, *nodeUrl)
	if err != nil {
		panic(err)
	}

	var blockNumUint64 uint64
	if *blockNumParam >= 0 {
		blockNumUint64 = uint64(*blockNumParam)
	} else {
		client := ethclient.NewClient(rpcClient)
		curHeader, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			panic(err)
		}
		blockNumUint64 = curHeader.Number.Uint64()
	}

	outDir := DirNameFor(*dataPath, blockNumUint64)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	_, err = os.Stat(filepath.Join(outDir, "header.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	} else if err == nil {
		panic("header.json already exists in output directory")
	}

	var inFileName string = ""
	if *prevBlockNumInt >= 0 {
		inFileName = filepath.Join(DirNameFor(*dataPath, uint64(*prevBlockNumInt)), "header.json")
	}

	err = statetransfer.ReadStateFromClassic(ctx, rpcClient, blockNumUint64, inFileName, outDir, *newAPI, *blocksOnly, *assumePrevValid)
	if err != nil {
		panic(err)
	}
}
