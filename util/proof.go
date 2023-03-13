package util

import (
	"encoding/binary"
	"errors"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math"
)

var (
	ErrInvalidLevel   = errors.New("invalid level")
	ErrInvalidHeight  = errors.New("invalid height")
	ErrMisaligned     = errors.New("misaligned")
	ErrIncorrectProof = errors.New("incorrect proof")
	ErrProofTooLong   = errors.New("merkle proof too long")
	ErrInvalidTree    = errors.New("invalid merkle tree")
	ErrInvalidLeaves  = errors.New("invalid number of leaves for merkle tree")
)

// Calculates a Merkle root from a Merkle proof, index, and leaf.
func CalculateRootFromProof(proof []common.Hash, index uint64, leaf common.Hash) (common.Hash, error) {
	if len(proof) > 256 {
		return common.Hash{}, ErrProofTooLong
	}
	h := crypto.Keccak256Hash(leaf[:])
	for i := 0; i < len(proof); i++ {
		node := proof[i]
		if index&(1<<i) == 0 {
			h = crypto.Keccak256Hash(h[:], node[:])
		} else {
			h = crypto.Keccak256Hash(node[:], h[:])
		}
	}
	return h, nil
}

// MerkleRoot from a tree.
func MerkleRoot(tree [][]common.Hash) (common.Hash, error) {
	if len(tree) == 0 || len(tree[len(tree)-1]) == 0 {
		return common.Hash{}, ErrInvalidTree
	}
	return tree[len(tree)-1][0], nil
}

// ComputeMerkleTree from a list of hashes. If not a power of two,
// pads with empty [32]byte{} until the length is a power of two.
// Creates a tree where the last level is the root.
func ComputeMerkleTree(items []common.Hash) [][]common.Hash {
	leaves := make([]common.Hash, len(items))
	for i, r := range items {
		// Rehash to match the Merkle expansion functions.
		leaves[i] = crypto.Keccak256Hash(r[:])
	}
	for !isPowerOfTwo(uint64(len(leaves))) {
		leaves = append(leaves, common.Hash{})
	}
	height := uint64(math.Log2(float64(len(leaves))))
	layers := make([][]common.Hash, height+1)
	layers[0] = leaves
	for i := uint64(0); i < height; i++ {
		updatedValues := make([]common.Hash, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			hashed := crypto.Keccak256Hash(layers[i][j][:], layers[i][j+1][:])
			updatedValues = append(updatedValues, hashed)
		}
		layers[i+1] = updatedValues
	}
	return layers
}

// GenerateMerkleProof for an index in a Merkle tree.
func GenerateMerkleProof(index uint64, tree [][]common.Hash) ([]common.Hash, error) {
	if len(tree) == 0 {
		return nil, ErrInvalidTree
	}
	proof := make([]common.Hash, len(tree)-1)
	leaves := tree[0]
	if index >= uint64(len(leaves)) {
		return nil, ErrInvalidLeaves
	}
	for i := 0; i < len(tree)-1; i++ {
		subIndex := (index / (1 << i)) ^ 1
		proof[i] = tree[i][subIndex]
	}
	return proof, nil
}

type MerkleExpansion []common.Hash

func NewEmptyMerkleExpansion() MerkleExpansion {
	return []common.Hash{}
}

func (me MerkleExpansion) Clone() MerkleExpansion {
	return append([]common.Hash{}, me...)
}

func (me MerkleExpansion) Root() common.Hash {
	accum := common.Hash{}
	empty := true
	for _, h := range me {
		if empty {
			if h != (common.Hash{}) {
				empty = false
				accum = h
			}
		} else {
			accum = crypto.Keccak256Hash(accum.Bytes(), h.Bytes())
		}
	}
	return accum
}

func (me MerkleExpansion) Compact() ([]common.Hash, uint64) {
	comp := []common.Hash{}
	size := uint64(0)
	for level, h := range me {
		if h != (common.Hash{}) {
			comp = append(comp, h)
			size += 1 << level
		}
	}
	return comp, size
}

func MerkleExpansionFromCompact(comp []common.Hash, size uint64) (MerkleExpansion, uint64) {
	me := []common.Hash{}
	numRead := uint64(0)
	i := uint64(1)
	for i <= size {
		if i&size != 0 {
			numRead++
			me = append(me, comp[0])
			comp = comp[1:]
		} else {
			me = append(me, common.Hash{})
		}
		i <<= 1
	}
	return me, numRead
}

func (me MerkleExpansion) AppendCompleteSubtree(level uint64, hash common.Hash) (MerkleExpansion, error) {
	if len(me) == 0 {
		exp := make([]common.Hash, level+1)
		exp[level] = hash
		return exp, nil
	}
	if level >= uint64(len(me)) {
		return nil, ErrInvalidLevel
	}
	ret := me.Clone()
	for i := uint64(0); i < uint64(len(me)); i++ {
		if i < level {
			if ret[i] != (common.Hash{}) {
				return nil, ErrMisaligned
			}
		} else {
			if ret[i] == (common.Hash{}) {
				ret[i] = hash
				return ret, nil
			} else {
				hash = crypto.Keccak256Hash(ret[i].Bytes(), hash.Bytes())
				ret[i] = common.Hash{}
			}
		}
	}
	return append(ret, hash), nil
}

func (me MerkleExpansion) AppendLeaf(leafHash common.Hash) MerkleExpansion {
	ret, _ := me.AppendCompleteSubtree(0, crypto.Keccak256Hash(leafHash.Bytes())) // re-hash to avoid collision with internal node hash
	return ret
}

