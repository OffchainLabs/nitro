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

contract ChallengeEdgeLibAccess {
    ChallengeEdge storageEdge;

    function getChallengeEdge() public view returns (ChallengeEdge memory) {
        return storageEdge;
    }

    function setChallengeEdge(ChallengeEdge memory edge) public {
        storageEdge = edge;
    }

    function deleteChallengeEdge() public {
        delete storageEdge;
    }

    function newEdgeChecks(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight
    ) public pure {
        return ChallengeEdgeLib.newEdgeChecks(originId, startHistoryRoot, startHeight, endHistoryRoot, endHeight);
    }

    function newLayerZeroEdge(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight,
        bytes32 claimId,
        address staker,
        uint8 level
    ) public view returns (ChallengeEdge memory) {
        return ChallengeEdgeLib.newLayerZeroEdge(
            originId, startHistoryRoot, startHeight, endHistoryRoot, endHeight, claimId, staker, level
        );
    }

    function newChildEdge(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight,
        uint8 level
    ) public view returns (ChallengeEdge memory) {
        return ChallengeEdgeLib.newChildEdge(originId, startHistoryRoot, startHeight, endHistoryRoot, endHeight, level);
    }

    function mutualIdComponent(
        uint8 level,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.mutualIdComponent(level, originId, startHeight, startHistoryRoot, endHeight);
    }

    function mutualId() public view returns (bytes32) {
        return ChallengeEdgeLib.mutualId(storageEdge);
    }

    function mutualIdMem(ChallengeEdge memory ce) public pure returns (bytes32) {
        return ChallengeEdgeLib.mutualIdMem(ce);
    }

    function idComponent(
        uint8 level,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.idComponent(level, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot);
    }

    function idMem(ChallengeEdge memory edge) public pure returns (bytes32) {
        return ChallengeEdgeLib.idMem(edge);
    }

    function id() public view returns (bytes32) {
        return ChallengeEdgeLib.id(storageEdge);
    }

    function exists() public view returns (bool) {
        return ChallengeEdgeLib.exists(storageEdge);
    }

    function length() public view returns (uint256) {
        return ChallengeEdgeLib.length(storageEdge);
    }

    function setChildren(bytes32 lowerChildId, bytes32 upperChildId) public {
        return ChallengeEdgeLib.setChildren(storageEdge, lowerChildId, upperChildId);
    }

    function setConfirmed() public {
        return ChallengeEdgeLib.setConfirmed(storageEdge);
    }

    function isLayerZero() public view returns (bool) {
        return ChallengeEdgeLib.isLayerZero(storageEdge);
    }

    function setRefunded() public {
        return ChallengeEdgeLib.setRefunded(storageEdge);
    }

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
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(EmptyOriginId.selector));
        access.newEdgeChecks(0, startRoot, 10, endRoot, 15);
    }

    function testEdgeChecksStartRoot() public {
        (bytes32 originId,, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(EmptyStartRoot.selector));
        access.newEdgeChecks(originId, 0, 10, endRoot, 15);
    }

    function testEdgeChecksEndRoot() public {
        (bytes32 originId, bytes32 startRoot,) = randCheckArgs();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(EmptyEndRoot.selector));
        access.newEdgeChecks(originId, startRoot, 10, 0, 15);
    }

    function testEdgeChecksHeightLessThan() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(InvalidHeights.selector, 10, 5));
        access.newEdgeChecks(originId, startRoot, 10, endRoot, 5);
    }

    function testEdgeChecksHeightEqual() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(InvalidHeights.selector, 10, 10));
        access.newEdgeChecks(originId, startRoot, 10, endRoot, 10);
    }

    function testNewLayerZeroEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        ChallengeEdge memory e =
            access.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1);
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
        assertEq(
            access.mutualIdMem(e),
            keccak256(abi.encodePacked(e.level, e.originId, e.startHeight, e.startHistoryRoot, e.endHeight)),
            "Id mem"
        );
    }

    function testNewLayerZeroEdgeZeroStaker() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(EmptyStaker.selector));
        access.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, address(0), NUM_BIGSTEP_LEVEL + 1);
    }

    function testNewLayerZeroEdgeZeroClaimId() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        address staker = rand.addr();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        vm.expectRevert(abi.encodeWithSelector(EmptyClaimId.selector));
        access.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, 0, staker, NUM_BIGSTEP_LEVEL + 1);
    }

    function testNewChildEdge() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        ChallengeEdge memory e = access.newChildEdge(originId, startRoot, 10, endRoot, 15, NUM_BIGSTEP_LEVEL + 1);
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

    function testEdgeExists() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();

        ChallengeEdgeLibAccess layerZero = new ChallengeEdgeLibAccess();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();

        assertFalse(layerZero.exists(), "Layer zero exists");
        assertFalse(child.exists(), "Child exists");

        ChallengeEdge memory layerZeroEdge =
            layerZero.newLayerZeroEdge(originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1);
        layerZero.setChallengeEdge(layerZeroEdge);
        ChallengeEdge memory childEdge = child.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        assertTrue(layerZero.exists(), "Layer zero exists");
        assertTrue(child.exists(), "Child exists");

        layerZero.deleteChallengeEdge();
        child.deleteChallengeEdge();

        assertFalse(layerZero.exists(), "Layer zero exists");
        assertFalse(child.exists(), "Child exists");
    }

    function testLength() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        bytes32 claimId = rand.hash();
        address staker = rand.addr();

        ChallengeEdgeLibAccess layerZero = new ChallengeEdgeLibAccess();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();

        ChallengeEdge memory layerZeroEdge = ChallengeEdgeLib.newLayerZeroEdge(
            originId, startRoot, 10, endRoot, 15, claimId, staker, NUM_BIGSTEP_LEVEL + 1
        );
        layerZero.setChallengeEdge(layerZeroEdge);
        ChallengeEdge memory childEdge =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        assertEq(layerZero.length(), 5, "L-zero len");
        assertEq(child.length(), 7, "Child len");

        layerZero.deleteChallengeEdge();
        child.deleteChallengeEdge();

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, layerZero.id()));
        layerZero.length();

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, child.id()));
        child.length();
    }

    function testSetChildren() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();
        ChallengeEdge memory childEdge =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        child.setChildren(lowerChildId, upperChildId);

        assertEq(child.getChallengeEdge().lowerChildId, lowerChildId, "Lower child id");
        assertEq(child.getChallengeEdge().upperChildId, upperChildId, "Upper child id");

        delete child;
    }

    function testSetChildrenTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();
        ChallengeEdge memory childEdge =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        bytes32 lowerChildId = rand.hash();
        bytes32 upperChildId = rand.hash();
        child.setChildren(lowerChildId, upperChildId);
        vm.expectRevert(abi.encodeWithSelector(ChildrenAlreadySet.selector, child.id(), lowerChildId, upperChildId));
        child.setChildren(lowerChildId, upperChildId);
    }

    function testSetConfirmed() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();
        ChallengeEdge memory childEdge =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        vm.roll(137);

        child.setConfirmed();
        assertTrue(child.getChallengeEdge().status == EdgeStatus.Confirmed, "Status confirmed");
        assertTrue(child.getChallengeEdge().confirmedAtBlock == 137, "Confirmed at block");
    }

    function testSetConfirmedTwice() public {
        (bytes32 originId, bytes32 startRoot, bytes32 endRoot) = randCheckArgs();
        ChallengeEdgeLibAccess child = new ChallengeEdgeLibAccess();
        ChallengeEdge memory childEdge =
            ChallengeEdgeLib.newChildEdge(originId, startRoot, 10, endRoot, 17, NUM_BIGSTEP_LEVEL + 1);
        child.setChallengeEdge(childEdge);

        child.setConfirmed();
        vm.expectRevert(abi.encodeWithSelector(EdgeNotPending.selector, child.id(), EdgeStatus.Confirmed));
        child.setConfirmed();
    }

    function testLevelToType() public {
        ChallengeEdgeLibAccess access = new ChallengeEdgeLibAccess();
        uint8 numBigStep = 4;
        assertTrue(access.levelToType(0, numBigStep) == EdgeType.Block, "Block");
        assertTrue(access.levelToType(1, numBigStep) == EdgeType.BigStep, "Big step 1");
        assertTrue(access.levelToType(2, numBigStep) == EdgeType.BigStep, "Big step 2");
        assertTrue(access.levelToType(3, numBigStep) == EdgeType.BigStep, "Big step 3");
        assertTrue(access.levelToType(4, numBigStep) == EdgeType.BigStep, "Big step 4");
        assertTrue(access.levelToType(5, numBigStep) == EdgeType.SmallStep, "Small step");

        TestChallengeEdge t = new TestChallengeEdge();
        vm.expectRevert(abi.encodeWithSelector(LevelTooHigh.selector, 6, 4));
        t.levelToType(6, numBigStep);
    }
}
