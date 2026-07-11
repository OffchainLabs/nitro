// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// makeTestBlock builds a block whose body holds txCount distinct txs.
func makeTestBlock(txCount int) (*types.Block, types.Transactions) {
	txes := make(types.Transactions, 0, txCount)
	for i := range txCount {
		txes = append(txes, types.NewTx(&types.LegacyTx{Nonce: uint64(i)}))
	}
	block := types.NewBlockWithHeader(&types.Header{}).WithBody(types.Body{Transactions: txes})
	return block, txes
}

func TestEmptyBlockMetadata(t *testing.T) {
	for _, tc := range []struct {
		txCount     int
		expectedLen int
	}{
		{txCount: 0, expectedLen: 1},
		{txCount: 1, expectedLen: 2},
		{txCount: 8, expectedLen: 2},
		{txCount: 9, expectedLen: 3},
		{txCount: 16, expectedLen: 3},
	} {
		block, _ := makeTestBlock(tc.txCount)
		metadata := emptyBlockMetadata(block)
		if len(metadata) != tc.expectedLen {
			t.Errorf("txCount %d: expected metadata length %d, got %d", tc.txCount, tc.expectedLen, len(metadata))
		}
		if !bytes.Equal(metadata, make([]byte, tc.expectedLen)) {
			t.Errorf("txCount %d: expected all-zero metadata, got %v", tc.txCount, metadata)
		}
	}
}

func TestBlockMetadataFromSequencedTxes(t *testing.T) {
	block, txes := makeTestBlock(10)
	for _, tc := range []struct {
		name     string
		results  []TxResult
		expected common.BlockMetadata
	}{
		{
			name:     "no results",
			results:  nil,
			expected: common.BlockMetadata{0, 0, 0},
		},
		{
			name: "no timeboosted txs",
			results: []TxResult{
				{Tx: txes[0]},
				{Tx: txes[1]},
			},
			expected: common.BlockMetadata{0, 0, 0},
		},
		{
			name: "timeboosted txs in both metadata bytes",
			results: []TxResult{
				{Tx: txes[1], Timeboosted: true},
				{Tx: txes[2]},
				{Tx: txes[8], Timeboosted: true},
			},
			expected: common.BlockMetadata{0, 1 << 1, 1 << 0},
		},
		{
			name: "errored timeboosted tx is skipped",
			results: []TxResult{
				{Tx: txes[1], Err: errors.New("nonce too low"), Timeboosted: true},
				{Tx: txes[2], Timeboosted: true},
			},
			expected: common.BlockMetadata{0, 1 << 2, 0},
		},
		{
			name: "timeboosted tx not in the block is ignored",
			results: []TxResult{
				{Tx: types.NewTx(&types.LegacyTx{Nonce: 999}), Timeboosted: true},
			},
			expected: common.BlockMetadata{0, 0, 0},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata := blockMetadataFromSequencedTxes(block, tc.results)
			if !bytes.Equal(metadata, tc.expected) {
				t.Errorf("expected metadata %v, got %v", tc.expected, metadata)
			}
		})
	}
}

// TestBlockMetadataBitPositions checks the documented bit layout: tx at block
// index i maps to metadata[1+i/8] bit (i%8).
func TestBlockMetadataBitPositions(t *testing.T) {
	block, txes := makeTestBlock(10)
	for i, tx := range txes {
		metadata := blockMetadataFromSequencedTxes(block, []TxResult{{Tx: tx, Timeboosted: true}})
		if metadata[1+i/8]&(1<<(i%8)) == 0 {
			t.Errorf("tx %d: expected bit %d of byte %d to be set, got %v", i, i%8, 1+i/8, metadata)
		}
		bitsSet := 0
		for _, b := range metadata[1:] {
			for ; b != 0; b &= b - 1 {
				bitsSet++
			}
		}
		if bitsSet != 1 {
			t.Errorf("tx %d: expected exactly one bit set, got %v", i, metadata)
		}
	}
}
