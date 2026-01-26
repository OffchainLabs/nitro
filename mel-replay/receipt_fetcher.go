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

	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type relevantTxIndicesFetcher interface {
	FetchRelevantTxIndices(ctx context.Context, parentChainBlockHash common.Hash) ([]uint, error)
}

type receiptFetcherForBlock struct {
	header           *types.Header
	preimageResolver PreimageResolver
	txIndicesFetcher relevantTxIndicesFetcher
}

func NewLogsFetcher(
	header *types.Header,
	preimageResolver PreimageResolver,
	txIndicesFetcher relevantTxIndicesFetcher,
) melextraction.LogsFetcher {
	return &receiptFetcherForBlock{header, preimageResolver, txIndicesFetcher}
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

// LogsForBlockHash first gets the txIndexes corresponding to the relevant logs by reading
// the key `parentChainBlockHash` from the preimages and then fetches logs for each of these txIndexes
func (rf *receiptFetcherForBlock) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if rf.header.Hash() != parentChainBlockHash {
		return nil, errors.New("parentChainBlockHash mismatch")
	}
	txIndices, err := rf.txIndicesFetcher.FetchRelevantTxIndices(ctx, parentChainBlockHash)
	if err != nil {
		return nil, err
	}
	var relevantLogs []*types.Log
	for _, txIndex := range txIndices {
		logs, err := rf.LogsForTxIndex(ctx, parentChainBlockHash, txIndex)
		if err != nil {
			return nil, err
		}
		relevantLogs = append(relevantLogs, logs...)
	}
	return relevantLogs, nil
}

// TODO: LogsForBlockHashAllLogs is kept, in case we go with an implementation of returning all logs present in a block
func (rf *receiptFetcherForBlock) LogsForBlockHashAllLogs(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
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
	entries, indices := collectTrieEntries(receiptsTrie)
	rawReceipts := reconstructOrderedData(entries, indices)
	receipts, err := decodeReceiptsData(rawReceipts)
	if err != nil {
		return nil, err
	}
	var relevantLogs []*types.Log
	for _, receipt := range receipts {
		relevantLogs = append(relevantLogs, receipt.Logs...)
	}
	return relevantLogs, nil
}

func collectTrieEntries(txTrie *trie.Trie) ([][]byte, []uint64) {
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
