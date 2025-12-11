package recording

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
)

type PreimageRecorder struct {
	preimages map[common.Hash][]byte
}

func NewPreimageRecorder() *PreimageRecorder {
	return &PreimageRecorder{
		preimages: make(map[common.Hash][]byte),
	}
}

func (pr *PreimageRecorder) GetPreimages() map[common.Hash][]byte {
	return pr.preimages
}

type RecordingDB struct {
	underlying triedb.Database
	recorder   *PreimageRecorder
}

func (rdb *RecordingDB) Get(key []byte) ([]byte, error) {
	hash := common.BytesToHash(key)
	value, err := rdb.underlying.Node(hash)
	if err != nil {
		return nil, err
	}
	if rdb.recorder != nil {
		rdb.recorder.preimages[hash] = value
	}

	return value, nil
}

func (rdb *RecordingDB) Has(key []byte) (bool, error) {
	hash := common.BytesToHash(key)
	_, err := rdb.underlying.Reader(hash).Node(common.Hash{}, key)
	return err == nil, nil
}

func (rdb *RecordingDB) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB")
}

func (rdb *RecordingDB) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB")
}

type TransactionRecorder struct {
	parentChainReader    melrunner.ParentChainReader
	parentChainBlockHash common.Hash
	preimages            map[common.Hash][]byte
	txs                  []*types.Transaction
}

func NewTransactionRecorder(
	parentChainReader melrunner.ParentChainReader,
	parentChainBlockHash common.Hash,
	preimages map[common.Hash][]byte,
) *TransactionRecorder {
	return &TransactionRecorder{
		parentChainReader:    parentChainReader,
		parentChainBlockHash: parentChainBlockHash,
		preimages:            preimages,
	}
}

func (tr *TransactionRecorder) Initialize(ctx context.Context) error {
	block, err := tr.parentChainReader.BlockByHash(ctx, tr.parentChainBlockHash)
	if err != nil {
		return err
	}
	tdb := triedb.NewDatabase(nil, &triedb.Config{
		Preimages: true,
	})
	txsTrie := trie.NewEmpty(tdb)
	txs := block.Body().Transactions
	for i, tx := range txs {
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
	return nil
}

func (tr *TransactionRecorder) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	if int(log.TxIndex) >= len(tr.txs) {
		return nil, fmt.Errorf("index out of range: %d", log.TxIndex)
	}
	recorder := NewPreimageRecorder()
	recordingDB := &RecordingDB{
		underlying: tl.tdb,
		recorder:   recorder,
	}
	recordingTDB := triedb.NewDatabase(recordingDB, nil)
	txsTrie, err := trie.New(trie.TrieID(log.TxHash), recordingTDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create trie: %w", err)
	}
	indexBytes, err := rlp.EncodeToBytes(log.TxIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to encode index: %w", err)
	}
	if _, err = tr.Get(indexBytes); err != nil {
		return nil, fmt.Errorf("failed to get transaction from trie: %w", err)
	}
	// TODO: Return the tx itself instead of nil, but also add the
	// tx marshaled binary by hash to the preimages map.
	return nil, nil
}
