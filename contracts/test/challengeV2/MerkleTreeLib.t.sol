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
        // some basic tests
        bytes32[] memory me = new bytes32[](0);
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

    function testVerifyPrefixProof() public {
        // similar tests to proof-tests.go
        proveVerify(1, 2);
        proveVerify(1, 3);
        proveVerify(2, 3);
        proveVerify(2, 13);
        proveVerify(17, 7052);
        proveVerify(23, 7052);
        proveVerify(20, 7052);
    }

    function testRoot(uint256 size) public {
        vm.assume(size > 0);
        vm.assume(size < 257);
        bytes32[] memory hashes = random.hashes(size);
        bytes32[] memory rehashed = ProofUtils.rehashed(hashes);
        bytes32[] memory expansion = ProofUtils.expansionFromLeaves(hashes, 0, size);

        bytes32[][] memory fullTree = ProofUtils.fullTree(rehashed);
        bytes32 root = fullTree[fullTree.length - 1][0];

        bytes32 expRoot = MerkleTreeLib.root(expansion);
        assertEq(root, expRoot, "Roots");
    }

    function getExpansion(uint256 leafCount) internal returns (bytes32[] memory) {
        bytes32[] memory hashes = random.hashes(leafCount);
        bytes32[] memory expansion = ProofUtils.expansionFromLeaves(hashes, 0, leafCount);
        return expansion;
    }

    function hashTogether(bytes32 a, bytes32 b) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(a, b));
    }

    function testRoot1() public {
        bytes32[] memory expansion = getExpansion(1);
        bytes32 expectedRoot = expansion[0];
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot2() public {
        bytes32[] memory expansion = getExpansion(2);
        bytes32 expectedRoot = expansion[1];
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot3() public {
        bytes32[] memory expansion = getExpansion(3);
        bytes32 expectedRoot = hashTogether(expansion[1], hashTogether(expansion[0], 0));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot4() public {
        bytes32[] memory expansion = getExpansion(4);
        bytes32 expectedRoot = expansion[2];
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot5() public {
        bytes32[] memory expansion = getExpansion(5);
        bytes32 expectedRoot = hashTogether(expansion[2], hashTogether(hashTogether(expansion[0], 0), 0));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot6() public {
        bytes32[] memory expansion = getExpansion(6);
        bytes32 expectedRoot = hashTogether(expansion[2], hashTogether(expansion[1], 0));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot7() public {
        bytes32[] memory expansion = getExpansion(7);
        bytes32 expectedRoot = hashTogether(expansion[2], hashTogether(expansion[1], hashTogether(expansion[0], 0)));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot8() public {
        bytes32[] memory expansion = getExpansion(8);
        bytes32 expectedRoot = expansion[3];
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot9() public {
        bytes32[] memory expansion = getExpansion(9);
        bytes32 expectedRoot =
            hashTogether(expansion[3], hashTogether(hashTogether(hashTogether(expansion[0], 0), 0), 0));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRoot11() public {
        bytes32[] memory expansion = getExpansion(10);
        bytes32 expectedRoot = hashTogether(expansion[3], hashTogether(hashTogether(expansion[1], 0), 0));
        assertEq(MerkleTreeLib.root(expansion), expectedRoot, "Invalid root");
    }

    function testRootEmpty() public {
        bytes32[] memory expansion = new bytes32[](0);
        vm.expectRevert("Empty merkle expansion");
        MerkleTreeLib.root(expansion);
    }

    function testRootTooLarge() public {
        bytes32[] memory expansion = new bytes32[](MerkleTreeLib.MAX_LEVEL + 1);
        vm.expectRevert("Merkle expansion too large");
        MerkleTreeLib.root(expansion);
    }

    function testAppendCS(uint256 treeSize) public {
        vm.assume(treeSize > 0);
        vm.assume(treeSize < 16);

        bytes32[] memory expansion = getExpansion(treeSize);
        bool lowestLevel = false;
        for (uint256 i = 0; i < expansion.length; i++) {
            bytes32 rand = random.hash();
            if (lowestLevel) {
                vm.expectRevert("Append above least significant bit");
            }
            bytes32[] memory post = MerkleTreeLib.appendCompleteSubTree(expansion, i, rand);

            if (expansion[i] != 0) {
                lowestLevel = true;
            } else {
                for (uint256 j = 0; j < expansion.length; j++) {
                    if (j == i) {
                        assertEq(post[j], rand, "Different level hash");
                    } else {
                        assertEq(post[j], expansion[j], "Conflicting level");
                    }
                }
            }

            if (expansion[expansion.length - 1] != post[expansion.length - 1]) {
                assertEq(post.length, expansion.length + 1, "Level increase");
            } else {
                assertEq(post.length, expansion.length, "Level same");
            }

            uint256 preSize = MerkleTreeLib.treeSize(expansion);
            uint256 postSize = MerkleTreeLib.treeSize(post);
            assertEq(postSize, preSize + (2 ** i), "Sizes");
        }
    }

    function plainAppend(uint256 level) internal {
        bytes32[] memory pre = getExpansion(44);

        bytes32 rand = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendCompleteSubTree(pre, level, rand);
        assertEq(pre.length, post.length, "Pre post len");
        for (uint256 i = 0; i < pre.length; i++) {
            if (i == level) {
                assertEq(post[i], rand, "Post equal");
            } else {
                assertEq(pre[i], post[i], "Pre post equal");
            }
        }
    }

    function testAppendCS0() public {
        plainAppend(0);
    }

    function testAppendCS1() public {
        plainAppend(1);
    }

    function testAppendCS2() public {
        // 101100 = 44
        bytes32[] memory pre = getExpansion(44);
        uint256 level = 2;

        bytes32 rand = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendCompleteSubTree(pre, level, rand);
        assertEq(pre.length, post.length, "Pre post len");
        for (uint256 i = 0; i < pre.length; i++) {
            if (i == level || i == level + 1) {
                assertEq(post[i], 0, "Post level equal");
            } else if (i == level + 2) {
                assertEq(post[i], hashTogether(pre[i - 1], hashTogether(pre[i - 2], rand)), "Post level plus 1 equal");
            } else {
                assertEq(pre[i], post[i], "Pre post equal");
            }
        }
    }

    function testAppendCS2IncreaseHeight() public {
        // 1100 = 12
        bytes32[] memory pre = getExpansion(12);
        uint256 level = 2;

        bytes32 rand = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendCompleteSubTree(pre, level, rand);
        assertEq(post.length, pre.length + 1, "Pre post len");
        for (uint256 i = 0; i < post.length; i++) {
            if (i == level || i == level + 1) {
                assertEq(post[i], 0, "Post level equal");
            } else if (i == level + 2) {
                assertEq(post[i], hashTogether(pre[i - 1], hashTogether(pre[i - 2], rand)), "Post level plus 1 equal");
            } else {
                assertEq(pre[i], post[i], "Pre post equal");
            }
        }
    }

    function testAppendCS3TooHigh() public {
        bytes32[] memory pre = getExpansion(12);
        bytes32 rand = random.hash();

        vm.expectRevert("Append above least significant bit");
        MerkleTreeLib.appendCompleteSubTree(pre, 3, rand);
    }

    function testAppendCS4GreaterLevel() public {
        bytes32[] memory pre = getExpansion(12);
        bytes32 rand = random.hash();

        vm.expectRevert("Level greater than highest level of current expansion");
        MerkleTreeLib.appendCompleteSubTree(pre, 4, rand);
    }

    function testAppendCsLevelTooHigh() public {
        bytes32[] memory pre = getExpansion(12);
        bytes32 rand = random.hash();

        vm.expectRevert("Level too high");
        MerkleTreeLib.appendCompleteSubTree(pre, MerkleTreeLib.MAX_LEVEL, rand);
    }

    function testAppendCsMeTooLargs() public {
        bytes32[] memory pre = new bytes32[](MerkleTreeLib.MAX_LEVEL + 1);

        bytes32 rand = random.hash();

        vm.expectRevert("Merkle expansion too large");
        MerkleTreeLib.appendCompleteSubTree(pre, 0, rand);
    }

    function testAppendCsEmptySubtree() public {
        bytes32[] memory pre = getExpansion(12);
        vm.expectRevert("Cannot append empty subtree");
        MerkleTreeLib.appendCompleteSubTree(pre, 1, 0);
    }

    function testAppendCsPostLevelHigh() public {
        bytes32[] memory pre = new bytes32[](MerkleTreeLib.MAX_LEVEL);
        pre[pre.length - 1] = random.hash();

        bytes32 rand2 = random.hash();
        // overflow
        vm.expectRevert("Append creates oversize tree");
        MerkleTreeLib.appendCompleteSubTree(pre, pre.length - 1, rand2);
    }

    function testAppendCsEmptyPre() public {
        bytes32[] memory pre = new bytes32[](0);
        uint256 level = 2;
        bytes32 rand = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendCompleteSubTree(pre, level, rand);

        for (uint256 i = 0; i < post.length; i++) {
            if (level == i) {
                assertEq(post[i], rand, "Post rand");
            } else {
                assertEq(post[i], 0, "Post empty");
            }
        }
    }

    function testAppendLeafEmpty() public {
        bytes32[] memory pre = new bytes32[](0);
        bytes32 leaf = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendLeaf(pre, leaf);
        assertEq(post.length, 1, "Post len");
        assertEq(post[0], keccak256(abi.encodePacked(leaf)), "Post slot");
    }

    function testAppendLeafOne() public {
        bytes32[] memory pre = new bytes32[](1);
        bytes32 slot0 = random.hash();
        pre[0] = slot0;
        bytes32 leaf = random.hash();
        bytes32[] memory post = MerkleTreeLib.appendLeaf(pre, leaf);
        assertEq(post.length, 2, "Post len");
        assertEq(post[0], 0, "Post slot 0");
        assertEq(post[1], hashTogether(slot0, keccak256(abi.encodePacked(leaf))), "Post slot 1");
    }

    function testMaximumAppendBetween() public {
        assertEq(MerkleTreeLib.maximumAppendBetween(0, 1), 0, "Max append 0,1");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 2), 1, "Max append 0,2");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 2), 0, "Max append 1,2");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 3), 1, "Max append 0,3");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 3), 0, "Max append 1,3");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 3), 0, "Max append 2,3");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 4), 2, "Max append 0,4");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 4), 0, "Max append 1,4");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 4), 1, "Max append 2,4");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 4), 0, "Max append 3,4");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 5), 2, "Max append 0,5");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 5), 0, "Max append 1,5");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 5), 1, "Max append 2,5");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 5), 0, "Max append 3,5");
        assertEq(MerkleTreeLib.maximumAppendBetween(4, 5), 0, "Max append 4,5");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 6), 2, "Max append 0,6");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 6), 0, "Max append 1,6");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 6), 1, "Max append 2,6");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 6), 0, "Max append 3,6");
        assertEq(MerkleTreeLib.maximumAppendBetween(4, 6), 1, "Max append 4,6");
        assertEq(MerkleTreeLib.maximumAppendBetween(5, 6), 0, "Max append 5,6");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 7), 2, "Max append 0,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 7), 0, "Max append 1,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 7), 1, "Max append 2,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 7), 0, "Max append 3,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(4, 7), 1, "Max append 4,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(5, 7), 0, "Max append 5,7");
        assertEq(MerkleTreeLib.maximumAppendBetween(6, 7), 0, "Max append 6,7");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 8), 3, "Max append 0,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 8), 0, "Max append 1,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 8), 1, "Max append 2,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 8), 0, "Max append 3,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(4, 8), 2, "Max append 4,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(5, 8), 0, "Max append 5,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(6, 8), 1, "Max append 6,8");
        assertEq(MerkleTreeLib.maximumAppendBetween(7, 8), 0, "Max append 7,8");

        assertEq(MerkleTreeLib.maximumAppendBetween(0, 9), 3, "Max append 0,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(1, 9), 0, "Max append 1,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(2, 9), 1, "Max append 2,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(3, 9), 0, "Max append 3,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(4, 9), 2, "Max append 4,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(5, 9), 0, "Max append 5,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(6, 9), 1, "Max append 6,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(7, 9), 0, "Max append 7,9");
        assertEq(MerkleTreeLib.maximumAppendBetween(8, 9), 0, "Max append 8,9");
    }

    function testMaxAppendBetweenStartTooLow() public {
        vm.expectRevert("Start not less than end");
        MerkleTreeLib.maximumAppendBetween(4, 4);
    }

    function testVerifyPrefixProofComp() public {
        proveVerify(1, 2);

        proveVerify(1, 3);
        proveVerify(2, 3);

        proveVerify(1, 4);
        proveVerify(2, 4);
        proveVerify(3, 4);

        proveVerify(1, 5);
        proveVerify(2, 5);
        proveVerify(3, 5);
        proveVerify(4, 5);

        proveVerify(1, 6);
        proveVerify(2, 6);
        proveVerify(3, 6);
        proveVerify(4, 6);
        proveVerify(5, 6);

        proveVerify(1, 7);
        proveVerify(2, 7);
        proveVerify(3, 7);
        proveVerify(4, 7);
        proveVerify(5, 7);
        proveVerify(6, 7);

        proveVerify(1, 8);
        proveVerify(2, 8);
        proveVerify(3, 8);
        proveVerify(4, 8);
        proveVerify(5, 8);
        proveVerify(6, 8);
        proveVerify(7, 8);

        proveVerify(1, 9);
        proveVerify(2, 9);
        proveVerify(3, 9);
        proveVerify(4, 9);
        proveVerify(5, 9);
        proveVerify(6, 9);
        proveVerify(7, 9);
        proveVerify(8, 9);

        proveVerify(1, 10);
        proveVerify(2, 10);
        proveVerify(3, 10);
        proveVerify(4, 10);
        proveVerify(5, 10);
        proveVerify(6, 10);
        proveVerify(7, 10);
        proveVerify(8, 10);
        proveVerify(9, 10);
    }

    function testVerifyPrefixProofManual() public {
        bytes32[] memory pre = getExpansion(5); // 101
        bytes32[] memory newLeaves = random.hashes(4); // 1001
        bytes32[] memory rehashedLeaves = ProofUtils.rehashed(newLeaves);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }

        // manually construct a proof from 5 to 9
        bytes32[] memory proof = new bytes32[](3);
        proof[0] = rehashedLeaves[0];
        proof[1] = hashTogether(rehashedLeaves[1], rehashedLeaves[2]);
        proof[2] = rehashedLeaves[3];
        MerkleTreeLib.verifyPrefixProof(MerkleTreeLib.root(pre), 5, MerkleTreeLib.root(post), 9, pre, proof);
    }

    function testVerifyPrefixProofPreZero() public {
        uint256 preSize = 5;
        uint256 newLeavesCount = 4;

        bytes32[] memory pre = getExpansion(preSize);
        bytes32[] memory newLeaves = random.hashes(newLeavesCount);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }
        bytes32[] memory proof = ProofUtils.generatePrefixProof(preSize, newLeaves);

        vm.expectRevert("Pre-size cannot be 0");
        MerkleTreeLib.verifyPrefixProof(
            MerkleTreeLib.root(pre), 0, MerkleTreeLib.root(post), preSize + newLeavesCount, pre, proof
        );
    }

    function testVerifyPrefixProofInvalidPreRoot() public {
        uint256 preSize = 5;
        uint256 newLeavesCount = 4;

        bytes32[] memory pre = getExpansion(preSize);
        bytes32[] memory newLeaves = random.hashes(newLeavesCount);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }
        bytes32[] memory proof = ProofUtils.generatePrefixProof(preSize, newLeaves);

        bytes32 randomHash = random.hash();
        vm.expectRevert("Pre expansion root mismatch");
        MerkleTreeLib.verifyPrefixProof(
            randomHash, preSize, MerkleTreeLib.root(post), preSize + newLeavesCount, pre, proof
        );
    }

    function testVerifyPrefixProofInvalidPreSize() public {
        uint256 preSize = 5;
        uint256 newLeavesCount = 4;

        bytes32[] memory pre = getExpansion(preSize);
        bytes32[] memory newLeaves = random.hashes(newLeavesCount);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }

        bytes32[] memory proof = ProofUtils.generatePrefixProof(preSize, newLeaves);

        vm.expectRevert("Pre size not less than post size");
        MerkleTreeLib.verifyPrefixProof(MerkleTreeLib.root(pre), preSize, MerkleTreeLib.root(post), preSize, pre, proof);
    }

    function testVerifyPrefixProofInvalidProofSize() public {
        uint256 preSize = 5;
        uint256 newLeavesCount = 4;

        bytes32[] memory pre = getExpansion(preSize);
        bytes32[] memory newLeaves = random.hashes(newLeavesCount);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }

        bytes32[] memory proof = ProofUtils.generatePrefixProof(preSize, newLeaves);
        proof = ArrayUtilsLib.append(proof, random.hash());

        vm.expectRevert("Incomplete proof usage");
        MerkleTreeLib.verifyPrefixProof(
            MerkleTreeLib.root(pre), preSize, MerkleTreeLib.root(post), preSize + newLeavesCount, pre, proof
        );
    }

    function testVerifyPrefixProofInvalidPreExpansionSize() public {
        uint256 preSize = 5;
        uint256 newLeavesCount = 4;

        bytes32[] memory pre = getExpansion(preSize);
        bytes32[] memory newLeaves = random.hashes(newLeavesCount);
        bytes32[] memory post = ArrayUtilsLib.slice(pre, 0, pre.length);
        for (uint256 i = 0; i < newLeaves.length; i++) {
            post = MerkleTreeLib.appendLeaf(post, newLeaves[i]);
        }

        bytes32[] memory proof = ProofUtils.generatePrefixProof(preSize, newLeaves);

        vm.expectRevert("Pre size does not match expansion");
        MerkleTreeLib.verifyPrefixProof(
            MerkleTreeLib.root(pre), preSize - 1, MerkleTreeLib.root(post), preSize + newLeavesCount, pre, proof
        );
    }

    function testVerifyInclusionProofManual() public {
        bytes32[] memory leaves = random.hashes(11); // 1011
        bytes32[] memory re = ProofUtils.rehashed(leaves);
        uint256 index = 4; // 100
        bytes32[] memory me = ProofUtils.expansionFromLeaves(leaves, 0, 11);

        // need 5 + (6,7) + ((0,1),(2,3)) ((8,9, 10,null), null)

        bytes32[] memory proof = new bytes32[](4);
        proof[0] = re[5];
        proof[1] = hashTogether(re[6], re[7]);
        proof[2] = hashTogether(hashTogether(re[0], re[1]), hashTogether(re[2], re[3]));
        proof[3] = hashTogether(hashTogether(hashTogether(re[8], re[9]), hashTogether(re[10], 0)), 0);

        bool r = MerkleTreeLib.verifyInclusionProof(MerkleTreeLib.root(me), leaves[index], index, proof);
        assertTrue(r, "Invalid root");
    }

    function verifyInclusion(uint256 index, uint256 treeSize) internal {
        bytes32[] memory leaves = random.hashes(treeSize);
        bytes32[] memory re = ProofUtils.rehashed(leaves);
        bytes32[] memory me = ProofUtils.expansionFromLeaves(leaves, 0, leaves.length);
        bytes32[] memory proof = ProofUtils.generateInclusionProof(re, index);

        bool v2 = MerkleTreeLib.verifyInclusionProof(MerkleTreeLib.root(me), leaves[index], index, proof);
        assertTrue(v2, "Invalid root");
    }

    function testProveInclusion() public {
        uint256 size = 16;
        for (uint256 i = 0; i < size; i++) {
            for (uint256 j = 0; j < i; j++) {
                verifyInclusion(j, i);
            }
        }
    }

    function testEmptyTreeSize() public {
        bytes32[] memory me = new bytes32[](0);
        assertEq(MerkleTreeLib.treeSize(me), 0, "Invalid zero tree size");
    }

    function testTreeSize() public {
        for (uint256 h = 1; h <= 256; h++) {
            bytes32[] memory me = getExpansion(h);
            assertEq(MerkleTreeLib.treeSize(me), h, "Invalid tree size");
        }
    }
}
