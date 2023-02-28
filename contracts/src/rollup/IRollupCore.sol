// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./Assertion.sol";
import "./RollupLib.sol";

interface IRollupCore {
    struct Staker {
        uint256 amountStaked;
        uint64 index;
        uint64 latestStakedAssertion;
        // currentChallenge is 0 if staker is not in a challenge
        uint64 currentChallenge;
        bool isStaked;
    }

    event RollupInitialized(bytes32 machineHash, uint256 chainId);

    event AssertionCreated(
        uint64 indexed assertionNum,
        uint256 indexed height,
        bytes32 indexed parentAssertionHash,
        bytes32 indexed assertionHash,
        bytes32 executionHash,
        AssertionInputs assertion,
        bytes32 afterInboxBatchAcc,
        bytes32 wasmModuleRoot,
        uint256 inboxMaxCount
    );

    event AssertionConfirmed(uint64 indexed assertionNum, bytes32 blockHash, bytes32 sendRoot);

    event AssertionRejected(uint64 indexed assertionNum);

    event RollupChallengeStarted(
        uint64 indexed challengeIndex,
        address asserter,
        address challenger,
        uint64 challengedAssertion
    );

    event UserStakeUpdated(address indexed user, uint256 initialBalance, uint256 finalBalance);

    event UserWithdrawableFundsUpdated(
        address indexed user,
        uint256 initialBalance,
        uint256 finalBalance
    );

    function confirmPeriodBlocks() external view returns (uint64);

    function extraChallengeTimeBlocks() external view returns (uint64);

    function chainId() external view returns (uint256);

    function baseStake() external view returns (uint256);

    function wasmModuleRoot() external view returns (bytes32);

    function bridge() external view returns (IBridge);

    function sequencerInbox() external view returns (ISequencerInbox);

    function outbox() external view returns (IOutbox);

    function rollupEventInbox() external view returns (IRollupEventInbox);

    function oldChallengeManager() external view returns (IOldChallengeManager);

    function loserStakeEscrow() external view returns (address);

    function stakeToken() external view returns (address);

    function minimumAssertionPeriod() external view returns (uint256);

    function isValidator(address) external view returns (bool);

    function validatorWhitelistDisabled() external view returns (bool);

    /**
     * @notice Get the Assertion for the given index.
     */
    function getAssertion(uint64 assertionNum) external view returns (AssertionNode memory);

    /**
     * @notice Check if the specified assertion has been staked on by the provided staker.
     * Only accurate at the latest confirmed assertion and afterwards.
     */
    function assertionHasStaker(uint64 assertionNum, address staker) external view returns (bool);

    /**
     * @notice Get the address of the staker at the given index
     * @param stakerNum Index of the staker
     * @return Address of the staker
     */
    function getStakerAddress(uint64 stakerNum) external view returns (address);

    /**
     * @notice Check whether the given staker is staked
     * @param staker Staker address to check
     * @return True or False for whether the staker was staked
     */
    function isStaked(address staker) external view returns (bool);

    /**
     * @notice Get the latest staked assertion of the given staker
     * @param staker Staker address to lookup
     * @return Latest assertion staked of the staker
     */
    function latestStakedAssertion(address staker) external view returns (uint64);

    /**
     * @notice Get the current challenge of the given staker
     * @param staker Staker address to lookup
     * @return Current challenge of the staker
     */
    function currentChallenge(address staker) external view returns (uint64);

    /**
     * @notice Get the amount staked of the given staker
     * @param staker Staker address to lookup
     * @return Amount staked of the staker
     */
    function amountStaked(address staker) external view returns (uint256);

    /**
     * @notice Retrieves stored information about a requested staker
     * @param staker Staker address to retrieve
     * @return A structure with information about the requested staker
     */
    function getStaker(address staker) external view returns (Staker memory);

    /**
     * @notice Get the original staker address of the zombie at the given index
     * @param zombieNum Index of the zombie to lookup
     * @return Original staker address of the zombie
     */
    function zombieAddress(uint256 zombieNum) external view returns (address);

    /**
     * @notice Get Latest assertion that the given zombie at the given index is staked on
     * @param zombieNum Index of the zombie to lookup
     * @return Latest assertion that the given zombie is staked on
     */
    function zombieLatestStakedAssertion(uint256 zombieNum) external view returns (uint64);

    /// @return Current number of un-removed zombies
    function zombieCount() external view returns (uint256);

    function isZombie(address staker) external view returns (bool);

    /**
     * @notice Get the amount of funds withdrawable by the given address
     * @param owner Address to check the funds of
     * @return Amount of funds withdrawable by owner
     */
    function withdrawableFunds(address owner) external view returns (uint256);

    /**
     * @return Index of the first unresolved assertion
     * @dev If all assertions have been resolved, this will be latestAssertionCreated + 1
     */
    function firstUnresolvedAssertion() external view returns (uint64);

    /// @return Index of the latest confirmed assertion
    function latestConfirmed() external view returns (uint64);

    /// @return Index of the latest rollup assertion created
    function latestAssertionCreated() external view returns (uint64);

    /// @return Ethereum block that the most recent stake was created
    function lastStakeBlock() external view returns (uint64);

    /// @return Number of active stakers currently staked
    function stakerCount() external view returns (uint64);
}
