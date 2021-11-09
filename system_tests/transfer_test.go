//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/offchainlabs/arbstate/arbnode"
)

func TestTransfer(t *testing.T) {
	_, l2info := CreateTestL2(t)

	client := l2info.Client

	ctx := context.Background()

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", 30000, big.NewInt(1e12), nil)

	err := client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	if err != nil {
		t.Fatal(err)
	}

	bal, err := client.BalanceAt(ctx, l2info.GetAddress("Owner"), nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Owner balance is: ", bal)
	bal2, err := client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected recipient balance: ", bal2)
	}

}
