// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrecording

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

// ReceiptRecorder records preimages corresponding to the receipts of a parent chain block
// needed during the message extraction. These preimages are needed for MEL validation and
// are used in creation of the validation entries by the MEL validator
type ReceiptRecorder struct {
	parentChainReader      BlockReader
	parentChainBlockHash   common.Hash
	parentChainBlockNumber uint64
	recordPreimages        daprovider.PreimageRecorder
	receipts               []*types.Receipt
	logs                   []*types.Log
	relevantLogsTxIndexes  map[uint]struct{}
	trieDB                 *triedb.Database
	blockReceiptHash       common.Hash
}

// NewReceiptRecorder returns ReceiptRecorder that records
// the receipt preimages into the given preimages map
func NewReceiptRecorder(
	parentChainReader BlockReader,
	parentChainBlockHash common.Hash,
	preimages daprovider.PreimagesMap,
) (*ReceiptRecorder, error) {
	if preimages == nil {
		return nil, errors.New("preimages recording destination cannot be nil")
	}
	return &ReceiptRecorder{
		parentChainReader:     parentChainReader,
		parentChainBlockHash:  parentChainBlockHash,
		recordPreimages:       daprovider.RecordPreimagesTo(preimages),
		relevantLogsTxIndexes: make(map[uint]struct{}),
	}, nil
}

// Initialize must be called first to setup the recording trie database and store all the
// block receipts into the triedb. Without this, preimage recording is not possible and
// the other functions will error out if called beforehand
func (rr *ReceiptRecorder) Initialize(ctx context.Context) error {
	block, err := rr.parentChainReader.BlockByHash(ctx, rr.parentChainBlockHash)
	if err != nil {
		return err
	}
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), &triedb.Config{
		Preimages: true,
	})
	receiptsTrie := trie.NewEmpty(tdb)
	var receipts []*types.Receipt
	txs := block.Body().Transactions
	for i, tx := range txs {
		receipt, err := rr.parentChainReader.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return fmt.Errorf("error fetching receipt for tx: %v, blockHash: %v", tx.Hash(), block.Hash())
		}
		receipts = append(receipts, receipt)
		rr.logs = append(rr.logs, receipt.Logs...)
		// #nosec G115
		indexBytes, err := rlp.EncodeToBytes(uint64(i))
		if err != nil {
			return fmt.Errorf("failed to encode index %d: %w", i, err)
		}
		receiptBytes, err := receipt.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal receipt %d: %w", i, err)
		}
		if err := receiptsTrie.Update(indexBytes, receiptBytes); err != nil {
			return fmt.Errorf("failed to update trie at index %d: %w", i, err)
		}
	}
	root, nodes := receiptsTrie.Commit(false)
	if root != block.ReceiptHash() {
		return fmt.Errorf("computed root %s doesn't match header root %s",
			root.Hex(), block.ReceiptHash().Hex())
	}
	if nodes != nil {
		if err := tdb.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), nil); err != nil {
			return fmt.Errorf("failed to commit trie nodes: %w", err)
		}
	}
	if err := tdb.Commit(root, false); err != nil {
		return fmt.Errorf("failed to commit database: %w", err)
	}
	rr.receipts = receipts
	rr.trieDB = tdb
	rr.blockReceiptHash = root
	rr.parentChainBlockNumber = block.NumberU64()
	return nil
}

func (rr *ReceiptRecorder) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if rr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if rr.parentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", rr.parentChainBlockHash, parentChainBlockHash)
	}
	if _, recorded := rr.relevantLogsTxIndexes[txIndex]; recorded {
		return rr.receipts[txIndex].Logs, nil
	}
	// #nosec G115
	if int(txIndex) >= len(rr.receipts) {
		return nil, fmt.Errorf("index out of range: %d", txIndex)
	}
	recordingDB := &TxsAndReceiptsDatabase{
		underlying: rr.trieDB,
		recorder:   rr.recordPreimages, // RecordingDB will record relevant preimages into the given preimagesmap
	}
	recordingTDB := triedb.NewDatabase(recordingDB, nil)
	receiptsTrie, err := trie.New(trie.TrieID(rr.blockReceiptHash), recordingTDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create trie: %w", err)
	}
	indexBytes, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to encode index: %w", err)
	}
	receiptBytes, err := receiptsTrie.Get(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt from trie: %w", err)
	}
	receipt := new(types.Receipt)
	if err = receipt.UnmarshalBinary(receiptBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
	}
	// Add the receipt marshaled binary by hash to the preimages map
	rr.recordPreimages(crypto.Keccak256Hash(receiptBytes), receiptBytes, arbutil.Keccak256PreimageType)
	// Fill in the TxIndex (give as input to this method) into the logs so that Tx recording
	// is possible. This field is one of the derived fields of Log hence won't be stored in trie.
	//
	// We use this same trick in validation as well in order to link a tx with its logs
	for _, log := range receipt.Logs {
		log.TxIndex = txIndex
		log.BlockHash = parentChainBlockHash
		log.BlockNumber = rr.parentChainBlockNumber
	}
	rr.relevantLogsTxIndexes[txIndex] = struct{}{}
	return receipt.Logs, nil
}

func (rr *ReceiptRecorder) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if rr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if rr.parentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", rr.parentChainBlockHash, parentChainBlockHash)
	}
	return rr.logs, nil
}

// CollectTxIndicesPreimage adds the array of relevant tx indexes to the preimages map as a value
// to the key represented by parentChainBlockHash.
func (rr *ReceiptRecorder) CollectTxIndicesPreimage() error {
	var relevantLogsTxIndexes []uint
	for k := range rr.relevantLogsTxIndexes {
		relevantLogsTxIndexes = append(relevantLogsTxIndexes, k)
	}
	var buf bytes.Buffer
	if err := rlp.Encode(&buf, relevantLogsTxIndexes); err != nil {
		return err
	}
	relevantTxIndicesKey := melreplay.RelevantTxIndexesKey(rr.parentChainBlockHash)
	rr.recordPreimages(relevantTxIndicesKey, buf.Bytes(), arbutil.Keccak256PreimageType)
	return nil
}
