// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dastree

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

const BinSize = 64 * 1024 // 64 kB
const LeafByte = 0xfe
const NodeByte = 0xff

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
		single := []byte{LeafByte}
		keccak := crypto.Keccak256Hash(single)
		record(keccak, single)
		return keccak
	}

	length := uint32(len(unrolled))
	leaves := []node{}
	for bin := uint32(0); bin < length; bin += BinSize {
		end := arbmath.MinUint32(bin+BinSize, length)
		single := append([]byte{LeafByte}, unrolled[bin:end]...)
		keccak := crypto.Keccak256Hash(single)
		record(keccak, single)
		leaves = append(leaves, node{keccak, end - bin})
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
			dataUnder := append([]byte{NodeByte}, firstHash...)
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

func ValidHash(hash bytes32, preimage []byte) bool {
	// TODO: remove keccak after committee upgrade
	return hash == Hash(preimage) || hash == crypto.Keccak256Hash(preimage)
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
	switch {
	case len(upper) > 0 && upper[0] == LeafByte:
		return upper[1:], nil
	case len(upper) == 69 && upper[0] == NodeByte:
		total = binary.BigEndian.Uint32(upper[65:])
	default:
		return nil, fmt.Errorf("invalid root with preimage of size %v: %v %v", len(upper), root, upper)
	}

	stack := []node{{hash: root, size: total}}
	preimage := []byte{}

	for len(stack) > 0 {
		place := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		colors.PrintYellow("here ", place.hash, place.size)

		under := oracle(place.hash)

		if len(under) == 0 || (under[0] == NodeByte && len(under) != 69) {
			return nil, fmt.Errorf("invalid node for hash %v: %v", place.hash, under)
		}

		kind := under[0]
		content := under[1:]

		switch kind {
		case LeafByte:
			if len(content) != int(place.size) {
				return nil, fmt.Errorf("leaf has a badly sized bin: %v vs %v", len(under), place.size)
			}
			preimage = append(preimage, content...)
		case NodeByte:
			count := binary.BigEndian.Uint32(content[64:])
			power := uint32(arbmath.NextOrCurrentPowerOf2(uint64(count)))

			if place.size != count {
				return nil, fmt.Errorf("invalid size data: %v vs %v for %v", count, place.size, under)
			}

			prior := node{
				hash: common.BytesToHash(content[:32]),
				size: power / 2,
			}
			after := node{
				hash: common.BytesToHash(content[32:64]),
				size: count - power/2,
			}

			// we want to expand leftward so we reverse their order
			stack = append(stack, after, prior)
		default:
			return nil, fmt.Errorf("failed to resolve preimage %v %v", place.hash, under)
		}
	}

	// Check the hash matches. Given the size data this should never fail but we'll check anyway
	if Hash(preimage) != root {
		return nil, fmt.Errorf("preimage not canonically hashed")
	}
	return preimage, nil
}
