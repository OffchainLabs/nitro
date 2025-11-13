// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbutil"
)

func TestFetchReceiptFromBlock_Multiple(t *testing.T) {
	ctx := context.Background()
	// Creates a block with 42 transactions and receipts.
	numReceipts := 42
	receipts := createTestReceipts(numReceipts)
	hasher := newRecordingHasher()
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)
	header := &types.Header{}
	txes := make([]*types.Transaction, numReceipts)
	for i := 0; i < numReceipts; i++ {
		txes[i] = types.NewTransaction(uint64(i), common.Address{}, big.NewInt(0), 21000, big.NewInt(1), nil) // #nosec G115
	}
	body := &types.Body{
		Transactions: txes,
	}
	blk := types.NewBlock(header, body, receipts, hasher)
	require.Equal(t, blk.ReceiptHash(), receiptsRoot)
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	receiptFetcher := &receiptFetcherForBlock{
		header:           blk.Header(),
		preimageResolver: mockPreimageResolver,
	}
	for i := 0; i < numReceipts; i++ {
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, uint(i)) // #nosec G115
		require.NoError(t, err)
		require.Equal(t, receipts[i].CumulativeGasUsed, receipt.CumulativeGasUsed)
	}
}

type mockPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (m *mockPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimage, exists := m.preimages[hash]; exists {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %s", hash.Hex())
}

// Implements a hasher that captures preimages of hashes as it computes them.
type preimageRecordingHasher struct {
	trie      *trie.StackTrie
	preimages map[common.Hash][]byte
}

func newRecordingHasher() *preimageRecordingHasher {
	h := &preimageRecordingHasher{
		preimages: make(map[common.Hash][]byte),
	}
	// OnTrieNode callback captures all trie nodes.
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		// Deep copy the blob since the callback warns contents may change, so this is required.
		h.preimages[hash] = common.CopyBytes(blob)
	}

	h.trie = trie.NewStackTrie(onTrieNode)
	return h
}

func (h *preimageRecordingHasher) Reset() {
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		h.preimages[hash] = common.CopyBytes(blob)
	}
	h.trie = trie.NewStackTrie(onTrieNode)
}

func (h *preimageRecordingHasher) Update(key, value []byte) error {
	valueHash := crypto.Keccak256Hash(value)
	h.preimages[valueHash] = common.CopyBytes(value)
	return h.trie.Update(key, value)
}

func (h *preimageRecordingHasher) Hash() common.Hash {
	return h.trie.Hash()
}

func (h *preimageRecordingHasher) GetPreimages() map[common.Hash][]byte {
	return h.preimages
}

func createTestReceipts(count int) types.Receipts {
	receipts := make(types.Receipts, count)
	for i := 0; i < count; i++ {
		receipt := &types.Receipt{
			Status:            1,
			CumulativeGasUsed: 50_000 + uint64(i), // #nosec G115
			TxHash:            common.Hash{},
			ContractAddress:   common.Address{},
			Logs:              []*types.Log{},
			BlockHash:         common.BytesToHash([]byte("foobar")),
			BlockNumber:       big.NewInt(100),
			TransactionIndex:  uint(i), // #nosec G115
		}
		receipt.Bloom = types.BytesToBloom(make([]byte, 256))
		receipts[i] = receipt
	}
	return receipts
}
