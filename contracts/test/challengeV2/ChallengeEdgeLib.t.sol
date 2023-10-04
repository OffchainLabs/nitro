// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "./Utils.sol";
import "../../src/challengeV2/libraries/ChallengeEdgeLib.sol";

contract TestChallengeEdge {
    function levelToType(uint8 level, uint8 numBigStepLevels) public pure returns (EdgeType eType) {
        return ChallengeEdgeLib.levelToType(level, numBigStepLevels);
    }
}

contract ChallengeEdgeLibTest is Test {
    Random rand = new Random();
    uint8 constant NUM_BIGSTEP_LEVEL = 3;

    function randCheckArgs() internal returns (bytes32, bytes32, bytes32) {
        return (rand.hash(), rand.hash(), rand.hash());
    }

    function testEdgeChecks() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 15);
    }

    function testEdgeChecksZeroOrigin() public {
        (, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert(abi.encodeWithSelector(EmptyOriginId.selector));
        ChallengeEdgeLib.newEdgeChecks(0, startRoot, 10, endRoot, 15);
    }

    function testEdgeChecksStartRoot() public {
        (bytes32 originId,, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert(abi.encodeWithSelector(EmptyStartRoot.selector));
        ChallengeEdgeLib.newEdgeChecks(originId, 0, 10, endRoot, 15);
    }

    function testEdgeChecksEndRoot() public {
        (bytes32 originId, bytes32 startRoot,) = randCheckArgs();
        vm.expectRevert(abi.encodeWithSelector(EmptyEndRoot.selector));
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, 0, 15);
    }

    function testEdgeChecksHeightLessThan() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert(abi.encodeWithSelector(InvalidHeights.selector, 10, 5));
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 5);
    }

    function testEdgeChecksHeightEqual() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        vm.expectRevert(abi.encodeWithSelector(InvalidHeights.selector, 10, 10));
        ChallengeEdgeLib.newEdgeChecks(originId, startRoot, 10, endRoot, 10);
    }

    function testNewLayerZeroEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();
        ChallengeEdge memory e = ChallengeEdgeLib.newLayerZeroEdge(
            originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1
        );
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
        assertTrue(e.level == NUM_BIGSTEP_LEVEL + 1, "EType");
    }

    function testNewLayerZeroEdgeZeroStaker() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        vm.expectRevert(abi.encodeWithSelector(EmptyStaker.selector));
        ChallengeEdgeLib.newLayerZeroEdge(
            originId, startRoot, 10, endRoot, 15, claimId, address(0), NUM_BIGSTEP_LEVEL + 1
        );
    }

    function testNewLayerZeroEdgeZeroClaimId() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        address staker = rand.addr();
        vm.expectRevert(abi.encodeWithSelector(EmptyClaimId.selector));
        ChallengeEdgeLib.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, 0, staker, NUM_BIGSTEP_LEVEL + 1);
    }

    function testNewChildEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdge memory e =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 15, NUM_BIGSTEP_LEVEL + 1);
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
        assertTrue(e.level == NUM_BIGSTEP_LEVEL + 1, "EType");
    }

    ChallengeEdge layerZero;
    ChallengeEdge child;

    function testEdgeExists() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();

        assertFalse(ChallengeEdgeLib.exists(layerZero), "Layer zero exists");
        assertFalse(ChallengeEdgeLib.exists(child), "Child exists");

        layerZero = ChallengeEdgeLib.newLayerZeroEdge(
            originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1
        );
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

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

        layerZero = ChallengeEdgeLib.newLayerZeroEdge(
            originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1
        );
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

        assertEq(ChallengeEdgeLib.length(layerZero), 5, "L-zero len");
        assertEq(ChallengeEdgeLib.length(child), 7, "Child len");

        delete layerZero;
        delete child;

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, ChallengeEdgeLib.id(layerZero)));
        ChallengeEdgeLib.length(layerZero);

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, ChallengeEdgeLib.id(child)));
        ChallengeEdgeLib.length(child);
    }

    function testSetChildren() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);

        assertEq(child.lowerChildId, lowerChildId, "Lower child id");
        assertEq(child.upperChildId, upperChildId, "Upper child id");

        delete child;
    }

    function testSetChildrenTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);
        vm.expectRevert(
            abi.encodeWithSelector(ChildrenAlreadySet.selector, ChallengeEdgeLib.id(child), lowerChildId, upperChildId)
        );
        ChallengeEdgeLib.setChildren(child, lowerChildId, upperChildId);
    }

    function testSetConfirmed() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

        vm.roll(137);

        ChallengeEdgeLib.setConfirmed(child);
        assertTrue(child.status == EdgeStatus.Confirmed, "Status confirmed");
        assertTrue(child.confirmedAtBlock == 137, "Confirmed at block");
    }

    function testSetConfirmedTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        child = ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);

        ChallengeEdgeLib.setConfirmed(child);
        vm.expectRevert(
            abi.encodeWithSelector(EdgeNotPending.selector, ChallengeEdgeLib.id(child), EdgeStatus.Confirmed)
        );
        ChallengeEdgeLib.setConfirmed(child);
    }

    function testLevelToType() public {
        uint8 numBigStep = 4;
        assertTrue(ChallengeEdgeLib.levelToType(0, numBigStep) == EdgeType.Block, "Block");
        assertTrue(ChallengeEdgeLib.levelToType(1, numBigStep) == EdgeType.BigStep, "Big step 1");
        assertTrue(ChallengeEdgeLib.levelToType(2, numBigStep) == EdgeType.BigStep, "Big step 2");
        assertTrue(ChallengeEdgeLib.levelToType(3, numBigStep) == EdgeType.BigStep, "Big step 3");
        assertTrue(ChallengeEdgeLib.levelToType(4, numBigStep) == EdgeType.BigStep, "Big step 4");
        assertTrue(ChallengeEdgeLib.levelToType(5, numBigStep) == EdgeType.SmallStep, "Small step");

        TestChallengeEdge t = new TestChallengeEdge();
        vm.expectRevert(abi.encodeWithSelector(LevelTooHigh.selector, 6, 4));
        t.levelToType(6, numBigStep);
    }
}
