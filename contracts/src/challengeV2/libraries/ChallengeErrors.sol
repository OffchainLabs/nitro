// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "./Enums.sol";

/// @dev The edge is not currently stored
error EdgeNotExists(bytes32 edgeId);
/// @dev The edge has already been stored
error EdgeAlreadyExists(bytes32 edgeId);
/// @dev The provided assertion hash was empty
error AssertionHashEmpty();
/// @dev The assertion hashes are not the same, but should have been
error AssertionHashMismatch(bytes32 h1, bytes32 h2);
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
/// @dev The claim edge has an invalid level
error ClaimEdgeInvalidLevel(uint8 argLevel, uint8 claimLevel);
/// @dev The val is not a power of two
error NotPowerOfTwo(uint256 val);
/// @dev The height has an unexpected value
error InvalidEndHeight(uint256 actualHeight, uint256 expectedHeight);
/// @dev The prefix proof is empty
error EmptyPrefixProof();
/// @dev The edge is not of type Block
error EdgeTypeNotBlock(uint8 level);
/// @dev The edge is not of type SmallStep
error EdgeTypeNotSmallStep(uint8 level);
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
/// @dev The edge level is unexpected
error EdgeLevelInvalid(bytes32 edgeId1, bytes32 edgeId2, uint8 level1, uint8 level2);
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
/// @dev Stake amounts does not match number of levels
error StakeAmountsMismatch(uint256 stakeLevels, uint256 numLevels);
/// @dev A rival edge is already confirmed
error RivalEdgeConfirmed(bytes32 edgeId, bytes32 confirmedRivalId);
/// @dev Thrown when big step levels is set to 0
error ZeroBigStepLevels();
/// @dev Thrown when there are too many big step levels requested
error BigStepLevelsTooMany(uint8 levels);
/// @dev Thrown when the level does not correspond to a valid type
error LevelTooHigh(uint8 level, uint8 numBigStepLevels);
/// @dev Thrown for unrecognised edge types
error InvalidEdgeType(EdgeType edgeType);
/// @dev Thrown when endHistoryRoot not matching the assertion
error EndHistoryRootMismatch(bytes32 endHistoryRoot, bytes32 assertionEndRoot);
/// @dev Thrown when the validator whitelist is enabled and the account attempting to create a layer zero edge is not whitelisted
error NotValidator(address account);
/// @dev Thrown when an account has already created a rivalling layer zero edge
error AccountHasMadeLayerZeroRival(address account, bytes32 mutualId);
/// @dev Thrown when the cached time is already sufficient
error CachedTimeSufficient(uint256 actual, uint256 expected);
/// @dev Thrown when the input is an empty array
error EmptyArray();