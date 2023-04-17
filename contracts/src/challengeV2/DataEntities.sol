// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

/// @title  Assertion chain interface
/// @notice The interface required by the EdgeChallengeManager for requesting assertion data from the AssertionChain
interface IAssertionChain {
    function getPredecessorId(bytes32 assertionId) external view returns (bytes32);
    function getHeight(bytes32 assertionId) external view returns (uint256);
    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256);
    function getStateHash(bytes32 assertionId) external view returns (bytes32);
    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32);
    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256);
    function isFirstChild(bytes32 assertionId) external view returns (bool);
    function isPending(bytes32 assertionId) external view returns (bool);
}
