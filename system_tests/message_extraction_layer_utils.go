package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// TODO: Code from cmd/mel-replay and cmd/replay packages for verification of preimages, should be deleted once we have validation wired
type blobPreimageReader struct {
	preimages daprovider.PreimagesMap
}

func (r *blobPreimageReader) Initialize(ctx context.Context) error { return nil }

func (r *blobPreimageReader) GetBlobs(
	ctx context.Context,
	batchBlockHash common.Hash,
	versionedHashes []common.Hash,
) ([]kzg4844.Blob, error) {
	var blobs []kzg4844.Blob
	for _, h := range versionedHashes {
		var blob kzg4844.Blob
		if _, ok := r.preimages[arbutil.EthVersionedHashPreimageType]; !ok {
			return nil, errors.New("no blobs found in preimages")
		}
		preimage, ok := r.preimages[arbutil.EthVersionedHashPreimageType][h]
		if !ok {
			return nil, errors.New("no blobs found in preimages")
		}
		if len(preimage) != len(blob) {
			return nil, fmt.Errorf("for blob %v got back preimage of length %v but expected blob length %v", h, len(preimage), len(blob))
		}
		copy(blob[:], preimage)
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

type testPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (r *testPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimageType != arbutil.Keccak256PreimageType {
		return nil, fmt.Errorf("unsupported preimageType: %d", preimageType)
	}
	if preimage, ok := r.preimages[hash]; ok {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %v", hash)
}

type preimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type delayedMessageDatabase struct {
	preimageResolver preimageResolver
}

func (d *delayedMessageDatabase) ReadDelayedMessage(
	ctx context.Context,
	state *mel.State,
	msgIndex uint64,
) (*mel.DelayedInboxMessage, error) {
	originalMsgIndex := msgIndex
	totalMsgsSeen := state.DelayedMessagesSeen
	if msgIndex >= totalMsgsSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, totalMsgsSeen)
	}
	treeSize := nextPowerOfTwo(totalMsgsSeen)
	merkleDepth := bits.TrailingZeros64(treeSize)

	// Start traversal from root, which is the delayed messages seen root.
	merkleRoot := state.DelayedMessagesSeenRoot
	currentHash := merkleRoot
	currentDepth := merkleDepth

	// Traverse down the Merkle tree to find the leaf at the given index.
	for currentDepth > 0 {
		// Resolve the preimage to get left and right children.
		result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
		if err != nil {
			return nil, err
		}
		if len(result) != 64 {
			return nil, fmt.Errorf("invalid preimage result length: %d, wanted 64", len(result))
		}
		// Split result into left and right halves.
		mid := len(result) / 2
		left := result[:mid]
		right := result[mid:]

		// Calculate which subtree contains our index.
		subtreeSize := uint64(1) << (currentDepth - 1)
		if msgIndex < subtreeSize {
			// Go left.
			currentHash = common.BytesToHash(left)
		} else {
			// Go right.
			currentHash = common.BytesToHash(right)
			msgIndex -= subtreeSize
		}
		currentDepth--
	}
	// At this point, currentHash should be the hash of the delayed message.
	delayedMsgBytes, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
	if err != nil {
		return nil, err
	}
	delayedMessage := new(mel.DelayedInboxMessage)
	if err = rlp.Decode(bytes.NewBuffer(delayedMsgBytes), &delayedMessage); err != nil {
		return nil, fmt.Errorf("failed to decode delayed message at index %d: %w", originalMsgIndex, err)
	}
	return delayedMessage, nil
}

func nextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len64(n)
}

type txFetcherForBlock struct {
	header           *types.Header
	preimageResolver preimageResolver
}

// TransactionByLog fetches the tx for a specific transaction index by walking
// the tx trie of the block header. It uses the preimage resolver to fetch the preimages
// of the trie nodes as needed.
func (tf *txFetcherForBlock) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	tx, err := fetchObjectFromTrie[types.Transaction](tf.header.TxHash, log.TxIndex, tf.preimageResolver)
	if err != nil {
		return nil, err
	}
	return tx, err
}

type receiptFetcherForBlock struct {
	header           *types.Header
	preimageResolver preimageResolver
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
	}
	return receipt.Logs, nil
}

// LogsForBlockHash first gets the txIndexes corresponding to the relevant logs by reading
// the key `parentChainBlockHash` from the preimages and then fetches logs for each of these txIndexes
func (rf *receiptFetcherForBlock) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	if rf.header.Hash() != parentChainBlockHash {
		return nil, errors.New("parentChainBlockHash mismatch")
	}
	txIndexData, err := rf.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, rf.header.Hash())
	if err != nil {
		return nil, err
	}
	var txIndexes []uint
	if err := rlp.DecodeBytes(txIndexData, &txIndexes); err != nil {
		return nil, err
	}
	var relevantLogs []*types.Log
	for _, txIndex := range txIndexes {
		logs, err := rf.LogsForTxIndex(ctx, parentChainBlockHash, txIndex)
		if err != nil {
			return nil, err
		}
		relevantLogs = append(relevantLogs, logs...)
	}
	return relevantLogs, nil
}

