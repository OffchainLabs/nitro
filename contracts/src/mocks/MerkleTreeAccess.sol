// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../challengeV2/libraries/MerkleTreeLib.sol";

contract MerkleTreeAccess {
    function root(bytes32[] memory me) internal pure returns (bytes32) {
        return MerkleTreeLib.root(me);
    }

    function appendCompleteSubTree(bytes32[] memory me, uint256 level, bytes32 subtreeRoot)
        internal
        pure
        returns (bytes32[] memory)
    {
        return MerkleTreeLib.appendCompleteSubTree(me, level, subtreeRoot);
    }

    function appendLeaf(bytes32[] memory me, bytes32 leaf) internal pure returns (bytes32[] memory) {
        return MerkleTreeLib.appendLeaf(me, leaf);
    }

    function maximumAppendBetween(uint256 startSize, uint256 endSize) internal pure returns (uint256) {
        return MerkleTreeLib.maximumAppendBetween(startSize, endSize);
    }

    function verifyPrefixProof(
        bytes32 preRoot,
        uint256 preSize,
        bytes32 postRoot,
        uint256 postSize,
        bytes32[] memory preExpansion,
        bytes32[] memory proof
    ) internal pure {
        MerkleTreeLib.verifyPrefixProof(preRoot, preSize, postRoot, postSize, preExpansion, proof);
    }

    function hasState(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] memory proof)
        internal
        pure
        returns (bool)
    {
        return MerkleTreeLib.hasState(rootHash, leaf, index, proof);
    }
}
