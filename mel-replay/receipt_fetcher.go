// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melreplay

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type receiptFetcherForBlock struct {
	header           *types.Header
	preimageResolver PreimageResolver
}

func NewLogsFetcher(
	header *types.Header,
	preimageResolver PreimageResolver,
) melextraction.LogsFetcher {
	return &receiptFetcherForBlock{header, preimageResolver}
}

// LogsForTxIndex fetches logs for a specific transaction index by walking
// the receipt trie of the block header. It uses the preimage resolver to fetch the preimages
// of the trie nodes as needed.
func (rf *receiptFetcherForBlock) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if rf.header.Hash() != parentChainBlockHash {
		return nil, errors.New("parentChainBlockHash mismatch")
	}
	receipt, err := fetchObjectFromTrie[types.Receipt](rf.header.ReceiptHash, txIndex, rf.preimageResolver)
	if err != nil {
		return nil, err
	}
	// This is needed to enable fetching corresponding tx from the txFetcher
	for _, log := range receipt.Logs {
		log.TxIndex = txIndex
		log.BlockHash = rf.header.Hash()
		log.BlockNumber = rf.header.Number.Uint64()
	}
	return receipt.Logs, nil
}

// TODO: LogsForBlockHashAllLogs is kept, in case we go with an implementation of returning all logs present in a block
func (rf *receiptFetcherForBlock) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if rf.header.Hash() != parentChainBlockHash {
		return nil, errors.New("parentChainBlockHash mismatch")
	}
	preimageDB := &DB{
		resolver: rf.preimageResolver,
	}
	tdb := triedb.NewDatabase(preimageDB, nil)
	receiptsTrie, err := trie.New(trie.TrieID(rf.header.ReceiptHash), tdb)
	if err != nil {
		return nil, err
	}
	entries, indices, maxIndex := collectTrieEntries(receiptsTrie)
	if len(indices) != 0 && len(indices) != int(maxIndex)+1 {
		return nil, fmt.Errorf("incorrect number of receipts in trie, want: %d, have: %d", int(maxIndex)+1, len(indices))
	}
	rawReceipts := reconstructOrderedData(entries, indices)
	receipts, err := decodeReceiptsData(rawReceipts)
	if err != nil {
		return nil, err
	}
	var relevantLogs []*types.Log
	for i, receipt := range receipts {
		// This is needed to enable fetching corresponding tx from the txFetcher
		for _, log := range receipt.Logs {
			log.TxIndex = uint(i)
			log.BlockHash = rf.header.Hash()
			log.BlockNumber = rf.header.Number.Uint64()
			relevantLogs = append(relevantLogs, log)
		}
	}
	return relevantLogs, nil
}

func collectTrieEntries(txTrie *trie.Trie) ([][]byte, []uint64, uint64) {
	nodeIterator, iterErr := txTrie.NodeIterator(nil)
	if iterErr != nil {
		panic(iterErr)
	}

	var rawValues [][]byte
	var indexKeys []uint64
	var maxIndex uint64

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
		maxIndex = max(maxIndex, decodedIndex)
		rawValues = append(rawValues, nodeIterator.LeafBlob())
	}

	return rawValues, indexKeys, maxIndex
}

func reconstructOrderedData(rawValues [][]byte, indices []uint64) []hexutil.Bytes {
	orderedData := make([]hexutil.Bytes, len(rawValues))
	for position, index := range indices {
		if index >= uint64(len(rawValues)) {
			panic(fmt.Sprintf("index out of bounds: %d", index-1))
		}
		if orderedData[index] != nil {
			panic(fmt.Sprintf("index collision detected: %d", index-1))
		}
		orderedData[index] = rawValues[position]
	}
	return orderedData
}

func decodeReceiptsData(encodedData []hexutil.Bytes) (types.Receipts, error) {
	receiptList := make(types.Receipts, 0, len(encodedData))
	for _, encodedReceipt := range encodedData {
		decodedReceipt := new(types.Receipt)
		if decodeErr := decodedReceipt.UnmarshalBinary(encodedReceipt); decodeErr != nil {
			return nil, fmt.Errorf("receipt decoding failed: %w", decodeErr)
		}
		receiptList = append(receiptList, decodedReceipt)
	}
	return receiptList, nil
}
