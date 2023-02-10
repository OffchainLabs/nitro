// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../../src/challengeV2/libraries/PsVerticesLib.sol";

import "forge-std/Test.sol";
import "./Utils.sol";
// CHRIS: TODO: should we have an addroot func on the ps vertices

contract PsVerticesLibTest is Test {
    Random rand = new Random();
    uint256 challengePeriodSec = 1000;

    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeVertexLib for ChallengeVertex;

    mapping(bytes32 => ChallengeVertex) vertices;

    function createPredecessorVertex() internal returns (bytes32, bytes32, uint256) {
        bytes32 challengeId = rand.hash();
        uint256 height = 10;

        ChallengeVertex memory v0 = ChallengeVertexLib.newVertex(challengeId, rand.hash(), height, 0);
        bytes32 v0Id = v0.id();
        vertices[v0Id] = v0;

        return (challengeId, v0Id, height);
    }

    // checkAtOneStepFork

    function testCheckAtOneStepFork() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        vertices.checkAtOneStepFork(vId);
    }

    function testCheckAtOneStepForkManySuccessors() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        ChallengeVertex memory v3 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v3, vId, challengePeriodSec);

        vertices.checkAtOneStepFork(vId);
    }

    function testCheckAtOneStepForkMultiHeight() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        ChallengeVertex memory v3 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v3, vId, challengePeriodSec);

        ChallengeVertex memory v4 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v4, vId, challengePeriodSec);

        ChallengeVertex memory v5 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 3, 0);
        vertices.addVertex(v5, vId, challengePeriodSec);

        vertices.checkAtOneStepFork(vId);
    }

    function testOneStepFailsOnePs() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.expectRevert("Has presumptive successor");
        vertices.checkAtOneStepFork(vId);
    }

    function testOneStepFailsDifferentHeights() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        vm.expectRevert("Has presumptive successor");
        vertices.checkAtOneStepFork(vId);
    }

    function testOneStepFailsWrongHeight() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        vm.expectRevert("Lowest height not one above the current height");
        vertices.checkAtOneStepFork(vId);
    }

    function testOneStepFailsNotExists() public {
        bytes32 challengeId = rand.hash();
        uint256 height = 10;

        ChallengeVertex memory v0 = ChallengeVertexLib.newVertex(challengeId, rand.hash(), height, 0);

        vm.expectRevert("Fork candidate vertex does not exist");
        vertices.checkAtOneStepFork(v0.id());
    }

    function testOneStepFailsNoSuccessors() public {
        (, bytes32 vId,) = createPredecessorVertex();

        vm.expectRevert("No successors");
        vertices.checkAtOneStepFork(vId);
    }

    function testCheckAtOneStepForkFailsLeaf() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 =
            ChallengeVertexLib.newLeaf(cId, rand.hash(), height + 1, rand.hash(), rand.addr(), 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.expectRevert("Leaf can never be a fork candidate");
        vertices.checkAtOneStepFork(v1.id());
    }

    // psExceedsChallengePeriod

    function testPsExceedsChallengePeriodTrue() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 4, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec + 1);

        assertTrue(vertices.psExceedsChallengePeriod(vId, challengePeriodSec), "Ps did not exceed");
    }

    function testPsExceedsChallengePeriodFalse() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 4, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        // skip forward to the boundary but dont cross it
        // CHRIS: TODO: could fuzz on this?
        vm.warp(block.timestamp + challengePeriodSec);

        assertFalse(vertices.psExceedsChallengePeriod(vId, challengePeriodSec), "Ps did exceed");
    }

    function testPsExceedsChallengePeriodFalseNoPs() public {
        (, bytes32 vId,) = createPredecessorVertex();

        vm.warp(block.timestamp + challengePeriodSec + 1);

        assertFalse(vertices.psExceedsChallengePeriod(vId, challengePeriodSec), "Ps did exceed");
    }

    function testPsExceedsChallengePeriodFailNotExists() public {
        bytes32 challengeId = rand.hash();
        uint256 height = 10;

        ChallengeVertex memory v0 = ChallengeVertexLib.newVertex(challengeId, rand.hash(), height, 0);

        vm.expectRevert("Predecessor vertex does not exist");
        vertices.psExceedsChallengePeriod(v0.id(), challengePeriodSec);
    }

    function testPsExceedsChallengePeriodFalseSameHeight() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec + 1);

        assertFalse(vertices.psExceedsChallengePeriod(vId, challengePeriodSec), "Ps did exceed");
    }

    function testPsExceedsChallengePeriodFailRoot() public {
        ChallengeVertex memory v0 = ChallengeVertexLib.newRoot(rand.hash(), rand.hash(), rand.hash());
        bytes32 v0Id = v0.id();
        vertices[v0Id] = v0;

        vm.expectRevert("Root has no ps timer");
        vertices.psExceedsChallengePeriod(v0Id, challengePeriodSec);
    }

    // getCurrentPsTimer

    function testGetCurrentPsTimer() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        bytes32 v1Id = v1.id();
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        assertEq(vertices.getCurrentPsTimer(v1Id), challengePeriodSec / 2);
    }

    function testGetCurrentPsTimerNotPsNow() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        // add a new ps
        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 1, 0);
        vertices.addVertex(v2, vId, challengePeriodSec);

        assertEq(vertices.getCurrentPsTimer(v1.id()), challengePeriodSec / 2);
    }

    function testGetCurrentPsTimerFlushed() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        bytes32 v1Id = v1.id();
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.flushPs(vId, 0);
        assertEq(vertices.getCurrentPsTimer(v1Id), 2 * challengePeriodSec / 4);

        vm.warp(block.timestamp + challengePeriodSec / 4);

        assertEq(vertices.getCurrentPsTimer(v1Id), 3 * challengePeriodSec / 4);
    }

    function testGetCurrentPsTimerFailsNoPredecessor() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Predecessor vertex does not exist");
        assertEq(vertices.getCurrentPsTimer(vId), challengePeriodSec / 2);
    }

    function testGetCurrentPsTimerFailsNotExist() public {
        (bytes32 cId,, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Vertex does not exist for ps timer");
        assertEq(vertices.getCurrentPsTimer(v1.id()), challengePeriodSec / 2);
    }

    // flushPs

    function testFlushPs() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        bytes32 v1Id = v1.id();
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        assertEq(vertices[vId].psId, v1Id, "Invalid last ps id before");
        assertEq(vertices[vId].lowestHeightSuccessorId, v1Id, "Invalid lowest height id before");

        vertices.flushPs(vId, 0);

        assertEq(vertices[vId].psLastUpdatedTimestamp, block.timestamp, "Invalid last updated time");
        assertEq(vertices[v1Id].flushedPsTimeSec, challengePeriodSec / 2, "Invalid flushed ps time");
        assertEq(vertices[vId].psId, v1Id, "Invalid last ps id after");
        assertEq(vertices[vId].lowestHeightSuccessorId, v1Id, "Invalid lowest height id after");
    }

    function testFlushPsNoPs() public {
        (, bytes32 vId,) = createPredecessorVertex();

        vm.warp(block.timestamp + challengePeriodSec / 2);

        assertEq(vertices[vId].psId, 0, "Invalid last ps id before");
        assertEq(vertices[vId].lowestHeightSuccessorId, 0, "Invalid lowest height id before");

        vertices.flushPs(vId, 0);

        assertEq(vertices[vId].psLastUpdatedTimestamp, block.timestamp, "Invalid last updated time");
        assertEq(vertices[vId].psId, 0, "Invalid last ps id after");
        assertEq(vertices[vId].lowestHeightSuccessorId, 0, "Invalid lowest height id after");
    }

    function testFlushPsWithMin() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        bytes32 v1Id = v1.id();
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        assertEq(vertices[vId].psId, v1Id, "Invalid last ps id before");
        assertEq(vertices[vId].lowestHeightSuccessorId, v1Id, "Invalid lowest height id before");

        vertices.flushPs(vId, 3 * challengePeriodSec / 4);

        assertEq(vertices[vId].psLastUpdatedTimestamp, block.timestamp, "Invalid last updated time");
        assertEq(vertices[v1Id].flushedPsTimeSec, 3 * challengePeriodSec / 4, "Invalid flushed ps time");
        assertEq(vertices[vId].psId, v1Id, "Invalid last ps id after");
        assertEq(vertices[vId].lowestHeightSuccessorId, v1Id, "Invalid lowest height id after");
    }

    function testFlushPsFailNotExist() public {
        (bytes32 cId,, uint256 height) = createPredecessorVertex();
        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Vertex does not exist");
        vertices.flushPs(v1.id(), 0);
    }

    function testFlushPsFailIsLeaf() public {
        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();
        ChallengeVertex memory v1 =
            ChallengeVertexLib.newLeaf(cId, rand.hash(), height + 2, rand.hash(), rand.addr(), 0);
        vertices.addVertex(v1, vId, challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Cannot flush leaf as it will never have a PS");
        vertices.flushPs(v1.id(), 0);
    }

    // connect

    function createTwoOneVertices(int256 heightOffset) internal returns (bytes32, bytes32, bytes32, bytes32, bytes32) {
        // create vertices like the following
        // a --> b --> c
        //  \
        //   \-------> d
        // where d is the vertex created from heightoffset
        // heightOffset is the difference between c and d

        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);
        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 4, 0);
        vertices.addVertex(v2, v1.id(), challengePeriodSec);
        ChallengeVertex memory v3 =
            ChallengeVertexLib.newVertex(cId, rand.hash(), uint256(int256(height + 4) + heightOffset), 0);
        vertices.addVertex(v3, vId, challengePeriodSec);

        return (vId, v1.id(), v2.id(), v3.id(), cId);
    }

    function createTwoRivalOneVertices(int256 heightOffset)
        internal
        returns (bytes32, bytes32, bytes32, bytes32, bytes32)
    {
        // create vertices like the following
        // a --> b --> c
        //  \     \--> d
        //   \-------> e
        // c and d are at the same height
        // where e is the vertex created from heightoffset
        // heightOffset is the difference between c and e

        (bytes32 cId, bytes32 vId, uint256 height) = createPredecessorVertex();

        ChallengeVertex memory v1 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v1, vId, challengePeriodSec);
        ChallengeVertex memory v2 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 4, 0);
        vertices.addVertex(v2, v1.id(), challengePeriodSec);
        ChallengeVertex memory v3 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 4, 0);
        vertices.addVertex(v3, v1.id(), challengePeriodSec);
        ChallengeVertex memory v4 =
            ChallengeVertexLib.newVertex(cId, rand.hash(), uint256(int256(height + 4) + heightOffset), 0);
        vertices.addVertex(v4, vId, challengePeriodSec);

        return (vId, v1.id(), v2.id(), v3.id(), v4.id());
    }

    function testConnectSameHeight() public {
        (, bytes32 v1Id,, bytes32 v3Id,) = createTwoOneVertices(0);

        vm.warp(block.timestamp + challengePeriodSec + 1);

        vm.expectRevert("Start vertex has ps with timer greater than challenge period, cannot set same height ps");
        vertices.connect(v1Id, v3Id, challengePeriodSec);
    }

    function testConnectGreaterHeight() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id,) = createTwoOneVertices(1);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v1Id, v3Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, v2Id, "Ps id");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v2Id, "LHS");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp - challengePeriodSec / 2, "Last updated");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), challengePeriodSec / 2, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
    }

    function testConnectGreaterHeightChallengePeriodExceeded() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id,) = createTwoOneVertices(1);

        vm.warp(block.timestamp + challengePeriodSec + 1);

        vertices.connect(v1Id, v3Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, v2Id, "Ps id");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v2Id, "LHS");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp - (challengePeriodSec + 1), "Last updated");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), challengePeriodSec + 1, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
    }

    function testConnectLowerHeight() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id,) = createTwoOneVertices(-1);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v1Id, v3Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, v3Id, "Ps id");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v3Id, "LHS");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp, "Last updated");
        assertEq(vertices[v2Id].flushedPsTimeSec, challengePeriodSec / 2, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), challengePeriodSec / 2, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
    }

    function testConnectLowerHeightFailsChallengePeriod() public {
        (, bytes32 v1Id,, bytes32 v3Id,) = createTwoOneVertices(-1);

        vm.warp(block.timestamp + challengePeriodSec + 1);

        vm.expectRevert("Start vertex has ps with timer greater than challenge period, cannot set lower ps");
        vertices.connect(v1Id, v3Id, challengePeriodSec);
    }

    function testConnectNoCompetition() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id,) = createTwoOneVertices(2);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v2Id, v3Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v2Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, v2Id, "Ps id v2");
        assertEq(vertices[v2Id].psId, v3Id, "Ps id v3");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v2Id, "LHS v2");
        assertEq(vertices[v2Id].lowestHeightSuccessorId, v3Id, "LHS v3");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp - challengePeriodSec / 2, "Last updated v1");
        assertEq(vertices[v2Id].psLastUpdatedTimestamp, block.timestamp, "Last updated v2");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), challengePeriodSec / 2, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
    }

    function testConnectGreaterThanRivals() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id, bytes32 v4Id) = createTwoRivalOneVertices(2);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v1Id, v4Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v4Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, 0, "Ps id v1");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v2Id, "LHS v2");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp - challengePeriodSec / 2, "Last updated v1");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), 0, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
        assertEq(vertices[v4Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v4Id), 0, "Current ps v3");
    }

    function testConnectLessThanRivals() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id, bytes32 v4Id) = createTwoRivalOneVertices(-1);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v1Id, v4Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v4Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, v4Id, "Ps id v1");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v4Id, "LHS v2");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp, "Last updated v1");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), 0, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
        assertEq(vertices[v4Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v4Id), 0, "Current ps v3");
    }

    function testConnectSameAsRivals() public {
        (, bytes32 v1Id, bytes32 v2Id, bytes32 v3Id, bytes32 v4Id) = createTwoRivalOneVertices(0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vertices.connect(v1Id, v4Id, challengePeriodSec);

        assertEq(vertices[v2Id].predecessorId, v1Id, "Invalid predecessor v2");
        assertEq(vertices[v3Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v4Id].predecessorId, v1Id, "Invalid predecessor v3");
        assertEq(vertices[v1Id].psId, 0, "Ps id v1");
        assertEq(vertices[v1Id].lowestHeightSuccessorId, v2Id, "LHS v2");
        assertEq(vertices[v1Id].psLastUpdatedTimestamp, block.timestamp, "Last updated v1");
        assertEq(vertices[v2Id].flushedPsTimeSec, 0, "V2 flushed");
        assertEq(vertices.getCurrentPsTimer(v2Id), 0, "Current ps v2");
        assertEq(vertices[v3Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v3Id), 0, "Current ps v3");
        assertEq(vertices[v4Id].flushedPsTimeSec, 0, "V3 flushed");
        assertEq(vertices.getCurrentPsTimer(v4Id), 0, "Current ps v3");
    }

    function testConnectFailAlreadyConnected() public {
        (, bytes32 v1Id, bytes32 v2Id,,) = createTwoOneVertices(0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Vertices already connected");
        vertices.connect(v1Id, v2Id, challengePeriodSec);
    }

    function testConnectFailDifferentChallengeId() public {
        (, bytes32 v1Id, bytes32 v2Id,,) = createTwoOneVertices(0);

        bytes32 cId = rand.hash();
        uint256 height = 100;
        ChallengeVertex memory v21 = ChallengeVertexLib.newVertex(cId, rand.hash(), height, 0);
        vertices[v21.id()] = v21;

        ChallengeVertex memory v22 = ChallengeVertexLib.newVertex(cId, rand.hash(), height + 2, 0);
        vertices.addVertex(v22, v21.id(), challengePeriodSec);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Predecessor and successor are in different challenges");
        vertices.connect(v1Id, v22.id(), challengePeriodSec);
    }

    function testConnectFailStartDoesntExist() public {
        (, bytes32 v1Id,,, bytes32 cId) = createTwoOneVertices(0);

        uint256 height = 100;
        ChallengeVertex memory v21 = ChallengeVertexLib.newVertex(cId, rand.hash(), height, 0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Start vertex does not exist");
        vertices.connect(v21.id(), v1Id, challengePeriodSec);
    }

    function testConnectFailEndDoesntExist() public {
        (, bytes32 v1Id,,, bytes32 cId) = createTwoOneVertices(0);

        uint256 height = 100;
        ChallengeVertex memory v21 = ChallengeVertexLib.newVertex(cId, rand.hash(), height, 0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("End vertex does not exist");
        vertices.connect(v1Id, v21.id(), challengePeriodSec);
    }

    function testConnectFailCannotConnectRootAsEnd() public {
        (, bytes32 v1Id,,, bytes32 cId) = createTwoOneVertices(0);

        ChallengeVertex memory root = ChallengeVertexLib.newRoot(cId, rand.hash(), rand.hash());
        vertices[root.id()] = root;

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Start height not lower than end height");
        vertices.connect(v1Id, root.id(), challengePeriodSec);
    }

    function testConnectFailCannotConnectLeafAsStart() public {
        (,, bytes32 v2Id,, bytes32 cId) = createTwoOneVertices(0);

        ChallengeVertex memory leaf = ChallengeVertexLib.newLeaf(cId, rand.hash(), 3, rand.hash(), rand.addr(), 0);
        vertices[leaf.id()] = leaf;

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("Cannot connect a successor to a leaf");
        vertices.connect(leaf.id(), v2Id, challengePeriodSec);
    }

    function testConnectFailEndDoesNotExist() public {
        (,, bytes32 v2Id,, bytes32 cId) = createTwoOneVertices(0);

        uint256 height = 100;
        ChallengeVertex memory v21 = ChallengeVertexLib.newVertex(cId, rand.hash(), height, 0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        vm.expectRevert("End vertex does not exist");
        vertices.connect(v2Id, v21.id(), challengePeriodSec);
    }

    function testConnectFailStartDoesNotExist() public {
        (,, bytes32 v2Id,,) = createTwoOneVertices(0);

        vm.warp(block.timestamp + challengePeriodSec / 2);

        bytes32 randId = rand.hash();
        vm.expectRevert("Start vertex does not exist");
        vertices.connect(randId, v2Id, challengePeriodSec);
    }

    // addVertex

    function testAddVertex() public {
        (,, bytes32 v2Id,, bytes32 cId) = createTwoOneVertices(0);

        ChallengeVertex memory v = ChallengeVertexLib.newVertex(cId, rand.hash(), 20, 0);

        assertFalse(vertices[v.id()].exists(), "Exists early");
        vertices.addVertex(v, v2Id, challengePeriodSec);
        assertTrue(vertices[v.id()].exists(), "Does not exist after");

        assertEq(vertices[v.id()].predecessorId, v2Id, "Predecessor set");
        assertEq(vertices[v2Id].lowestHeightSuccessorId, v.id(), "LHS set");
        assertEq(vertices[v2Id].psId, v.id(), "ps id set");
        assertEq(vertices[v2Id].psLastUpdatedTimestamp, block.timestamp, "ps last updated set");
    }

    function testAddVertexFailsNotExists() public {
        (,, bytes32 v2Id,, bytes32 cId) = createTwoOneVertices(0);

        ChallengeVertex memory v = ChallengeVertexLib.newVertex(cId, rand.hash(), 20, 0);
        vertices[v.id()] = v;

        vm.expectRevert("Vertex already exists");
        vertices.addVertex(v, v2Id, challengePeriodSec);
    }
}
