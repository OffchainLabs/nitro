// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	bal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), nil)
	Require(t, err)
	fmt.Println("Owner balance is: ", bal)
	bal2, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if bal2.Cmp(big.NewInt(1e12)) != 0 {
		Fatal(t, "Unexpected recipient balance: ", bal2)
	}
}

func TestP256Verify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = 30
	cleanup := builder.Build(t)
	defer cleanup()
	addr := common.BytesToAddress([]byte{0x01, 0x00})
	got, err := builder.L2.Client.CallContract(ctx, ethereum.CallMsg{
		From:  builder.L2Info.GetAddress("Owner"),
		To:    &addr,
		Gas:   builder.L2Info.TransferGas,
		Data:  common.Hex2Bytes("4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"),
		Value: big.NewInt(1e12),
	}, nil)
	if err != nil {
		t.Fatalf("Calling p256 precompile, unexpected error: %v", err)
	}
	want := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	if !bytes.Equal(got, want) {
		t.Errorf("P256Verify() = %v, want: %v", got, want)
	}
}
