// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../bridge/IBridge.sol";
import "../osp/IOneStepProofEntry.sol";
import "../rollup/Assertion.sol";

/// @title  Assertion chain interface
/// @notice The interface required by the EdgeChallengeManager for requesting assertion data from the AssertionChain
interface IAssertionChain {
    function bridge() external view returns (IBridge);
    function validateAssertionId(
        bytes32 assertionId,
        ExecutionState calldata state,
        bytes32 prevAssertionId,
        bytes32 inboxAcc
    ) external view;
    function validateConfig(bytes32 assertionId, ConfigData calldata configData) external view;
    function getFirstChildCreationBlock(bytes32 assertionId) external view returns (uint256);
    function getSecondChildCreationBlock(bytes32 assertionId) external view returns (uint256);
    function isFirstChild(bytes32 assertionId) external view returns (bool);
    function isPending(bytes32 assertionId) external view returns (bool);
}
