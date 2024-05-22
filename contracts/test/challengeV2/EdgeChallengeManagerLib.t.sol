// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
import "./Utils.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    using GlobalStateLib for GlobalState;

    constructor(uint256 _testMachineStep) {
        testMachineStep = _testMachineStep;
    }

    uint256 public testMachineStep;

    function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot) external pure returns (bytes32) {
        return keccak256(abi.encodePacked("Machine:", globalStateHash, wasmModuleRoot));
    }

    function proveOneStep(ExecutionContext calldata, uint256 machineStep, bytes32, bytes calldata proof)
        external
        view
        returns (bytes32 afterHash)
    {
        if (testMachineStep != 0) require(testMachineStep == machineStep, "Invalid machine step");
        return bytes32(proof);
    }

    function getMachineHash(ExecutionState calldata execState) external pure override returns (bytes32) {
        require(execState.machineStatus == MachineStatus.FINISHED, "BAD_MACHINE_STATUS");
        return GlobalStateLib.hash(execState.globalState);
    }
}

contract EdgeChallengeManagerLibAccess {
    using EdgeChallengeManagerLib for EdgeStore;
    using ChallengeEdgeLib for ChallengeEdge;

    EdgeStore private store;

    function exists(bytes32 edgeId) public view returns (bool) {
        return store.get(edgeId).exists();
    }

    function get(bytes32 edgeId) public view returns (ChallengeEdge memory) {
        return store.get(edgeId);
    }

    function getNoCheck(bytes32 edgeId) public view returns (ChallengeEdge memory) {
        return store.getNoCheck(edgeId);
    }

    function add(ChallengeEdge memory edge) public returns (EdgeAddedData memory) {
        return store.add(edge);
    }

    function isPowerOfTwo(uint256 x) public pure returns (bool) {
        return EdgeChallengeManagerLib.isPowerOfTwo(x);
    }

    function createLayerZeroEdge(
        CreateEdgeArgs calldata args,
        AssertionReferenceData memory ard,
        IOneStepProofEntry oneStepProofEntry,
        uint256 expectedEndHeight,
        uint8 numBigStepLevel,
        bool whitelistEnabled
    ) public returns (EdgeAddedData memory) {
        return store.createLayerZeroEdge(
            args, ard, oneStepProofEntry, expectedEndHeight, numBigStepLevel, whitelistEnabled
        );
    }

    function getPrevAssertionHash(bytes32 edgeId) public view returns (bytes32) {
        return store.getPrevAssertionHash(edgeId);
    }

    function hasRival(bytes32 edgeId) public view returns (bool) {
        return store.hasRival(edgeId);
    }

    function setFirstRival(bytes32 edgeId, bytes32 firstRival) public {
        store.firstRivals[edgeId] = firstRival;
    }

    function hasLengthOneRival(bytes32 edgeId) public view returns (bool) {
        return store.hasLengthOneRival(edgeId);
    }

    function timeUnrivaled(bytes32 edgeId) public view returns (uint256) {
        return store.timeUnrivaled(edgeId);
    }

    function timeUnrivaledTotal(bytes32 edgeId) public view returns (uint256) {
        return store.timeUnrivaledTotal(edgeId);
    }

    function updateTimerCacheByChildren(bytes32 edgeId, uint256 requiredTime) public {
        store.updateTimerCacheByChildren(edgeId, requiredTime);
    }

    function mandatoryBisectionHeight(uint256 start, uint256 end) public pure returns (uint256) {
        return EdgeChallengeManagerLib.mandatoryBisectionHeight(start, end);
    }

    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        public
        returns (bytes32, EdgeAddedData memory, EdgeAddedData memory)
    {
        return store.bisectEdge(edgeId, bisectionHistoryRoot, prefixProof);
    }

    function setConfirmed(bytes32 id) public {
        store.get(id).setConfirmed();
    }

    function setConfirmedRival(bytes32 edgeId) public {
        return EdgeChallengeManagerLib.setConfirmedRival(store, edgeId);
    }

    function setClaimId(bytes32 edgeId, bytes32 claimId) public {
        store.get(edgeId).claimId = claimId;
    }

    function getConfirmedRival(bytes32 mutualId) public view returns (bytes32) {
        return store.confirmedRivals[mutualId];
    }

    function setLevel(bytes32 edgeId, uint8 level) public {
        store.get(edgeId).level = level;
    }

    function nextEdgeLevel(uint8 level, uint8 numBigStepLevel) public pure returns (uint8) {
        return EdgeChallengeManagerLib.nextEdgeLevel(level, numBigStepLevel);
    }

    function firstRivals(bytes32 mutualId) public view returns (bytes32) {
        return store.firstRivals[mutualId];
    }

    function hasMadeLayerZeroRival(address account, bytes32 mutualId) public view returns (bool) {
        return store.hasMadeLayerZeroRival[account][mutualId];
    }

    function setHasMadeLayerZeroRival(address account, bytes32 mutualId, bool x) public {
        store.hasMadeLayerZeroRival[account][mutualId] = x;
    }

    function remove(bytes32 edgeId) public {
        delete store.edges[edgeId];
    }

    function confirmedRivals(bytes32 mutualId) public view returns (bytes32) {
        return store.confirmedRivals[mutualId];
    }

    function confirmEdgeByTime(
        bytes32 edgeId,
        uint64 claimedAssertionUnrivaledBlocks,
        uint64 confirmationThresholdBlock
    ) public returns (uint256) {
        return store.confirmEdgeByTime(edgeId, claimedAssertionUnrivaledBlocks, confirmationThresholdBlock);
    }

    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        IOneStepProofEntry oneStepProofEntry,
        OneStepData calldata oneStepData,
        ExecutionContext memory execCtx,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof,
        uint8 numBigStepLevel,
        uint256 bigStepHeight,
        uint256 smallStepHeight
    ) public {
        store.confirmEdgeByOneStepProof(
            edgeId,
            oneStepProofEntry,
            oneStepData,
            execCtx,
            beforeHistoryInclusionProof,
            afterHistoryInclusionProof,
            numBigStepLevel,
            bigStepHeight,
            smallStepHeight
        );
    }
}

