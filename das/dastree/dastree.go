// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dastree

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const binSize = 64 * 1024 // 64 kB

type bytes32 = common.Hash

func RecordHash(record func(bytes32, []byte), preimage ...[]byte) bytes32 {
	// Algorithm
	//  1. split the preimage into 64kB bins and hash them to produces the tree's leaves
	//  2. repeatedly hash pairs over and over, bubbling up any odd-one's out, to form the root
	//
	//            r         <=>  hash(hash(0, 1), 2)           step 2
	//           / \
	//          *   2       <=>  hash(0, 1), 2                 step 1
	//         / \
	//        0   1         <=>  0, 1, 2                       step 0
	//
	//  Intermediate hashes like '*' from above may be recorded via the `record` closure
	//

	unrolled := []byte{}
	for _, slice := range preimage {
		unrolled = append(unrolled, slice...)
	}
	if len(unrolled) == 0 {
		innerKeccak := crypto.Keccak256Hash([]byte{})
		outerKeccak := crypto.Keccak256Hash(innerKeccak.Bytes())
		record(outerKeccak, innerKeccak.Bytes())
		record(innerKeccak, []byte{})
		return outerKeccak
	}

	length := int64(len(unrolled))
	leaves := []bytes32{}
	for bin := int64(0); bin < length; bin += binSize {
		end := arbmath.MinInt(bin+binSize, length)
		content := unrolled[bin:end]
		innerKeccak := crypto.Keccak256Hash(content)
		outerKeccak := crypto.Keccak256Hash(innerKeccak.Bytes())
		record(outerKeccak, innerKeccak.Bytes())
		record(innerKeccak, content)
		leaves = append(leaves, outerKeccak)
	}

	layer := leaves
	for len(layer) > 1 {
		prior := len(layer)
		after := prior/2 + prior%2
		paired := make([]bytes32, after)
		for i := 0; i < prior-1; i += 2 {
			leftChild := layer[i].Bytes()
			rightChild := layer[i+1].Bytes()
			parent := crypto.Keccak256Hash(leftChild, rightChild)
			record(parent, append(leftChild, rightChild...))
			paired[i/2] = parent
		}
		if prior%2 == 1 {
			paired[after-1] = layer[prior-1]
		}
		layer = paired
	}
	return layer[0]
}

func Hash(preimage ...[]byte) bytes32 {
	// Merkelizes without recording anything. All but the replay binary's DAS will call this
	return RecordHash(func(bytes32, []byte) {}, preimage...)
}

func HashBytes(preimage ...[]byte) []byte {
	return Hash(preimage...).Bytes()
}

func Content(root common.Hash, oracle func(common.Hash) []byte) ([]byte, error) {
	leaves := []common.Hash{}
	stack := []common.Hash{root}

	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		under := oracle(node)

		switch len(under) {
		case 32:
			leaves = append(leaves, common.BytesToHash(under))
		case 64:
			prior := common.BytesToHash(under[:32]) // we want to expand leftward,
			after := common.BytesToHash(under[32:]) // so we reverse their order
			stack = append(stack, after, prior)
		default:
			return nil, fmt.Errorf("failed to resolve preimage %v %v", len(under), node)
		}
	}

	preimage := []byte{}
	for _, leaf := range leaves {
		preimage = append(preimage, oracle(leaf)...)
	}
	return preimage, nil
}
