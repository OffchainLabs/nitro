// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupCore.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/IOutbox.sol";
import "../bridge/IOwnable.sol";

interface IRollupUserAbs is IRollupCore, IOwnable {
    /// @dev the user logic just validated configuration and shouldn't write to state during init
    /// this allows the admin logic to ensure consistency on parameters.
    function initialize(address stakeToken) external view;

    function removeWhitelistAfterFork() external;

    function removeWhitelistAfterValidatorAfk() external;

    function isERC20Enabled() external view returns (bool);

    function rejectNextAssertion(address stakerAddress) external;

    function confirmNextAssertion(bytes32 blockHash, bytes32 sendRoot, bytes32 winningEdge) external;

    function stakeOnExistingAssertion(uint64 assertionNum, bytes32 assertionHash) external;

    function stakeOnNewAssertion(
        AssertionInputs memory assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) external;

    function returnOldDeposit(address stakerAddress) external;

    function reduceDeposit(uint256 target) external;

    function removeZombie(uint256 zombieNum, uint256 maxAssertions) external;

    function removeOldZombies(uint256 startIndex) external;

    function requiredStake(
        uint256 blockNumber,
        uint64 firstUnresolvedAssertionNum,
        uint64 latestCreatedAssertion
    ) external view returns (uint256);

    function currentRequiredStake() external view returns (uint256);

    function countStakedZombies(uint64 assertionNum) external view returns (uint256);

    function countZombiesStakedOnChildren(uint64 assertionNum) external view returns (uint256);

    function requireUnresolvedExists() external view;

    function requireUnresolved(uint256 assertionNum) external view;

    function withdrawStakerFunds() external returns (uint256);

}

interface IRollupUser is IRollupUserAbs {
    function newStakeOnExistingAssertion(uint64 assertionNum, bytes32 assertionHash) external payable;

    function newStakeOnNewAssertion(
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) external payable;

    function addToDeposit(address stakerAddress) external payable;
}

interface IRollupUserERC20 is IRollupUserAbs {
    function newStakeOnExistingAssertion(
        uint256 tokenAmount,
        uint64 assertionNum,
        bytes32 assertionHash
    ) external;

    function newStakeOnNewAssertion(
        uint256 tokenAmount,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) external;

    function addToDeposit(address stakerAddress, uint256 tokenAmount) external;
}
