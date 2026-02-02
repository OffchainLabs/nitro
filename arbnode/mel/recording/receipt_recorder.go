package melrecording

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// Implements a hasher that captures preimages of hashes as it computes them.
type preimageRecordingTrieHasher struct {
	trie            *trie.StackTrie
	recordPreimages daprovider.PreimageRecorder
}

func newPreimageRecordingTrieHasher(recordPreimages daprovider.PreimageRecorder) *preimageRecordingTrieHasher {
	h := &preimageRecordingTrieHasher{
		recordPreimages: recordPreimages,
	}
	// OnTrieNode callback captures all trie nodes.
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		// Deep copy the blob since the callback warns contents may change, so this is required.
		recordPreimages(hash, common.CopyBytes(blob), arbutil.Keccak256PreimageType)
	}

	h.trie = trie.NewStackTrie(onTrieNode)
	return h
}

func (h *preimageRecordingTrieHasher) Reset() {
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		h.recordPreimages(hash, common.CopyBytes(blob), arbutil.Keccak256PreimageType)
	}
	h.trie = trie.NewStackTrie(onTrieNode)
}

func (h *preimageRecordingTrieHasher) Update(key, value []byte) error {
	valueHash := crypto.Keccak256Hash(value)
	h.recordPreimages(valueHash, common.CopyBytes(value), arbutil.Keccak256PreimageType)
	return h.trie.Update(key, value)
}

func (h *preimageRecordingTrieHasher) Hash() common.Hash {
	return h.trie.Hash()
}

// recordedLogsFetcher holds the logs of recorded receipt preimages. These preimages are
// needed for MEL validation and is used in creation of the validation entries by the MEL validator
type recordedLogsFetcher struct {
	parentChainBlockHash common.Hash
	receipts             []*types.Receipt
	logs                 []*types.Log
}

// RecordReceipts records preimages of all the receipts in a block and returns a LogsFetcher
// that provides these logs during message extraction
func RecordReceipts(ctx context.Context, parentChainReader BlockReader, parentChainBlockHash common.Hash, preimages daprovider.PreimagesMap) (melextraction.LogsFetcher, error) {
	if preimages == nil {
		return nil, errors.New("preimages recording destination cannot be nil")
	}
	block, err := parentChainReader.BlockByHash(ctx, parentChainBlockHash)
	if err != nil {
		return nil, err
	}
	txs := block.Body().Transactions
	var receipts []*types.Receipt
	var logs []*types.Log
	for _, tx := range txs {
		receipt, err := parentChainReader.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, fmt.Errorf("error fetching receipt for tx: %v, blockHash: %v", tx.Hash(), block.Hash())
		}
		receipts = append(receipts, receipt)
		logs = append(logs, receipt.Logs...)
	}
	hasher := newPreimageRecordingTrieHasher(daprovider.RecordPreimagesTo(preimages))
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)
	if receiptsRoot != block.ReceiptHash() {
		return nil, fmt.Errorf("computed root %s doesn't match header root %s", receiptsRoot.Hex(), block.ReceiptHash().Hex())
	}
	return &recordedLogsFetcher{
		logs:                 logs,
		receipts:             receipts,
		parentChainBlockHash: parentChainBlockHash,
	}, nil
}

func (rr *recordedLogsFetcher) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if rr.parentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", rr.parentChainBlockHash, parentChainBlockHash)
	}
	// #nosec G115
	if int(txIndex) >= len(rr.receipts) {
		return nil, fmt.Errorf("index out of range: %d", txIndex)
	}
	return rr.receipts[txIndex].Logs, nil
}

func (rr *recordedLogsFetcher) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if rr.parentChainBlockHash != parentChainBlockHash {
		return nil, fmt.Errorf("parentChainBlockHash mismatch. expected: %v got: %v", rr.parentChainBlockHash, parentChainBlockHash)
	}
	return rr.logs, nil
}
