package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos"
)

func TestTransfer(t *testing.T) {
	backend, client, ownerKey := CreateTestBackendWithBalance(t)

	ctx := context.Background()

	ownerAddress := crypto.PubkeyToAddress(ownerKey.PublicKey)

	signer := types.NewLondonSigner(arbos.ChainConfig.ChainID)
	user2Key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	user2Address := crypto.PubkeyToAddress(user2Key.PublicKey)

	accesses := types.AccessList{types.AccessTuple{
		Address:     user2Address,
		StorageKeys: []common.Hash{{0}},
	}}

	txdata := &types.DynamicFeeTx{
		ChainID:    arbos.ChainConfig.ChainID,
		Nonce:      0,
		To:         &user2Address,
		Gas:        30000,
		GasFeeCap:  big.NewInt(5e+09),
		GasTipCap:  big.NewInt(2),
		Value:      big.NewInt(1e12),
		AccessList: accesses,
		Data:       []byte{},
	}
	tx := types.NewTx(txdata)
	tx, err = types.SignTx(tx, signer, ownerKey)
	if err != nil {
		t.Fatal(err)
	}

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	WaitForTx(t, tx.Hash(), backend, client, 0)

	bal, err := client.BalanceAt(ctx, ownerAddress, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Owner balance is: ", bal)
	bal2, err := client.BalanceAt(ctx, user2Address, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected recipient balance: ", bal2)
	}
}