func VerifyProof(pre, post HistoryCommitment, compactPre []common.Hash, proof common.Hash) error {
	preExpansion, _ := MerkleExpansionFromCompact(compactPre, pre.Height)
	if pre.Height >= post.Height {
		return ErrInvalidHeight
	}
	diff := post.Height - pre.Height
	if bits.OnesCount64(diff) != 1 {
		return ErrMisaligned
	}
	level := bits.TrailingZeros64(diff)
	postExpansion, err := preExpansion.AppendCompleteSubtree(uint64(level), proof)
	if err != nil {
		return err
	}
	if postExpansion.Root() != post.Merkle {
		return ErrIncorrectProof
	}
	return nil
}

func VerifyPrefixProof(pre, post HistoryCommitment, proof []common.Hash) error {
	if pre.Height >= post.Height {
		return ErrInvalidHeight
	}
	if len(proof) == 0 {
		return ErrIncorrectProof
	}
	expHeight := pre.Height
	expansion, numRead := MerkleExpansionFromCompact(proof, expHeight)
	proof = proof[numRead:]
	height := post.Height + 1
	for expHeight < height {
		if len(proof) == 0 {
			return ErrIncorrectProof
		}
		// extHeight looks like   xxxxxxx0yyy
		// post.height looks like xxxxxxx1zzz
		firstDiffBit := 63 - bits.LeadingZeros64(expHeight^height)
		mask := (uint64(1) << firstDiffBit) - 1
		yyy := expHeight & mask
		zzz := height & mask
		if yyy != 0 {
			lowBit := bits.TrailingZeros64(yyy)
			exp, err := expansion.AppendCompleteSubtree(uint64(lowBit), proof[0])
			if err != nil {
				return err
			}
			expansion = exp
			expHeight += 1 << lowBit
			proof = proof[1:]
		} else if zzz != 0 {
			highBit := 63 - bits.LeadingZeros64(zzz)
			exp, err := expansion.AppendCompleteSubtree(uint64(highBit), proof[0])
			if err != nil {
				return err
			}
			expansion = exp
			expHeight += 1 << highBit
			proof = proof[1:]
		} else {
			exp, err := expansion.AppendCompleteSubtree(uint64(firstDiffBit), proof[0])
			if err != nil {
				return err
			}
			expansion = exp
			expHeight = height
			proof = proof[1:]
		}
	}
	if expansion.Root() != post.Merkle {
		return ErrIncorrectProof
	}
	return nil
}

func GeneratePrefixProof(preHeight uint64, preExpansion MerkleExpansion, leaves []common.Hash) []common.Hash {
	height := preHeight
	postHeight := height + uint64(len(leaves))
	proof, _ := preExpansion.Compact()
	for height < postHeight {
		// extHeight looks like   xxxxxxx0yyy
		// post.height looks like xxxxxxx1zzz
		firstDiffBit := 63 - bits.LeadingZeros64(height^postHeight)
		mask := (uint64(1) << firstDiffBit) - 1
		yyy := height & mask
		zzz := postHeight & mask
		if yyy != 0 {
			lowBit := bits.TrailingZeros64(yyy)
			numLeaves := uint64(1) << lowBit
			proof = append(proof, ExpansionFromLeaves(leaves[:numLeaves]).Root())
			leaves = leaves[numLeaves:]
			height += numLeaves
		} else if zzz != 0 {
			highBit := 63 - bits.LeadingZeros64(zzz)
			numLeaves := uint64(1) << highBit
			proof = append(proof, ExpansionFromLeaves(leaves[:numLeaves]).Root())
			leaves = leaves[numLeaves:]
			height += numLeaves
		} else {
			proof = append(proof, ExpansionFromLeaves(leaves).Root())
			height = postHeight
		}
	}
	return proof
}

func GeneratePrefixProofBackend(preHeight uint64, preExpansion MerkleExpansion, hi uint64, backendCall func(lo uint64, hi uint64) (common.Hash, error)) []common.Hash {
	height := preHeight
	postHeight := hi
	proof, _ := preExpansion.Compact()
	for height < postHeight {
		// extHeight looks like   xxxxxxx0yyy
		// post.height looks like xxxxxxx1zzz
		firstDiffBit := 63 - bits.LeadingZeros64(height^postHeight)
		mask := (uint64(1) << firstDiffBit) - 1
		yyy := height & mask
		zzz := postHeight & mask
		if yyy != 0 {
			lowBit := bits.TrailingZeros64(yyy)
			numLeaves := uint64(1) << lowBit
			root, err := backendCall(height, height+numLeaves-1)
			if err != nil {
				return nil
			}
			proof = append(proof, root)
			height += numLeaves
		} else if zzz != 0 {
			highBit := 63 - bits.LeadingZeros64(zzz)
			numLeaves := uint64(1) << highBit
			root, err := backendCall(height, height+numLeaves-1)
			if err != nil {
				return nil
			}
			proof = append(proof, root)
			height += numLeaves
		} else {
			root, err := backendCall(height, postHeight-1)
			if err != nil {
				return nil
			}
			proof = append(proof, root)
			height = postHeight
		}
	}
	return proof
}

func ExpansionFromLeaves(leaves []common.Hash) MerkleExpansion {
	ret := NewEmptyMerkleExpansion()
	for _, leaf := range leaves {
		ret = ret.AppendLeaf(leaf)
	}
	return ret
}

func HashForUint(x uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, x))
}

func isPowerOfTwo(x uint64) bool {
	if x == 0 {
		return false
	}
	return x&(x-1) == 0
}
