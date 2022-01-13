//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"github.com/offchainlabs/arbstate/statetransfer"
)

const SampleRate = 0.00001

func main() {
	nodeUrl := flag.String("nodeurl", "", "URL of classic chain node")
	cachePath := flag.String("cachepath", "cache", "path to state cache directory")
	sampleRate := flag.Float64("samplerate", SampleRate, "fraction of accounts to load")
	flag.Parse()

	var maybeUrl *string
	if *nodeUrl != "" {
		maybeUrl = nodeUrl
	}
	jsonState, err := statetransfer.GetDataFromClassicAsJson(maybeUrl, cachePath, sampleRate)
	if err != nil {
		panic(err)
	}
	fmt.Print(jsonState)
}
