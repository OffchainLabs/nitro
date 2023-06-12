// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./Enums.sol";

/// @dev The edge is not currently stored
error EdgeNotExists(bytes32 edgeId);
/// @dev The edge has already been stored
error EdgeAlreadyExists(bytes32 edgeId);
/// @dev The provided assertion id was empty
error AssertionIdEmpty();
/// @dev The assertion ids are not the same, but should have been
error AssertionIdMismatch(bytes32 id1, bytes32 id2);
/// @dev The assertion is not currently pending
error AssertionNotPending();
/// @dev The assertion has no sibling
error AssertionNoSibling();
/// @dev The edge type specific proof data is empty
error EmptyEdgeSpecificProof();
/// @dev The start machine status is empty
error EmptyStartMachineStatus();
/// @dev The end machine status is empty
error EmptyEndMachineStatus();
/// @dev The claim edge is not pending
error ClaimEdgeNotPending();
/// @dev The claim edge does not have a length one rival
error ClaimEdgeNotLengthOneRival(bytes32 claimId);
/// @dev The claim edge has an invalid type
error ClaimEdgeInvalidType(EdgeType argType, EdgeType claimType);
/// @dev The val is not a power of two
error NotPowerOfTwo(uint256 val);
/// @dev The height has an unexpected value
error InvalidEndHeight(uint256 actualHeight, uint256 expectedHeight);
/// @dev The prefix proof is empty
error EmptyPrefixProof();
/// @dev The edge is not of type Block
error EdgeTypeNotBlock(EdgeType eType);
/// @dev The edge is not of type SmallStep
error EdgeTypeNotSmallStep(EdgeType eType);
/// @dev The first rival record is empty
error EmptyFirstRival();
/// @dev The difference between two heights is less than 2
error HeightDiffLtTwo(uint256 h1, uint256 h2);
/// @dev The edge is not pending
error EdgeNotPending(bytes32 edgeId, EdgeStatus status);
/// @dev The edge is unrivaled
error EdgeUnrivaled(bytes32 edgeId);
/// @dev The edge is not confirmed
error EdgeNotConfirmed(bytes32 edgeId, EdgeStatus);
/// @dev The edge type is unexpected
error EdgeTypeInvalid(bytes32 edgeId1, bytes32 edgeId2, EdgeType type1, EdgeType type2);
/// @dev The claim id on the claimingEdge does not match the provided edge id
error EdgeClaimMismatch(bytes32 edgeId, bytes32 claimingEdgeId);
/// @dev The origin id is not equal to the mutual id
error OriginIdMutualIdMismatch(bytes32 mutualId, bytes32 originId);
/// @dev The edge does not have a valid ancestor link
error EdgeNotAncestor(
    bytes32 edgeId, bytes32 lowerChildId, bytes32 upperChildId, bytes32 ancestorEdgeId, bytes32 claimId
);
/// @dev The total number of blocks is not above the threshold
error InsufficientConfirmationBlocks(uint256 totalBlocks, uint256 thresholdBlocks);
/// @dev The edge is not of length one
error EdgeNotLengthOne(uint256 length);
/// @dev No origin id supplied when creating an edge
error EmptyOriginId();
/// @dev Invalid heights supplied when creating an edge
error InvalidHeights(uint256 start, uint256 end);
/// @dev No start root supplied when creating an edge
error EmptyStartRoot();
/// @dev No end root supplied when creating an edge
error EmptyEndRoot();
/// @dev No staker supplied when creating a layer zero edge
error EmptyStaker();
/// @dev No claim id supplied when creating a layer zero edge
error EmptyClaimId();
/// @dev Children already set on edge
error ChildrenAlreadySet(bytes32 edgeId, bytes32 lowerChildId, bytes32 upperChildId);
/// @dev Edge is not a layer zero edge
error EdgeNotLayerZero(bytes32 edgeId, address staker, bytes32 claimId);
/// @dev The edge staker has already been refunded
error EdgeAlreadyRefunded(bytes32 edgeId);
/// @dev No assertion chain address supplied
error EmptyAssertionChain();
/// @dev No one step proof entry address supplied
error EmptyOneStepProofEntry();
/// @dev No challenge period supplied
error EmptyChallengePeriod();
/// @dev No stake receiver address supplied
error EmptyStakeReceiver();
