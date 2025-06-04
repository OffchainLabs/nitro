package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/stretchr/testify/require"
)

func TestMPTNavigation(t *testing.T) {
	receipts := createTestReceipts(1)
	hasher := newRecordingHasher()
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	fetcher(t, receiptsRoot, mockPreimageResolver, []byte{0x80})
}

func fetcher(
	t *testing.T,
	receiptsRoot common.Hash,
	preimageResolver preimageResolver,
	receiptKey []byte,
) (*types.Receipt, error) {
	currentNodeHash := receiptsRoot
	currentPath := []byte{} // Track nibbles consumed so far
	targetNibbles := keyToNibbles(receiptKey)
	for {
		nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentNodeHash)
		require.NoError(t, err)
		var node []any
		require.NoError(t, rlp.DecodeBytes(nodeData, &node))
		switch len(node) {
		case 17:
			// Branch node.
			return nil, nil
		case 2:
			keyPath, ok := node[0].([]byte)
			require.True(t, ok)
			// Check if it is a leaf or extension node.
			if isLeaf(keyPath) {
				// Check that the keyPath matches the target nibbles,
				// otherwise, the receipt does not exist in the trie.
				leafKey := extractKeyNibbles(keyPath)
				expectedPath := append(currentPath, leafKey...)
				require.True(t, bytes.Equal(expectedPath, targetNibbles))

				receipt := new(types.Receipt)
				receiptData := bytes.NewBuffer(node[1].([]byte))
				// We skip the first byte which is the receipt type, as we will only care
				// about non-legacy transaction receipts for MEL.
				require.NoError(t, rlp.Decode(receiptData, &receipt))
				return receipt, nil
			}
			// Otherwise, we have an extension node to parse here.
			return nil, nil
		default:
			return nil, errors.New("not found")
		}
	}
}

func keyToNibbles(key []byte) []byte {
	nibbles := make([]byte, len(key)*2)
	for i, b := range key {
		nibbles[i*2] = b >> 4
		nibbles[i*2+1] = b & 0x0f
	}
	return nibbles
}

func extractKeyNibbles(keyPath []byte) []byte {
	if len(keyPath) == 0 {
		return nil
	}

	firstByte := keyPath[0]
	isOdd := (firstByte & 0x10) != 0

	var nibbles []byte
	if isOdd {
		// Odd length: first nibble is in the first byte.
		nibbles = append(nibbles, firstByte&0x0f)
		keyPath = keyPath[1:]
	} else {
		// Even length: skip the first byte
		keyPath = keyPath[1:]
	}
	// Convert remaining bytes to nibbles.
	for _, b := range keyPath {
		nibbles = append(nibbles, b>>4)
		nibbles = append(nibbles, b&0x0f)
	}

	return nibbles
}

func isLeaf(keyPath []byte) bool {
	firstByte := keyPath[0]
	firstNibble := firstByte >> 4
	return firstNibble >= 2 // 2 or 3 indicates leaf, while 0 or 1 indicates extension nodes.
}

func TestFetchReceiptFromBlock(t *testing.T) {
	receipts := createTestReceipts(1)
	hasher := newRecordingHasher()
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)
	header := &types.Header{}
	body := &types.Body{}
	blk := types.NewBlock(header, body, receipts, hasher)
	require.Equal(t, blk.ReceiptHash(), receiptsRoot)
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	receipt, err := fetchReceiptFromBlock(len(receipts), receiptsRoot, uint(0), mockPreimageResolver)
	require.NoError(t, err)
	require.Equal(t, receipts[0].CumulativeGasUsed, receipt.CumulativeGasUsed)
}

func TestFetchReceiptFromBlock_Multiple(t *testing.T) {
	numReceipts := 4
	receipts := createTestReceipts(numReceipts)
	hasher := newRecordingHasher()
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)
	header := &types.Header{}
	body := &types.Body{}
	blk := types.NewBlock(header, body, receipts, hasher)
	require.Equal(t, blk.ReceiptHash(), receiptsRoot)
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	for i := 1; i < numReceipts; i++ {
		receipt, err := fetchReceiptFromBlock(numReceipts, receiptsRoot, uint(i), mockPreimageResolver)
		require.NoError(t, err)
		require.Equal(t, receipts[i].CumulativeGasUsed, receipt.CumulativeGasUsed)
	}
}

