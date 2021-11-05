//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

func TestRedeemNonExistentRetryable(t *testing.T) {
	arbstate.RequireHookedGeth()
	rand.Seed(time.Now().UTC().UnixNano())

	arbRetryableAddress := common.HexToAddress("0x6e")

	backend, l2info := CreateTestL2(t)
	client := ClientForArbBackend(t, backend)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(arbRetryableAddress, client)
	if err != nil {
		t.Fatal(err)
	}
	ownerOps := l2info.GetDefaultTransactOpts("Owner")

	ctx := context.Background()

	tx, err := arbRetryableTx.Redeem(&ownerOps, [32]byte{})
	failOnError(t, err, "Error executing redeem")

	time.Sleep(4 * time.Millisecond) // allow some time for the receipt to show up
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	failOnError(t, err, "Error getting receipt")
	if receipt.Status != 0 {
		t.Fatal("redeem of non-existent retryable reported success")
	}
}
