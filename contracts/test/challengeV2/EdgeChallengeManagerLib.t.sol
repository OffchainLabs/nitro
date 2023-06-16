// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
import "./Utils.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    using GlobalStateLib for GlobalState;

    function proveOneStep(ExecutionContext calldata, uint256, bytes32, bytes calldata proof)
        external
        pure
        returns (bytes32 afterHash)
    {
        return bytes32(proof);
    }

    function getMachineHash(ExecutionState calldata execState) external pure override returns (bytes32) {
        if (execState.machineStatus == MachineStatus.FINISHED) {
            return keccak256(abi.encodePacked("Machine finished:", execState.globalState.hash()));
        } else if (execState.machineStatus == MachineStatus.ERRORED) {
            return keccak256(abi.encodePacked("Machine errored:", execState.globalState.hash()));
        } else {
            revert("BAD_MACHINE_STATUS");
        }
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

    function setConfirmed(bytes32 id) public {
        store.edges[id].status = EdgeStatus.Confirmed;
    }

    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        IOneStepProofEntry oneStepProofEntry,
        OneStepData calldata oneStepData,
        ExecutionContext memory execCtx,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) public {
        store.confirmEdgeByOneStepProof(
            edgeId, oneStepProofEntry, oneStepData, execCtx, beforeHistoryInclusionProof, afterHistoryInclusionProof
        );
    }

    function createLayerZeroEdge(
        CreateEdgeArgs calldata args,
        AssertionReferenceData calldata ard,
        IOneStepProofEntry oneStepProofEntry,
        uint256 expectedEndHeight,
        uint256 challengePeriodBlocks,
        uint256 stakeAmount
    ) public returns (EdgeAddedData memory) {
        return store.createLayerZeroEdge(args, ard, oneStepProofEntry, expectedEndHeight);
    }
}