contract EdgeChallengeManagerLibTest is Test {
    using ChallengeEdgeLib for ChallengeEdge;
    using AssertionStateLib for AssertionState;

    MockOneStepProofEntry mockOsp = new MockOneStepProofEntry(0);

    uint64 challengePeriodBlocks = 7;
    uint256 stakeAmount = 13;

    EdgeChallengeManagerLibAccess store = new EdgeChallengeManagerLibAccess();
    Random rand = new Random();

    uint8 constant NUM_BIGSTEP_LEVEL = 3;

    function twoNonRivals() internal returns (ChallengeEdge memory, ChallengeEdge memory) {
        bytes32 originId = rand.hash();

        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, 0);

        return (edge1, edge2);
    }

    function twoRivals() internal returns (ChallengeEdge memory, ChallengeEdge memory) {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();

        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, 0);

        return (edge1, edge2);
    }

    function checkEdgeAddedData(ChallengeEdge memory edge, bool hasRival, EdgeAddedData memory d) internal {
        bytes32 id = edge.idMem();
        bytes32 mutualId = ChallengeEdgeLib.mutualIdComponent(
            edge.level, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight
        );
        assertEq(id, d.edgeId, "invalid edge id");
        assertEq(mutualId, d.mutualId, "invalid mutual id");
        assertEq(edge.originId, d.originId, "invalid origin id");
        assertEq(hasRival, d.hasRival, "invalid has rival");
        assertEq(edge.endHeight - edge.startHeight, d.length, "invalid length");
        assertEq(uint256(edge.level), uint256(d.level), "invalid eType");
        assertEq(false, d.isLayerZero, "invalid is layer zero");
    }

    function testAdd() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        EdgeAddedData memory d = store.add(edge);

        ChallengeEdge memory se = store.get(edge.idMem());
        assertTrue(store.exists(edge.idMem()), "Edge exists");
        assertTrue(store.firstRivals(se.mutualIdMem()) == EdgeChallengeManagerLib.UNRIVALED, "NO_RIVAL first rival");

        checkEdgeAddedData(se, false, d);
    }

    function testGet() public {
        (ChallengeEdge memory edge,) = twoNonRivals();

        store.add(edge);

        ChallengeEdge memory se = store.get(edge.idMem());
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

        ChallengeEdge memory se = store.get(edge1.idMem());
        ChallengeEdge memory se2 = store.get(edge2.idMem());
        assertTrue(store.exists(se2.idMem()), "Edge exists");
        assertTrue(store.firstRivals(se2.mutualIdMem()) == edge2.idMem(), "First rival1");
        assertTrue(store.firstRivals(se.mutualIdMem()) == edge2.idMem(), "First rival2");
        checkEdgeAddedData(se, false, d1);
        checkEdgeAddedData(se2, true, d2);
    }

    function testAddMoreRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, 0
        );

        store.add(edge1);
        store.add(edge2);
        store.add(edge3);

        ChallengeEdge memory se = store.get(edge1.idMem());
        ChallengeEdge memory se2 = store.get(edge2.idMem());
        ChallengeEdge memory se3 = store.get(edge3.idMem());
        assertTrue(store.exists(se3.idMem()), "Edge exists");
        assertTrue(store.firstRivals(se.mutualIdMem()) == edge2.idMem(), "First rival1");
        assertTrue(store.firstRivals(se2.mutualIdMem()) == edge2.idMem(), "First rival2");
        assertTrue(store.firstRivals(se3.mutualIdMem()) == edge2.idMem(), "First rival3");
    }

    function testAddNonRivals() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoNonRivals();

        store.add(edge1);
        store.add(edge2);

        ChallengeEdge memory se = store.get(edge1.idMem());
        ChallengeEdge memory se2 = store.get(edge2.idMem());
        assertTrue(store.exists(se2.idMem()), "Edge exists");
        assertTrue(store.firstRivals(se.mutualIdMem()) == EdgeChallengeManagerLib.UNRIVALED, "First rival1");
        assertTrue(store.firstRivals(se2.mutualIdMem()) == EdgeChallengeManagerLib.UNRIVALED, "First rival2");
    }

    function testCannotAddSameEdgeTwice() public {
        ChallengeEdge memory edge = ChallengeEdgeLib.newChildEdge(rand.hash(), rand.hash(), 0, rand.hash(), 10, 0);

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

    function testHasRivalEmpty() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);
        store.add(edge2);

        assertTrue(store.hasRival(edge1.idMem()), "Edge1 rival");
        store.setFirstRival(edge2.mutualIdMem(), 0);
        vm.expectRevert(abi.encodeWithSelector(EmptyFirstRival.selector));
        store.hasRival(edge2.idMem());
    }

    function testHasRivalMore() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();
        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(
            edge1.originId, edge1.startHistoryRoot, edge1.startHeight, rand.hash(), edge1.endHeight, 0
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
        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 9, rand.hash(), 10, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 9, rand.hash(), 10, 0);

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasLengthOneRival(edge1.idMem()));
        assertFalse(store.hasLengthOneRival(edge2.idMem()));
    }

    function testSingleStepRivalNotHeight() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 8, rand.hash(), 10, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 8, rand.hash(), 10, 0);

        store.add(edge1);
        store.add(edge2);

        assertFalse(store.hasLengthOneRival(edge1.idMem()));
        assertFalse(store.hasLengthOneRival(edge2.idMem()));
    }

    function testSingleStepRival() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, 0);

        store.add(edge1);
        store.add(edge2);

        assertTrue(store.hasLengthOneRival(edge1.idMem()));
        assertTrue(store.hasLengthOneRival(edge2.idMem()));
    }

    function testSingleStepRivalNotExist() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, 0);
        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 9, rand.hash(), 10, 0);

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

    function testTimeUnrivaledFirstRivalEdgeNotExist() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2) = twoRivals();

        store.add(edge1);
        vm.roll(block.number + 3);
        store.add(edge2);

        store.remove(edge2.idMem());

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, edge2.idMem()));
        store.timeUnrivaled(edge1.idMem());
    }

    function testTimeUnrivaledFirstRivalNotExist() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        store.add(edge1);
        vm.roll(block.number + 3);

        store.setFirstRival(edge1.mutualIdMem(), 0);
        vm.expectRevert(abi.encodeWithSelector(EmptyFirstRival.selector));
        store.timeUnrivaled(edge1.idMem());
    }

    function testTimeUnrivaledNotExist() public {
        (ChallengeEdge memory edge1,) = twoRivals();

        vm.roll(block.number + 3);

        bytes32 id1 = edge1.idMem();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, id1));
        store.timeUnrivaled(id1);
    }

    function testTimeUnrivaledAfterRival() public {
        bytes32 originId = rand.hash();
        bytes32 startRoot = rand.hash();
        ChallengeEdge memory edge1 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, 0);

        store.add(edge1);
        vm.roll(block.number + 4);

        ChallengeEdge memory edge2 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, 0);

        store.add(edge2);
        vm.roll(block.number + 5);

        ChallengeEdge memory edge3 = ChallengeEdgeLib.newChildEdge(originId, startRoot, 3, rand.hash(), 9, 0);
        store.add(edge3);

        vm.roll(block.number + 6);

        ChallengeEdge memory edge4 = ChallengeEdgeLib.newChildEdge(originId, rand.hash(), 3, rand.hash(), 9, 0);
        store.add(edge4);

        vm.roll(block.number + 7);

        assertEq(store.timeUnrivaled(edge1.idMem()), 4, "Time unrivaled 1");
        assertEq(store.timeUnrivaled(edge2.idMem()), 0, "Time unrivaled 2");
        assertEq(store.timeUnrivaled(edge3.idMem()), 0, "Time unrivaled 3");
        assertEq(store.timeUnrivaled(edge4.idMem()), 7, "Time unrivaled 4");
    }

    function testMandatoryBisectHeightSizeOne() public {
        vm.expectRevert(abi.encodeWithSelector(HeightDiffLtTwo.selector, 1, 2));
        store.mandatoryBisectionHeight(1, 2);
    }

    function testMandatoryBisectHeightSizes() public {
        assertEq(store.mandatoryBisectionHeight(1, 3), 2);
        assertEq(store.mandatoryBisectionHeight(0, 4), 2);
        assertEq(store.mandatoryBisectionHeight(1, 4), 2);
        assertEq(store.mandatoryBisectionHeight(2, 4), 3);
        assertEq(store.mandatoryBisectionHeight(0, 5), 4);
        assertEq(store.mandatoryBisectionHeight(1, 5), 4);
        assertEq(store.mandatoryBisectionHeight(2, 5), 4);
        assertEq(store.mandatoryBisectionHeight(3, 5), 4);
        assertEq(store.mandatoryBisectionHeight(0, 6), 4);
        assertEq(store.mandatoryBisectionHeight(1, 6), 4);
        assertEq(store.mandatoryBisectionHeight(2, 6), 4);
        assertEq(store.mandatoryBisectionHeight(3, 6), 4);
        assertEq(store.mandatoryBisectionHeight(4, 6), 5);
        assertEq(store.mandatoryBisectionHeight(0, 7), 4);
        assertEq(store.mandatoryBisectionHeight(1, 7), 4);
        assertEq(store.mandatoryBisectionHeight(2, 7), 4);
        assertEq(store.mandatoryBisectionHeight(2, 7), 4);
        assertEq(store.mandatoryBisectionHeight(3, 7), 4);
        assertEq(store.mandatoryBisectionHeight(4, 7), 6);
        assertEq(store.mandatoryBisectionHeight(5, 7), 6);
        assertEq(store.mandatoryBisectionHeight(0, 8), 4);
        assertEq(store.mandatoryBisectionHeight(1, 8), 4);
        assertEq(store.mandatoryBisectionHeight(2, 8), 4);
        assertEq(store.mandatoryBisectionHeight(3, 8), 4);
        assertEq(store.mandatoryBisectionHeight(4, 8), 6);
        assertEq(store.mandatoryBisectionHeight(5, 8), 6);
        assertEq(store.mandatoryBisectionHeight(6, 8), 7);
        assertEq(store.mandatoryBisectionHeight(0, 9), 8);
        assertEq(store.mandatoryBisectionHeight(1, 9), 8);
        assertEq(store.mandatoryBisectionHeight(2, 9), 8);
        assertEq(store.mandatoryBisectionHeight(2, 9), 8);
        assertEq(store.mandatoryBisectionHeight(3, 9), 8);
        assertEq(store.mandatoryBisectionHeight(4, 9), 8);
        assertEq(store.mandatoryBisectionHeight(5, 9), 8);
        assertEq(store.mandatoryBisectionHeight(6, 9), 8);
        assertEq(store.mandatoryBisectionHeight(7, 9), 8);
        assertEq(store.mandatoryBisectionHeight(0, 10), 8);
        assertEq(store.mandatoryBisectionHeight(1, 10), 8);
        assertEq(store.mandatoryBisectionHeight(2, 10), 8);
        assertEq(store.mandatoryBisectionHeight(3, 10), 8);
        assertEq(store.mandatoryBisectionHeight(4, 10), 8);
        assertEq(store.mandatoryBisectionHeight(5, 10), 8);
        assertEq(store.mandatoryBisectionHeight(6, 10), 8);
        assertEq(store.mandatoryBisectionHeight(7, 10), 8);
        assertEq(store.mandatoryBisectionHeight(8, 10), 9);
        assertEq(store.mandatoryBisectionHeight(0, 11), 8);
        assertEq(store.mandatoryBisectionHeight(1, 11), 8);
        assertEq(store.mandatoryBisectionHeight(2, 11), 8);
        assertEq(store.mandatoryBisectionHeight(3, 11), 8);
        assertEq(store.mandatoryBisectionHeight(4, 11), 8);
        assertEq(store.mandatoryBisectionHeight(5, 11), 8);
        assertEq(store.mandatoryBisectionHeight(6, 11), 8);
        assertEq(store.mandatoryBisectionHeight(7, 11), 8);
        assertEq(store.mandatoryBisectionHeight(8, 11), 10);
        assertEq(store.mandatoryBisectionHeight(9, 11), 10);
        assertEq(store.mandatoryBisectionHeight(7, 73), 64);
        assertEq(store.mandatoryBisectionHeight(765273563, 10898783768364), 8796093022208);
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

        return ChallengeEdgeLib.newChildEdge(originId, startRoot, start, endRoot, end, 0);
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
            edge.originId, edge.startHistoryRoot, edge.startHeight, bisectionRoot, bisectionHeight, edge.level
        );
        ChallengeEdge memory upperChild = ChallengeEdgeLib.newChildEdge(
            edge.originId, bisectionRoot, bisectionHeight, edge.endHistoryRoot, edge.endHeight, edge.level
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
        bytes memory prefixProof = abi.encode(
            ProofUtils.expansionFromLeaves(states, 0, bisectionPoint + 1),
            ProofUtils.generatePrefixProof(
                bisectionPoint + 1, ArrayUtilsLib.slice(states, bisectionPoint + 1, states.length)
            )
        );
        (bytes32 lowerChildId, EdgeAddedData memory lowerChildAdded, EdgeAddedData memory upperChildAdded) =
            store.bisectEdge(edge.idMem(), bisectionRoot, prefixProof);
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
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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
                    edge1.level
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.level
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
                    edge2.level
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.level
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
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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
                    edge1.level
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge1.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge1.originId, bisectionRoot1, bisectionPoint, edge1.endHistoryRoot, edge1.endHeight, edge1.level
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
                    edge2.level
                )
            ).idMem(),
            "Lower child id"
        );

        assertEq(
            store.get(edge2.idMem()).upperChildId,
            (
                ChallengeEdgeLib.newChildEdge(
                    edge2.originId, bisectionRoot2, bisectionPoint, edge2.endHistoryRoot, edge2.endHeight, edge2.level
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
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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

        vm.expectRevert(abi.encodeWithSelector(EdgeAlreadyExists.selector, store.get(edgeId).upperChildId));
        store.bisectEdge(edgeId, bisectionRoot1, proof);
    }

    function testBisectInvalidProof() public {
        uint256 start = 3;
        uint256 agree = 5;
        uint256 end = 11;
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);

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
        view
        returns (uint256, bytes32, bytes memory)
    {
        uint256 bisectionPoint = store.mandatoryBisectionHeight(start, end);
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

    function testNextlevel() public {
        assertTrue(store.nextEdgeLevel(0, NUM_BIGSTEP_LEVEL) == 1);
        assertTrue(store.nextEdgeLevel(NUM_BIGSTEP_LEVEL, NUM_BIGSTEP_LEVEL) == NUM_BIGSTEP_LEVEL + 1);
        vm.expectRevert(abi.encodeWithSelector(LevelTooHigh.selector, NUM_BIGSTEP_LEVEL + 2, NUM_BIGSTEP_LEVEL));
        store.nextEdgeLevel(NUM_BIGSTEP_LEVEL + 1, NUM_BIGSTEP_LEVEL);
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
        bytes32 edge2Id;
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

        return BArgs(
            edge1.idMem(), edge2.idMem(), lowerChildId1, upperChildId1, lowerChildId2, upperChildId2, states1, states2
        );
    }

    // todo:
    // updateTimerCacheByClaim

    function testTimeUnrivaledTotalAndUpdateTimerCacheByChildren() public {
        (ChallengeEdge memory edge1, ChallengeEdge memory edge2, bytes32[] memory states1, bytes32[] memory states2) =
            twoRivalsFromLeaves(2, 5, 8);

        // create 2 two parent edges
        store.add(edge1);
        vm.roll(block.number + 2);
        edge2.createdAtBlock = uint64(block.number);
        store.add(edge2);

        // bisect first parent
        (, bytes32 bisectionRoot1, bytes memory bisectionProof1) = bisectArgs(states1, 2, 8);
        (bytes32 lowerChildId1,, EdgeAddedData memory upperChildAdded1) =
            store.bisectEdge(edge1.idMem(), bisectionRoot1, bisectionProof1);

        // roll forward a bit
        vm.roll(block.number + 20);

        // bisect second parent
        (, bytes32 bisectionRoot2, bytes memory bisectionProof2) = bisectArgs(states2, 2, 8);
        (bytes32 lowerChildId2,, EdgeAddedData memory upperChildAdded2) =
            store.bisectEdge(edge2.idMem(), bisectionRoot2, bisectionProof2);

        // roll forward a bit
        vm.roll(block.number + 200);

        // make sure we have expected time unrivaled
        assertEq(store.timeUnrivaled(edge1.idMem()), 2);
        assertEq(store.timeUnrivaled(edge2.idMem()), 0);
        assertEq(store.timeUnrivaled(lowerChildId1), 220);
        assertEq(store.timeUnrivaled(upperChildAdded1.edgeId), 20);
        assertEq(store.timeUnrivaled(lowerChildId2), 220);
        assertEq(store.timeUnrivaled(upperChildAdded2.edgeId), 0);

        // make sure caches are 0
        assertEq(store.get(edge1.idMem()).totalTimeUnrivaledCache, 0);
        assertEq(store.get(edge2.idMem()).totalTimeUnrivaledCache, 0);
        assertEq(store.get(lowerChildId1).totalTimeUnrivaledCache, 0);
        assertEq(store.get(upperChildAdded1.edgeId).totalTimeUnrivaledCache, 0);
        assertEq(store.get(lowerChildId2).totalTimeUnrivaledCache, 0);
        assertEq(store.get(upperChildAdded2.edgeId).totalTimeUnrivaledCache, 0);

        // make sure leaves just return their time unrivaled for total time unrivaled
        assertEq(store.timeUnrivaledTotal(lowerChildId1), 220);
        assertEq(store.timeUnrivaledTotal(upperChildAdded1.edgeId), 20);
        assertEq(store.timeUnrivaledTotal(lowerChildId2), 220);
        assertEq(store.timeUnrivaledTotal(upperChildAdded2.edgeId), 0);

        // make sure parents return their time unrivaled for total time unrivaled (since we haven't updated caches yet)
        assertEq(store.timeUnrivaledTotal(edge1.idMem()), 2);
        assertEq(store.timeUnrivaledTotal(edge2.idMem()), 0);

        // update the child caches, since they are leaves, this should just set the cache to the time unrivaled
        store.updateTimerCacheByChildren(lowerChildId1, 220);
        store.updateTimerCacheByChildren(upperChildAdded1.edgeId, 20);
        vm.expectRevert(abi.encodeWithSelector(CachedTimeSufficient.selector, 220, 220));
        store.updateTimerCacheByChildren(lowerChildId2, 220);
        vm.expectRevert(abi.encodeWithSelector(CachedTimeSufficient.selector, 0, 0));
        store.updateTimerCacheByChildren(upperChildAdded2.edgeId, 0);
        assertEq(store.get(lowerChildId1).totalTimeUnrivaledCache, 220);
        assertEq(store.get(upperChildAdded1.edgeId).totalTimeUnrivaledCache, 20);
        assertEq(store.get(lowerChildId2).totalTimeUnrivaledCache, 220);
        assertEq(store.get(upperChildAdded2.edgeId).totalTimeUnrivaledCache, 0);

        // time unrivaled total should now return the parent's time unrivaled plus the lower child's time unrivaled cache
        assertEq(store.timeUnrivaledTotal(edge1.idMem()), 22);
        assertEq(store.timeUnrivaledTotal(edge2.idMem()), 0);

        // updating the cache should set the cache to the time unrivaled total
        store.updateTimerCacheByChildren(edge1.idMem(), 22);
        vm.expectRevert(abi.encodeWithSelector(CachedTimeSufficient.selector, 0, 0));
        store.updateTimerCacheByChildren(edge2.idMem(), 0);
        assertEq(store.get(edge1.idMem()).totalTimeUnrivaledCache, 22);
        assertEq(store.get(edge2.idMem()).totalTimeUnrivaledCache, 0);
    }

    struct ConfirmByOneStepData {
        ChallengeEdge e1;
        ChallengeEdge e2;
        bytes32[] beforeProof;
        bytes32[] afterProof;
        bytes revertArg;
    }

    uint256 BIGSTEPHEIGHT = 1 << 4;
    uint256 SMALLSTEPHEIGHT = 1 << 6;

    function addUpLastLevel() internal returns (bytes32 originId, uint256[] memory startHeights) {
        originId = rand.hash();

        startHeights = new uint256[](NUM_BIGSTEP_LEVEL + 1);
        for (uint256 i = 0; i < NUM_BIGSTEP_LEVEL + 1; i++) {
            uint256 startHeight = rand.unsignedInt(BIGSTEPHEIGHT);
            ChallengeEdge memory e1 = ChallengeEdgeLib.newChildEdge(
                originId, rand.hash(), startHeight, rand.hash(), startHeight + 1, uint8(i)
            );
            store.add(e1);
            ChallengeEdge memory e2 = ChallengeEdgeLib.newChildEdge(
                originId, e1.startHistoryRoot, startHeight, rand.hash(), startHeight + 1, uint8(i)
            );
            store.add(e2);

            originId = e1.mutualIdMem();
            startHeights[i] = startHeight;
        }
    }

    function getLayerZeroStepSize(
        uint256 numBigStepLevel,
        uint256 bigStepHeight,
        uint256 smallStepHeight,
        uint256 level
    ) internal returns (uint256) {
        uint256 stepSize = 1;
        uint256 maxLevelIndex = numBigStepLevel + 1;
        for (uint256 i = level; i < maxLevelIndex; i++) {
            if (i == maxLevelIndex - 1) {
                stepSize *= smallStepHeight;
            } else {
                stepSize *= bigStepHeight;
            }
        }
        return stepSize;
    }

    function confirmByOneStep(uint256 flag) internal {
        uint256 startHeight = rand.unsignedInt(SMALLSTEPHEIGHT);
        (bytes32[] memory states1, bytes32[] memory states2) = rivalStates(startHeight, startHeight, startHeight + 1);

        (bytes32 originId, uint256[] memory startHeights) = addUpLastLevel();
        ConfirmByOneStepData memory data;
        data.e1 = ChallengeEdgeLib.newChildEdge(
            originId,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, startHeight + 1)),
            startHeight,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states1, 0, startHeight + 2)),
            startHeight + 1,
            NUM_BIGSTEP_LEVEL + 1
        );

        data.e2 = ChallengeEdgeLib.newChildEdge(
            originId,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, startHeight + 1)),
            startHeight,
            MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states2, 0, startHeight + 2)),
            startHeight + 1,
            NUM_BIGSTEP_LEVEL + 1
        );
        if (flag == 3) {
            data.e1.level = NUM_BIGSTEP_LEVEL;
            data.revertArg = abi.encodeWithSelector(EdgeTypeNotSmallStep.selector, data.e1.level);
        }
        if (flag == 5) {
            data.e1.endHeight = data.e1.endHeight + 1;
            data.revertArg = abi.encodeWithSelector(EdgeNotLengthOne.selector, 2);
        }
        bytes32 eid = data.e1.idMem();
        if (flag == 2) {
            data.e1.status = EdgeStatus.Confirmed;
            data.revertArg = abi.encodeWithSelector(EdgeNotPending.selector, eid, data.e1.status);
        }

        uint256 expectedStartMachineStep = startHeight;
        {
            for (uint256 i = 1; i < startHeights.length; i++) {
                expectedStartMachineStep +=
                    getLayerZeroStepSize(NUM_BIGSTEP_LEVEL, BIGSTEPHEIGHT, SMALLSTEPHEIGHT, i) * startHeights[i];
            }
        }

        if (flag != 1) {
            store.add(data.e1);
        } else {
            data.revertArg = abi.encodeWithSelector(EdgeNotExists.selector, eid);
        }
        if (flag != 4) {
            store.add(data.e2);
        }
        OneStepData memory d =
            OneStepData({beforeHash: states1[startHeight], proof: abi.encodePacked(states1[startHeight + 1])});
        ExecutionContext memory e =
            ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0)), initialWasmModuleRoot: bytes32(0)});
        data.beforeProof = ProofUtils.generateInclusionProof(
            ProofUtils.rehashed(ArrayUtilsLib.slice(states1, 0, startHeight + 1)), startHeight
        );
        if (flag == 6) {
            data.beforeProof[0] = rand.hash();
            data.revertArg = "Invalid inclusion proof";
        }
        data.afterProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), startHeight + 1);
        if (flag == 7) {
            data.afterProof[0] = rand.hash();
            data.revertArg = "Invalid inclusion proof";
        }
        if (flag == 8) {
            d.proof = abi.encodePacked(rand.hash());
            data.revertArg = "Invalid inclusion proof";
        }
        if (flag == 9) {
            store.setConfirmed(data.e2.idMem());
            store.setConfirmedRival(data.e2.idMem());
            data.revertArg = abi.encodeWithSelector(RivalEdgeConfirmed.selector, data.e1.idMem(), data.e2.idMem());
        }

        MockOneStepProofEntry entry = new MockOneStepProofEntry(expectedStartMachineStep);

        if (data.revertArg.length != 0) {
            vm.expectRevert(data.revertArg);
        }
        store.confirmEdgeByOneStepProof(
            eid, entry, d, e, data.beforeProof, data.afterProof, NUM_BIGSTEP_LEVEL, 1 << 4, 1 << 6
        );

        if (bytes(data.revertArg).length != 0) {
            // for flag one the edge does not exist
            // for flag two we set the status to confirmed anyway
            if (flag != 1 && flag != 2) {
                assertTrue(store.get(eid).status == EdgeStatus.Pending, "Edge pending");
            }
        } else {
            assertTrue(store.get(eid).status == EdgeStatus.Confirmed, "Edge confirmed");
            assertEq(store.getConfirmedRival(ChallengeEdgeLib.mutualIdMem(data.e1)), eid, "Confirmed rival");
        }
    }

    error MachineStep(uint256 actual, uint256 expected, uint256[] startHeights, uint256 smallStartHeight);

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

    function testConfirmByOneStepRivalConfirmed() public {
        confirmByOneStep(9);
    }

    function testPowerOfTwo() public {
        assertEq(store.isPowerOfTwo(0), false);
        assertEq(store.isPowerOfTwo(1), true);
        assertEq(store.isPowerOfTwo(2), true);
        assertEq(store.isPowerOfTwo(3), false);
        assertEq(store.isPowerOfTwo(4), true);
        assertEq(store.isPowerOfTwo(5), false);
        assertEq(store.isPowerOfTwo(6), false);
        assertEq(store.isPowerOfTwo(7), false);
        assertEq(store.isPowerOfTwo(8), true);
        assertEq(store.isPowerOfTwo(2 ** 17), true);
        assertEq(store.isPowerOfTwo(1 << 255), true);
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
        AssertionState assertionState;
        bytes32 machineHash;
    }

    function randomAssertionState(IOneStepProofEntry os) private returns (ExecStateVars memory) {
        AssertionState memory assertionState = AssertionState(
            GlobalState([rand.hash(), rand.hash()], [uint64(uint256(rand.hash())), uint64(uint256(rand.hash()))]),
            MachineStatus.FINISHED,
            bytes32(0)
        );

        bytes32 machineHash = os.getMachineHash(assertionState.toExecutionState());
        return ExecStateVars(assertionState, machineHash);
    }

    function createZeroBlockEdge(uint256 mode) internal returns (EdgeAddedData memory) {
        return createZeroBlockEdge(mode, "");
    }

    function createZeroBlockEdge(uint256 mode, bytes memory extraData) internal returns (EdgeAddedData memory) {
        bytes memory revertArg;
        MockOneStepProofEntry entry = new MockOneStepProofEntry(0);
        uint256 expectedEndHeight = 2 ** 2;
        if (mode == 139) {
            expectedEndHeight = 2 ** 5 - 1;
            revertArg = abi.encodeWithSelector(NotPowerOfTwo.selector, expectedEndHeight);
        }

        bool whitelistEnabled = mode == 150 || mode == 151;

        if (mode == 151) {
            bytes32 expectedMutualId = abi.decode(extraData, (bytes32));
            revertArg = abi.encodeWithSelector(AccountHasMadeLayerZeroRival.selector, address(this), expectedMutualId);
        }

        ExecStateVars memory startExec = randomAssertionState(entry);
        ExecStateVars memory endExec = randomAssertionState(entry);
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
                assertionHash: claimId,
                predecessorId: rand.hash(),
                isPending: true,
                hasSibling: true,
                startState: startExec.assertionState,
                endState: endExec.assertionState
            });
            if (mode == 141) {
                ard.assertionHash = rand.hash();
                revertArg = abi.encodeWithSelector(AssertionHashMismatch.selector, ard.assertionHash, claimId);
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
            revertArg = abi.encodeWithSelector(AssertionHashEmpty.selector);
        }

        if (mode == 145) {
            AssertionState memory s;
            ard.startState = s;
            revertArg = abi.encodeWithSelector(EmptyStartMachineStatus.selector);
        }
        if (mode == 146) {
            AssertionState memory e;
            ard.endState = e;
            revertArg = abi.encodeWithSelector(EmptyEndMachineStatus.selector);
        }
        CreateEdgeArgs memory args;
        {
            bytes memory proof = abi.encode(
                ProofUtils.generateInclusionProof(ProofUtils.rehashed(roots.states), expectedEndHeight),
                AssertionStateData(ard.startState, bytes32(0), bytes32(0)),
                AssertionStateData(ard.endState, bytes32(0), bytes32(0))
            );
            if (mode == 147) {
                proof = "";
                revertArg = abi.encodeWithSelector(EmptyEdgeSpecificProof.selector);
            }

            args = CreateEdgeArgs({
                level: 0,
                endHistoryRoot: endRoot,
                endHeight: expectedEndHeight,
                claimId: claimId,
                prefixProof: abi.encode(roots.startExp, roots.prefixProof),
                proof: proof
            });
        }
        if (mode == 138) {
            args.endHeight = 2 ** 4;
            revertArg = abi.encodeWithSelector(InvalidEndHeight.selector, 2 ** 4, expectedEndHeight);
        }
        if (mode == 148) {
            args.prefixProof = "";
            revertArg = abi.encodeWithSelector(EmptyPrefixProof.selector);
        }

        if (revertArg.length != 0) {
            vm.expectRevert(revertArg);
        }
        EdgeAddedData memory addedEdge =
            store.createLayerZeroEdge(args, ard, entry, expectedEndHeight, NUM_BIGSTEP_LEVEL, whitelistEnabled);
        if (revertArg.length == 0) {
            assertEq(
                store.get(addedEdge.edgeId).startHistoryRoot,
                MerkleTreeLib.root(
                    MerkleTreeLib.appendLeaf(
                        new bytes32[](0), mockOsp.getMachineHash(startExec.assertionState.toExecutionState())
                    )
                ),
                "Start history root"
            );
        }

        return addedEdge;
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

    function testCreateLayerZeroEdgeEmptyEdgeSpecificProof() public {
        createZeroBlockEdge(147);
    }

    function testCreateLayerZeroEdgeEmptyPrefixProof() public {
        createZeroBlockEdge(148);
    }

    function testPerAccountRivalRestriction() public {
        uint256 snapshot = vm.snapshot();
        EdgeAddedData memory edgeAdded = createZeroBlockEdge(150);
        assertTrue(store.hasMadeLayerZeroRival(address(this), edgeAdded.mutualId));
        vm.revertTo(snapshot);

        store.setHasMadeLayerZeroRival(address(this), edgeAdded.mutualId, true);
        createZeroBlockEdge(151, abi.encode(edgeAdded.mutualId));
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
            NUM_BIGSTEP_LEVEL
        );
        c.add(ce);
        // and give it a rival
        if (includeRival) {
            c.add(
                ChallengeEdgeLib.newChildEdge(
                    ce.originId, ce.startHistoryRoot, ce.startHeight, rand.hash(), ce.endHeight, ce.level
                )
            );
        }

        return (ce.idMem(), claimRoots);
    }

    struct CreateSmallStepEdgeData {
        bytes revertArg;
        uint256 claimStartHeight;
        uint256 claimEndHeight;
        uint256 expectedEndHeight;
        bytes32 claimId;
        ExpsAndProofs claimRoots;
        ExpsAndProofs roots;
        bytes proof;
        MockOneStepProofEntry a;
        AssertionReferenceData emptyArd;
    }

    function createSmallStepEdge(uint256 mode) internal {
        CreateSmallStepEdgeData memory vars;

        vars.claimStartHeight = 4;
        vars.claimEndHeight = mode == 161 ? 6 : 5;

        vars.expectedEndHeight = 2 ** 5;
        (vars.claimId, vars.claimRoots) =
            createClaimEdge(store, vars.claimStartHeight, vars.claimEndHeight, mode == 160 ? false : true);
        if (mode == 160) {
            vars.revertArg = abi.encodeWithSelector(ClaimEdgeNotLengthOneRival.selector, vars.claimId);
        }
        if (mode == 161) {
            vars.revertArg = abi.encodeWithSelector(ClaimEdgeNotLengthOneRival.selector, vars.claimId);
        }

        vars.roots = newRootsAndProofs(
            0,
            vars.expectedEndHeight,
            vars.claimRoots.states[vars.claimStartHeight],
            vars.claimRoots.states[vars.claimEndHeight]
        );
        if (mode == 164) {
            bytes32[] memory b = new bytes32[](1);
            b[0] = rand.hash();
            vars.claimRoots.startInclusionProof = ArrayUtilsLib.concat(vars.claimRoots.startInclusionProof, b);
            vars.revertArg = "Invalid inclusion proof";
        }
        if (mode == 165) {
            bytes32[] memory b = new bytes32[](1);
            b[0] = rand.hash();
            vars.claimRoots.endInclusionProof = ArrayUtilsLib.concat(vars.claimRoots.endInclusionProof, b);
            vars.revertArg = "Invalid inclusion proof";
        }
        vars.proof = abi.encode(
            vars.roots.states[0],
            vars.roots.states[vars.expectedEndHeight],
            vars.claimRoots.startInclusionProof,
            vars.claimRoots.endInclusionProof,
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(vars.roots.states), vars.expectedEndHeight)
        );
        if (mode == 166) {
            vars.proof = "";
            vars.revertArg = abi.encodeWithSelector(EmptyEdgeSpecificProof.selector);
        }
        if (mode == 162) {
            store.setConfirmed(vars.claimId);
            vars.revertArg = abi.encodeWithSelector(ClaimEdgeNotPending.selector);
        }

        vars.a = new MockOneStepProofEntry(0);
        vars.emptyArd;

        if (mode == 163) {
            vars.revertArg =
                abi.encodeWithSelector(ClaimEdgeInvalidLevel.selector, NUM_BIGSTEP_LEVEL, NUM_BIGSTEP_LEVEL);
        }
        if (vars.revertArg.length != 0) {
            vm.expectRevert(vars.revertArg);
        }
        store.createLayerZeroEdge(
            CreateEdgeArgs({
                level: mode == 163 ? NUM_BIGSTEP_LEVEL : NUM_BIGSTEP_LEVEL + 1,
                endHistoryRoot: MerkleTreeLib.root(vars.roots.endExp),
                endHeight: vars.expectedEndHeight,
                claimId: vars.claimId,
                prefixProof: abi.encode(vars.roots.startExp, vars.roots.prefixProof),
                proof: vars.proof
            }),
            vars.emptyArd,
            vars.a,
            vars.expectedEndHeight,
            NUM_BIGSTEP_LEVEL,
            false
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

    function testCreateLayerZeroEdgeSmallSteplevel() public {
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

    function testCreateLayerZeroEdgeSmallStepEmptySpecificProof() public {
        createSmallStepEdge(166);
    }

    struct CreateBlockEdgesBisectArgs {
        bytes32 claim1Id;
        bytes32 claim2Id;
        AssertionState endState1;
        AssertionState endState2;
        bool skipLast;
    }

    bytes32 genesisBlockHash = rand.hash();
    AssertionState genesisState = StateToolsLib.randomState(rand, 4, genesisBlockHash, MachineStatus.FINISHED);
    bytes32 genesisStateHash = StateToolsLib.mockMachineHash(genesisState);
    AssertionStateData genesisStateData = AssertionStateData(genesisState, bytes32(0), bytes32(0));
    bytes32 genesisAssertionHash = rand.hash();
    uint256 height1 = 32;

    function genesisStates() internal view returns (bytes32[] memory) {
        bytes32[] memory genStates = new bytes32[](1);
        genStates[0] = genesisStateHash;
        return genStates;
    }

    function createLayerZeroEdge(
        bytes32 claimId,
        AssertionState memory endState,
        bytes32[] memory states,
        bytes32[] memory exp,
        AssertionReferenceData memory ard,
        uint256 expectedEndHeight,
        uint8 numBigStepLevel
    ) internal returns (bytes32) {
        bytes memory typeSpecificProof1 = abi.encode(
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1),
            genesisStateData,
            AssertionStateData(endState, genesisAssertionHash, bytes32(0))
        );
        bytes memory prefixProof = abi.encode(
            ProofUtils.expansionFromLeaves(states, 0, 1),
            ProofUtils.generatePrefixProof(1, ArrayUtilsLib.slice(states, 1, states.length))
        );

        return store.createLayerZeroEdge(
            CreateEdgeArgs({
                level: 0,
                endHistoryRoot: MerkleTreeLib.root(exp),
                endHeight: height1,
                claimId: claimId,
                prefixProof: prefixProof,
                proof: typeSpecificProof1
            }),
            ard,
            mockOsp,
            expectedEndHeight,
            numBigStepLevel,
            false
        ).edgeId;
    }

    struct BisectionChildren {
        bytes32 lowerChildId;
        bytes32 upperChildId;
    }

    struct BisectToForkOnlyArgs {
        bytes32 winningId;
        bytes32 losingId;
        bytes32[] winningLeaves;
        bytes32[] losingLeaves;
        bool skipLast;
    }

    function bisect(bytes32 edgeId, bytes32[] memory states, uint256 bisectionSize, uint256 endSize)
        internal
        returns (BisectionChildren memory)
    {
        bytes32[] memory middleExp = ProofUtils.expansionFromLeaves(states, 0, bisectionSize + 1);
        bytes32[] memory upperStates = ArrayUtilsLib.slice(states, bisectionSize + 1, endSize + 1);

        (bytes32 lowerChildId,, EdgeAddedData memory upperChild) = store.bisectEdge(
            edgeId,
            MerkleTreeLib.root(middleExp),
            abi.encode(middleExp, ProofUtils.generatePrefixProof(bisectionSize + 1, upperStates))
        );

        return BisectionChildren(lowerChildId, upperChild.edgeId);
    }

    function bisectToForkOnly(BisectToForkOnlyArgs memory args)
        internal
        returns (BisectionChildren[6] memory, BisectionChildren[6] memory)
    {
        BisectionChildren[6] memory winningEdges;
        BisectionChildren[6] memory losingEdges;

        winningEdges[5] = BisectionChildren(args.winningId, 0);
        losingEdges[5] = BisectionChildren(args.losingId, 0);

        // height 16
        winningEdges[4] = bisect(winningEdges[5].lowerChildId, args.winningLeaves, 16, args.winningLeaves.length - 1);
        losingEdges[4] = bisect(losingEdges[5].lowerChildId, args.losingLeaves, 16, args.losingLeaves.length - 1);

        // height 8
        winningEdges[3] = bisect(winningEdges[4].lowerChildId, args.winningLeaves, 8, 16);
        losingEdges[3] = bisect(losingEdges[4].lowerChildId, args.losingLeaves, 8, 16);

        // height 4
        winningEdges[2] = bisect(winningEdges[3].lowerChildId, args.winningLeaves, 4, 8);
        losingEdges[2] = bisect(losingEdges[3].lowerChildId, args.losingLeaves, 4, 8);

        winningEdges[1] = bisect(winningEdges[2].lowerChildId, args.winningLeaves, 2, 4);
        losingEdges[1] = bisect(losingEdges[2].lowerChildId, args.losingLeaves, 2, 4);

        // height 2
        winningEdges[0] = bisect(winningEdges[1].lowerChildId, args.winningLeaves, 1, 2);
        if (!args.skipLast) {
            losingEdges[0] = bisect(losingEdges[1].lowerChildId, args.losingLeaves, 1, 2);
        }

        return (winningEdges, losingEdges);
    }

    function createBlockEdgesAndBisectToFork(CreateBlockEdgesBisectArgs memory args)
        internal
        returns (bytes32[] memory, bytes32[] memory, BisectionChildren[6] memory, BisectionChildren[6] memory)
    {
        bytes32[] memory states1;
        bytes32 edge1Id;
        {
            bytes32[] memory exp1;
            (states1, exp1) =
                appendRandomStatesBetween(genesisStates(), StateToolsLib.mockMachineHash(args.endState1), height1);

            edge1Id = createLayerZeroEdge(
                args.claim1Id,
                args.endState1,
                states1,
                exp1,
                AssertionReferenceData(args.claim1Id, genesisAssertionHash, true, true, genesisState, args.endState1),
                32,
                1
            );

            vm.roll(block.number + 1);

            assertEq(store.timeUnrivaled(edge1Id), 1, "Edge1 timer");
        }

        bytes32[] memory states2;
        bytes32 edge2Id;
        {
            bytes32[] memory exp2;
            (states2, exp2) =
                appendRandomStatesBetween(genesisStates(), StateToolsLib.mockMachineHash(args.endState2), height1);
            AssertionReferenceData memory ard2 =
                AssertionReferenceData(args.claim2Id, genesisAssertionHash, true, true, genesisState, args.endState2);
            edge2Id = createLayerZeroEdge(args.claim2Id, args.endState2, states2, exp2, ard2, 32, 1);

            vm.roll(block.number + 2);

            assertEq(store.timeUnrivaled(edge1Id), 1, "Edge1 timer 2");
            assertEq(store.timeUnrivaled(edge2Id), 0, "Edge2 timer 2");
        }

        (BisectionChildren[6] memory edges1, BisectionChildren[6] memory edges2) =
            bisectToForkOnly(BisectToForkOnlyArgs(edge1Id, edge2Id, states1, states2, args.skipLast));

        return (states1, states2, edges1, edges2);
    }

    struct CreateMachineEdgesBisectArgs {
        uint8 eType;
        bytes32 claim1Id;
        bytes32 claim2Id;
        bytes32 endState1;
        bytes32 endState2;
        bool skipLast;
        bytes32[] forkStates1;
        bytes32[] forkStates2;
    }

    struct BisectionData {
        bytes32[] states1;
        bytes32[] states2;
        BisectionChildren[6] edges1;
        BisectionChildren[6] edges2;
    }

    AssertionReferenceData emptyArd;

    function createMachineEdgesAndBisectToFork(CreateMachineEdgesBisectArgs memory args)
        internal
        returns (BisectionData memory)
    {
        (bytes32[] memory states1, bytes32[] memory exp1) =
            appendRandomStatesBetween(genesisStates(), args.endState1, height1);
        bytes32 edge1Id;
        {
            bytes memory typeSpecificProof1;
            {
                bytes32[] memory claimStartInclusionProof = ProofUtils.generateInclusionProof(
                    ProofUtils.rehashed(ArrayUtilsLib.slice(args.forkStates1, 0, 1)), 0
                );
                bytes32[] memory claimEndInclusionProof =
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(args.forkStates1), 1);
                bytes32[] memory edgeInclusionProof =
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), states1.length - 1);
                typeSpecificProof1 = abi.encode(
                    genesisStateHash,
                    args.endState1,
                    claimStartInclusionProof,
                    claimEndInclusionProof,
                    edgeInclusionProof
                );
            }
            edge1Id = store.createLayerZeroEdge(
                CreateEdgeArgs({
                    level: args.eType,
                    endHistoryRoot: MerkleTreeLib.root(exp1),
                    endHeight: height1,
                    claimId: args.claim1Id,
                    prefixProof: abi.encode(
                        ProofUtils.expansionFromLeaves(states1, 0, 1),
                        ProofUtils.generatePrefixProof(1, ArrayUtilsLib.slice(states1, 1, states1.length))
                    ),
                    proof: typeSpecificProof1
                }),
                emptyArd,
                mockOsp,
                32,
                1,
                false
            ).edgeId;
        }

        vm.roll(block.number + 1);

        assertEq(store.timeUnrivaled(edge1Id), 1, "Edge1 timer");

        (bytes32[] memory states2, bytes32[] memory exp2) =
            appendRandomStatesBetween(genesisStates(), args.endState2, height1);
        bytes32 edge2Id;
        {
            bytes memory typeSpecificProof2;
            {
                bytes32[] memory claimStartInclusionProof = ProofUtils.generateInclusionProof(
                    ProofUtils.rehashed(ArrayUtilsLib.slice(args.forkStates2, 0, 1)), 0
                );
                bytes32[] memory claimEndInclusionProof =
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(args.forkStates2), 1);
                bytes32[] memory edgeInclusionProof =
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(states2), states2.length - 1);
                typeSpecificProof2 = abi.encode(
                    genesisStateHash,
                    args.endState2,
                    claimStartInclusionProof,
                    claimEndInclusionProof,
                    edgeInclusionProof
                );
            }
            edge2Id = store.createLayerZeroEdge(
                CreateEdgeArgs({
                    level: args.eType,
                    endHistoryRoot: MerkleTreeLib.root(exp2),
                    endHeight: height1,
                    claimId: args.claim2Id,
                    prefixProof: abi.encode(
                        ProofUtils.expansionFromLeaves(states2, 0, 1),
                        ProofUtils.generatePrefixProof(1, ArrayUtilsLib.slice(states2, 1, states2.length))
                    ),
                    proof: typeSpecificProof2
                }),
                emptyArd,
                mockOsp,
                32,
                1,
                false
            ).edgeId;
        }

        vm.roll(block.number + 2);

        (BisectionChildren[6] memory edges1, BisectionChildren[6] memory edges2) =
            bisectToForkOnly(BisectToForkOnlyArgs(edge1Id, edge2Id, states1, states2, args.skipLast));

        return BisectionData(states1, states2, edges1, edges2);
    }

    function testGetPrevAssertionHashCorrectly() public {
        bytes32 a1 = rand.hash();
        bytes32 a2 = rand.hash();
        bytes32 h1 = rand.hash();
        bytes32 h2 = rand.hash();
        AssertionState memory a1State = StateToolsLib.randomState(
            rand, GlobalStateLib.getInboxPosition(genesisState.globalState), h1, MachineStatus.FINISHED
        );
        AssertionState memory a2State = StateToolsLib.randomState(
            rand, GlobalStateLib.getInboxPosition(genesisState.globalState), h2, MachineStatus.FINISHED
        );

        (
            bytes32[] memory blockStates1,
            bytes32[] memory blockStates2,
            BisectionChildren[6] memory blockEdges1,
            BisectionChildren[6] memory blockEdges2
        ) = createBlockEdgesAndBisectToFork(CreateBlockEdgesBisectArgs(a1, a2, a1State, a2State, false));

        BisectionData memory bsbd = createMachineEdgesAndBisectToFork(
            CreateMachineEdgesBisectArgs(
                1,
                blockEdges1[0].lowerChildId,
                blockEdges2[0].lowerChildId,
                blockStates1[1],
                blockStates2[1],
                false,
                ArrayUtilsLib.slice(blockStates1, 0, 2),
                ArrayUtilsLib.slice(blockStates2, 0, 2)
            )
        );

        BisectionData memory ssbd = createMachineEdgesAndBisectToFork(
            CreateMachineEdgesBisectArgs(
                2,
                bsbd.edges1[0].lowerChildId,
                bsbd.edges2[0].lowerChildId,
                bsbd.states1[1],
                bsbd.states2[1],
                false,
                ArrayUtilsLib.slice(bsbd.states1, 0, 2),
                ArrayUtilsLib.slice(bsbd.states2, 0, 2)
            )
        );

        assertEq(
            store.getPrevAssertionHash(blockEdges1[5].lowerChildId), genesisAssertionHash, "Block level winning edge"
        );
        assertEq(
            store.getPrevAssertionHash(blockEdges2[5].lowerChildId), genesisAssertionHash, "Block level losing edge"
        );

        for (uint256 x = 0; x < 5; x++) {
            assertEq(
                store.getPrevAssertionHash(blockEdges1[x].lowerChildId),
                genesisAssertionHash,
                "Block level winning edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(blockEdges2[x].lowerChildId),
                genesisAssertionHash,
                "Block level losing edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(blockEdges1[x].upperChildId),
                genesisAssertionHash,
                "Block level winning edge upper"
            );
            assertEq(
                store.getPrevAssertionHash(blockEdges2[x].upperChildId),
                genesisAssertionHash,
                "Block level losing edge upper"
            );
        }

        assertEq(
            store.getPrevAssertionHash(bsbd.edges1[5].lowerChildId), genesisAssertionHash, "Block level winning edge"
        );
        assertEq(
            store.getPrevAssertionHash(bsbd.edges2[5].lowerChildId), genesisAssertionHash, "Block level losing edge"
        );

        for (uint256 x = 0; x < 5; x++) {
            assertEq(
                store.getPrevAssertionHash(bsbd.edges1[x].lowerChildId),
                genesisAssertionHash,
                "Block level winning edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(bsbd.edges2[x].lowerChildId),
                genesisAssertionHash,
                "Block level losing edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(bsbd.edges1[x].upperChildId),
                genesisAssertionHash,
                "Block level winning edge upper"
            );
            assertEq(
                store.getPrevAssertionHash(bsbd.edges2[x].upperChildId),
                genesisAssertionHash,
                "Block level losing edge upper"
            );
        }

        assertEq(
            store.getPrevAssertionHash(ssbd.edges1[5].lowerChildId), genesisAssertionHash, "Block level winning edge"
        );
        assertEq(
            store.getPrevAssertionHash(ssbd.edges2[5].lowerChildId), genesisAssertionHash, "Block level losing edge"
        );

        for (uint256 x = 0; x < 5; x++) {
            assertEq(
                store.getPrevAssertionHash(ssbd.edges1[x].lowerChildId),
                genesisAssertionHash,
                "Block level winning edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(ssbd.edges2[x].lowerChildId),
                genesisAssertionHash,
                "Block level losing edge lower"
            );
            assertEq(
                store.getPrevAssertionHash(ssbd.edges1[x].upperChildId),
                genesisAssertionHash,
                "Block level winning edge upper"
            );
            assertEq(
                store.getPrevAssertionHash(ssbd.edges2[x].upperChildId),
                genesisAssertionHash,
                "Block level losing edge upper"
            );
        }
    }
}
