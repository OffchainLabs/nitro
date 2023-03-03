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

    function logArr(bytes32[] memory arr1) internal view {
        console.log("----------------------");
        for (uint256 i = 0; i < arr1.length; i++) {
            console.logBytes32(arr1[i]);
        }
    }

    function eq(bytes32[] memory arr1, bytes32[] memory arr2) internal view {
        require(keccak256(abi.encode(arr1)) == keccak256(abi.encode(arr2)), "Arrays not equal");
    }

    function rehashed(bytes32[] memory arr) internal pure returns (bytes32[] memory) {
        bytes32[] memory arr2 = new bytes32[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            arr2[i] = keccak256(abi.encodePacked(arr[i]));
        }
        return arr2;
    }

    function testRootsMatch(uint256 length) public {
        vm.assume(length > 0);
        vm.assume(length < 1000);
        // uint256 length = 5;
        bytes32[] memory h = random.hashes(length);
        bytes32[] memory rh = rehashed(h);

        bytes32[] memory me = MerkleTreeLib.expansionFromLeaves(h, 0, length);
        bytes32 root = MerkleLib.generateRoot(rh);

        assertEq(root, MerkleTreeLib.root(me), "Equal roots");
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

    function expansionsFromLeaves(bytes32[] memory leaves, uint256 lowHeight)
        public
        view
        returns (bytes32[] memory, bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory lowExpansion = new bytes32[](0);
        bytes32[] memory highExpansion = new bytes32[](0);
        bytes32[] memory difference = new bytes32[](leaves.length - lowHeight);

        for (uint256 i = 0; i < leaves.length; i++) {
            if (i < lowHeight) {
                lowExpansion = MerkleTreeLib.appendLeaf(lowExpansion, leaves[i]);
            } else {
                difference[i - lowHeight] = leaves[i];
            }

            highExpansion = MerkleTreeLib.appendLeaf(highExpansion, leaves[i]);
        }

        return (lowExpansion, highExpansion, difference);
    }

    function proveVerify(uint256 startHeight, uint256 endHeight) internal {
        bytes32[] memory leaves = random.hashes(endHeight);
        (bytes32[] memory lowExp, bytes32[] memory highExp, bytes32[] memory diff) =
            expansionsFromLeaves(leaves, startHeight);

        bytes32[] memory proof = MerkleTreeLib.generatePrefixProof(startHeight, diff);

        MerkleTreeLib.verifyPrefixProof(
            MerkleTreeLib.root(lowExp), startHeight, MerkleTreeLib.root(highExp), endHeight, lowExp, proof
        );
    }

    function testDoesProve() public {
        proveVerify(1, 2);
        // proveVerify(1, 3);
        // proveVerify(2, 3);
        // proveVerify(2, 13);
        // proveVerify(17, 7052);
        // proveVerify(23, 7052);
        // proveVerify(20, 7052);
    }
}
