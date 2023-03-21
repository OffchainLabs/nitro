// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../../src/challengeV2/libraries/MerkleTreeLib.sol";
import "../../src/challengeV2/libraries/UintUtilsLib.sol";
import "../../src/challengeV2/libraries/ArrayUtilsLib.sol";
import "forge-std/Test.sol";

contract Random {
    bytes32 private seed = 0xf19f64ef5b8c788ff3f087b4f75bc6596a6aaa3c9048bbbbe990fa0870261385;

    function hash() public returns (bytes32) {
        seed = keccak256(abi.encodePacked(seed));
        return seed;
    }

    function hashes(uint256 count) public returns (bytes32[] memory) {
        bytes32[] memory h = new bytes32[](count);
        for (uint256 i = 0; i < h.length; i++) {
            h[i] = hash();
        }
        return h;
    }

    function addr() public returns (address) {
        seed = keccak256(abi.encodePacked(seed));
        return address(bytes20(seed));
    }
}

library Logger {
    function bytes32Array(string memory name, bytes32[] memory arr) internal view {
        console.log(name);
        for (uint256 i = 0; i < arr.length; i++) {
            console.logBytes32(arr[i]);
        }
        console.log("-----------------------------");
    }
}

library ProofUtils {
    /// @notice Create a merkle expansion from an array of leaves
    /// @param leaves The leaves to form into an expansion
    /// @param leafStartIndex The subset of the leaves to start the expansion from - inclusive
    /// @param leafEndIndex The subset of the leaves to end the expansion from - exclusive
    function expansionFromLeaves(bytes32[] memory leaves, uint256 leafStartIndex, uint256 leafEndIndex)
        internal
        pure
        returns (bytes32[] memory)
    {
        require(leafStartIndex < leafEndIndex, "Leaf start not less than leaf end");
        require(leafEndIndex <= leaves.length, "Leaf end not less than leaf length");

        bytes32[] memory expansion = new bytes32[](0);
        for (uint256 i = leafStartIndex; i < leafEndIndex; i++) {
            expansion = MerkleTreeLib.appendLeaf(expansion, leaves[i]);
        }

        return expansion;
    }

    /// @notice Generate a proof that a tree of size preSize when appended to with newLeaves
    ///         results in the tree at size preSize + newLeaves.length
    /// @dev    The proof is the minimum number of complete sub trees that must
    ///         be appended to the pre tree in order to produce the post tree.
    function generatePrefixProof(uint256 preSize, bytes32[] memory newLeaves)
        internal
        pure
        returns (bytes32[] memory)
    {
        require(preSize > 0, "Pre-size cannot be 0");
        require(newLeaves.length > 0, "No new leaves added");

        uint256 size = preSize;
        uint256 postSize = size + newLeaves.length;
        bytes32[] memory proof = new bytes32[](0);

        // We always want to append the subtrees at the maximum level, so that we cover the most
        // leaves possible. We do this by finding the maximum level between the start and the end
        // that we can append at, then append these leaves, then repeat the process.

        while (size < postSize) {
            uint256 level = MerkleTreeLib.maximumAppendBetween(size, postSize);
            // add 2^level leaves to create a subtree
            uint256 numLeaves = 1 << level;

            uint256 startIndex = size - preSize;
            uint256 endIndex = startIndex + numLeaves;
            // create a complete sub tree at the specified level
            bytes32[] memory exp = expansionFromLeaves(newLeaves, startIndex, endIndex);
            proof = ArrayUtilsLib.append(proof, MerkleTreeLib.root(exp));

            size += numLeaves;

            assert(size <= postSize);
        }

        return proof;
    }

    function generateInclusionProof(bytes32[] memory leaves, uint256 index) internal pure returns (bytes32[] memory) {
        require(leaves.length >= 1, "No leaves");
        require(index < leaves.length, "Index too high");
        bytes32[][] memory fullT = fullTree(leaves);
        if (leaves.length == 1) return new bytes32[](0);

        uint256 maxLevel = UintUtilsLib.mostSignificantBit(leaves.length - 1);

        bytes32[] memory proof = new bytes32[](maxLevel + 1);

        for (uint256 level = 0; level <= maxLevel; level++) {
            uint256 levelIndex = index >> level;

            uint256 counterpartIndex = levelIndex ^ 1;
            bytes32[] memory layer = fullT[level];
            bytes32 counterpart = counterpartIndex > layer.length - 1 ? bytes32(0) : layer[counterpartIndex];

            proof[level] = counterpart;
        }
        return proof;
    }

    function fullTree(bytes32[] memory leaves) internal pure returns (bytes32[][] memory) {
        uint256 msb = UintUtilsLib.mostSignificantBit(leaves.length);
        uint256 lsb = UintUtilsLib.leastSignificantBit(leaves.length);

        uint256 maxLevel = msb == lsb ? msb : msb + 1;

        bytes32[][] memory layers = new bytes32[][](maxLevel + 1);
        layers[0] = leaves;
        uint256 l = 1;

        bytes32[] memory prevLayer = leaves;
        while (prevLayer.length > 1) {
            bytes32[] memory nextLayer = new bytes32[]((prevLayer.length + 1) / 2);
            for (uint256 i = 0; i < nextLayer.length; i++) {
                if (2 * i + 1 < prevLayer.length) {
                    nextLayer[i] = keccak256(abi.encodePacked(prevLayer[2 * i], prevLayer[2 * i + 1]));
                } else {
                    nextLayer[i] = keccak256(abi.encodePacked(prevLayer[2 * i], bytes32(0)));
                }
            }
            layers[l] = nextLayer;
            prevLayer = nextLayer;
            l++;
        }
        return layers;
    }

    function rehashed(bytes32[] memory arr) internal pure returns (bytes32[] memory) {
        bytes32[] memory arr2 = new bytes32[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            arr2[i] = keccak256(abi.encodePacked(arr[i]));
        }
        return arr2;
    }
}
