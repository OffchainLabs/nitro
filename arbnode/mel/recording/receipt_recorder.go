package melrecording

import (
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
)

type ReceiptRecorder struct {
	parentChainReader    BlockReader
	parentChainBlockHash common.Hash
	preimages            daprovider.PreimagesMap
	receipts             []*types.Receipt
	logs                 []*types.Log
	trieDB               *triedb.Database
	blockReceiptHash     common.Hash
}

func NewReceiptRecorder(
	parentChainReader BlockReader,
	parentChainBlockHash common.Hash,
	preimages daprovider.PreimagesMap,
) *ReceiptRecorder {
	return &ReceiptRecorder{
		parentChainReader:    parentChainReader,
		parentChainBlockHash: parentChainBlockHash,
		preimages:            preimages,
	}
}

func (lr *ReceiptRecorder) Initialize(ctx context.Context) error {
	block, err := lr.parentChainReader.BlockByHash(ctx, lr.parentChainBlockHash)
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
		receipt, err := lr.parentChainReader.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return fmt.Errorf("error fetching receipt for tx: %v", tx.Hash())
		}
		receipts = append(receipts, receipt)
		lr.logs = append(lr.logs, receipt.Logs...)
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
	lr.receipts = receipts
	lr.trieDB = tdb
	lr.blockReceiptHash = root
	return nil
}

func (lr *ReceiptRecorder) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if lr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if lr.parentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", lr.parentChainBlockHash, parentChainBlockHash)
	}
	// #nosec G115
	if int(txIndex) >= len(lr.receipts) {
		return nil, fmt.Errorf("index out of range: %d", txIndex)
	}
	recordingDB := &TxAndLogsDatabase{
		underlying: lr.trieDB,
		recorder:   daprovider.RecordPreimagesTo(lr.preimages), // RecordingDB will record relevant preimages into tr.preimages
	}
	recordingTDB := triedb.NewDatabase(recordingDB, nil)
	receiptsTrie, err := trie.New(trie.TrieID(lr.blockReceiptHash), recordingTDB)
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
	if _, ok := lr.preimages[arbutil.Keccak256PreimageType]; !ok {
		lr.preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	lr.preimages[arbutil.Keccak256PreimageType][crypto.Keccak256Hash(receiptBytes)] = receiptBytes
	// Fill in the TxIndex (give as input to this method) into the logs so that Tx recording
	// is possible. This field is one of the derived fields of Log hence won't be stored in trie.
	//
	// We use this same trick in validation as well in order to link a tx with its logs
	for _, log := range receipt.Logs {
		log.TxIndex = txIndex
	}
	return receipt.Logs, nil
}

func (lr *ReceiptRecorder) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if lr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if lr.parentChainBlockHash == parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", lr.parentChainBlockHash, parentChainBlockHash)
	}
	return lr.logs, nil
}

func (tr *ReceiptRecorder) GetPreimages() daprovider.PreimagesMap { return tr.preimages }
