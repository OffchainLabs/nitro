package recording

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

type BlockReader interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
}

type TransactionRecorder struct {
	parentChainReader    BlockReader
	parentChainBlockHash common.Hash
	preimages            map[common.Hash][]byte
	txs                  []*types.Transaction
	trieDB               *triedb.Database
}

func NewTransactionRecorder(
	parentChainReader BlockReader,
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
	tr.trieDB = tdb
	return nil
}

func (tr *TransactionRecorder) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if tr.trieDB == nil {
		return nil, errors.New("TransactionRecorder not initialized")
	}
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	if int(log.TxIndex) >= len(tr.txs) {
		return nil, fmt.Errorf("index out of range: %d", log.TxIndex)
	}
	recorder := NewPreimageRecorder()
	recordingDB := &RecordingDB{
		underlying: tr.trieDB,
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
	txBytes, err := txsTrie.Get(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from trie: %w", err)
	}
	// Return the tx itself instead of nil, but also add the
	// tx marshaled binary by hash to the preimages map.
	tr.preimages[crypto.Keccak256Hash(txBytes)] = txBytes
	tx := new(types.Transaction)
	if err = tx.UnmarshalBinary(txBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	return tx, nil
}

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
	underlying *triedb.Database
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
	_, err := rdb.underlying.Node(hash)
	return err == nil, nil
}
func (rdb *RecordingDB) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB")
}
func (rdb *RecordingDB) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB")
}
func (rdb *RecordingDB) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB")
}
func (rdb *RecordingDB) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *RecordingDB) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *RecordingDB) SyncAncient() error {
	return fmt.Errorf("SyncAncient not supported on recording DB")
}
func (rdb *RecordingDB) TruncateHead(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateHead not supported on recording DB")
}
func (rdb *RecordingDB) TruncateTail(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateTail not supported on recording DB")
}
func (rdb *RecordingDB) Append(kind string, number uint64, item interface{}) error {
	return fmt.Errorf("Append not supported on recording DB")
}
func (rdb *RecordingDB) AppendRaw(kind string, number uint64, item []byte) error {
	return fmt.Errorf("AppendRaw not supported on recording DB")
}
func (rdb *RecordingDB) AncientDatadir() (string, error) {
	return "", fmt.Errorf("AncientDatadir not supported on recording DB")
}
func (rdb *RecordingDB) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, fmt.Errorf("Ancient not supported on recording DB")
}
func (rdb *RecordingDB) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, fmt.Errorf("AncientRange not supported on recording DB")
}
func (rdb *RecordingDB) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, fmt.Errorf("AncientBytes not supported on recording DB")
}
func (rdb *RecordingDB) Ancients() (uint64, error) {
	return 0, fmt.Errorf("Ancients not supported on recording DB")
}
func (rdb *RecordingDB) Tail() (uint64, error) {
	return 0, fmt.Errorf("Tail not supported on recording DB")
}
func (rdb *RecordingDB) AncientSize(kind string) (uint64, error) {
	return 0, fmt.Errorf("AncientSize not supported on recording DB")
}
func (rdb *RecordingDB) Compact(start []byte, limit []byte) error {
	return nil
}
func (rdb *RecordingDB) SyncKeyValue() error {
	return nil
}
func (rdb *RecordingDB) Stat() (string, error) {
	return "", nil
}
func (rdb *RecordingDB) WasmDataBase() ethdb.KeyValueStore {
	return nil
}
func (rdb *RecordingDB) NewBatch() ethdb.Batch {
	return nil
}
func (rdb *RecordingDB) NewBatchWithSize(size int) ethdb.Batch {
	return nil
}
func (rdb *RecordingDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return nil
}
func (rdb *RecordingDB) Close() error {
	return nil
}
