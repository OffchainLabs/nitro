package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbos"
)

func TestTransfer(t *testing.T) {
	backend, l2info, _, _ := CreateTestBackendWithBalance(t)

	client := ClientForArbBackend(t, backend)

	ctx := context.Background()

	l2info.GenerateAccount("User2")

	accesses := types.AccessList{types.AccessTuple{
		Address:     l2info.GetAddress("User2"),
		StorageKeys: []common.Hash{{0}},
	}}

	l2addr := l2info.GetAddress("User2")
	txdata := &types.DynamicFeeTx{
		ChainID:    arbos.ChainConfig.ChainID,
		Nonce:      0,
		To:         &l2addr,
		Gas:        30000,
		GasFeeCap:  big.NewInt(5e+09),
		GasTipCap:  big.NewInt(2),
		Value:      big.NewInt(1e12),
		AccessList: accesses,
		Data:       []byte{},
	}
	tx := l2info.SignTxAs("Owner", txdata)

	err := client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	WaitForTx(t, tx.Hash(), backend, client, 0)

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