func Test_compactDecoding(t *testing.T) {
	// Test the compact decoding with the actual bytes from the debug output
	pathBytes := []byte{0x20, 0x80}

	nibbles, isLeaf := decodeCompact(pathBytes)

	t.Logf("Path bytes: %x", pathBytes)
	t.Logf("Decoded nibbles: %v", nibbles)
	t.Logf("Is leaf: %v", isLeaf)

	// Our key nibbles
	var keyBuf []byte
	keyBuf = rlp.AppendUint64(keyBuf, 0)
	keyNibbles := keybytesToHex(keyBuf)

	t.Logf("Our key bytes: %x", keyBuf)
	t.Logf("Our key nibbles: %v", keyNibbles)

	// They should match now
	if len(keyNibbles) == len(nibbles) {
		match := true
		for i := range keyNibbles {
			if keyNibbles[i] != nibbles[i] {
				match = false
				break
			}
		}
		t.Logf("Nibbles match: %v", match)
	} else {
		t.Logf("Length mismatch: key=%d, path=%d", len(keyNibbles), len(nibbles))
	}
}

func Test_comprehensiveDebug(t *testing.T) {

	t.Logf("=== COMPREHENSIVE TRIE DEBUG ===")

	// Create a single receipt
	receipts := createTestReceipts(1)

	// Test different key encodings
	t.Logf("\n--- KEY ENCODING ANALYSIS ---")
	for i := 0; i < 3; i++ {
		var keyBuf []byte
		keyBuf = rlp.AppendUint64(keyBuf, uint64(i))
		nibbles := keybytesToHex(keyBuf)
		t.Logf("Index %d: RLP=%x, nibbles=%v, len=%d", i, keyBuf, nibbles, len(nibbles))
	}

	// Test what happens with empty bytes
	emptyKey := []byte{}
	t.Logf("Empty key: RLP=%x, nibbles=%v, len=%d", emptyKey, keybytesToHex(emptyKey), len(keybytesToHex(emptyKey)))

	zeroKey := []byte{0}
	t.Logf("Zero byte key: RLP=%x, nibbles=%v, len=%d", zeroKey, keybytesToHex(zeroKey), len(keybytesToHex(zeroKey)))

	// Build the trie and examine its structure
	t.Logf("\n--- TRIE STRUCTURE ANALYSIS ---")
	hasher := newRecordingHasher()
	receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)

	preimages := hasher.GetPreimages()
	t.Logf("Receipts root: %x", receiptsRoot)
	t.Logf("Number of preimages: %d", len(preimages))

	// Examine each preimage
	for hash, data := range preimages {
		t.Logf("\n--- Preimage %x ---", hash)
		t.Logf("Data: %x", data)
		t.Logf("Length: %d", len(data))

		// Try to decode as RLP to see the structure
		var node []interface{}
		if err := rlp.DecodeBytes(data, &node); err != nil {
			t.Logf("Failed to decode as RLP: %v", err)
			continue
		}

		t.Logf("RLP structure: %d elements", len(node))

		if len(node) == 2 {
			// Extension or leaf node
			pathData, err := extractBytes(node[0])
			if err != nil {
				t.Logf("Failed to extract path: %v", err)
				continue
			}

			t.Logf("Path data: %x", pathData)

			// Decode compact encoding
			pathNibbles, isLeaf := decodeCompact(pathData)
			t.Logf("Decoded path nibbles: %v", pathNibbles)
			t.Logf("Is leaf: %v", isLeaf)
			t.Logf("Path length: %d", len(pathNibbles))

			// Show what our key looks like
			var ourKeyBuf []byte
			ourKeyBuf = rlp.AppendUint64(ourKeyBuf, 0)
			ourKeyNibbles := keybytesToHex(ourKeyBuf)
			t.Logf("Our key nibbles: %v (length: %d)", ourKeyNibbles, len(ourKeyNibbles))

			// Show the mismatch
			if len(ourKeyNibbles) != len(pathNibbles) {
				t.Logf("LENGTH MISMATCH: our=%d, path=%d", len(ourKeyNibbles), len(pathNibbles))
				t.Logf("This is why we get 'remaining key too short'!")
			}

			// Try different key encodings to see what would match
			t.Logf("\n--- TRYING DIFFERENT KEY ENCODINGS ---")
			for testIdx := 0; testIdx < 5; testIdx++ {
				var testKeyBuf []byte
				testKeyBuf = rlp.AppendUint64(testKeyBuf, uint64(testIdx))
				testNibbles := keybytesToHex(testKeyBuf)

				matches := len(testNibbles) == len(pathNibbles)
				if matches && len(testNibbles) > 0 {
					for i := range testNibbles {
						if testNibbles[i] != pathNibbles[i] {
							matches = false
							break
						}
					}
				}

				t.Logf("Index %d -> key=%x, nibbles=%v, matches=%v", testIdx, testKeyBuf, testNibbles, matches)
			}

			// Try direct byte encodings
			t.Logf("\n--- TRYING DIRECT BYTE ENCODINGS ---")
			directKeys := [][]byte{
				{},
				{0},
				{0, 0},
				{0, 0, 0},
			}

			for i, directKey := range directKeys {
				directNibbles := keybytesToHex(directKey)
				matches := len(directNibbles) == len(pathNibbles)
				if matches && len(directNibbles) > 0 {
					for j := range directNibbles {
						if directNibbles[j] != pathNibbles[j] {
							matches = false
							break
						}
					}
				}
				t.Logf("Direct key %d -> key=%x, nibbles=%v, matches=%v", i, directKey, directNibbles, matches)
			}
		}
	}

	// Now test our fixed function
	t.Logf("\n--- TESTING FIXED FUNCTION ---")
	mockPreimageResolver := &mockPreimageResolver{preimages: preimages}

	receipt, err := fetchReceiptFromBlock(len(receipts), receiptsRoot, 0, mockPreimageResolver)
	if err != nil {
		t.Logf("Error: %v", err)
	} else {
		t.Logf("Success! Retrieved receipt with TxHash: %x", receipt.TxHash)
	}
}

