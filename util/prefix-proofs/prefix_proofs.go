// Package prefixproofs defines utilities for creating Merkle prefix proofs for binary
// trees. It is used extensively in the challenge protocol for making challenge moves on-chain.
// These utilities also have equivalent counterparts written in Solidity under
// MerkleTreeLib.sol, which must be thoroughly tested against to ensure safety.
//
// Binary trees
// --------------------------------------------------------------------------------------------
// A complete tree is a balanced binary tree - each node has two children except the leaf
// Leaves have no children, they are a complete tree of size one
// Any tree (can be incomplete) can be represented as a collection of complete sub trees.
// Since the tree is binary only one or zero complete tree at each level is enough to define any size of tree.
// The root of a tree (incomplete or otherwise) is defined as the cumulative hashing of all of the
// roots of each of it's complete and empty subtrees.
// ---------
// eg. Below a tree of size 3 is represented as the composition of 2 complete subtrees, one of size
// 2 (AB) and one of size one (C).
//    AB
//   /  \
//  A    B    C

// Merkle expansions and roots
// --------------------------------------------------------------------------------------------
// The minimal amount of information we need to keep in order to compute the root of a tree
// is the roots of each of it's sub trees, and the levels of each of those trees
// A "merkle expansion" (ME) is this information - it is a vector of roots of each complete subtree,
// the level of the tree being the index in the vector, the subtree root being the value.
// The root is calculated by hashing each of the levels of the subtree together, adding zero hashes
// where relevant to make a balanced tree.
// ---------
// eg. from the example above
// ME of the AB tree = (0, AB), root=AB
// ME of the C tree = (C), root=(C, 0)
// ME of the composed ABC tree = (AB, C), root=hash(AB, hash(C, 0)) - here C is hashed with 0
// to balance the tree, before then being hashed with AB.

// Tree operations
// --------------------------------------------------------------------------------------------
// Binary trees are modified by adding or subtracting complete subtrees, however this libary
// supports additive only trees since we dont have a specific use for subtraction at the moment.
// We call adding a complete subtree to an existing tree "appending", appending has the following
// rules:
// 1. Only a complete sub trees can be appended
// 2. Complete sub trees can only be appended at the level of the lowest complete subtree in the tree, or below
// 3. If the existing tree is empty a sub tree can be appended at any level
// When appending a sub tree we may increase the size of the merkle expansion vector, in the same
// that adding 1 to a binary number may increase the index of its most significant bit
// ---------
// eg. A complete subtree can only be appended to the ABC tree at level 0, since the its lowest complete
// subtree (C) is at level 0. Doing so would create a complete sub tree at level 1, which would in turn
// cause the creation of new size 4 sub tree
//
//	                               ABCD
//	                             /     \
//	  AB                        AB     CD
//	 /  \         +       =    /  \   /  \
//	A    B    C       D       A    B C    D
//
// ME of ABCD = (0, 0, ABCD), root=hash(AB, CD)
// --------------------------------------------------------------------------------------------
package prefixproofs

