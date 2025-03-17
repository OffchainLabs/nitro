package nethexec

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

func DumpState(db ethdb.Database, stateRoot common.Hash) error {
	fmt.Printf("State root: %s\n\n", stateRoot.Hex())

	// Create trie database
	trieDb := triedb.NewDatabase(db, &triedb.Config{
		Preimages: true,
	})

	// Create dumper
	dumper := NewTrieDumper(trieDb)

	// Dump state trie
	output, err := dumper.DumpTrie(stateRoot, common.Hash{}, false, "")
	if err != nil {
		return fmt.Errorf("failed to dump trie: %w", err)
	}

	fmt.Println(output)
	return nil
}

func DumpAccountStorage(db ethdb.Database, stateRoot common.Hash, accountAddr common.Address) error {
	trieDb := triedb.NewDatabase(db, &triedb.Config{
		Preimages: true,
	})

	stateDb := state.NewDatabase(trieDb, nil)
	stateDbReader, err := stateDb.Reader(stateRoot)
	if err != nil {
		return fmt.Errorf("failed to get state db reader: %w", err)
	}

	acc, err := stateDbReader.Account(accountAddr)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	fmt.Printf("Account: %s\n", accountAddr.Hex())
	fmt.Printf("Storage Root: %s\n\n", acc.Root.Hex())

	if acc.Root != types.EmptyRootHash && acc.Root != (common.Hash{}) {
		dumper := NewTrieDumper(trieDb)

		ownerHash := crypto.Keccak256Hash(accountAddr.Bytes())

		output, err := dumper.DumpTrie(acc.Root, ownerHash, true, "")
		if err != nil {
			return fmt.Errorf("failed to dump storage trie: %w", err)
		}

		fmt.Println(output)
	} else {
		fmt.Println("Account has empty storage")
	}

	return nil
}

// NodeInfo stores information about a trie node
type NodeInfo struct {
	Path       []byte
	PathStr    string
	Hash       common.Hash
	ParentHash common.Hash
	Parent     *NodeInfo
	IsLeaf     bool
	Key        []byte
	Value      []byte
	Children   map[byte]*NodeInfo
}

// TrieDumper provides functionality to dump trie in Nethermind-compatible format
type TrieDumper struct {
	db *triedb.Database
}

// NewTrieDumper creates a new trie dumper
func NewTrieDumper(db *triedb.Database) *TrieDumper {
	return &TrieDumper{db: db}
}

// DumpTrie dumps a trie with the given root in Nethermind format
func (td *TrieDumper) DumpTrie(rootHash common.Hash, owner common.Hash, isStorage bool, indent string) (string, error) {
	var output strings.Builder

	if isStorage {
		output.WriteString(fmt.Sprintf("%s STORAGE TREE  owner=%s\n", indent, owner.String()))
	} else {
		output.WriteString(fmt.Sprintf("%s STATE TREE    owner=%s\n", indent, owner.String()))
	}

	// Create trie based on type
	var t *trie.Trie
	var err error

	if isStorage {
		t, err = trie.New(trie.StorageTrieID(rootHash, owner, rootHash), td.db)
	} else {
		t, err = trie.New(trie.StateTrieID(rootHash), td.db)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create trie: %w", err)
	}

	// Collect all nodes using iterator
	nodes, rootNode, err := td.collectNodes(t, rootHash)
	if err != nil {
		return "", err
	}

	// Dump tree recursively
	td.dumpNode(&output, rootNode, nodes, indent+"++", isStorage, owner)

	return output.String(), nil
}

// collectNodes collects all nodes from the trie
func (td *TrieDumper) collectNodes(t *trie.Trie, rootHash common.Hash) (map[string]*NodeInfo, *NodeInfo, error) {
	nodesByPath := make(map[string]*NodeInfo)
	nodesByHash := make(map[common.Hash]*NodeInfo)

	it := t.MustNodeIterator(nil)
	for it.Next(true) {
		path := it.Path()
		pathStr := hex.EncodeToString(path)

		// Skip if we already have this node
		if _, exists := nodesByPath[pathStr]; exists {
			continue
		}

		node := &NodeInfo{
			Path:       path,
			PathStr:    pathStr,
			Hash:       it.Hash(),
			ParentHash: it.Parent(),
			IsLeaf:     it.Leaf(),
			Children:   make(map[byte]*NodeInfo),
		}

		if it.Leaf() {
			node.Key = it.LeafKey()
			node.Value = it.LeafBlob()
		}

		nodesByPath[pathStr] = node
		nodesByHash[node.Hash] = node
	}

	if err := it.Error(); err != nil {
		return nil, nil, err
	}

	for _, node := range nodesByPath {
		if node.ParentHash != (common.Hash{}) {
			if parent, exists := nodesByHash[node.ParentHash]; exists {
				// Determine child index from path
				node.Parent = parent

				if len(node.Path) > 0 && len(parent.Path) < len(node.Path) {
					childIndex := node.Path[len(parent.Path)]
					parent.Children[childIndex] = node
				}
			}
		}
	}

	return nodesByPath, nodesByHash[rootHash], nil
}

