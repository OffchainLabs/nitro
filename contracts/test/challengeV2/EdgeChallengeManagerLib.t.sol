    // SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
// import "../../src/challengeV2/DataEntities.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
// import "../src/osp/IOneStepProofEntry.sol";
import "./Utils.sol";
// import "./StateTools.sol";

contract EdgeChallengeManagerLibTest is Test {
    using ChallengeEdgeLib for ChallengeEdge;
    using EdgeChallengeManagerLib for EdgeStore;

    EdgeStore store;
    Random rand = new Random();

    function twoNonRivals() internal returns (ChallengeEdge memory, ChallengeEdge memory) {
        bytes32 originId = rand.hash();

        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, EdgeType.Block);

        return (edge1, edge2);
    }

    function twoRivals() internal returns (ChallengeEdge memory, ChallengeEdge memory) {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();

        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);

        return (edge1, edge2);
    }

    function testAdd() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        store.add(edge);

        ChallengeEdge storage se = store.edges[edge.id()];
        assertTrue(se.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == EdgeChallengeManagerLib.UNRIVALED, "NO_RIVAL first rival");
    }

    function testGet() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        store.add(edge);

        ChallengeEdge storage se = store.get(edge.id());
        assertEq(edge.id(), se.id(), "Id's are equal");
    }

    function testGetNotExist() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        bytes32 edgeId = edge.id();

        vm.expectRevert("Edge does not exist");
        store.get(edgeId);
    }

    function testAddRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);
        store.add(edge2);

        ChallengeEdge storage se = store.edges[edge1.id()];
        ChallengeEdge storage se2 = store.edges[edge2.id()];
        assertTrue(se2.exists(), "Edge exists");
        assertTrue(store.firstRivals[se2.mutualId()] == edge2.id(), "First rival1");
        assertTrue(store.firstRivals[se.mutualId()] == edge2.id(), "First rival2");
    }

    function testAddMoreRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, EdgeType.Block
        );

        store.add(edge1);
        store.add(edge2);
        store.add(edge3);

        ChallengeEdge storage se = store.edges[edge1.id()];
        ChallengeEdge storage se2 = store.edges[edge2.id()];
        ChallengeEdge storage se3 = store.edges[edge3.id()];
        assertTrue(se3.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == edge2.id(), "First rival1");
        assertTrue(store.firstRivals[se2.mutualId()] == edge2.id(), "First rival2");
        assertTrue(store.firstRivals[se3.mutualId()] == edge2.id(), "First rival3");
    }

    function testAddNonRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoNonRivals();

        store.add(edge1);
        store.add(edge2);

        ChallengeEdge storage se = store.edges[edge1.id()];
        ChallengeEdge storage se2 = store.edges[edge2.id()];
        assertTrue(se2.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == EdgeChallengeManagerLib.UNRIVALED, "First rival1");
        assertTrue(store.firstRivals[se2.mutualId()] == EdgeChallengeManagerLib.UNRIVALED, "First rival2");
    }

    function testCannotAddSameEdgeTwice() public {
        ChallengeEdge memory edge =
            ChallengeEdgeLib.newChildEdge(rand.hash(), rand.hash(), 0, rand.hash(), 10, EdgeType.Block);

        store.add(edge);
        vm.expectRevert("Edge already exists");
        store.add(edge);
    }

    function testHasRival() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);
        store.add(edge2);

        assertTrue(store.hasRival(edge1.id()), "Edge1 rival");
        assertTrue(store.hasRival(edge2.id()), "Edge2 rival");
    }

    function testHasRivalMore() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, EdgeType.Block
        );

        store.add(edge1);
        store.add(edge2);
        store.add(edge3);

        assertTrue(store.hasRival(edge1.id()), "Edge1 rival");
        assertTrue(store.hasRival(edge2.id()), "Edge2 rival");
        assertTrue(store.hasRival(edge3.id()), "Edge2 rival");
    }

    function testNoRival() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoNonRivals();

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasRival(edge1.id()), "Edge1 rival");
        assertFalse(store.hasRival(edge2.id()), "Edge2 rival");
    }

    function testRivalNotExist() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);

        bytes32 edge2Id = edge2.id();
        vm.expectRevert("Edge does not exist");
        store.hasRival(edge2Id);
    }

    function testSingleStepRivalNotRival() public {
        bytes32 originId = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 9, rand.hash(), 10, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 9, rand.hash(), 10, EdgeType.Block);

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasLengthOneRival(edge1.id()));
        assertFalse(store.hasLengthOneRival(edge2.id()));
    }

    function testSingleStepRivalNotHeight() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 8, rand.hash(), 10, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 8, rand.hash(), 10, EdgeType.Block);

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasLengthOneRival(edge1.id()));
        assertFalse(store.hasLengthOneRival(edge2.id()));
    }

    function testSingleStepRival() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);

        store.add(edge1);
        store.add(edge2);

        assertTrue(store.hasLengthOneRival(edge1.id()));
        assertTrue(store.hasLengthOneRival(edge2.id()));
    }

    function testSingleStepRivalNotExist() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);

        store.add(edge1);

        bytes32 edge2Id = edge2.id();
        vm.expectRevert("Edge does not exist");
        store.hasLengthOneRival(edge2Id);
    }

    function testTimeUnrivaled() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        store.add(edge1);
        vm.warp(block.timestamp + 3);

        assertEq(store.timeUnrivaled(edge1.id()), 3, "Time unrivaled");
    }

    function testTimeUnrivaledNotExist() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        vm.warp(block.timestamp + 3);

        bytes32 id1 = edge1.id();
        vm.expectRevert("Edge does not exist");
        assertEq(store.timeUnrivaled(id1), 3, "Time unrivaled");
    }

    function testTimeUnrivaledAfterRival() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);

        store.add(edge1);
        vm.warp(block.timestamp + 4);

        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);

        store.add(edge2);
        vm.warp(block.timestamp + 5);

        ChallengeEdge memory edge3 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);
        store.add(edge3);

        vm.warp(block.timestamp + 6);

        ChallengeEdge memory edge4 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, EdgeType.Block);
        store.add(edge4);

        vm.warp(block.timestamp + 7);

        assertEq(store.timeUnrivaled(edge1.id()), 4, "Time unrivaled 1");
        assertEq(store.timeUnrivaled(edge2.id()), 0, "Time unrivaled 2");
        assertEq(store.timeUnrivaled(edge3.id()), 0, "Time unrivaled 3");
        assertEq(store.timeUnrivaled(edge4.id()), 7, "Time unrivaled 4");
    }

    function testMandatoryBisectHeightSizeOne() public {
        vm.expectRevert("Height difference not two or more");
        EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 2);
    }

    function testMandatoryBisectHeightSizes() public {
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 3), 2);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 4), 2);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 4), 2);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 4), 3);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 5), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 5), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 5), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 5), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 6), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 6), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 6), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 6), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 6), 5);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 7), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 7), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 7), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 7), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 7), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 7), 6);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(5, 7), 6);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 8), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 8), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 8), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 8), 4);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 8), 6);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(5, 8), 6);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(6, 8), 7);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(5, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(6, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(7, 9), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(5, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(6, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(7, 10), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(8, 10), 9);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(0, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(1, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(2, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(3, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(4, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(5, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(6, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(7, 11), 8);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(8, 11), 10);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(9, 11), 10);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(7, 73), 64);
        assertEq(EdgeChallengeManagerLib.mandatoryBisectionHeight(765273563, 10898783768364), 8796093022208);
    }

    function getExpansion(uint256 leafCount) internal returns (bytes32[] memory) {
        bytes32[] memory hashes = rand.hashes(leafCount);
        bytes32[] memory expansion = ProofUtils.expansionFromLeaves(hashes, 0, leafCount);
        return expansion;
    }

    function appendRandomStates(bytes32[] memory currentStates, uint256 numStates)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory newStates = rand.hashes(numStates);
        bytes32[] memory full = ArrayUtilsLib.concat(currentStates, newStates);
        bytes32[] memory exp = ProofUtils.expansionFromLeaves(full, 0, full.length);

        return (full, exp);
    }

    function appendRandomStatesBetween(bytes32[] memory currentStates, bytes32 endState, uint256 numStates)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        (bytes32[] memory states, bytes32[] memory exp) = appendRandomStates(currentStates, numStates - 1);
        bytes32[] memory fullStates = ArrayUtilsLib.append(states, endState);
        bytes32[] memory fullExp = MerkleTreeLib.appendLeaf(exp, endState);
        return (fullStates, fullExp);
    }

    struct BisectionDefArgs {
        uint256 start;
        uint256 agreePoint;
        uint256 end;
    }

    function rivalStates(uint256 start, uint256 agreePoint, uint256 end)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory preStates = rand.hashes(start + 1);
        (bytes32[] memory agreeStates,) = appendRandomStates(preStates, agreePoint - start);
        (bytes32[] memory states1,) = appendRandomStates(agreeStates, end - agreePoint);

        (bytes32[] memory states2,) = appendRandomStates(agreeStates, end - agreePoint);

        return (states1, states2);
    }

    function edgeFromStates(bytes32 originId, uint256 start, uint256 end, bytes32[] memory states)
        internal
        returns (ChallengeEdge memory)
    {
        bytes32 startRoot = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, start + 1));
        bytes32 endRoot = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, end + 1));

        return ChallengeEdgeLib.newChildEdge(originId, startRoot, start, endRoot, end, EdgeType.Block);
    }

    function proofGen(uint256 start, bytes32[] memory states) internal pure returns (bytes32[] memory) {
        return ProofUtils.generatePrefixProof(start, ArrayUtilsLib.slice(states, start, states.length));
    }

    function twoRivalsFromLeaves(uint256 start, uint256 agreePoint, uint256 end)
        internal
        returns (ChallengeEdge memory, ChallengeEdge memory, bytes32[] memory, bytes32[] memory)
    {
        (bytes32[] memory states1, bytes32[] memory states2) = rivalStates(start, agreePoint, end);

        ChallengeEdge memory edge1 = edgeFromStates(rand.hash(), start, end, states1);
        ChallengeEdge memory edge2 = edgeFromStates(edge1.originId, start, end, states2);

        return (edge1, edge2, states1, states2);
    }

    function testBisectEdge() public {
        uint256 start = 3;
        uint256 agree = 5; // agree point is below the bisection point of 8
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1, bytes32[] memory states2) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge1.id(),
            bisectionRoot1,
            abi.encode(
                ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge1.id()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId,
                    edge1.startHistoryRoot,
                    edge1.startHeight,
                    bisectionRoot1,
                    bisectionPoint,
                    edge1.eType
                )
            ).id(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.id()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.eType
                )
            ).id(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge1.id()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge1.id()).upperChildId), "Upper child rival");

        bytes32 bisectionRoot2 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge2.id(),
            bisectionRoot2,
            abi.encode(
                ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states2, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge2.id()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId,
                    edge2.startHistoryRoot,
                    edge2.startHeight,
                    bisectionRoot2,
                    bisectionPoint,
                    edge2.eType
                )
            ).id(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.id()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.eType
                )
            ).id(),
            "Lower child id"
        );

        assertTrue(store.hasRival(store.get(edge2.id()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge2.id()).upperChildId), "Upper child rival");
    }

    function bisectMergeEdge(uint256 agree) internal {
        uint256 start = 3;
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1, bytes32[] memory states2) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge1.id(),
            bisectionRoot1,
            abi.encode(
                ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge1.id()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId,
                    edge1.startHistoryRoot,
                    edge1.startHeight,
                    bisectionRoot1,
                    bisectionPoint,
                    edge1.eType
                )
            ).id(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.id()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.eType
                )
            ).id(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge1.id()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge1.id()).upperChildId), "Upper child rival");

        bytes32 bisectionRoot2 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge2.id(),
            bisectionRoot2,
            abi.encode(
                ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states2, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge2.id()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId,
                    edge2.startHistoryRoot,
                    edge2.startHeight,
                    bisectionRoot2,
                    bisectionPoint,
                    edge2.eType
                )
            ).id(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.id()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.eType
                )
            ).id(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge2.id()).lowerChildId), "Lower child rival");
        assertTrue(store.hasRival(store.get(edge2.id()).upperChildId), "Upper child rival");

        assertEq(
            store.hasRival(store.get(edge1.id()).lowerChildId),
            store.hasRival(store.get(edge2.id()).lowerChildId),
            "Lower children equal"
        );
    }

    function testBisectMergeEdge() public {
        bisectMergeEdge(9);
    }

    function testBisectMergeEdgeEqualBisection() public {
        bisectMergeEdge(8);
    }

    function testBisectNoRival() public {
        uint256 start = 3;
        uint256 agree = 5;
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1,, bytes32[] memory states1,) = twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        bytes32 edgeId = edge1.id();
        bytes memory proof = abi.encode(
            ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
            ProofUtils.generatePrefixProof(
                bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
            )
        );
        vm.expectRevert("Cannot bisect an unrivaled edge");
        store.bisectEdge(edgeId, bisectionRoot1, proof);
    }

    function testBisectTwice() public {
        uint256 start = 3;
        uint256 agree = 5;
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1,) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        bytes32 edgeId = edge1.id();
        bytes memory proof = abi.encode(
            ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
            ProofUtils.generatePrefixProof(
                bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
            )
        );
        store.bisectEdge(edgeId, bisectionRoot1, proof);

        vm.expectRevert("Edge already has children");
        store.bisectEdge(edgeId, bisectionRoot1, proof);
    }

    function testBisectInvalidProof() public {
        uint256 start = 3;
        uint256 agree = 5;
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1,) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        bytes32 edgeId = edge1.id();
        bytes memory proof = abi.encode(
            ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint),
            ProofUtils.generatePrefixProof(
                bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
            )
        );
        vm.expectRevert("Pre expansion root mismatch");
        store.bisectEdge(edgeId, bisectionRoot1, proof);
    }

    function testBisectEdgeConfirmed() public {
        uint256 start = 3;
        uint256 agree = 5; // agree point is below the bisection point of 8
        uint256 end = 11;
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);

        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1,) =
            twoRivalsFromLeaves(start, agree, end);

        edge1.status = EdgeStatus.Confirmed;

        store.add(edge1);
        store.add(edge2);

        bytes32 bisectionRoot1 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1));
        bytes memory proof = abi.encode(
                ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
                )
            );
        bytes32 edgeId = edge1.id();
        vm.expectRevert("Edge not pending");
        store.bisectEdge(
            edgeId,
            bisectionRoot1,
            proof
        );
    }

    function testBisectEdgeLengthOne() public {
        uint256 start = 0;
        uint256 end = 1;

        bytes32[] memory states1 = new bytes32[](2);
        states1[0] = rand.hash();
        states1[1] = rand.hash();
        bytes32[] memory states2 = new bytes32[](2);
        states2[0] = states1[0];
        states2[1] = rand.hash();
        ChallengeEdge memory edge1 = edgeFromStates(rand.hash(), start, end, states1);
        ChallengeEdge memory edge2 = edgeFromStates(edge1.originId, start, end, states2);

        store.add(edge1);
        store.add(edge2);

        bytes32 edgeId = edge1.id();
        vm.expectRevert("Height difference not two or more");
        store.bisectEdge(
            edgeId,
            edge1.endHistoryRoot,
            ""
        );
    }
}
