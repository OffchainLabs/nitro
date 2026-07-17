// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"math/big"
	"strings"
	"testing"

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
		{"receipt missing EffectiveGasPrice", header, tx, &types.Receipt{}, "missing EffectiveGasPrice"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := BuildFeedMessage(tc.header, tc.tx, tc.receipt)
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
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, EffectiveGasPrice: big.NewInt(1)}

	msg, err := BuildFeedMessage(header, tx, receipt)
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
