package main

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/stretchr/testify/require"
)

func TestFetchTransactionsForBlockHeader(t *testing.T) {
	ctx := context.Background()
	total := 42
	txes := make([]*types.Transaction, total)
	for i := 0; i < total; i++ {
		txes[i] = types.NewTransaction(uint64(i), common.Address{}, big.NewInt(0), 21000, big.NewInt(1), nil)
	}
	hasher := newRecordingHasher()
	txsRoot := types.DeriveSha(types.Transactions(txes), hasher)
	header := &types.Header{
		TxHash: txsRoot,
	}
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	txsFetcher := &txsFetcherForBlock{
		header:           header,
		preimageResolver: mockPreimageResolver,
	}
	fetched, err := txsFetcher.TransactionsByHeader(ctx, header.Hash())
	require.NoError(t, err)
	require.Len(t, fetched, total)
	for i, tx := range fetched {
		require.Equal(t, txes[i].Hash(), tx.Hash())
		require.Equal(t, uint64(i), tx.Nonce())
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
