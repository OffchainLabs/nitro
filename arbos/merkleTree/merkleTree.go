//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package merkleTree

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/util"
	"io"
)

type MerkleTree interface {
	Hash() common.Hash
	Size() uint64
	Capacity() uint64
	Append(common.Hash) MerkleTree
	SummarizeUpTo(num uint64) MerkleTree
	Prove(leafIndex uint64) *MerkleProof
	Serialize(wr io.Writer) error
}

const (
	SerializedLeaf byte = iota
	SerializedEmptySubtree
	SerializedInternalNode
	SerializedSubtreeSummary
)

func NewEmptyMerkleTree() MerkleTree {
	return newMerkleEmpty(0)
}

type merkleTreeLeaf struct {
	hash common.Hash
}

func newMerkleLeaf(hash common.Hash) MerkleTree {
	return &merkleTreeLeaf{hash}
}

func newMerkleLeafFromReader(rd io.Reader) (MerkleTree, error) {
	hash, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	return newMerkleLeaf(hash), nil
}

func (leaf *merkleTreeLeaf) Hash() common.Hash {
	return leaf.hash
}

func (leaf *merkleTreeLeaf) Size() uint64 {
	return 1
}

func (leaf *merkleTreeLeaf) Capacity() uint64 {
	return 1
}

func (leaf *merkleTreeLeaf) Append(newHash common.Hash) MerkleTree {
	return newMerkleInternal(leaf, newMerkleLeaf(newHash))
}

func (leaf *merkleTreeLeaf) SummarizeUpTo(num uint64) MerkleTree {
	return leaf
}

func (leaf *merkleTreeLeaf) Prove(leafIndex uint64) *MerkleProof {
	if leafIndex != 0 {
		return nil
	}
	return &MerkleProof{
		leaf.hash,
		leaf.hash,
		0,
		[]common.Hash{},
	}
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

func newMerkleEmpty(capacity uint64) MerkleTree {
	return &merkleEmpty{capacity}
}

func newMerkleEmptyFromReader(rd io.Reader) (MerkleTree, error) {
	capacity, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	return newMerkleEmpty(capacity), nil
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
		return newMerkleLeaf(newHash)
	} else {
		halfSizeEmpty := newMerkleEmpty(me.capacity / 2)
		return newMerkleInternal(halfSizeEmpty.Append(newHash), halfSizeEmpty)
	}
}

func (me *merkleEmpty) SummarizeUpTo(num uint64) MerkleTree {
	return me
}

func (me *merkleEmpty) Prove(leafIndex uint64) *MerkleProof {
	return nil
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

func newMerkleInternal(left, right MerkleTree) MerkleTree {
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
	return newMerkleInternal(left, right), nil
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
		return newMerkleInternal(mi, newMerkleEmpty(mi.capacity).Append(newHash))
	} else if 2*mi.size < mi.capacity {
		return newMerkleInternal(mi.left.Append(newHash), mi.right)
	} else {
		return newMerkleInternal(mi.left, mi.right.Append(newHash))
	}
}

func (mi *merkleInternal) SummarizeUpTo(num uint64) MerkleTree {
	if num == mi.size {
		return summaryFromMerkleTree(mi)
	} else {
		leftSize := mi.left.Size()
		if num <= leftSize {
			return newMerkleInternal(mi.left.SummarizeUpTo(num), mi.right)
		} else {
			return newMerkleInternal(summaryFromMerkleTree(mi.left), mi.right.SummarizeUpTo(num-leftSize))
		}
	}
}

func (mi *merkleInternal) Prove(leafIndex uint64) *MerkleProof {
	if leafIndex >= mi.size {
		return nil
	}
	leftSize := mi.left.Size()
	var proof *MerkleProof
	if leafIndex < leftSize {
		proof = mi.left.Prove(leafIndex)
		proof.Proof = append(proof.Proof, mi.right.Hash())
	} else {
		proof = mi.right.Prove(leafIndex - leftSize)
		proof.Proof = append(proof.Proof, mi.left.Hash())
	}
	proof.LeafIndex = leafIndex
	proof.RootHash = mi.hash
	return proof
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
	size     uint64
	capacity uint64
}

func summaryFromMerkleTree(subtree MerkleTree) MerkleTree {
	if subtree.Size() == 1 {
		return subtree
	}
	return &merkleCompleteSubtreeSummary{subtree.Hash(), subtree.Size(), subtree.Capacity()}
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
	return &merkleCompleteSubtreeSummary{ hash, capacity, capacity }, nil
}

func (sum *merkleCompleteSubtreeSummary) Hash() common.Hash {
	return sum.hash
}

func (sum *merkleCompleteSubtreeSummary) Size() uint64 {
	return sum.size
}

func (sum *merkleCompleteSubtreeSummary) Capacity() uint64 {
	return sum.capacity
}

func (sum *merkleCompleteSubtreeSummary) Append(newHash common.Hash) MerkleTree {
	return newMerkleInternal(sum, newMerkleEmpty(sum.size).Append(newHash))
}

func (sum *merkleCompleteSubtreeSummary) SummarizeUpTo(num uint64) MerkleTree {
	return sum
}

func (sum *merkleCompleteSubtreeSummary) Prove(leafIndex uint64) *MerkleProof {
	return nil
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
		panic(index)
		return false
	}
	return hash == proof.RootHash
}
