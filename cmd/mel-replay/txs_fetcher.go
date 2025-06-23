package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

type txsFetcherForBlock struct {
	header           *types.Header
	preimageResolver preimageResolver
}

func (tf *txsFetcherForBlock) TransactionsByHeader(
	ctx context.Context,
	parentChainHeaderHash common.Hash,
) (types.Transactions, error) {
	preimageDB := &DB{
		resolver: tf.preimageResolver,
	}
	tdb := triedb.NewDatabase(preimageDB, nil)
	tr, err := trie.New(trie.TrieID(tf.header.TxHash), tdb)
	if err != nil {
		panic(err)
	}
	entries, indices := tf.collectTrieEntries(tr)
	rawTxs := tf.reconstructOrderedData(entries, indices)
	return tf.decodeTransactionData(rawTxs)
}

func (btr *txsFetcherForBlock) collectTrieEntries(txTrie *trie.Trie) ([][]byte, []uint64) {
	nodeIterator, iterErr := txTrie.NodeIterator(nil)
	if iterErr != nil {
		panic(iterErr)
	}

	var rawValues [][]byte
	var indexKeys []uint64

	for nodeIterator.Next(true) {
		if !nodeIterator.Leaf() {
			continue
		}

		leafKey := nodeIterator.LeafKey()
		var decodedIndex uint64

		decodeErr := rlp.DecodeBytes(leafKey, &decodedIndex)
		if decodeErr != nil {
			panic(fmt.Errorf("key decoding error: %w", decodeErr))
		}

		indexKeys = append(indexKeys, decodedIndex)
		rawValues = append(rawValues, nodeIterator.LeafBlob())
	}

	return rawValues, indexKeys
}

func (btr *txsFetcherForBlock) reconstructOrderedData(rawValues [][]byte, indices []uint64) []hexutil.Bytes {
	orderedData := make([]hexutil.Bytes, len(rawValues))
	for position, index := range indices {
		if index >= uint64(len(rawValues)) {
			panic(fmt.Sprintf("index out of bounds: %d", index))
		}
		if orderedData[index] != nil {
			panic(fmt.Sprintf("index collision detected: %d", index))
		}
		orderedData[index] = rawValues[position]
	}
	return orderedData
}

func (btr *txsFetcherForBlock) decodeTransactionData(encodedData []hexutil.Bytes) (types.Transactions, error) {
	transactionList := make(types.Transactions, 0, len(encodedData))
	for _, encodedTx := range encodedData {
		decodedTx := new(types.Transaction)
		if decodeErr := decodedTx.UnmarshalBinary(encodedTx); decodeErr != nil {
			return nil, fmt.Errorf("transaction decoding failed: %w", decodeErr)
		}
		transactionList = append(transactionList, decodedTx)
	}
	return transactionList, nil
}
