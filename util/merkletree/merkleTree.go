// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package merkletree

import (
	"errors"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/util"
)

type MerkleTree interface {
	Hash() common.Hash
	Size() uint64
	Capacity() uint64
	Append(common.Hash) MerkleTree
	SummarizeUpTo(num uint64) MerkleTree
	Serialize(wr io.Writer) error
}

const (
	SerializedLeaf byte = iota
	SerializedEmptySubtree
	SerializedInternalNode
	SerializedSubtreeSummary
)

type LevelAndLeaf struct {
	Level uint64
	Leaf  uint64
}

func NewLevelAndLeaf(level, leaf uint64) LevelAndLeaf {
	return LevelAndLeaf{
		Level: level,
		Leaf:  leaf,
	}
}

func NewLevelAndLeafFromPostion(position *big.Int) LevelAndLeaf {
	leaf := position.Uint64()                     // lower 8 bytes
	level := position.Rsh(position, 192).Uint64() // upper 8 bytes
	return LevelAndLeaf{
		Level: level,
		Leaf:  leaf,
	}
}

func (place LevelAndLeaf) ToBigInt() *big.Int {
	return new(big.Int).Add(
		new(big.Int).Lsh(big.NewInt(int64(place.Level)), 192),
		big.NewInt(int64(place.Leaf)),
	)
}

func NewEmptyMerkleTree() MerkleTree {
	return NewMerkleEmpty(0)
}

type merkleTreeLeaf struct {
	hash common.Hash
}

func NewMerkleLeaf(hash common.Hash) MerkleTree {
	return &merkleTreeLeaf{hash}
}

func newMerkleLeafFromReader(rd io.Reader) (MerkleTree, error) {
	hash, err := util.HashFromReader(rd)
	return NewMerkleLeaf(hash), err
}

func (leaf *merkleTreeLeaf) Hash() common.Hash {
	return crypto.Keccak256Hash(leaf.hash.Bytes())
}

func (leaf *merkleTreeLeaf) Size() uint64 {
	return 1
}

func (leaf *merkleTreeLeaf) Capacity() uint64 {
	return 1
}

func (leaf *merkleTreeLeaf) Append(newHash common.Hash) MerkleTree {
	return NewMerkleInternal(leaf, NewMerkleLeaf(newHash))
}

func (leaf *merkleTreeLeaf) SummarizeUpTo(num uint64) MerkleTree {
	return leaf
}

func (leaf *merkleTreeLeaf) Serialize(wr io.Writer) error {
	if _, err := wr.Write([]byte{SerializedLeaf}); err != nil {
		return err
	}
	_, err := wr.Write(leaf.hash.Bytes())
	return err
}

type merkleEmpty struct {
	capacity uint64
}

func NewMerkleEmpty(capacity uint64) MerkleTree {
	return &merkleEmpty{capacity}
}

func newMerkleEmptyFromReader(rd io.Reader) (MerkleTree, error) {
	capacity, err := util.Uint64FromReader(rd)
	return NewMerkleEmpty(capacity), err
}

func (me *merkleEmpty) Hash() common.Hash {
	return common.Hash{}
}

func (me *merkleEmpty) Size() uint64 {
	return 0
}

func (me *merkleEmpty) Capacity() uint64 {
	return me.capacity
}

func (me *merkleEmpty) Append(newHash common.Hash) MerkleTree {
	if me.capacity <= 1 {
		return NewMerkleLeaf(newHash)
	} else {
		halfSizeEmpty := NewMerkleEmpty(me.capacity / 2)
		return NewMerkleInternal(halfSizeEmpty.Append(newHash), halfSizeEmpty)
	}
}

func (me *merkleEmpty) SummarizeUpTo(num uint64) MerkleTree {
	return me
}

func (me *merkleEmpty) Serialize(wr io.Writer) error {
	if _, err := wr.Write([]byte{SerializedEmptySubtree}); err != nil {
		return err
	}
	return util.Uint64ToWriter(me.capacity, wr)
}

type merkleInternal struct {
	hash     common.Hash
	size     uint64
	capacity uint64
	left     MerkleTree
	right    MerkleTree
}

func NewMerkleInternal(left, right MerkleTree) MerkleTree {
	return &merkleInternal{
		crypto.Keccak256Hash(left.Hash().Bytes(), right.Hash().Bytes()),
		left.Size() + right.Size(),
		left.Capacity() + right.Capacity(),
		left,
		right,
	}
}

func newMerkleInternalFromReader(rd io.Reader) (MerkleTree, error) {
	left, err := NewMerkleTreeFromReader(rd)
	if err != nil {
		return nil, err
	}
	right, err := NewMerkleTreeFromReader(rd)
	if err != nil {
		return nil, err
	}
	return NewMerkleInternal(left, right), nil
}

