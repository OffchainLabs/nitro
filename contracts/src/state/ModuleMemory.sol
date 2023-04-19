// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./MerkleProof.sol";
import "./Deserialize.sol";
import "./ModuleMemoryCompact.sol";

library ModuleMemoryLib {
    using MerkleProofLib for MerkleProof;

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
}
