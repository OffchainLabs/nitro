//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"fmt"
	"github.com/offchainlabs/arbstate/statetransfer"
)


const SampleRate = 0.00001

func main() {
	sampleRate := SampleRate
	jsonState, err := statetransfer.GetDataFromClassicAsJson(nil, &sampleRate)
	if err != nil {
		panic("failed to get state from Arbitrum One")
	}
	fmt.Print(jsonState)
}
