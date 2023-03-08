// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/libraries/MerkleTreeLib.sol";
import "../../src/libraries/MerkleLib.sol";
import "./Utils.sol";

contract MerkleTreeLibTest is Test {
    Random random = new Random();

    function clone(bytes32[] memory arr) internal pure returns (bytes32[] memory) {
        bytes32[] memory newArr = new bytes32[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            newArr[i] = arr[i];
        }
        return newArr;
    }

    function eq(bytes32[] memory arr1, bytes32[] memory arr2) internal pure {
        require(keccak256(abi.encode(arr1)) == keccak256(abi.encode(arr2)), "Arrays not equal");
    }

    function testDoesAppend() public {
        // CHRIS: TODO: add more assertions to this test
        bytes32[] memory me = new bytes32[](0);
        assertEq(MerkleTreeLib.root(me), 0, "Zero root");

        bytes32 h0 = random.hash();
        me = MerkleTreeLib.appendCompleteSubTree(me, 0, h0);

        bytes32 h1 = random.hash();
        me = MerkleTreeLib.appendCompleteSubTree(me, 0, h1);
        bytes32[] memory me2 = clone(me);

        bytes32 h2 = random.hash();
        bytes32 h3 = random.hash();
        bytes32 h23 = keccak256(abi.encodePacked(h2, h3));
        me = MerkleTreeLib.appendCompleteSubTree(me, 1, h23);

        bytes32[] memory me4 = clone(me);
        me = MerkleTreeLib.appendCompleteSubTree(me2, 0, h2);
        me = MerkleTreeLib.appendCompleteSubTree(me, 0, h3);
        eq(me4, me);
    }

    function expansionsFromLeaves(bytes32[] memory leaves, uint256 lowSize)
        public
        pure
        returns (bytes32[] memory, bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory lowExpansion = new bytes32[](0);
        bytes32[] memory highExpansion = new bytes32[](0);
        bytes32[] memory difference = new bytes32[](leaves.length - lowSize);

        for (uint256 i = 0; i < leaves.length; i++) {
            if (i < lowSize) {
                lowExpansion = MerkleTreeLib.appendLeaf(lowExpansion, leaves[i]);
            } else {
                difference[i - lowSize] = leaves[i];
            }

            highExpansion = MerkleTreeLib.appendLeaf(highExpansion, leaves[i]);
        }

        return (lowExpansion, highExpansion, difference);
    }

    function proveVerify(uint256 startSize, uint256 endSize) internal {
        bytes32[] memory leaves = random.hashes(endSize);
        (bytes32[] memory lowExp, bytes32[] memory highExp, bytes32[] memory diff) =
            expansionsFromLeaves(leaves, startSize);

        bytes32[] memory proof = ProofUtils.generatePrefixProof(startSize, diff);

        MerkleTreeLib.verifyPrefixProof(
            MerkleTreeLib.root(lowExp), startSize, MerkleTreeLib.root(highExp), endSize, lowExp, proof
        );
    }

    function testDoesProve() public {
        proveVerify(1, 2);
        proveVerify(1, 3);
        proveVerify(2, 3);
        proveVerify(2, 13);
        proveVerify(17, 7052);
        proveVerify(23, 7052);
        proveVerify(20, 7052);
    }

    function verifyInclusion(uint256 index, uint256 treeSize) internal {
        bytes32[] memory leaves = random.hashes(treeSize);
        bytes32[] memory re = ProofUtils.rehashed(leaves);
        bytes32[] memory me = ProofUtils.expansionFromLeaves(leaves, 0, leaves.length);
        bytes32[] memory proof = ProofUtils.generateInclusionProof(re, index);

        bool v2 = MerkleTreeLib.hasState(MerkleTreeLib.root(me), leaves[index], index, proof);
        assertTrue(v2, "Invalid v2 root");
    }

    function testProveInclusion() public {
        uint256 size = 16;
        for (uint256 i = 0; i < size; i++) {
            for (uint256 j = 0; j < i; j++) {
                verifyInclusion(j, i);
            }
        }
    }
}