import (
	"math"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

const (
	// MAX_LEVEL for our binary trees.
	MAX_LEVEL = uint64(64)
)

var (
	ErrRootForEmpty                      = errors.New("cannot calculate root for empty")
	ErrExpansionTooLarge                 = errors.New("merkle expansion to large")
	ErrLevelTooHigh                      = errors.New("level too high")
	ErrTreeSize                          = errors.New("tree size incorrect")
	ErrCannotAppendEmpty                 = errors.New("cannot append empty")
	ErrCannotAppendAboveLeastSignificant = errors.New("cannot append above least significant")
	ErrStartNotLessThanEnd               = errors.New("start not less than end")
	ErrCannotBeZero                      = errors.New("cannot be zero")
	ErrRootMismatch                      = errors.New("root mismatch")
	ErrIncompleteProof                   = errors.New("incomplete proof usage")
	ErrSizeNotLeqPostSize                = errors.New("size not <= post size")
	ErrIndexOutOfRange                   = errors.New("index out of range")
)

// LeastSignificantBit of a 64bit unsigned integer.
func LeastSignificantBit(x uint64) (uint64, error) {
	if x == 0 {
		return 0, ErrCannotBeZero
	}
	return uint64(bits.TrailingZeros64(x)), nil
}

// MostSignificantBit of a 64bit unsigned integer.
func MostSignificantBit(x uint64) (uint64, error) {
	if x == 0 {
		return 0, ErrCannotBeZero
	}
	return uint64(63 - bits.LeadingZeros64(x)), nil
}

// The root of the subtree. A collision free commitment to the contents of the tree.
// The root of a tree is defined as the cumulative hashing of the roots of
// all its subtrees. Returns error for empty tree.
func Root(me []common.Hash) (common.Hash, error) {
	if uint64(len(me)) >= MAX_LEVEL {
		return common.Hash{}, ErrLevelTooHigh
	}

	var accum common.Hash
	for i := 0; i < len(me); i++ {
		val := me[i]
		if accum == (common.Hash{}) {
			if val != (common.Hash{}) {
				accum = val

				// the tree is balanced if the only non zero entry in the merkle extension
				// us the last entry
				// otherwise the lowest level entry needs to be combined with a zero to balance the bottom
				// level, after which zeros in the merkle extension above that will balance the rest
				if i != len(me)-1 {
					accum = crypto.Keccak256Hash(accum.Bytes(), (common.Hash{}).Bytes())
				}
			}
		} else if (val != common.Hash{}) {
			// accum represents the smaller sub trees, since it is earlier in the expansion we put
			// the larger subtrees on the left
			accum = crypto.Keccak256Hash(val.Bytes(), accum.Bytes())
		} else {
			// by definition we always complete trees by appending zeros to the right
			accum = crypto.Keccak256Hash(accum.Bytes(), (common.Hash{}).Bytes())
		}
	}
	return accum, nil
}

// Calculate the full tree size represented by a merkle expansion
func TreeSize(me []common.Hash) uint64 {
	sum := uint64(0)
	for i := 0; i < len(me); i++ {
		if me[i] != (common.Hash{}) {
			sum += uint64(math.Pow(2, float64(i)))
		}
	}
	return sum
}

// Append a complete subtree to an existing tree
// See above description of trees for rules on how appending can occur.
// Briefly, appending works like binary addition only that the value being added be an
// exact power of two (complete), and must equal to or less than the least signficant bit
// in the existing tree.
// If the me is empty, will just append directly.
func AppendCompleteSubTree(
	me []common.Hash, level uint64, subtreeRoot common.Hash,
) ([]common.Hash, error) {
	// we use number representations of the levels elsewhere, so we need to ensure we're appending a leve
	// that's too high to use in uint
	if level >= MAX_LEVEL {
		return nil, ErrLevelTooHigh
	}
	if subtreeRoot == (common.Hash{}) {
		return nil, ErrCannotAppendEmpty
	}
	if uint64(len(me)) > MAX_LEVEL {
		return nil, ErrExpansionTooLarge
	}

	if len(me) == 0 {
		empty := make([]common.Hash, level+1)
		empty[level] = subtreeRoot
		return empty, nil
	}

	if level >= uint64(len(me)) {
		// This technically isn't necessary since it would be caught by the i < level check
		// on the last loop of the for-loop below, but we add it for a clearer error message
		return nil, errors.Wrap(ErrLevelTooHigh, "failing before for loop")
	}

	accumHash := subtreeRoot
	next := make([]common.Hash, len(me))

	// loop through all the levels in self and try to append the new subtree
	// since each node has two children by appending a subtree we may complete another one
	// in the level above. So we move through the levels updating the result at each level
	for i := uint64(0); i < uint64(len(me)); i++ {
		// we can only append at the level of the smallest complete sub tree or below
		// appending above this level would mean create "holes" in the tree
		// we can find the smallest complete sub tree by looking for the first entry in the merkle expansion
		if i < level {
			// we're below the level we want to append - no complete sub trees allowed down here
			// if the level is 0 there are no complete subtrees, and we therefore cannot be too low
			if me[i] != (common.Hash{}) {
				return nil, ErrCannotAppendAboveLeastSignificant
			}
		} else {
			// we're at or above the level
			if accumHash == (common.Hash{}) {
				// no more changes to propagate upwards - just fill the tree
				next[i] = me[i]
			} else {
				// we have a change to propagate
				if me[i] == (common.Hash{}) {
					// if the level is currently empty we can just add the change
					next[i] = accumHash
					// and then there's nothing more to propagate
					accumHash = common.Hash{}
				} else {
					// if the level is not currently empty then we combine it with propagation
					// change, and propagate that to the level above. This level is now part of a complete subtree
					// so we zero it out
					next[i] = common.Hash{}
					accumHash = crypto.Keccak256Hash(me[i].Bytes(), accumHash.Bytes())
				}
			}
		}
	}

	// we had a final change to propagate above the existing highest complete sub tree
	// so we have a new highest complete sub tree in the level above
	if accumHash != (common.Hash{}) {
		next = append(next, accumHash)
	}

	if uint64(len(next)) >= MAX_LEVEL+1 {
		return nil, ErrLevelTooHigh
	}
	return next, nil
}

// Append a leaf to a subtree
// Leaves are just complete subtrees at level 0, however we hash the leaf before putting it
// into the tree to avoid root collisions.
func AppendLeaf(
	me []common.Hash, leaf [32]byte,
) ([]common.Hash, error) {
	// it's important that we hash the leaf, this ensures that this leaf cannot be a collision with any other non leaf
	// or root node, since these are always the hash of 64 bytes of data, and we're hashing 32 bytes
	return AppendCompleteSubTree(me, 0, crypto.Keccak256Hash(leaf[:]))
}

// Find the highest level which can be appended to tree of size startSize without
// creating a tree with size greater than end size (inclusive)
// Subtrees can only be appended according to certain rules, see tree description at top of file
// for details. A subtree can only be appended if it is at the same level, or below, the current lowest
// subtree in the expansion
func MaximumAppendBetween(startSize, endSize uint64) (uint64, error) {
	// Since the tree is binary we can represent it using the binary representation of a number
	// We use size here instead of height since height is zero indexed
	// As described above, subtrees can only be appended to a tree if they are at the same level, or below,
	// the current lowest subtree.
	// In this function we want to find the level of the highest tree that can be appended to the current
	// tree, without the resulting tree surpassing the end point. We do this by looking at the difference
	// between the start and end size, and iteratively reducing it in the maximal way.

	// The start and end size will share some higher order bits, below that they differ, and it is this
	// difference that we need to fill in the minimum number of appends
	// startSize looks like: xxxxxxyyyy
	// endSize looks like:   xxxxxxzzzz
	// where x are the complete sub trees they share, and y and z are the subtrees they dont
	if startSize >= endSize {
		return 0, errors.Wrapf(ErrStartNotLessThanEnd, "start %d, end %d", startSize, endSize)
	}

	// remove the high order bits that are shared
	msb, err := MostSignificantBit(startSize ^ endSize)
	if err != nil {
		return 0, err
	}

	mask := uint64((1 << (msb + 1)) - 1)
	y := startSize & mask
	z := endSize & mask

	// Since in the verification we will be appending at start size, the highest level at which we
	// can append is the lowest complete subtree - the least significant bit
	if y != 0 {
		return LeastSignificantBit(y)
	}
	// y == 0, therefore we can append at any of levels where start and end differ
	// The highest level that we can append at without surpassing the end, is the most significant
	// bit of the end
	if z != 0 {
		return MostSignificantBit(z)
	}
	// since we enforce that start < end, we know that y and z cannot both be 0
	return 0, errors.Wrap(ErrCannotBeZero, "y and z cannot both be 0")
}

func GeneratePrefixProof(
	prefixHeight uint64,
	prefixExpansion MerkleExpansion,
	leaves []common.Hash,
	rootFetcher MerkleExpansionRootFetcherFunc,
) ([]common.Hash, error) {
	height := prefixHeight
	postHeight := height + uint64(len(leaves))
	proof, _ := prefixExpansion.Compact()
	for height < postHeight {
		// extHeight looks like   xxxxxxx0yyy
		// post.height looks like xxxxxxx1zzz
		firstDiffBit, err := MostSignificantBit(height ^ postHeight)
		if err != nil {
			return nil, err
		}
		mask := (uint64(1) << firstDiffBit) - 1
		yyy := height & mask
		zzz := postHeight & mask
		if yyy != 0 {
			lowBit, err := LeastSignificantBit(yyy)
			if err != nil {
				return nil, err
			}
			numLeaves := uint64(1) << lowBit
			root, err := rootFetcher(leaves, numLeaves)
			if err != nil {
				return nil, err
			}
			proof = append(proof, root)
			leaves = leaves[numLeaves:]
			height += numLeaves
		} else if zzz != 0 {
			highBit, err := MostSignificantBit(yyy)
			if err != nil {
				return nil, err
			}
			numLeaves := uint64(1) << highBit
			root, err := rootFetcher(leaves, numLeaves)
			if err != nil {
				return nil, err
			}
			proof = append(proof, root)
			leaves = leaves[numLeaves:]
			height += numLeaves
		} else {
			root, err := rootFetcher(leaves, uint64(len(leaves)))
			if err != nil {
				return nil, err
			}
			proof = append(proof, root)
			height = postHeight
		}
	}
	return proof, nil
}

type VerifyPrefixProofConfig struct {
	PreRoot      common.Hash
	PreSize      uint64
	PostRoot     common.Hash
	PostSize     uint64
	PreExpansion []common.Hash
	PrefixProof  []common.Hash
}

// Verify that a pre-root commits to a prefix of the leaves committed by a post-root
// Verifies by appending sub trees to the pre tree until we get to the size of the post tree
// and then checking that the root of the calculated post tree is equal to the supplied one
func VerifyPrefixProof(cfg *VerifyPrefixProofConfig) error {
	if cfg.PreSize == 0 {
		return errors.Wrap(ErrCannotBeZero, "presize was 0")
	}
	root, rootErr := Root(cfg.PreExpansion)
	if rootErr != nil {
		return errors.Wrap(rootErr, "pre expansion root error")
	}
	if root != cfg.PreRoot {
		return errors.Wrap(ErrRootMismatch, "pre expansion root mismatch")
	}
	if cfg.PreSize != TreeSize(cfg.PreExpansion) {
		return errors.Wrap(ErrTreeSize, "pre expansion tree size")
	}
	if cfg.PreSize >= cfg.PostSize {
		return errors.Wrapf(
			ErrStartNotLessThanEnd,
			"presize %d >= postsize %d",
			cfg.PreSize,
			cfg.PostSize,
		)
	}

	exp := make([]common.Hash, len(cfg.PreExpansion))
	copy(exp, cfg.PreExpansion)
	size := cfg.PreSize
	proofIndex := uint64(0)

	for size < cfg.PostSize {
		level, err := MaximumAppendBetween(size, cfg.PostSize)
		if err != nil {
			return err
		}
		if proofIndex >= uint64(len(cfg.PrefixProof)) {
			return ErrIndexOutOfRange
		}
		exp, err = AppendCompleteSubTree(
			exp, level, cfg.PrefixProof[proofIndex],
		)
		if err != nil {
			return err
		}
		numLeaves := 1 << level
		size += uint64(numLeaves)
		if size > cfg.PostSize {
			return errors.Wrapf(
				ErrSizeNotLeqPostSize,
				"size %d > postsize %d",
				size,
				cfg.PostSize,
			)
		}
		proofIndex++
	}
	gotRoot, gotRootErr := Root(exp)
	if gotRootErr != nil {
		return errors.Wrap(gotRootErr, "post root error")
	}
	if gotRoot != cfg.PostRoot {
		return errors.Wrapf(
			ErrRootMismatch,
			"post expansion root mismatch got %#x, wanted %#x",
			gotRoot,
			cfg.PostRoot,
		)
	}
	if proofIndex != uint64(len(cfg.PrefixProof)) {
		return errors.Wrapf(
			ErrIncompleteProof,
			"proof index %d, proof length %d",
			proofIndex,
			len(cfg.PrefixProof),
		)
	}
	return nil
}
