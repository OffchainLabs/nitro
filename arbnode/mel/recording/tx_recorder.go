// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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

type BlockReader interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// TransactionRecorder records preimages corresponding to the transactions of a parent chain block
// needed during the message extraction. These preimages are needed for MEL validation and
// is used in creation of the validation entries by the MEL validator
type TransactionRecorder struct {
	parentChainReader    BlockReader
	parentChainBlockHash common.Hash
	recordPreimages      daprovider.PreimageRecorder
	txs                  []*types.Transaction
	trieDB               *triedb.Database
	blockTxHash          common.Hash
}

// NewTransactionRecorder returns TransactionRecorder that records
// the transaction preimages into the given preimages map
func NewTransactionRecorder(
	parentChainReader BlockReader,
	parentChainBlockHash common.Hash,
	preimages daprovider.PreimagesMap,
) (*TransactionRecorder, error) {
	if preimages == nil {
		return nil, errors.New("preimages recording destination cannot be nil")
	}
	return &TransactionRecorder{
		parentChainReader:    parentChainReader,
		parentChainBlockHash: parentChainBlockHash,
		recordPreimages:      daprovider.RecordPreimagesTo(preimages),
	}, nil
}

// Initialize must be called first to setup the recording trie database and store all the
// transactions into the triedb. Without this, preimage recording is not possible and
// the other functions will error out if called beforehand
func (tr *TransactionRecorder) Initialize(ctx context.Context) error {
	block, err := tr.parentChainReader.BlockByHash(ctx, tr.parentChainBlockHash)
	if err != nil {
		return err
	}
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), &triedb.Config{
		Preimages: true,
	})
	txsTrie := trie.NewEmpty(tdb)
	txs := block.Body().Transactions
	for i, tx := range txs {
		// #nosec G115
		indexBytes, err := rlp.EncodeToBytes(uint64(i))
		if err != nil {
			return fmt.Errorf("failed to encode index %d: %w", i, err)
		}
		txBytes, err := tx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal transaction %d: %w", i, err)
		}
		if err := txsTrie.Update(indexBytes, txBytes); err != nil {
			return fmt.Errorf("failed to update trie at index %d: %w", i, err)
		}
	}
	root, nodes := txsTrie.Commit(false)
	if root != block.TxHash() {
		return fmt.Errorf("computed root %s doesn't match header root %s",
			root.Hex(), block.TxHash().Hex())
	}
	if nodes != nil {
		if err := tdb.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), nil); err != nil {
			return fmt.Errorf("failed to commit trie nodes: %w", err)
		}
	}
	if err := tdb.Commit(root, false); err != nil {
		return fmt.Errorf("failed to commit database: %w", err)
	}
	tr.txs = txs
	tr.trieDB = tdb
	tr.blockTxHash = root
	return nil
}

func (tr *TransactionRecorder) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if tr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	// #nosec G115
	if int(log.TxIndex) >= len(tr.txs) {
		return nil, fmt.Errorf("index out of range: %d", log.TxIndex)
	}
	recordingDB := &TxsAndReceiptsDatabase{
		underlying: tr.trieDB,
		recorder:   tr.recordPreimages, // RecordingDB will record relevant preimages into the given preimagesmap
	}
	recordingTDB := triedb.NewDatabase(recordingDB, nil)
	txsTrie, err := trie.New(trie.TrieID(tr.blockTxHash), recordingTDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create trie: %w", err)
	}
	indexBytes, err := rlp.EncodeToBytes(log.TxIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to encode index: %w", err)
	}
	txBytes, err := txsTrie.Get(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from trie: %w", err)
	}
	// Return the tx itself instead of nil
	tx := new(types.Transaction)
	if err = tx.UnmarshalBinary(txBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	// Add the tx marshaled binary by hash to the preimages map
	tr.recordPreimages(crypto.Keccak256Hash(txBytes), txBytes, arbutil.Keccak256PreimageType)
	return tx, nil
}
