    // SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
import "./Utils.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    function proveOneStep(ExecutionContext calldata, uint256, bytes32, bytes calldata proof)
        external
        view
        returns (bytes32 afterHash)
    {
        return bytes32(proof);
    }
}

contract EdgeChallengeManagerLibAccess {
    using EdgeChallengeManagerLib for EdgeStore;

    EdgeStore private store;

    function add(ChallengeEdge memory edge) public {
        store.add(edge);
    }

    function getEdge(bytes32 id) public view returns (ChallengeEdge memory) {
        return store.get(id);
    }

    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        IOneStepProofEntry oneStepProofEntry,
        OneStepData memory oneStepData,
        bytes32[] memory beforeHistoryInclusionProof,
        bytes32[] memory afterHistoryInclusionProof
    ) public {
        store.confirmEdgeByOneStepProof(
            edgeId, oneStepProofEntry, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof
        );
    }
}

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

        ChallengeEdge storage se = store.edges[edge.idMem()];
        assertTrue(se.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == EdgeChallengeManagerLib.UNRIVALED, "NO_RIVAL first rival");
    }

    function testGet() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        store.add(edge);

        ChallengeEdge storage se = store.get(edge.idMem());
        assertEq(edge.idMem(), se.idMem(), "Id's are equal");
    }

    function testGetNotExist() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        bytes32 edgeId = edge.idMem();

        vm.expectRevert("Edge does not exist");
        store.get(edgeId);
    }

    function testAddRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);
        store.add(edge2);

        ChallengeEdge storage se = store.edges[edge1.idMem()];
        ChallengeEdge storage se2 = store.edges[edge2.idMem()];
        assertTrue(se2.exists(), "Edge exists");
        assertTrue(store.firstRivals[se2.mutualId()] == edge2.idMem(), "First rival1");
        assertTrue(store.firstRivals[se.mutualId()] == edge2.idMem(), "First rival2");
    }

    function testAddMoreRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, EdgeType.Block
        );

        store.add(edge1);
        store.add(edge2);
        store.add(edge3);

        ChallengeEdge storage se = store.edges[edge1.idMem()];
        ChallengeEdge storage se2 = store.edges[edge2.idMem()];
        ChallengeEdge storage se3 = store.edges[edge3.idMem()];
        assertTrue(se3.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == edge2.idMem(), "First rival1");
        assertTrue(store.firstRivals[se2.mutualId()] == edge2.idMem(), "First rival2");
        assertTrue(store.firstRivals[se3.mutualId()] == edge2.idMem(), "First rival3");
    }

    function testAddNonRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoNonRivals();

        store.add(edge1);
        store.add(edge2);

        ChallengeEdge storage se = store.edges[edge1.idMem()];
        ChallengeEdge storage se2 = store.edges[edge2.idMem()];
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

        assertTrue(store.hasRival(edge1.idMem()), "Edge1 rival");
        assertTrue(store.hasRival(edge2.idMem()), "Edge2 rival");
    }

    function testHasRivalMore() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, EdgeType.Block
        );

        store.add(edge1);
        store.add(edge2);
        store.add(edge3);

        assertTrue(store.hasRival(edge1.idMem()), "Edge1 rival");
        assertTrue(store.hasRival(edge2.idMem()), "Edge2 rival");
        assertTrue(store.hasRival(edge3.idMem()), "Edge2 rival");
    }

    function testNoRival() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoNonRivals();

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasRival(edge1.idMem()), "Edge1 rival");
        assertFalse(store.hasRival(edge2.idMem()), "Edge2 rival");
    }

    function testRivalNotExist() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);

        bytes32 edge2Id = edge2.idMem();
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

        assertFalse(store.hasLengthOneRival(edge1.idMem()));
        assertFalse(store.hasLengthOneRival(edge2.idMem()));
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

        assertFalse(store.hasLengthOneRival(edge1.idMem()));
        assertFalse(store.hasLengthOneRival(edge2.idMem()));
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

        assertTrue(store.hasLengthOneRival(edge1.idMem()));
        assertTrue(store.hasLengthOneRival(edge2.idMem()));
    }

    function testSingleStepRivalNotExist() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);
        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, EdgeType.Block);

        store.add(edge1);

        bytes32 edge2Id = edge2.idMem();
        vm.expectRevert("Edge does not exist");
        store.hasLengthOneRival(edge2Id);
    }

    function testTimeUnrivaled() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        store.add(edge1);
        vm.warp(block.timestamp + 3);

        assertEq(store.timeUnrivaled(edge1.idMem()), 3, "Time unrivaled");
    }

    function testTimeUnrivaledNotExist() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        vm.warp(block.timestamp + 3);

        bytes32 id1 = edge1.idMem();
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

        assertEq(store.timeUnrivaled(edge1.idMem()), 4, "Time unrivaled 1");
        assertEq(store.timeUnrivaled(edge2.idMem()), 0, "Time unrivaled 2");
        assertEq(store.timeUnrivaled(edge3.idMem()), 0, "Time unrivaled 3");
        assertEq(store.timeUnrivaled(edge4.idMem()), 7, "Time unrivaled 4");
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
        view
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
            edge1.idMem(),
            bisectionRoot1,
            abi.encode(
                ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge1.idMem()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId,
                    edge1.startHistoryRoot,
                    edge1.startHeight,
                    bisectionRoot1,
                    bisectionPoint,
                    edge1.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge1.idMem()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge1.idMem()).upperChildId), "Upper child rival");

        bytes32 bisectionRoot2 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge2.idMem(),
            bisectionRoot2,
            abi.encode(
                ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states2, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge2.idMem()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId,
                    edge2.startHistoryRoot,
                    edge2.startHeight,
                    bisectionRoot2,
                    bisectionPoint,
                    edge2.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertTrue(store.hasRival(store.get(edge2.idMem()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge2.idMem()).upperChildId), "Upper child rival");
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
            edge1.idMem(),
            bisectionRoot1,
            abi.encode(
                ProofUtils.expansionFromLeaves(states1, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states1, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge1.idMem()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId,
                    edge1.startHistoryRoot,
                    edge1.startHeight,
                    bisectionRoot1,
                    bisectionPoint,
                    edge1.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge1.idMem()).lowerChildId), "Lower child rival");
        assertFalse(store.hasRival(store.get(edge1.idMem()).upperChildId), "Upper child rival");

        bytes32 bisectionRoot2 = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1));
        store.bisectEdge(
            edge2.idMem(),
            bisectionRoot2,
            abi.encode(
                ProofUtils.expansionFromLeaves(states2, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states2, bisectionPoint + 1, states1.length)
                )
            )
        );

        assertEq(
            store.get(edge2.idMem()).lowerChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId,
                    edge2.startHistoryRoot,
                    edge2.startHeight,
                    bisectionRoot2,
                    bisectionPoint,
                    edge2.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.eType
                )
            ).idMem(),
            "Lower child id"
        );

        assertFalse(store.hasRival(store.get(edge2.idMem()).lowerChildId), "Lower child rival");
        assertTrue(store.hasRival(store.get(edge2.idMem()).upperChildId), "Upper child rival");

        assertEq(
            store.hasRival(store.get(edge1.idMem()).lowerChildId),
            store.hasRival(store.get(edge2.idMem()).lowerChildId),
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
        bytes32 edgeId = edge1.idMem();
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
        bytes32 edgeId = edge1.idMem();
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
        bytes32 edgeId = edge1.idMem();
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
        bytes32 edgeId = edge1.idMem();
        vm.expectRevert("Edge not pending");
        store.bisectEdge(edgeId, bisectionRoot1, proof);
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

        bytes32 edgeId = edge1.idMem();
        vm.expectRevert("Height difference not two or more");
        store.bisectEdge(edgeId, edge1.endHistoryRoot, "");
    }

    function bisectArgs(bytes32[] memory states, uint256 start, uint256 end)
        internal
        pure
        returns (uint256, bytes32, bytes memory)
    {
        uint256 bisectionPoint = EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);
        bytes32 bisectionRoot = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, bisectionPoint + 1));
        bytes memory proof = abi.encode(
            ProofUtils.expansionFromLeaves(states, 0, bisectionPoint + 1),
            ProofUtils.generatePrefixProof(bisectionPoint + 1, ArrayUtilsLib.slice(states, bisectionPoint + 1, end + 1))
        );

        return (bisectionPoint, bisectionRoot, proof);
    }

    function addParentAndChildren(uint256 start, uint256 agree, uint256 end)
        internal
        returns (bytes32, bytes32, bytes32)
    {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1,) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        (, bytes32 bisectionRoot, bytes memory bisectionProof) = bisectArgs(states1, start, end);

        (bytes32 lowerChildId, bytes32 upperChildId) = store.bisectEdge(edge1.idMem(), bisectionRoot, bisectionProof);

        return (edge1.idMem(), lowerChildId, upperChildId);
    }

    function testConfirmEdgeByChildren() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        store.confirmEdgeByChildren(parentEdgeId);
        assertTrue(store.get(parentEdgeId).status == EdgeStatus.Confirmed);
    }

    function testConfirmEdgeByChildrenAlreadyConfirmed() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        store.confirmEdgeByChildren(parentEdgeId);
        vm.expectRevert("Edge not pending");
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenNotExist() public {
        bytes32 randId = rand.hash();

        vm.expectRevert("Edge does not exist");
        store.confirmEdgeByChildren(randId);
    }

    function testConfirmEdgeByChildrenLowerChildNotExist() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        delete store.edges[lowerChildId];
        vm.expectRevert("Lower child does not exist");
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenUpperChildNotExist() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        delete store.edges[upperChildId];
        vm.expectRevert("Upper child does not exist");
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenLowerChildPending() public {
        (bytes32 parentEdgeId,, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[upperChildId].setConfirmed();

        vm.expectRevert("Lower child not confirmed");
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenUpperChildPending() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId,) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();

        vm.expectRevert("Upper child not confirmed");
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testNextEdgeType() public {
        assertTrue(EdgeChallengeManagerLib.nextEdgeType(EdgeType.Block) == EdgeType.BigStep);
        assertTrue(EdgeChallengeManagerLib.nextEdgeType(EdgeType.BigStep) == EdgeType.SmallStep);
        vm.expectRevert("No next type after SmallStep");
        EdgeChallengeManagerLib.nextEdgeType(EdgeType.SmallStep);
    }

    function testCheckClaimIdLink() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        store.checkClaimIdLink(upperChildId, ce.idMem());
    }

    function testCheckClaimIdLinkOrigin() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            rand.hash(), rand.hash(), 3, rand.hash(), 4, upperChildId, rand.addr(), EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        vm.expectRevert("Origin id-mutual id mismatch");
        store.checkClaimIdLink(upperChildId, eid);
    }

    function testCheckClaimIdLinkEdgeType() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.Block
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        vm.expectRevert("Edge type does not match claiming edge type");
        store.checkClaimIdLink(upperChildId, eid);
    }

    function testConfirmClaim() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        store.confirmEdgeByClaim(upperChildId, eid);

        assertTrue(store.edges[upperChildId].status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function testConfirmClaimWrongClaimId() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            rand.hash(),
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        vm.expectRevert("Claim does not match edge");
        store.confirmEdgeByClaim(upperChildId, eid);
    }

    function testConfirmClaimNotExist() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        bytes32 eid = ce.idMem();
        bytes32 randId = rand.hash();
        vm.expectRevert("Edge does not exist");
        store.confirmEdgeByClaim(randId, eid);
    }

    function testConfirmClaimNotPending() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        store.edges[upperChildId].setConfirmed();
        vm.expectRevert("Edge not pending");
        store.confirmEdgeByClaim(upperChildId, eid);
    }

    function testConfirmClaimClaimerNotExist() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        bytes32 eid = ce.idMem();
        vm.expectRevert("Claiming edge does not exist");
        store.confirmEdgeByClaim(upperChildId, eid);
    }

    function testConfirmClaimClaimerNotConfirmed() public {
        (,, bytes32 upperChildId) = addParentAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(upperChildId).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            upperChildId,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        vm.expectRevert("Claiming edge not confirmed");
        store.confirmEdgeByClaim(upperChildId, eid);
    }

    function bisect(ChallengeEdge memory edge, bytes32[] memory states, uint256 start, uint256 end)
        internal
        returns (bytes32, bytes32)
    {
        (, bytes32 bisectionRoot, bytes memory bisectionProof) = bisectArgs(states, start, end);
        return store.bisectEdge(edge.idMem(), bisectionRoot, bisectionProof);
    }

    struct BArgs {
        bytes32 edge1Id;
        bytes32 lowerChildId1;
        bytes32 upperChildId1;
        bytes32 lowerChildId2;
        bytes32 upperChildId2;
        bytes32[] states1;
        bytes32[] states2;
    }

    function addParentsAndChildren(uint256 start, uint256 agree, uint256 end) internal returns (BArgs memory) {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1, bytes32[] memory states2) =
            twoRivalsFromLeaves(start, agree, end);

        store.add(edge1);
        store.add(edge2);

        (bytes32 lowerChildId1, bytes32 upperChildId1) = bisect(edge1, states1, start, end);
        (bytes32 lowerChildId2, bytes32 upperChildId2) = bisect(edge2, states2, start, end);

        return BArgs(edge1.idMem(), lowerChildId1, upperChildId1, lowerChildId2, upperChildId2, states1, states2);
    }

    function claimWithMixedAncestors(
        uint256 challengePeriodSec,
        uint256 timeAfterParent1,
        uint256 timeAfterParent2,
        uint256 timeAfterZeroLayer,
        string memory revertArg
    ) internal {
        BArgs memory pc = addParentsAndChildren(2, 5, 8);

        (, bytes32 bisectionRoot, bytes memory bisectionProof) = bisectArgs(pc.states1, 4, 8);
        (bytes32 lowerChildId148,) = store.bisectEdge(pc.upperChildId1, bisectionRoot, bisectionProof);

        vm.warp(block.timestamp + timeAfterParent1);
        bytes32 lowerChildId146;
        {
            (, bytes32 bisectionRoot2, bytes memory bisectionProof2) = bisectArgs(pc.states2, 4, 8);
            store.bisectEdge(pc.upperChildId2, bisectionRoot2, bisectionProof2);

            (, bytes32 bisectionRoot3, bytes memory bisectionProof3) = bisectArgs(pc.states1, 4, 6);
            (bytes32 lowerChildId146X,) = store.bisectEdge(lowerChildId148, bisectionRoot3, bisectionProof3);
            lowerChildId146 = lowerChildId146X;
        }

        vm.warp(block.timestamp + timeAfterParent2);

        ChallengeEdge memory bigStepZero = ChallengeEdgeLib.newLayerZeroEdge(
            store.edges[lowerChildId146].mutualId(),
            store.edges[lowerChildId146].startHistoryRoot,
            store.edges[lowerChildId146].startHeight,
            rand.hash(),
            100,
            lowerChildId146,
            rand.addr(),
            EdgeType.BigStep
        );
        if (timeAfterParent1 != 139) {
            store.add(bigStepZero);
        }
        bytes32 bsId = bigStepZero.idMem();

        vm.warp(block.timestamp + timeAfterZeroLayer);

        bytes32[] memory ancestorIds = new bytes32[](3);
        ancestorIds[0] = lowerChildId146;
        ancestorIds[1] = lowerChildId148;
        ancestorIds[2] = pc.upperChildId1;

        if (timeAfterParent1 == 137) {
            ChallengeEdge memory childE1 =
                ChallengeEdgeLib.newChildEdge(rand.hash(), rand.hash(), 10, rand.hash(), 100, EdgeType.Block);

            store.add(childE1);

            ancestorIds[1] = childE1.idMem();
        }

        if (timeAfterParent1 == 138) {
            store.edges[bsId].claimId = rand.hash();
        }

        if (timeAfterParent1 == 140) {
            store.edges[bsId].setConfirmed();
        }

        if (bytes(revertArg).length != 0) {
            vm.expectRevert(bytes(revertArg));
        }
        store.confirmEdgeByTime(bsId, ancestorIds, challengePeriodSec);

        assertTrue(store.edges[bsId].status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function testConfirmByTimeGrandParent() public {
        claimWithMixedAncestors(10, 11, 0, 0, "");
    }

    function testConfirmByTimeParent() public {
        claimWithMixedAncestors(10, 0, 11, 0, "");
    }

    function testConfirmByTimeSelf() public {
        claimWithMixedAncestors(10, 0, 0, 11, "");
    }

    function testConfirmByTimeCombined() public {
        claimWithMixedAncestors(10, 5, 6, 0, "");
    }

    function testConfirmByTimeCombinedClaimAll() public {
        claimWithMixedAncestors(10, 3, 5, 3, "");
    }

    function testConfirmByTimeNoTime() public {
        claimWithMixedAncestors(10, 1, 1, 1, "Total time unrivaled not greater than confirmation threshold");
    }

    function testConfirmByTimeBrokenAncestor() public {
        claimWithMixedAncestors(10, 137, 1, 1, "Current is not a child of ancestor");
    }

    function testConfirmByTimeBrokenClaim() public {
        claimWithMixedAncestors(10, 138, 1, 1, "Current is not a child of ancestor");
    }

    function testConfirmByTimeEdgeNotExist() public {
        claimWithMixedAncestors(10, 139, 1, 1, "Edge does not exist");
    }

    function testConfirmByTimeEdgeNotPending() public {
        claimWithMixedAncestors(10, 140, 1, 1, "Edge not pending");
    }

    function confirmByOneStep(uint256 flag, string memory revertArg) internal {
        uint256 startHeight = 5;
        (bytes32[] memory states1, bytes32[] memory states2) = rivalStates(startHeight, startHeight, startHeight + 1);

        ChallengeEdge memory e1 = ChallengeEdgeLib.newChildEdge(
            rand.hash(),
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, startHeight + 1)),
            startHeight,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, startHeight + 2)),
            startHeight + 1,
            EdgeType.SmallStep
        );
        ChallengeEdge memory e2 = ChallengeEdgeLib.newChildEdge(
            e1.originId,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, startHeight + 1)),
            startHeight,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, startHeight + 2)),
            startHeight + 1,
            EdgeType.SmallStep
        );
        if (flag == 2) {
            e1.status = EdgeStatus.Confirmed;
        }
        if (flag == 3) {
            e1.eType = EdgeType.BigStep;
        }
        if (flag == 5) {
            e1.endHeight = e1.endHeight + 1;
        }
        bytes32 eid = e1.idMem();

        MockOneStepProofEntry entry = new MockOneStepProofEntry();
        // confirm by one step makes external calls to the mock one step proof entry
        // since the edgechallengemanager lib is internal these calls will be the ones
        // that trigger the expectRevert behaviour - so we have a problem if we expect
        // a revert after this call. Instead we use this access contract to ensure that
        // just calling the lib creates an external call
        EdgeChallengeManagerLibAccess a = new EdgeChallengeManagerLibAccess();

        if (flag != 1) {
            a.add(e1);
        }
        if (flag != 4) {
            a.add(e2);
        }

        OneStepData memory d = OneStepData({
            execCtx: ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0))}),
            machineStep: e1.startHeight,
            beforeHash: states1[startHeight],
            proof: abi.encodePacked(states1[startHeight + 1])
        });
        bytes32[] memory beforeProof = ProofUtils.generateInclusionProof(
            ProofUtils.rehashed(ArrayUtilsLib.slice(states1, 0, startHeight + 1)), startHeight
        );
        if (flag == 6) {
            beforeProof[0] = rand.hash();
        }
        bytes32[] memory afterProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), startHeight + 1);
        if (flag == 7) {
            afterProof[0] = rand.hash();
        }
        if (flag == 8) {
            d.proof = abi.encodePacked(rand.hash());
        }

        if (bytes(revertArg).length != 0) {
            vm.expectRevert(bytes(revertArg));
        }
        a.confirmEdgeByOneStepProof(eid, entry, d, beforeProof, afterProof);

        if (bytes(revertArg).length != 0) {
            // for flag one the edge does not exist
            // for flag two we set the status to confirmed anyway
            if (flag != 1 && flag != 2) {
                assertTrue(a.getEdge(eid).status == EdgeStatus.Pending, "Edge pending");
            }
        } else {
            assertTrue(a.getEdge(eid).status == EdgeStatus.Confirmed, "Edge confirmed");
        }
    }

    function testConfirmByOneStep() public {
        confirmByOneStep(0, "");
    }

    function testConfirmByOneStepNotExist() public {
        confirmByOneStep(1, "Edge does not exist");
    }

    function testConfirmByOneStepNotPending() public {
        confirmByOneStep(2, "Edge not pending");
    }

    function testConfirmByOneStepNotSmallStep() public {
        confirmByOneStep(3, "Edge is not a small step");
    }

    function testConfirmByOneStepNoRival() public {
        confirmByOneStep(4, "Edge does not have single step rival");
    }

    function testConfirmByOneStepNotLengthOne() public {
        confirmByOneStep(5, "Edge does not have single step rival");
    }

    function testConfirmByOneStepBadStartProof() public {
        confirmByOneStep(6, "Before state not in history");
    }

    function testConfirmByOneStepBadAfterProof() public {
        confirmByOneStep(7, "After state not in history");
    }

    function testConfirmByOneStepBadOneStepReturn() public {
        confirmByOneStep(8, "After state not in history");
    }
}