func (mi *merkleInternal) Hash() common.Hash {
	return mi.hash
}

func (mi *merkleInternal) Size() uint64 {
	return mi.size
}

func (mi *merkleInternal) Capacity() uint64 {
	return mi.capacity
}

func (mi *merkleInternal) Append(newHash common.Hash) MerkleTree {
	if mi.size == mi.capacity {
		return NewMerkleInternal(mi, NewMerkleEmpty(mi.capacity).Append(newHash))
	} else if 2*mi.size < mi.capacity {
		return NewMerkleInternal(mi.left.Append(newHash), mi.right)
	} else {
		return NewMerkleInternal(mi.left, mi.right.Append(newHash))
	}
}

func (mi *merkleInternal) SummarizeUpTo(num uint64) MerkleTree {
	if num == mi.capacity {
		return summaryFromMerkleTree(mi)
	} else {
		leftSize := mi.left.Size()
		if num <= leftSize {
			return NewMerkleInternal(mi.left.SummarizeUpTo(num), mi.right)
		} else {
			return NewMerkleInternal(summaryFromMerkleTree(mi.left), mi.right.SummarizeUpTo(num-leftSize))
		}
	}
}

func (mi *merkleInternal) Serialize(wr io.Writer) error {
	if _, err := wr.Write([]byte{SerializedInternalNode}); err != nil {
		return err
	}
	if err := mi.left.Serialize(wr); err != nil {
		return err
	}
	return mi.right.Serialize(wr)
}

type merkleCompleteSubtreeSummary struct {
	hash     common.Hash
	capacity uint64
}

func NewSummaryMerkleTree(hash common.Hash, capacity uint64) MerkleTree {
	return &merkleCompleteSubtreeSummary{hash, capacity}
}

func summaryFromMerkleTree(subtree MerkleTree) MerkleTree {
	if subtree.Size() == 1 {
		return subtree
	}
	if subtree.Size() != subtree.Capacity() {
		panic("tried to summarize a non-full MerkleTree node")
	}
	return &merkleCompleteSubtreeSummary{subtree.Hash(), subtree.Capacity()}
}

func newMerkleSummaryFromReader(rd io.Reader) (MerkleTree, error) {
	capacity, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	hash, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	return &merkleCompleteSubtreeSummary{hash, capacity}, nil
}

func (sum *merkleCompleteSubtreeSummary) Hash() common.Hash {
	return sum.hash
}

func (sum *merkleCompleteSubtreeSummary) Size() uint64 {
	return sum.capacity
}

func (sum *merkleCompleteSubtreeSummary) Capacity() uint64 {
	return sum.capacity
}

func (sum *merkleCompleteSubtreeSummary) Append(newHash common.Hash) MerkleTree {
	return NewMerkleInternal(sum, NewMerkleEmpty(sum.capacity).Append(newHash))
}

func (sum *merkleCompleteSubtreeSummary) SummarizeUpTo(num uint64) MerkleTree {
	return sum
}

func (sum *merkleCompleteSubtreeSummary) Serialize(wr io.Writer) error {
	if _, err := wr.Write([]byte{SerializedSubtreeSummary}); err != nil {
		return err
	}
	if err := util.Uint64ToWriter(sum.capacity, wr); err != nil {
		return err
	}
	_, err := wr.Write(sum.hash.Bytes())
	return err
}

func NewMerkleTreeFromReader(rd io.Reader) (MerkleTree, error) {
	var typeBuf [1]byte
	if _, err := rd.Read(typeBuf[:]); err != nil {
		return nil, err
	}
	switch typeBuf[0] {
	case SerializedLeaf:
		return newMerkleLeafFromReader(rd)
	case SerializedInternalNode:
		return newMerkleInternalFromReader(rd)
	case SerializedEmptySubtree:
		return newMerkleEmptyFromReader(rd)
	case SerializedSubtreeSummary:
		return newMerkleSummaryFromReader(rd)
	default:
		return nil, errors.New("invalid node type in deserializing Merkle tree")
	}
}

type MerkleProof struct {
	RootHash  common.Hash
	LeafHash  common.Hash
	LeafIndex uint64
	Proof     []common.Hash
}

func (proof *MerkleProof) IsCorrect() bool {
	hash := proof.LeafHash
	index := proof.LeafIndex
	for _, hashFromProof := range proof.Proof {

		if index&1 == 0 {
			hash = crypto.Keccak256Hash(hash.Bytes(), hashFromProof.Bytes())
		} else {
			hash = crypto.Keccak256Hash(hashFromProof.Bytes(), hash.Bytes())
		}
		index = index / 2
	}
	if index != 0 {
		return false
	}
	return hash == proof.RootHash
}
