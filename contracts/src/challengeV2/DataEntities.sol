// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../bridge/IBridge.sol";

/// @title  Assertion chain interface
/// @notice The interface required by the EdgeChallengeManager for requesting assertion data from the AssertionChain
interface IAssertionChain {
    function bridge() external view returns (IBridge);
    function getPredecessorId(bytes32 assertionId) external view returns (bytes32);
    function getHeight(bytes32 assertionId) external view returns (uint256);
    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256);
    function getStateHash(bytes32 assertionId) external view returns (bytes32);
    function hasSibling(bytes32 assertionId) external view returns (bool);
    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256);
    function getWasmModuleRoot(bytes32 assertionId) external view returns (bytes32);
    function isFirstChild(bytes32 assertionId) external view returns (bool);
    function isPending(bytes32 assertionId) external view returns (bool);
}
