// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "./Utils.sol";
import "../../src/challengeV2/libraries/ChallengeEdgeLib.sol";

contract ChallengeEdgeLibTest is Test {
    Random rand = new Random();

    function randCheckArgs() internal returns (bytes32, bytes32, bytes32) {
        return (rand.hash(), rand.hash(), rand.hash());
    }

    function testEdgeChecks() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 15);
    }

    function testEdgeChecksZeroOrigin() public {
        (, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert("Empty origin id");
        ChallengeEdgeLib.newEdgeChecks(0, startRoot, 10, endRoot, 15);
    }

    function testEdgeChecksStartRoot() public {
        (bytes32 originId,, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert("Empty start history root");
        ChallengeEdgeLib.newEdgeChecks(originId, 0, 10, endRoot, 15);
    }

    function testEdgeChecksEndRoot() public {
        (bytes32 originId, bytes32 startRoot,) = randCheckArgs();
        vm.expectRevert("Empty end history root");
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, 0, 15);
    }

    function testEdgeChecksHeightLessThan() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert();
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 5);
    }

    function testEdgeChecksHeightEqual() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert("Invalid heights");
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 10);
    }

    function testNewLayerZeroEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();
        ChallengeEdge memory e =
            ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, staker, EdgeType.BigStep);
        assertEq(e.originId, originId, "Origin id");
        assertEq(e.startHeight, 10, "Start height");
        assertEq(e.startHistoryRoot, startRoot, "Start root");
        assertEq(e.endHeight, 15, "end height");
        assertEq(e.endHistoryRoot, endRoot, "End root");
        assertEq(e.lowerChildId, 0, "Lower child");
        assertEq(e.upperChildId, 0, "Upper child");
        assertEq(e.createdAtBlock, block.number, "Block number");
        assertEq(e.createdAtBlock, 1, "Block number 1");
        assertEq(e.claimId, claimId, "Claim id");
        assertEq(e.staker, staker, "Staker");
        assertTrue(e.status == EdgeStatus.Pending, "Status");
        assertTrue(e.eType == EdgeType.BigStep, "EType");
    }

    function testNewLayerZeroEdgeZeroStaker() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        vm.expectRevert("Empty staker");
        ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, address(0), EdgeType.BigStep);
    }

    function testNewLayerZeroEdgeZeroClaimId() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        address staker = rand.addr();
        vm.expectRevert("Empty claim id");
        ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, 0, staker, EdgeType.BigStep);
    }

    function testNewChildEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdge memory e = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 15, EdgeType.BigStep);
        assertEq(e.originId, originId, "Origin id");
        assertEq(e.startHeight, 10, "Start height");
        assertEq(e.startHistoryRoot, startRoot, "Start root");
        assertEq(e.endHeight, 15, "end height");
        assertEq(e.endHistoryRoot, endRoot, "End root");
        assertEq(e.lowerChildId, 0, "Lower child");
        assertEq(e.upperChildId, 0, "Upper child");
        assertEq(e.createdAtBlock, block.number, "Block number");
        assertEq(e.createdAtBlock, 1, "Block number 1");
        assertEq(e.claimId, 0, "Claim id");
        assertEq(e.staker, address(0), "Staker");
        assertTrue(e.status == EdgeStatus.Pending, "Status");
        assertTrue(e.eType == EdgeType.BigStep, "EType");
    }

    ChallengeEdge layerZero;
    ChallengeEdge child;

    function testEdgeExists() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();

        assertFalse(ChallengeEdgeLib.exists(layerZero), "Layer zero exists");
        assertFalse(ChallengeEdgeLib.exists(child), "Child exists");

        layerZero =
            ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, staker, EdgeType.BigStep);
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        assertTrue(ChallengeEdgeLib.exists(layerZero), "Layer zero exists");
        assertTrue(ChallengeEdgeLib.exists(child), "Child exists");

        delete layerZero;
        delete child;

        assertFalse(ChallengeEdgeLib.exists(layerZero), "Layer zero exists");
        assertFalse(ChallengeEdgeLib.exists(child), "Child exists");
    }

    function testLength() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();

        layerZero =
            ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, staker, EdgeType.BigStep);
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        assertEq(ChallengeEdgeLib.length(layerZero), 5, "L-zero len");
        assertEq(ChallengeEdgeLib.length(child), 7, "Child len");

        delete layerZero;
        delete child;

        vm.expectRevert("Edge does not exist");
        ChallengeEdgeLib.length(layerZero);

        vm.expectRevert("Edge does not exist");
        ChallengeEdgeLib.length(child);
    }

    function testSetChildren() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);

        assertEq(child.lowerChildId, lowerChildId, "Lower child id");
        assertEq(child.upperChildId, upperChildId, "Upper child id");

        delete child;
    }

    function testSetChildrenTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);
        vm.expectRevert("Children already set");
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);
    }

    function testSetConfirmed() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        ChallengeEdgeLib.setConfirmed(child);
        assertTrue(child.status == EdgeStatus.Confirmed, "Status confirmed");
    }

    function testSetConfirmedTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, EdgeType.BigStep);

        ChallengeEdgeLib.setConfirmed(child);
        vm.expectRevert("Only Pending edges can be Confirmed");
        ChallengeEdgeLib.setConfirmed(child);
    }
}
