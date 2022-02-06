//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func JsonFileNameFor(dirPath string, blockNum uint64) string {
	return path.Join(dirPath, fmt.Sprintf("state.%09d.json", blockNum))
}

func main() {
	nodeUrl := flag.String("nodeurl", "https://arb1-graph.arbitrum.io/rpc", "URL of classic chain node")
	dataPath := flag.String("dir", "target/data", "path to data directory")
	blockNumParam := flag.Int64("blocknum", -1, "negative for current")
	prevBlockNumInt := flag.Int64("prevblock", -1, "-1 for no previous data")

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

	outFile, err := os.OpenFile(JsonFileNameFor(*dataPath, blockNumUint64), os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		panic(err)
	}

	var inFile io.Reader = nil
	if *prevBlockNumInt >= 0 {
		inFile, err = os.OpenFile(JsonFileNameFor(*dataPath, uint64(*prevBlockNumInt)), os.O_RDONLY, 0664)
		if err != nil {
			panic(err)
		}
	}

	err = statetransfer.ReadStateFromClassic(ctx, rpcClient, blockNumUint64, inFile, outFile, true)
	if err != nil {
		panic(err)
	}
}
