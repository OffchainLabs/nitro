// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../challengeV2/libraries/MerkleTreeLib.sol";

contract MerkleTreeAccess {
    function mostSignificantBit(uint256 x) external pure returns (uint256) {
        return UintUtils.mostSignificantBit(x);
    }
    function leastSignificantBit(uint256 x) external pure returns (uint256) {
        return UintUtils.leastSignificantBit(x);
    }
    function root(bytes32[] memory me) external pure returns (bytes32) {
        return MerkleTreeLib.root(me);
    }

    function maskMsb(uint256 msb) external pure returns (uint64) {
        return uint64((1<<(msb) + 1) - 1);
    }

    function appendCompleteSubTree(bytes32[] memory me, uint256 level, bytes32 subtreeRoot)
        external
        pure
        returns (bytes32[] memory)
    {
        return MerkleTreeLib.appendCompleteSubTree(me, level, subtreeRoot);
    }

    function appendLeaf(bytes32[] memory me, bytes32 leaf) external pure returns (bytes32[] memory) {
        return MerkleTreeLib.appendLeaf(me, leaf);
    }

    function maximumAppendBetween(uint256 startSize, uint256 endSize) external pure returns (uint256) {
        return MerkleTreeLib.maximumAppendBetween(startSize, endSize);
    }

    function verifyPrefixProof(
        bytes32 preRoot,
        uint256 preSize,
        bytes32 postRoot,
        uint256 postSize,
        bytes32[] memory preExpansion,
        bytes32[] memory proof
    ) external pure {
        return MerkleTreeLib.verifyPrefixProof(preRoot, preSize, postRoot, postSize, preExpansion, proof);
    }

    function partialCompute(
        bytes32 preRoot,
        uint256 preSize,
        bytes32 postRoot,
        uint256 postSize,
        bytes32[] memory preExpansion,
        bytes32[] memory proof
    ) external pure returns (bytes32[] memory) {
        return MerkleTreeLib.partialCompute(preRoot, preSize, postRoot, postSize, preExpansion, proof);
    }

    function hasState(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] memory proof)
        external
        pure
        returns (bool)
    {
        return MerkleTreeLib.hasState(rootHash, leaf, index, proof);
    }
}