contract EdgeChallengeManagerLibTest is Test {
    using ChallengeEdgeLib for ChallengeEdge;
    using EdgeChallengeManagerLib for EdgeStore;

    uint256 challengePeriodBlocks = 7;
    uint256 stakeAmount = 13;

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

    function checkEdgeAddedData(ChallengeEdge memory edge, bool hasRival, EdgeAddedData memory d) internal {
        bytes32 id = edge.idMem();
        bytes32 mutualId = ChallengeEdgeLib.mutualIdComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight
        );
        assertEq(id, d.edgeId, "invalid edge id");
        assertEq(mutualId, d.mutualId, "invalid mutual id");
        assertEq(edge.originId, d.originId, "invalid origin id");
        assertEq(hasRival, d.hasRival, "invalid has rival");
        assertEq(edge.endHeight - edge.startHeight, d.length, "invalid length");
        assertEq(uint256(edge.eType), uint256(d.eType), "invalid eType");
        assertEq(false, d.isLayerZero, "invalid is layer zero");
    }

    function testAdd() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        EdgeAddedData memory d = store.add(edge);

        ChallengeEdge storage se = store.edges[edge.idMem()];
        assertTrue(se.exists(), "Edge exists");
        assertTrue(store.firstRivals[se.mutualId()] == EdgeChallengeManagerLib.UNRIVALED, "NO_RIVAL first rival");

        checkEdgeAddedData(se, false, d);
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

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, edgeId));
        store.get(edgeId);
    }

    function testAddRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        EdgeAddedData memory d1 = store.add(edge1);

        EdgeAddedData memory d2 = store.add(edge2);

        ChallengeEdge storage se = store.edges[edge1.idMem()];
        ChallengeEdge storage se2 = store.edges[edge2.idMem()];
        assertTrue(se2.exists(), "Edge exists");
        assertTrue(store.firstRivals[se2.mutualId()] == edge2.idMem(), "First rival1");
        assertTrue(store.firstRivals[se.mutualId()] == edge2.idMem(), "First rival2");
        checkEdgeAddedData(se, false, d1);
        checkEdgeAddedData(se2, true, d2);
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

        vm.expectRevert(abi.encodeWithSelector(EdgeAlreadyExists.selector, edge.idMem()));
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
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, edge2Id));
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
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, edge2Id));
        store.hasLengthOneRival(edge2Id);
    }

    function testTimeUnrivaled() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        store.add(edge1);
        vm.roll(block.number + 3);

        assertEq(store.timeUnrivaled(edge1.idMem()), 3, "Time unrivaled");
    }

    function testTimeUnrivaledNotExist() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        vm.roll(block.number + 3);

        bytes32 id1 = edge1.idMem();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, id1));
        assertEq(store.timeUnrivaled(id1), 3, "Time unrivaled");
    }

    function testTimeUnrivaledAfterRival() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);

        store.add(edge1);
        vm.roll(block.number + 4);

        ChallengeEdge memory edge2 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);

        store.add(edge2);
        vm.roll(block.number + 5);

        ChallengeEdge memory edge3 =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, EdgeType.Block);
        store.add(edge3);

        vm.roll(block.number + 6);

        ChallengeEdge memory edge4 =
            ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, EdgeType.Block);
        store.add(edge4);

        vm.roll(block.number + 7);

        assertEq(store.timeUnrivaled(edge1.idMem()), 4, "Time unrivaled 1");
        assertEq(store.timeUnrivaled(edge2.idMem()), 0, "Time unrivaled 2");
        assertEq(store.timeUnrivaled(edge3.idMem()), 0, "Time unrivaled 3");
        assertEq(store.timeUnrivaled(edge4.idMem()), 7, "Time unrivaled 4");
    }

    function testMandatoryBisectHeightSizeOne() public {
        vm.expectRevert(abi.encodeWithSelector(HeightDiffLtTwo.selector, 1, 2));
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

    function bisectEdgeEmitted(
        ChallengeEdge memory edge,
        bytes32 bisectionRoot,
        uint256 bisectionHeight,
        bool lowerChildHasRival,
        bool upperChildHasRival,
        bytes32 lowerChildId,
        EdgeAddedData memory lowerChildAdded,
        EdgeAddedData memory upperChildAdded
    ) internal {
        ChallengeEdge memory lowerChild = ChallengeEdgeLib.newChildEdge(
            edge.originId, edge.startHistoryRoot, edge.startHeight, bisectionRoot, bisectionHeight, edge.eType
        );
        ChallengeEdge memory upperChild = ChallengeEdgeLib.newChildEdge(
            edge.originId, bisectionRoot, bisectionHeight, edge.endHistoryRoot, edge.endHeight, edge.eType
        );

        if (lowerChildAdded.edgeId != 0) {
            checkEdgeAddedData(lowerChild, lowerChildHasRival, lowerChildAdded);
        }
        checkEdgeAddedData(upperChild, upperChildHasRival, upperChildAdded);

        bytes32 lowerId = lowerChild.idMem();
        assertEq(lowerId, lowerChildId, "Invalid lower child id");
    }

    function bisectAndCheck(
        ChallengeEdge memory edge,
        bytes32 bisectionRoot,
        uint256 bisectionPoint,
        bytes32[] memory states,
        bool lowerHasRival,
        bool upperHasRival
    ) internal {
        (bytes32 lowerChildId, EdgeAddedData memory lowerChildAdded, EdgeAddedData memory upperChildAdded) = store
            .bisectEdge(
            edge.idMem(),
            bisectionRoot,
            abi.encode(
                ProofUtils.expansionFromLeaves(states, 0, bisectionPoint + 1),
                ProofUtils.generatePrefixProof(
                    bisectionPoint + 1, ArrayUtilsLib.slice(states, bisectionPoint + 1, states.length)
                )
            )
        );
        bisectEdgeEmitted(
            edge,
            bisectionRoot,
            bisectionPoint,
            lowerHasRival,
            upperHasRival,
            lowerChildId,
            lowerChildAdded,
            upperChildAdded
        );
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
        bisectAndCheck(edge1, bisectionRoot1, bisectionPoint, states1, false, false);

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
        bisectAndCheck(edge2, bisectionRoot2, bisectionPoint, states2, true, false);

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
        bisectAndCheck(edge1, bisectionRoot1, bisectionPoint, states1, false, false);

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
        bisectAndCheck(edge2, bisectionRoot2, bisectionPoint, states2, false, true);

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
        vm.expectRevert(abi.encodeWithSelector(EdgeUnrivaled.selector, edgeId));
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

        vm.expectRevert(abi.encodeWithSelector(EdgeAlreadyExists.selector, store.edges[edgeId].upperChildId));
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
        vm.expectRevert(abi.encodeWithSelector(EdgeNotPending.selector, edgeId, EdgeStatus.Confirmed));
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
        vm.expectRevert(abi.encodeWithSelector(HeightDiffLtTwo.selector, start, end));
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

        (bytes32 lowerChildId,, EdgeAddedData memory upperChildAdded) =
            store.bisectEdge(edge1.idMem(), bisectionRoot, bisectionProof);

        return (edge1.idMem(), lowerChildId, upperChildAdded.edgeId);
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
        vm.expectRevert(abi.encodeWithSelector(EdgeNotPending.selector, parentEdgeId, EdgeStatus.Confirmed));
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenNotExist() public {
        bytes32 randId = rand.hash();

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, randId));
        store.confirmEdgeByChildren(randId);
    }

    function testConfirmEdgeByChildrenLowerChildNotExist() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        delete store.edges[lowerChildId];
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, lowerChildId));
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenUpperChildNotExist() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();
        store.edges[upperChildId].setConfirmed();

        delete store.edges[upperChildId];
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, upperChildId));
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenLowerChildPending() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[upperChildId].setConfirmed();

        vm.expectRevert(abi.encodeWithSelector(EdgeNotConfirmed.selector, lowerChildId, EdgeStatus.Pending));
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testConfirmEdgeByChildrenUpperChildPending() public {
        (bytes32 parentEdgeId, bytes32 lowerChildId, bytes32 upperChildId) = addParentAndChildren(3, 5, 11);

        store.edges[lowerChildId].setConfirmed();

        vm.expectRevert(abi.encodeWithSelector(EdgeNotConfirmed.selector, upperChildId, EdgeStatus.Pending));
        store.confirmEdgeByChildren(parentEdgeId);
    }

    function testNextEdgeType() public {
        assertTrue(EdgeChallengeManagerLib.nextEdgeType(EdgeType.Block) == EdgeType.BigStep);
        assertTrue(EdgeChallengeManagerLib.nextEdgeType(EdgeType.BigStep) == EdgeType.SmallStep);
        vm.expectRevert("No next type after SmallStep");
        EdgeChallengeManagerLib.nextEdgeType(EdgeType.SmallStep);
    }

    function testConfirmClaim() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();

        store.confirmEdgeByClaim(bargs.upperChildId1, eid);

        assertTrue(store.edges[bargs.upperChildId1].status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function testCheckClaimIdLinkOrigin() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            rand.hash(), rand.hash(), 3, rand.hash(), 4, bargs.upperChildId1, rand.addr(), EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        vm.expectRevert(
            abi.encodeWithSelector(
                OriginIdMutualIdMismatch.selector,
                store.edges[bargs.upperChildId1].mutualId(),
                store.edges[eid].originId
            )
        );
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function testCheckClaimIdLinkEdgeType() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.Block
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        vm.expectRevert(
            abi.encodeWithSelector(
                EdgeTypeInvalid.selector,
                bargs.upperChildId1,
                eid,
                EdgeChallengeManagerLib.nextEdgeType(store.edges[bargs.upperChildId1].eType),
                store.edges[eid].eType
            )
        );
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function testConfirmClaimWrongClaimId() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
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
        vm.expectRevert(
            abi.encodeWithSelector(EdgeClaimMismatch.selector, bargs.upperChildId1, store.edges[eid].claimId)
        );
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function testConfirmClaimNotExist() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.BigStep
        );

        bytes32 eid = ce.idMem();
        bytes32 randId = rand.hash();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, randId));
        store.confirmEdgeByClaim(randId, eid);
    }

    function testConfirmClaimNotPending() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        store.edges[eid].setConfirmed();
        store.edges[bargs.upperChildId1].setConfirmed();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotPending.selector, bargs.upperChildId1, EdgeStatus.Confirmed));
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function testConfirmClaimClaimerNotExist() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.BigStep
        );

        bytes32 eid = ce.idMem();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, bargs.upperChildId1));
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function testConfirmClaimClaimerNotConfirmed() public {
        BArgs memory bargs = addParentsAndChildren(2, 3, 4);

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            store.get(bargs.upperChildId1).mutualId(),
            rand.hash(),
            3,
            rand.hash(),
            4,
            bargs.upperChildId1,
            rand.addr(),
            EdgeType.BigStep
        );

        store.add(ce);
        bytes32 eid = ce.idMem();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotConfirmed.selector, eid, EdgeStatus.Pending));
        store.confirmEdgeByClaim(bargs.upperChildId1, eid);
    }

    function bisect(ChallengeEdge memory edge, bytes32[] memory states, uint256 start, uint256 end)
        internal
        returns (bytes32, bytes32)
    {
        (, bytes32 bisectionRoot, bytes memory bisectionProof) = bisectArgs(states, start, end);
        (bytes32 lowerChildId,, EdgeAddedData memory upperChildAdded) =
            store.bisectEdge(edge.idMem(), bisectionRoot, bisectionProof);
        return (lowerChildId, upperChildAdded.edgeId);
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
        uint256 claimedAssertionBlocks
    ) internal {
        BArgs memory pc = addParentsAndChildren(2, 5, 8);
        bytes memory revertArg;
        bytes32 lowerChildId148;
        {
            (, bytes32 bisectionRoot, bytes memory bisectionProof) = bisectArgs(pc.states1, 4, 8);
            (bytes32 lowerChildId148X,,) = store.bisectEdge(pc.upperChildId1, bisectionRoot, bisectionProof);
            lowerChildId148 = lowerChildId148X;
        }
        vm.roll(block.number + timeAfterParent1);

        bytes32 upperChildId146;
        {
            bytes32 lowerChildId248;
            {
                (, bytes32 bisectionRoot2, bytes memory bisectionProof2) = bisectArgs(pc.states2, 4, 8);
                (bytes32 lowerChildId248X,,) = store.bisectEdge(pc.upperChildId2, bisectionRoot2, bisectionProof2);
                lowerChildId248 = lowerChildId248X;
            }

            (, bytes32 bisectionRoot3, bytes memory bisectionProof3) = bisectArgs(pc.states1, 4, 6);
            (,, EdgeAddedData memory upperChildId146Data) =
                store.bisectEdge(lowerChildId148, bisectionRoot3, bisectionProof3);
            upperChildId146 = upperChildId146Data.edgeId;
            vm.roll(block.number + timeAfterParent2);

            (, bytes32 bisectionRoot4, bytes memory bisectionProof4) = bisectArgs(pc.states2, 4, 6);
            store.bisectEdge(lowerChildId248, bisectionRoot4, bisectionProof4);
        }

        bytes32 bsId;
        {
            ChallengeEdge memory bigStepZero = ChallengeEdgeLib.newLayerZeroEdge(
                store.edges[upperChildId146].mutualId(),
                store.edges[upperChildId146].startHistoryRoot,
                store.edges[upperChildId146].startHeight,
                rand.hash(),
                100,
                upperChildId146,
                rand.addr(),
                EdgeType.BigStep
            );
            if (timeAfterParent1 != 139) {
                store.add(bigStepZero);
            } else {
                revertArg = abi.encodeWithSelector(EdgeNotExists.selector, bigStepZero.idMem());
            }
            bsId = bigStepZero.idMem();
        }

        vm.roll(block.number + timeAfterZeroLayer);

        bytes32[] memory ancestorIds = new bytes32[](3);
        ancestorIds[0] = upperChildId146;
        ancestorIds[1] = lowerChildId148;
        ancestorIds[2] = pc.upperChildId1;

        if (timeAfterParent1 == 137) {
            ChallengeEdge memory childE1 =
                ChallengeEdgeLib.newChildEdge(rand.hash(), rand.hash(), 10, rand.hash(), 100, EdgeType.Block);

            store.add(childE1);

            ancestorIds[1] = childE1.idMem();

            revertArg = abi.encodeWithSelector(
                EdgeNotAncestor.selector,
                ancestorIds[0],
                store.edges[bsId].lowerChildId,
                store.edges[bsId].upperChildId,
                ancestorIds[1],
                store.edges[ancestorIds[0]].claimId
            );
        }

        if (timeAfterParent1 == 138) {
            store.edges[bsId].claimId = rand.hash();
            revertArg = abi.encodeWithSelector(
                EdgeNotAncestor.selector,
                bsId,
                store.edges[bsId].lowerChildId,
                store.edges[bsId].upperChildId,
                ancestorIds[0],
                store.edges[bsId].claimId
            );
        }

        if (timeAfterParent1 == 140) {
            store.edges[bsId].setConfirmed();
            revertArg = abi.encodeWithSelector(EdgeNotPending.selector, bsId, EdgeStatus.Confirmed);
        }

        if (timeAfterParent1 + timeAfterParent2 + timeAfterZeroLayer + claimedAssertionBlocks == 4) {
            revertArg = abi.encodeWithSelector(InsufficientConfirmationBlocks.selector, 4, challengePeriodSec);
        }

        if (revertArg.length != 0) {
            vm.expectRevert(revertArg);
        }
        uint256 totalTime = store.confirmEdgeByTime(bsId, ancestorIds, claimedAssertionBlocks, challengePeriodSec);

        assertTrue(store.edges[bsId].status == EdgeStatus.Confirmed, "Edge confirmed");
        assertEq(
            totalTime,
            timeAfterParent1 + timeAfterParent2 + timeAfterZeroLayer + claimedAssertionBlocks,
            "Invalid total time"
        );
    }

    function testConfirmByTimeGrandParent() public {
        claimWithMixedAncestors(10, 11, 0, 0, 0);
    }

    function testConfirmByTimeParent() public {
        claimWithMixedAncestors(10, 0, 11, 0, 0);
    }

    function testConfirmByTimeSelf() public {
        claimWithMixedAncestors(10, 0, 0, 11, 0);
    }

    function testConfirmByTimeAssertion() public {
        claimWithMixedAncestors(10, 0, 0, 0, 11);
    }

    function testConfirmByTimeCombined() public {
        claimWithMixedAncestors(10, 5, 6, 0, 0);
    }

    function testConfirmByTimeCombinedClaimAll() public {
        claimWithMixedAncestors(10, 3, 3, 3, 3);
    }

    function testConfirmByTimeNoTime() public {
        claimWithMixedAncestors(10, 1, 1, 1, 1);
    }

    function testConfirmByTimeBrokenAncestor() public {
        claimWithMixedAncestors(10, 137, 1, 1, 1);
    }

    function testConfirmByTimeBrokenClaim() public {
        claimWithMixedAncestors(10, 138, 1, 1, 1);
    }

    function testConfirmByTimeEdgeNotExist() public {
        claimWithMixedAncestors(10, 139, 1, 1, 1);
    }

    function testConfirmByTimeEdgeNotPending() public {
        claimWithMixedAncestors(10, 140, 1, 1, 1);
    }

    function confirmByOneStep(uint256 flag) internal {
        bytes memory revertArg;
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
        if (flag == 3) {
            e1.eType = EdgeType.BigStep;
            revertArg = abi.encodeWithSelector(EdgeTypeNotSmallStep.selector, e1.eType);
        }
        if (flag == 5) {
            e1.endHeight = e1.endHeight + 1;
            revertArg = abi.encodeWithSelector(EdgeNotLengthOne.selector, 2);
        }
        bytes32 eid = e1.idMem();
        if (flag == 2) {
            e1.status = EdgeStatus.Confirmed;
            revertArg = abi.encodeWithSelector(EdgeNotPending.selector, eid, e1.status);
        }

        MockOneStepProofEntry entry = new MockOneStepProofEntry();
        // confirm by one step makes external calls to the mock one step proof entry
        // since the edgechallengemanager lib is internal these calls will be the ones
        // that trigger the expectRevert behaviour - so we have a problem if we expect
        // a revert after this call. Instead we use this access contract to ensure that
        // just calling the lib creates an external call
        EdgeChallengeManagerLibAccess a = new EdgeChallengeManagerLibAccess();

        if (flag != 1) {
            a.add(e1);
        } else {
            revertArg = abi.encodeWithSelector(EdgeNotExists.selector, eid);
        }
        if (flag != 4) {
            a.add(e2);
        }
        OneStepData memory d =
            OneStepData({beforeHash: states1[startHeight], proof: abi.encodePacked(states1[startHeight + 1])});
        ExecutionContext memory e =
            ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0)), initialWasmModuleRoot: bytes32(0)});
        bytes32[] memory beforeProof = ProofUtils.generateInclusionProof(
            ProofUtils.rehashed(ArrayUtilsLib.slice(states1, 0, startHeight + 1)), startHeight
        );
        if (flag == 6) {
            beforeProof[0] = rand.hash();
            revertArg = "Invalid inclusion proof";
        }
        bytes32[] memory afterProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), startHeight + 1);
        if (flag == 7) {
            afterProof[0] = rand.hash();
            revertArg = "Invalid inclusion proof";
        }
        if (flag == 8) {
            d.proof = abi.encodePacked(rand.hash());
            revertArg = "Invalid inclusion proof";
        }

        if (revertArg.length != 0) {
            vm.expectRevert(revertArg);
        }
        a.confirmEdgeByOneStepProof(eid, entry, d, e, beforeProof, afterProof);

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
        confirmByOneStep(0);
    }

    function testConfirmByOneStepNotExist() public {
        confirmByOneStep(1);
    }

    function testConfirmByOneStepNotPending() public {
        confirmByOneStep(2);
    }

    function testConfirmByOneStepNotSmallStep() public {
        confirmByOneStep(3);
    }

    function testConfirmByOneStepNotLengthOne() public {
        confirmByOneStep(5);
    }

    function testConfirmByOneStepBadStartProof() public {
        confirmByOneStep(6);
    }

    function testConfirmByOneStepBadAfterProof() public {
        confirmByOneStep(7);
    }

    function testConfirmByOneStepBadOneStepReturn() public {
        confirmByOneStep(8);
    }

    function testPowerOfTwo() public {
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(0), false);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(1), true);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(2), true);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(3), false);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(4), true);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(5), false);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(6), false);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(7), false);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(8), true);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(2 ** 17), true);
        assertEq(EdgeChallengeManagerLib.isPowerOfTwo(1 << 255), true);
    }

    struct ExpsAndProofs {
        bytes32[] states;
        bytes32[] startExp;
        bytes32[] endExp;
        bytes32[] startInclusionProof;
        bytes32[] endInclusionProof;
        bytes32[] prefixProof;
    }

    function newRootsAndProofs(uint256 startHeight, uint256 endHeight, bytes32 startState, bytes32 endState)
        internal
        returns (ExpsAndProofs memory)
    {
        bytes32[] memory states;
        {
            if (startState == 0) {
                startState = rand.hash();
            }
            if (endState == 0) {
                endState = rand.hash();
            }

            bytes32[] memory innerStates = rand.hashes(endHeight - 1);
            bytes32[] memory startStates = new bytes32[](1);
            startStates[0] = startState;
            bytes32[] memory endStates = new bytes32[](1);
            endStates[0] = endState;
            states = ArrayUtilsLib.concat(ArrayUtilsLib.concat(startStates, innerStates), endStates);
        }
        bytes32[] memory startExp = ProofUtils.expansionFromLeaves(states, 0, startHeight + 1);
        bytes32[] memory expansion = ProofUtils.expansionFromLeaves(states, 0, endHeight + 1);

        // inclusion in the start root
        bytes32[] memory startInclusionProof = ProofUtils.generateInclusionProof(
            ProofUtils.rehashed(ArrayUtilsLib.slice(states, 0, startHeight + 1)), startHeight
        );
        bytes32[] memory endInclusionProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), endHeight);

        bytes32[] memory prefixProof =
            ProofUtils.generatePrefixProof(startHeight + 1, ArrayUtilsLib.slice(states, startHeight + 1, endHeight + 1));

        return ExpsAndProofs(states, startExp, expansion, startInclusionProof, endInclusionProof, prefixProof);
    }

    struct ExecStateVars {
        ExecutionState execState;
        bytes32 machineHash;
        bytes32 stateHash;
    }

    function randomExecutionState(IOneStepProofEntry os, uint256 inboxMaxCount)
        private
        returns (ExecStateVars memory)
    {
        ExecutionState memory execState = ExecutionState(
            GlobalState([rand.hash(), rand.hash()], [uint64(uint256(rand.hash())), uint64(uint256(rand.hash()))]),
            MachineStatus.FINISHED
        );

        bytes32 machineHash = os.getMachineHash(execState);
        bytes32 stateHash = RollupLib.stateHashMem(execState, inboxMaxCount);
        return ExecStateVars(execState, machineHash, stateHash);
    }

    function createZeroBlockEdge(uint256 mode) internal {
        bytes memory revertArg;
        MockOneStepProofEntry entry = new MockOneStepProofEntry();
        uint256 expectedEndHeight = 2 ** 2;
        if (mode == 139) {
            expectedEndHeight = 2 ** 5 - 1;
            revertArg = abi.encodeWithSelector(NotPowerOfTwo.selector, expectedEndHeight);
        }

        ExecStateVars memory startExec = randomExecutionState(entry, 10);
        ExecStateVars memory endExec = randomExecutionState(entry, 20);
        ExpsAndProofs memory roots = newRootsAndProofs(0, expectedEndHeight, startExec.machineHash, endExec.machineHash);
        bytes32 claimId = rand.hash();
        bytes32 endRoot;
        if (mode == 137) {
            endRoot = rand.hash();
            revertArg = "Invalid inclusion proof";
        } else {
            endRoot = MerkleTreeLib.root(roots.endExp);
        }
        AssertionReferenceData memory ard;
        if (mode != 144) {
            ard = AssertionReferenceData({
                assertionId: claimId,
                predecessorId: rand.hash(),
                isPending: true,
                hasSibling: true,
                startState: startExec.execState,
                endState: endExec.execState
            });
            if (mode == 141) {
                ard.assertionId = rand.hash();
                revertArg = abi.encodeWithSelector(AssertionIdMismatch.selector, ard.assertionId, claimId);
            }
            if (mode == 142) {
                ard.isPending = false;
                revertArg = abi.encodeWithSelector(AssertionNotPending.selector);
            }
            if (mode == 143) {
                ard.hasSibling = false;
                revertArg = abi.encodeWithSelector(AssertionNoSibling.selector);
            }
        } else {
            revertArg = abi.encodeWithSelector(AssertionIdEmpty.selector);
        }

        if (mode == 145) {
            ExecutionState memory s;
            ard.startState = s;
            revertArg = abi.encodeWithSelector(EmptyStartMachineStatus.selector);
        }
        if (mode == 146) {
            ExecutionState memory e;
            ard.endState = e;
            revertArg = abi.encodeWithSelector(EmptyEndMachineStatus.selector);
        }

        bytes memory proof = abi.encode(
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(roots.states), expectedEndHeight),
            ExecutionStateData(ard.startState, bytes32(0), bytes32(0)),
            ExecutionStateData(ard.endState, bytes32(0), bytes32(0))
        );
        if (mode == 140) {
            bytes32[] memory b = new bytes32[](1);
            b[0] = rand.hash();
            roots.prefixProof = ArrayUtilsLib.concat(roots.prefixProof, b);
            revertArg = bytes("Incomplete proof usage");
        }

        EdgeChallengeManagerLibAccess a = new EdgeChallengeManagerLibAccess();

        CreateEdgeArgs memory args = CreateEdgeArgs({
            edgeType: EdgeType.Block,
            endHistoryRoot: endRoot,
            endHeight: expectedEndHeight,
            claimId: claimId,
            prefixProof: abi.encode(roots.startExp, roots.prefixProof),
            proof: proof
        });
        if (mode == 138) {
            args.endHeight = 2 ** 4;
            revertArg = abi.encodeWithSelector(InvalidEndHeight.selector, 2 ** 4, expectedEndHeight);
        }

        if (revertArg.length != 0) {
            vm.expectRevert(revertArg);
        }
        a.createLayerZeroEdge(args, ard, entry, expectedEndHeight, challengePeriodBlocks, stakeAmount);
    }

    function testCreateLayerZeroEdgeBlockA() public {
        createZeroBlockEdge(0);
    }

    function testCreateLayerZeroEdgeBlockInvalidInclusionProof() public {
        createZeroBlockEdge(137);
    }

    function testCreateLayerZeroEdgeBlockEndHeight() public {
        createZeroBlockEdge(138);
    }

    function testCreateLayerZeroEdgeBlockPowerHeight() public {
        createZeroBlockEdge(139);
    }

    function testCreateLayerZeroEdgeInvalidPrefixProof() public {
        createZeroBlockEdge(140);
    }

    function testCreateLayerZeroEdgeClaimId() public {
        createZeroBlockEdge(141);
    }

    function testCreateLayerZeroEdgeIsPending() public {
        createZeroBlockEdge(142);
    }

    function testCreateLayerZeroEdgeHasSibling() public {
        createZeroBlockEdge(143);
    }

    function testCreateLayerZeroEdgeEmptyAssertion() public {
        createZeroBlockEdge(144);
    }

    function testCreateLayerZeroEdgeEmptyStartState() public {
        createZeroBlockEdge(145);
    }

    function testCreateLayerZeroEdgeEmptyEndState() public {
        createZeroBlockEdge(146);
    }

    function createClaimEdge(EdgeChallengeManagerLibAccess c, uint256 start, uint256 end, bool includeRival)
        public
        returns (bytes32, ExpsAndProofs memory)
    {
        // create a claim edge
        ExpsAndProofs memory claimRoots = newRootsAndProofs(start, end, 0, 0);
        ChallengeEdge memory ce = ChallengeEdgeLib.newChildEdge(
            rand.hash(),
            MerkleTreeLib.root(claimRoots.startExp),
            start,
            MerkleTreeLib.root(claimRoots.endExp),
            end,
            EdgeType.BigStep
        );
        c.add(ce);
        // and give it a rival
        if (includeRival) {
            c.add(
                ChallengeEdgeLib.newChildEdge(
                    ce.originId, ce.startHistoryRoot, ce.startHeight, rand.hash(), ce.endHeight, ce.eType
                )
            );
        }

        return (ce.idMem(), claimRoots);
    }

    function createSmallStepEdge(uint256 mode) internal {
        bytes memory revertArg;
        uint256 claimStartHeight = 4;
        uint256 claimEndHeight = mode == 161 ? 6 : 5;

        uint256 expectedEndHeight = 2 ** 5;
        EdgeChallengeManagerLibAccess c = new EdgeChallengeManagerLibAccess();
        (bytes32 claimId, ExpsAndProofs memory claimRoots) =
            createClaimEdge(c, claimStartHeight, claimEndHeight, mode == 160 ? false : true);
        if (mode == 160) {
            revertArg = abi.encodeWithSelector(ClaimEdgeNotLengthOneRival.selector, claimId);
        }
        if (mode == 161) {
            revertArg = abi.encodeWithSelector(ClaimEdgeNotLengthOneRival.selector, claimId);
        }

        ExpsAndProofs memory roots = newRootsAndProofs(
            0, expectedEndHeight, claimRoots.states[claimStartHeight], claimRoots.states[claimEndHeight]
        );
        if (mode == 164) {
            bytes32[] memory b = new bytes32[](1);
            b[0] = rand.hash();
            claimRoots.startInclusionProof = ArrayUtilsLib.concat(claimRoots.startInclusionProof, b);
            revertArg = "Invalid inclusion proof";
        }
        if (mode == 165) {
            bytes32[] memory b = new bytes32[](1);
            b[0] = rand.hash();
            claimRoots.endInclusionProof = ArrayUtilsLib.concat(claimRoots.endInclusionProof, b);
            revertArg = "Invalid inclusion proof";
        }
        bytes memory proof = abi.encode(
            roots.states[0],
            roots.states[expectedEndHeight],
            claimRoots.startInclusionProof,
            claimRoots.endInclusionProof,
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(roots.states), expectedEndHeight)
        );
        if (mode == 162) {
            c.setConfirmed(claimId);
            revertArg = abi.encodeWithSelector(ClaimEdgeNotPending.selector);
        }

        MockOneStepProofEntry a = new MockOneStepProofEntry();
        AssertionReferenceData memory emptyArd;

        if (mode == 163) {
            revertArg = abi.encodeWithSelector(ClaimEdgeInvalidType.selector, EdgeType.BigStep, EdgeType.BigStep);
        }
        if (revertArg.length != 0) {
            vm.expectRevert(revertArg);
        }
        c.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: mode == 163 ? EdgeType.BigStep : EdgeType.SmallStep,
                endHistoryRoot: MerkleTreeLib.root(roots.endExp),
                endHeight: expectedEndHeight,
                claimId: claimId,
                prefixProof: abi.encode(roots.startExp, roots.prefixProof),
                proof: proof
            }),
            emptyArd,
            a,
            expectedEndHeight,
            challengePeriodBlocks,
            stakeAmount
        );
    }

    function testCreateLayerZeroEdgeSmallStep() public {
        createSmallStepEdge(0);
    }

    function testCreateLayerZeroEdgeSmallStepNoRival() public {
        createSmallStepEdge(160);
    }

    function testCreateLayerZeroEdgeSmallStepNotLength1() public {
        createSmallStepEdge(161);
    }

    function testCreateLayerZeroEdgeSmallStepEdgeType() public {
        createSmallStepEdge(162);
    }

    function testCreateLayerZeroEdgeSmallStepNotPending() public {
        createSmallStepEdge(163);
    }

    function testCreateLayerZeroEdgeSmallStepStartInclusion() public {
        createSmallStepEdge(164);
    }

    function testCreateLayerZeroEdgeSmallStepEndInclusion() public {
        createSmallStepEdge(165);
    }
}
