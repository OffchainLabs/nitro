// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestBuildFeedMessageNilInputs(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{})
	header := &types.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1)}
	headerNoBaseFee := &types.Header{Number: big.NewInt(1)}
	receipt := &types.Receipt{}

	tests := []struct {
		name    string
		header  *types.Header
		tx      *types.Transaction
		receipt *types.Receipt
		wantErr string
	}{
		{"nil tx", header, nil, receipt, "nil transaction"},
		{"nil header", nil, tx, receipt, "nil header"},
		{"header missing BaseFee", headerNoBaseFee, tx, receipt, "missing BaseFee"},
		{"nil receipt", header, tx, nil, "nil receipt"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := BuildFeedMessage(tc.header, tc.tx, tc.receipt, false)
			if err == nil {
				t.Fatalf("expected error, got nil (msg=%v)", msg)
			}
			if msg != nil {
				t.Fatalf("expected nil message on error, got %v", msg)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestBuildFeedMessageHappyPath(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{})
	header := &types.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1)}
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful}

	msg, err := BuildFeedMessage(header, tx, receipt, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Transaction.TxHash != tx.Hash().Hex() {
		t.Fatalf("tx hash mismatch: got %s want %s", msg.Transaction.TxHash, tx.Hash().Hex())
	}
	if msg.Transaction.BlockNumber != header.Number.Uint64() {
		t.Fatalf("block number mismatch: got %d want %d", msg.Transaction.BlockNumber, header.Number.Uint64())
	}
}

func TestFeedMessageEffectiveGasPriceByTxType(t *testing.T) {
	baseFee := big.NewInt(100)
	header := &types.Header{Number: big.NewInt(1), BaseFee: baseFee}

	tests := []struct {
		name        string
		tx          *types.Transaction
		collectTips bool
		want        int64
	}{
		{
			"no tips collected -- always base fee",
			types.NewTx(&types.DynamicFeeTx{GasTipCap: big.NewInt(50), GasFeeCap: big.NewInt(500)}),
			false, 100,
		},
		{
			"no tips collected -- internal tx also base fee",
			types.NewTx(&types.ArbitrumInternalTx{ChainId: big.NewInt(1), Data: []byte{}}),
			false, 100,
		},
		{
			"dynamic fee tx pays base fee plus tip",
			types.NewTx(&types.DynamicFeeTx{GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(500)}),
			true, 102,
		},
		{
			"dynamic fee tx tip capped by fee cap",
			types.NewTx(&types.DynamicFeeTx{GasTipCap: big.NewInt(50), GasFeeCap: big.NewInt(120)}),
			true, 120,
		},
		{
			"legacy tx pays its gas price",
			types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(130)}),
			true, 130,
		},
		{
			"access list tx pays its gas price",
			types.NewTx(&types.AccessListTx{GasPrice: big.NewInt(140)}),
			true, 140,
		},
		{
			"arbitrum unsigned tx pays base fee",
			types.NewTx(&types.ArbitrumUnsignedTx{GasFeeCap: big.NewInt(200)}),
			true, 100,
		},
		{
			"arbitrum contract tx pays base fee",
			types.NewTx(&types.ArbitrumContractTx{GasFeeCap: big.NewInt(200)}),
			true, 100,
		},
		{
			"arbitrum retry tx pays base fee",
			types.NewTx(&types.ArbitrumRetryTx{GasFeeCap: big.NewInt(200)}),
			true, 100,
		},
		{
			"arbitrum submit retryable tx pays base fee",
			types.NewTx(&types.ArbitrumSubmitRetryableTx{GasFeeCap: big.NewInt(200)}),
			true, 100,
		},
		{
			"arbitrum deposit tx pays zero",
			types.NewTx(&types.ArbitrumDepositTx{Value: big.NewInt(0)}),
			true, 0,
		},
		{
			"arbitrum internal tx pays zero",
			types.NewTx(&types.ArbitrumInternalTx{ChainId: big.NewInt(1), Data: []byte{}}),
			true, 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := BuildFeedMessage(header, tc.tx, &types.Receipt{}, tc.collectTips)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := msg.Transaction.Receipt.EffectiveGasPrice
			if want := hexutil.EncodeBig(big.NewInt(tc.want)); got != want {
				t.Fatalf("got %s want %s", got, want)
			}
		})
	}
}

func TestBuildFeedMessageEffectiveGasPrice(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(500)})
	header := &types.Header{Number: big.NewInt(1), BaseFee: big.NewInt(100)}

	msg, err := BuildFeedMessage(header, tx, &types.Receipt{EffectiveGasPrice: big.NewInt(7)}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Transaction.Receipt.EffectiveGasPrice != "0x64" { // 100, the base fee
		t.Fatalf("got %s want 0x64", msg.Transaction.Receipt.EffectiveGasPrice)
	}

	msg, err = BuildFeedMessage(header, tx, &types.Receipt{}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Transaction.Receipt.EffectiveGasPrice != "0x66" { // 102, base fee + tip
		t.Fatalf("got %s want 0x66", msg.Transaction.Receipt.EffectiveGasPrice)
	}

	msg, err = BuildFeedMessage(header, tx, &types.Receipt{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Transaction.Receipt.EffectiveGasPrice != "0x64" { // 100, the base fee
		t.Fatalf("got %s want 0x64", msg.Transaction.Receipt.EffectiveGasPrice)
	}
}
