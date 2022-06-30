// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dastree

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const binSize = 64 * 1024 // 64 kB

type bytes32 = common.Hash

func Hash(preimage ...[]byte) []byte {
	// Algorithm
	//  1. split the preimage into 64kB bins and hash them to produces the tree's leaves
	//  2. repeatedly hash pairs over and over, bubbling up any odd-one's out, forming the root
	//
	//            r         <=>  hash(hash(0, 1), 2)           step 2
	//           / \
	//          *   2       <=>  hash(0, 1), 2                 step 1
	//         / \
	//        0   1         <=>  0, 1, 2                       step 0

	unrolled := []byte{}
	for _, slice := range preimage {
		unrolled = append(unrolled, slice...)
	}
	if len(unrolled) == 0 {
		return crypto.Keccak256([]byte{})
	}

	length := int64(len(unrolled))
	leaves := []bytes32{}
	for bin := int64(0); bin < length; bin += binSize {
		end := arbmath.MinInt(bin+binSize, length)
		keccak := crypto.Keccak256Hash(unrolled[bin:end])
		leaves = append(leaves, keccak)
	}

	layer := leaves
	for len(layer) > 1 {
		prior := len(layer)
		after := prior/2 + prior%2
		paired := make([]bytes32, after)
		for i := 0; i < prior-1; i += 2 {
			paired[i/2] = crypto.Keccak256Hash(layer[i][:], layer[i+1][:])
		}
		if prior%2 == 1 {
			paired[after-1] = layer[prior-1]
		}
		layer = paired
	}

	return layer[0][:]
}
