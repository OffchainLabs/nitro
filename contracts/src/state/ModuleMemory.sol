// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./MerkleProof.sol";
import "./Deserialize.sol";
import "./ModuleMemoryCompact.sol";

library ModuleMemoryLib {
    using MerkleProofLib for MerkleProof;

    uint256 private constant LEAF_SIZE = 32;

    function hash(ModuleMemory memory mem) internal pure returns (bytes32) {
        return ModuleMemoryCompactLib.hash(mem);
    }

    function proveLeaf(
        ModuleMemory memory mem,
        uint256 leafIdx,
        bytes calldata proof,
        uint256 startOffset
    )
        internal
        pure
        returns (
            bytes32 contents,
            uint256 offset,
            MerkleProof memory merkle
        )
    {
        offset = startOffset;
        (contents, offset) = Deserialize.b32(proof, offset);
        (merkle, offset) = Deserialize.merkleProof(proof, offset);
        bytes32 recomputedRoot = merkle.computeRootFromMemory(leafIdx, contents);
        require(recomputedRoot == mem.merkleRoot, "WRONG_MEM_ROOT");
    }

    function isValidLeaf(ModuleMemory memory mem, uint256 pointer) internal pure returns (bool) {
        return pointer + 32 <= mem.size && pointer % LEAF_SIZE == 0;
    }

    function pullLeafByte(bytes32 leaf, uint256 idx) internal pure returns (uint8) {
        require(idx < LEAF_SIZE, "BAD_PULL_LEAF_BYTE_IDX");
        // Take into account that we are casting the leaf to a big-endian integer
        uint256 leafShift = (LEAF_SIZE - 1 - idx) * 8;
        return uint8(uint256(leaf) >> leafShift);
    }

    // loads a big-endian value from memory
    function load(
        ModuleMemory memory mem,
        uint256 start,
        uint256 width,
        bytes calldata proof,
        uint256 proofOffset
    )
        internal
        pure
        returns (
            bool err,
            uint256 value,
            uint256 offset
        )
    {
        if (start + width > mem.size) {
            return (true, 0, proofOffset);
        }

        uint256 lastProvedLeafIdx = ~uint256(0);
        bytes32 lastProvedLeafContents;
        uint256 readValue;
        for (uint256 i = 0; i < width; i++) {
            uint256 idx = start + i;
            uint256 leafIdx = idx / LEAF_SIZE;
            if (leafIdx != lastProvedLeafIdx) {
                (lastProvedLeafContents, proofOffset, ) = proveLeaf(
                    mem,
                    leafIdx,
                    proof,
                    proofOffset
                );
                lastProvedLeafIdx = leafIdx;
            }
            uint256 indexWithinLeaf = idx % LEAF_SIZE;
            readValue |= uint256(pullLeafByte(lastProvedLeafContents, indexWithinLeaf)) << (i * 8);
        }
        return (false, readValue, proofOffset);
    }
}
