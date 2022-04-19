//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/statetransfer"
)

func DirNameFor(dirPath string, blockNum uint64) string {
	return path.Join(dirPath, fmt.Sprintf("state.%09d", blockNum))
}

func main() {
	nodeUrl := flag.String("nodeurl", "https://arb1-graph.arbitrum.io/rpc", "URL of classic chain node")
	dataPath := flag.String("dir", "target/data", "path to data directory")
	blockNumParam := flag.Int64("blocknum", -1, "negative for current")
	prevBlockNumInt := flag.Int64("prevblock", -1, "-1 for no previous data")
	newAPI := flag.Bool("newapi", false, "use new api")

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
	entries, err := os.ReadDir(outDir)
	if err != nil {
		panic(err)
	}
	if len(entries) > 0 {
		panic("out dir not empty")
	}
	var inFileName string = ""
	if *prevBlockNumInt >= 0 {
		inFileName = path.Join(DirNameFor(*dataPath, uint64(*prevBlockNumInt)), "header.json")
	}

	err = statetransfer.ReadStateFromClassic(ctx, rpcClient, blockNumUint64, inFileName, outDir, !*newAPI)
	if err != nil {
		panic(err)
	}
}