func Test_compareTrieStructures(t *testing.T) {
	t.Logf("=== COMPARING TRIE STRUCTURES ===")

	// Test with 1, 2, and 3 receipts to see how structure changes
	for numReceipts := 1; numReceipts <= 3; numReceipts++ {
		t.Logf("\n--- TESTING WITH %d RECEIPTS ---", numReceipts)

		receipts := createTestReceipts(numReceipts)
		hasher := newRecordingHasher()
		receiptsRoot := types.DeriveSha(types.Receipts(receipts), hasher)

		preimages := hasher.GetPreimages()
		t.Logf("Root: %x", receiptsRoot)
		t.Logf("Preimages: %d", len(preimages))

		// Try to fetch receipt 0 in each case
		mockResolver := &mockPreimageResolver{preimages: preimages}

		_, err := fetchReceiptFromBlock(numReceipts, receiptsRoot, 0, mockResolver)
		if err != nil {
			t.Logf("Error fetching receipt 0: %v", err)
		} else {
			t.Logf("Successfully fetched receipt 0!")
		}

		// Examine the root preimage structure
		if rootData, exists := preimages[receiptsRoot]; exists {
			var node []interface{}
			if err := rlp.DecodeBytes(rootData, &node); err == nil {
				t.Logf("Root node has %d elements", len(node))

				if len(node) == 2 {
					// Leaf or extension
					pathData, _ := extractBytes(node[0])
					pathNibbles, isLeaf := decodeCompact(pathData)
					t.Logf("Root path: %x -> nibbles %v (leaf: %v)", pathData, pathNibbles, isLeaf)
				} else if len(node) == 17 {
					// Branch node
					t.Logf("Root is a branch node")
					nonNilChildren := 0
					for i := 0; i < 16; i++ {
						if node[i] != nil {
							nonNilChildren++
						}
					}
					t.Logf("Branch has %d non-nil children", nonNilChildren)

					// Check if there's a value at position 16
					if node[16] != nil {
						t.Logf("Branch has value at position 16")
					}
				}
			}
		}
	}
}

// Implements a hasher that captures preimages of hashes as it computes them.
type preimageRecordingHasher struct {
	trie      *trie.StackTrie
	preimages map[common.Hash][]byte
}

func newRecordingHasher() *preimageRecordingHasher {
	h := &preimageRecordingHasher{
		preimages: make(map[common.Hash][]byte),
	}
	// OnTrieNode callback captures all trie nodes.
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		// Deep copy the blob since the callback warns contents may change.
		h.preimages[hash] = common.CopyBytes(blob)
	}

	h.trie = trie.NewStackTrie(onTrieNode)
	return h
}

func (h *preimageRecordingHasher) Reset() {
	onTrieNode := func(path []byte, hash common.Hash, blob []byte) {
		h.preimages[hash] = common.CopyBytes(blob)
	}
	h.trie = trie.NewStackTrie(onTrieNode)
}

func (h *preimageRecordingHasher) Update(key, value []byte) error {
	valueHash := crypto.Keccak256Hash(value)
	h.preimages[valueHash] = common.CopyBytes(value)
	return h.trie.Update(key, value)
}

func (h *preimageRecordingHasher) Hash() common.Hash {
	return h.trie.Hash()
}

func (h *preimageRecordingHasher) GetPreimages() map[common.Hash][]byte {
	return h.preimages
}

func createTestReceipts(count int) types.Receipts {
	receipts := make(types.Receipts, count)
	for i := 0; i < count; i++ {
		receipt := &types.Receipt{
			Status:            1,
			CumulativeGasUsed: 50_000 + uint64(i),
			TxHash:            randomHash(),
			ContractAddress:   common.Address{},
			Logs:              []*types.Log{},
			BlockHash:         common.BytesToHash([]byte("foobar")),
			BlockNumber:       big.NewInt(100),
			TransactionIndex:  uint(i),
		}
		receipt.Bloom = types.BytesToBloom(make([]byte, 256))
		receipts[i] = receipt
	}
	return receipts
}

func randomHash() common.Hash {
	var hash common.Hash
	rand.Read(hash[:])
	return hash
}