// Fetches a specific object at index from a block's Receipt/Tx trie by navigating its
// Merkle Patricia Trie structure. It uses the preimage resolver to fetch preimages
// of trie nodes as needed, and determines how to navigate depending on the structure of the trie nodes.
func fetchObjectFromTrie[T any](root common.Hash, index uint, preimageResolver preimageResolver) (*T, error) {
	var empty *T
	currentNodeHash := root
	currentPath := []byte{} // Track nibbles consumed so far.
	receiptKey, err := rlp.EncodeToBytes(index)
	if err != nil {
		return empty, err
	}
	targetNibbles := keyToNibbles(receiptKey)
	for {
		nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentNodeHash)
		if err != nil {
			return empty, err
		}
		var node []any
		if err = rlp.DecodeBytes(nodeData, &node); err != nil {
			return empty, fmt.Errorf("failed to decode RLP node: %w", err)
		}
		switch len(node) {
		case 17:
			// We hit a branch node, which has 16 children and a value.
			if len(currentPath) == len(targetNibbles) {
				// A branch node's 17th item could be the value, so we check if it contains the receipt.
				if valueBytes, ok := node[16].([]byte); ok && len(valueBytes) > 0 {
					// This branch node has the actual value as the last item, so we decode the receipt
					return decodeBinary[T](valueBytes)
				}
				return empty, fmt.Errorf("no receipt found at target key")
			}
			// Get the next nibble to follow.
			targetNibble := targetNibbles[len(currentPath)]
			childData, ok := node[targetNibble].([]byte)
			if !ok || len(childData) == 0 {
				return empty, fmt.Errorf("no child at nibble %d", targetNibble)
			}
			// Move to the child node, which is the next hash we have to navigate.
			currentNodeHash = common.BytesToHash(childData)
			currentPath = append(currentPath, targetNibble)
		case 2:
			keyPath, ok := node[0].([]byte)
			if !ok {
				return empty, fmt.Errorf("invalid key path in node")
			}
			key := extractKeyNibbles(keyPath)
			expectedPath := make([]byte, 0)
			expectedPath = append(expectedPath, currentPath...)
			expectedPath = append(expectedPath, key...)

			// Check if it is a leaf or extension node.
			leaf, err := isLeaf(keyPath)
			if err != nil {
				return empty, err
			}
			if leaf {
				// Check that the keyPath matches the target nibbles,
				// otherwise, the receipt does not exist in the trie.
				if !bytes.Equal(expectedPath, targetNibbles) {
					return empty, fmt.Errorf("leaf key does not match target nibbles")
				}
				rawData, ok := node[1].([]byte)
				if !ok {
					return empty, fmt.Errorf("invalid receipt data in leaf node")
				}
				return decodeBinary[T](rawData)
			}
			// If the node is not a leaf node, it is an extension node.
			// Check if our target key matches this extension path.
			if len(expectedPath) > len(targetNibbles) || !bytes.Equal(expectedPath, targetNibbles[:len(expectedPath)]) {
				return empty, fmt.Errorf("extension path mismatch")
			}
			nextNodeBytes, ok := node[1].([]byte)
			if !ok {
				return empty, fmt.Errorf("invalid next node in extension")
			}
			// We navigate to the next node in the trie.
			currentNodeHash = common.BytesToHash(nextNodeBytes)
			currentPath = expectedPath
		default:
			return empty, fmt.Errorf("invalid node structure: unexpected length %d", len(node))
		}
	}
}

// Converts a byte slice key into a slice of nibbles (4-bit values).
// Keys are encoded in big endian format, which is required by Ethereum MPTs.
func keyToNibbles(key []byte) []byte {
	nibbles := make([]byte, len(key)*2)
	for i, b := range key {
		nibbles[i*2] = b >> 4
		nibbles[i*2+1] = b & 0x0f
	}
	return nibbles
}

// Extracts the key nibbles from a key path, handling odd/even length cases.
func extractKeyNibbles(keyPath []byte) []byte {
	if len(keyPath) == 0 {
		return nil
	}
	nibbles := keyToNibbles(keyPath)
	if nibbles[0]&1 != 0 {
		return nibbles[1:]
	}
	return nibbles[2:]
}

func isLeaf(keyPath []byte) (bool, error) {
	firstByte := keyPath[0]
	firstNibble := firstByte >> 4
	// 2 or 3 indicates leaf, while 0 or 1 indicates extension nodes in the Ethereum MPT specification.
	if firstNibble > 3 {
		return false, errors.New("first nibble cannot be greater than 3")
	}
	return firstNibble >= 2, nil
}

func decodeBinary[T any](data []byte) (*T, error) {
	var empty *T
	if len(data) == 0 {
		return empty, errors.New("empty data cannot be decoded")
	}
	v := new(T)
	u, ok := any(v).(interface{ UnmarshalBinary([]byte) error })
	if !ok {
		return empty, errors.New("decodeBinary is called on a type that doesnt implement UnmarshalBinary")
	}
	if err := u.UnmarshalBinary(data); err != nil {
		return empty, err
	}
	return v, nil
}