// dumpNode recursively dumps a node
func (td *TrieDumper) dumpNode(output *strings.Builder, node *NodeInfo, tree map[string]*NodeInfo, indent string, isStorage bool, owner common.Hash) {
	if node == nil {
		return
	}

	if node.IsLeaf {
		// Leaf node
		nodeType := "ACCOUNT"
		if isStorage {
			nodeType = "STORAGE"
		}

		keyStr := td.formatKey(node.Key)
		output.WriteString(fmt.Sprintf("%s %s  %s -> %s\n",
			indent,
			nodeType,
			keyStr,
			node.Hash.Hex()[2:]))

		// Add pre-hash for modified nodes (if available)
		// This would require tracking node modifications

		// Decode and display data
		if !isStorage && len(node.Value) > 0 {
			td.decodeAccountData(output, node, indent+"++")
		} else if isStorage && len(node.Value) > 0 {
			// Decode storage value
			var value common.Hash
			if len(node.Value) == 32 {
				value = common.BytesToHash(node.Value)
			} else {
				// RLP decode for smaller values
				rlp.DecodeBytes(node.Value, &value)
			}
			output.WriteString(fmt.Sprintf("%s VALUE: %s\n", indent, value.Hex()))
		}
	} else {
		// Branch or Extension node
		if len(node.Children) > 1 || (len(node.Children) == 1 && node.Children[16] != nil) {
			// Branch node
			output.WriteString(fmt.Sprintf("%s BRANCH | -> %s\n", indent, node.Hash.Hex()[2:]))

			// Sort children for consistent output
			var indices []byte
			for idx := range node.Children {
				indices = append(indices, idx)
			}
			sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

			// Dump children
			for _, idx := range indices {
				child := node.Children[idx]
				if child != nil {
					td.dumpNode(output, child, tree, fmt.Sprintf("%s%02x", indent+"++", idx), isStorage, owner)
				}
			}
		} else {
			// Extension node
			// Extract the extension key from path difference
			output.WriteString(fmt.Sprintf("%s EXTENSION -> %s\n", indent, node.Hash.Hex()[2:]))

			// Sort children for consistent output
			var indices []byte
			for idx := range node.Children {
				indices = append(indices, idx)
			}
			sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

			// Dump children
			for _, idx := range indices {
				child := node.Children[idx]
				if child != nil {
					td.dumpNode(output, child, tree, fmt.Sprintf("%s%02x", indent+"++", idx), isStorage, owner)
				}
			}
		}
	}
}

// formatKey formats a key in Nethermind hex style
func (td *TrieDumper) formatKey(key []byte) string {
	hexStr := hex.EncodeToString(key)
	// Ensure even length
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	return hexStr
}

// decodeAccountData decodes and displays account information
func (td *TrieDumper) decodeAccountData(output *strings.Builder, node *NodeInfo, indent string) {
	acc := new(types.StateAccount)
	if err := rlp.DecodeBytes(node.Value, &acc); err != nil {
		output.WriteString(fmt.Sprintf("%s  [Failed to decode account: %v]\n", indent, err))
		return
	}

	output.WriteString(fmt.Sprintf("%s  NONCE: %d\n", indent, acc.Nonce))
	output.WriteString(fmt.Sprintf("%s  BALANCE: %s\n", indent, acc.Balance.String()))

	emptyCodeHash := crypto.Keccak256(nil)
	isContract := !bytes.Equal(acc.CodeHash, emptyCodeHash)
	output.WriteString(fmt.Sprintf("%s  IS_CONTRACT: %t\n", indent, isContract))
	output.WriteString(fmt.Sprintf("%s  CODE_HASH: %s\n", indent, common.BytesToHash(acc.CodeHash).Hex()))
	output.WriteString(fmt.Sprintf("%s  STORAGE_ROOT: %s\n", indent, acc.Root.Hex()))

	if acc.Root != types.EmptyRootHash && acc.Root != (common.Hash{}) {
		keyHash := common.BytesToHash(node.Key)
		storageDump, err := td.DumpTrie(acc.Root, keyHash, true, indent+"++")
		if err != nil {
			output.WriteString(fmt.Sprintf("[Failed to dump storage: %v]\n", err))
		} else {
			output.WriteString(storageDump)
		}
	}
}

// PathToNibbles converts a path to nibbles for display
func PathToNibbles(path []byte) []byte {
	nibbles := make([]byte, 0, len(path)*2)
	for _, b := range path {
		nibbles = append(nibbles, b>>4, b&0x0f)
	}
	return nibbles
}

// CompactToNibbles converts compact encoded path to nibbles
func CompactToNibbles(compact []byte) []byte {
	if len(compact) == 0 {
		return nil
	}

	base := keybytesToNibbles(compact)
	// delete terminator flag
	if base[len(base)-1] == 16 {
		base = base[:len(base)-1]
	}

	// apply odd flag
	chop := 2 - base[0]&1
	return base[chop:]
}

func keybytesToNibbles(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}
