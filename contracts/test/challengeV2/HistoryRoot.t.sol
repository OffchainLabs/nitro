// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/libraries/HistoryRootLib.sol";
import "./Utils.sol";

contract MerklePrefixProofLibTest is Test {
    Random random = new Random();

    function clone(bytes32[] memory arr) internal pure returns (bytes32[] memory) {
        bytes32[] memory newArr = new bytes32[](arr.length);
        for (uint256 i = 0; i < arr.length; i++) {
            newArr[i] = arr[i];
        }
        return newArr;
    }

    function eq(bytes32[] memory arr1, bytes32[] memory arr2) internal pure {
        require(keccak256(abi.encode(arr1)) == keccak256(abi.encode(arr2)));
    }

    function testDoesAppend() public {
        // CHRIS: TODO: add more assertions to this test
        bytes32[] memory me = new bytes32[](0);
        assertEq(MerkleExpansionLib.root(me), 0, "Zero root");

        bytes32 h0 = random.hash();
        me = MerkleExpansionLib.appendCompleteSubTree(me, 0, h0);

        bytes32 h1 = random.hash();
        me = MerkleExpansionLib.appendCompleteSubTree(me, 0, h1);

        bytes32[] memory me2 = clone(me);

        bytes32 h2 = random.hash();
        bytes32 h3 = random.hash();
        bytes32 h23 = keccak256(abi.encodePacked(h2, h3));
        me = MerkleExpansionLib.appendCompleteSubTree(me, 1, h23);

        bytes32[] memory me4 = clone(me);
        me = MerkleExpansionLib.appendCompleteSubTree(me2, 0, h2);
        me = MerkleExpansionLib.appendCompleteSubTree(me, 0, h3);
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
                lowExpansion = MerkleExpansionLib.appendLeaf(lowExpansion, leaves[i]);
            } else {
                difference[i - lowHeight] = leaves[i];
            }

            highExpansion = MerkleExpansionLib.appendLeaf(highExpansion, leaves[i]);
        }

        return (lowExpansion, highExpansion, difference);
    }

    function proveVerify(uint256 startHeight, uint256 endHeight) internal {
        bytes32[] memory leaves = random.hashes(endHeight);
        (bytes32[] memory lowExp, bytes32[] memory highExp, bytes32[] memory diff) =
            expansionsFromLeaves(leaves, startHeight);

        bytes32[] memory proof = HistoryRootLib.generatePrefixProof(startHeight, diff);

        HistoryRootLib.verifyPrefixProof(
            MerkleExpansionLib.root(lowExp), startHeight, MerkleExpansionLib.root(highExp), endHeight, lowExp, proof
        );
    }

    function testDoesProve() public {
        proveVerify(1, 2);
        proveVerify(1, 3);
        proveVerify(2, 3);
        proveVerify(2, 4);
        proveVerify(2, 13);
        proveVerify(18, 398);
        proveVerify(24, 3983);
        proveVerify(17, 3983);
    }

    function testTesty() public {
        uint256 a = HistoryRootLib.mostSignificantBit(10);

        uint256 b = 1 << a;

        uint256 c = HistoryRootLib.leastSignificantBit(1);

        // uint256 b = 1 << a;
    }
}
