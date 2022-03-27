//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: seq-coordinator-invalidate [redis url] [signing key] [msg index]\n")
		os.Exit(1)
	}
	redisUrl := os.Args[1]
	signingKey := os.Args[2]
	msgIndex, err := strconv.ParseUint(os.Args[3], 10, 64)
	if err != nil {
		panic("Failed to parse msg index: " + err.Error())
	}
	err = arbnode.StandaloneSeqCoordinatorInvalidateMsgIndex(context.Background(), redisUrl, signingKey, arbutil.MessageIndex(msgIndex))
	if err != nil {
		panic(err)
	}
}
