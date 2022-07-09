// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dastree

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const BinSize = 64 * 1024 // 64 kB

type bytes32 = common.Hash

type node struct {
	hash bytes32
	size uint32
}

func RecordHash(record func(bytes32, []byte), preimage ...[]byte) bytes32 {
	// Algorithm
	//  1. split the preimage into 64kB bins and hash them to produce the tree's leaves
	//  2. repeatedly hash pairs and their combined length, bubbling up any odd-one's out, to form the root
	//
	//            r         <=>  hash(hash(0, 1), 2, len(0:2))        step 2
	//           / \
	//          *   2       <=>  hash(0, 1, len(0:1)), 2              step 1
	//         / \
	//        0   1         <=>  0, 1, 2                              step 0
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

	length := uint32(len(unrolled))
	leaves := []node{}
	for bin := uint32(0); bin < length; bin += BinSize {
		end := arbmath.MinUint32(bin+BinSize, length)
		content := unrolled[bin:end]
		innerKeccak := crypto.Keccak256Hash(content)
		outerKeccak := crypto.Keccak256Hash(innerKeccak.Bytes())
		record(outerKeccak, innerKeccak.Bytes())
		record(innerKeccak, content)
		leaves = append(leaves, node{outerKeccak, end - bin})
	}

	layer := leaves
	for len(layer) > 1 {
		prior := len(layer)
		after := prior/2 + prior%2
		paired := make([]node, after)
		for i := 0; i < prior-1; i += 2 {
			firstHash := layer[i].hash.Bytes()
			otherHash := layer[i+1].hash.Bytes()
			sizeUnder := layer[i].size + layer[i+1].size
			dataUnder := firstHash
			dataUnder = append(dataUnder, otherHash...)
			dataUnder = append(dataUnder, arbmath.Uint32ToBytes(sizeUnder)...)
			parent := node{
				crypto.Keccak256Hash(dataUnder),
				sizeUnder,
			}
			record(parent.hash, dataUnder)
			paired[i/2] = parent
		}
		if prior%2 == 1 {
			paired[after-1] = layer[prior-1]
		}
		layer = paired
	}
	return layer[0].hash
}

func Hash(preimage ...[]byte) bytes32 {
	// Merkelizes without recording anything. All but the replay binary's DAS will call this
	return RecordHash(func(bytes32, []byte) {}, preimage...)
}

func HashBytes(preimage ...[]byte) []byte {
	return Hash(preimage...).Bytes()
}

func FlatHashToTreeHash(flat bytes32) bytes32 {
	// Forms a degenerate dastree that's just a single leaf
	// note: the inner preimage may be larger than the 64 kB standard
	return crypto.Keccak256Hash(flat[:])
}

func Content(root bytes32, oracle func(bytes32) []byte) ([]byte, error) {
	// Reverses hashes to reveal the full preimage under the root using the preimage oracle.
	// This function also checks that the size-data is consistent and that the hash is canonical.
	//
	// Notes
	//     1. Because we accept degenerate dastrees, we can't check that single-leaf trees are canonical.
	//     2. For any canonical dastree, there exists a degenerate single-leaf equivalent that we accept.
	//     3. Only the committee can produce trees unwrapped by this function
	//

	total := uint32(0)
	upper := oracle(root)
	switch len(upper) {
	case 32:
		return oracle(common.BytesToHash(upper)), nil
	case 68:
		total = binary.BigEndian.Uint32(upper[64:])
	default:
		return nil, fmt.Errorf("invalid root with preimage of size %v: %v %v", len(upper), root, upper)
	}

	leaves := []node{}
	stack := []node{{hash: root, size: total}}

	for len(stack) > 0 {
		place := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		under := oracle(place.hash)

		switch len(under) {
		case 32:
			leaf := node{
				hash: common.BytesToHash(under),
				size: place.size,
			}
			leaves = append(leaves, leaf)
		case 68:
			count := binary.BigEndian.Uint32(under[64:])
			power := uint32(arbmath.NextOrCurrentPowerOf2(uint64(count)))

			if place.size != count {
				return nil, fmt.Errorf("invalid size data: %v vs %v for %v", count, place.size, under)
			}

			prior := node{
				hash: common.BytesToHash(under[:32]),
				size: power / 2,
			}
			after := node{
				hash: common.BytesToHash(under[32:64]),
				size: count - power/2,
			}

			// we want to expand leftward so we reverse their order
			stack = append(stack, after, prior)
		default:
			return nil, fmt.Errorf("failed to resolve preimage %v %v", place.hash, under)
		}
	}

	preimage := []byte{}
	for i, leaf := range leaves {
		bin := oracle(leaf.hash)
		if len(bin) != int(leaf.size) {
			return nil, fmt.Errorf("leaf %v has an incorrectly sized bin: %v vs %v", i, len(bin), leaf.size)
		}
		preimage = append(preimage, bin...)
	}

	// Check the hash matches. Given the size data this should never fail but we'll check anyway
	if Hash(preimage) != root {
		return nil, fmt.Errorf("preimage not canonically hashed")
	}
	return preimage, nil
}
