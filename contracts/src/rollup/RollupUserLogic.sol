// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";

import {IRollupUser} from "./IRollupLogic.sol";
import "../libraries/UUPSNotUpgradeable.sol";
import "./RollupCore.sol";
import "./IRollupLogic.sol";
import {ETH_POS_BLOCK_TIME} from "../libraries/Constants.sol";

abstract contract AbsRollupUserLogic is RollupCore, UUPSNotUpgradeable, IRollupUserAbs {
    using AssertionNodeLib for AssertionNode;
    using GlobalStateLib for GlobalState;

    modifier onlyValidator() {
        require(isValidator[msg.sender] || validatorWhitelistDisabled, "NOT_VALIDATOR");
        _;
    }

    uint256 internal immutable deployTimeChainId = block.chainid;

    function _chainIdChanged() internal view returns (bool) {
        return deployTimeChainId != block.chainid;
    }

    /**
     * @notice Extra number of blocks the validator can remain idle before considered idle
     *         This is 21 days assuming a 13.2 seconds block time // TODO: determine the value here
     */
    uint256 public constant VALIDATOR_AFK_BLOCKS = 137454;

    function _validatorIsAfk() internal view returns (bool) {
        AssertionNode memory latestConfirmedAssertion = getAssertionStorage(latestConfirmed());
        if (latestConfirmedAssertion.createdAtBlock == 0) return false;
        // We consider the validator is gone if the last known assertion is older than VALIDATOR_AFK_BLOCKS
        // Which is either the latest confirmed assertion or the first child of the latest confirmed assertion
        if (latestConfirmedAssertion.firstChildBlock > 0) {
            return latestConfirmedAssertion.firstChildBlock + VALIDATOR_AFK_BLOCKS < block.number;
        }
        return latestConfirmedAssertion.createdAtBlock + VALIDATOR_AFK_BLOCKS < block.number;
    }

    function removeWhitelistAfterFork() external {
        require(!validatorWhitelistDisabled, "WHITELIST_DISABLED");
        require(_chainIdChanged(), "CHAIN_ID_NOT_CHANGED");
        validatorWhitelistDisabled = true;
    }

    /**
     * @notice Remove the whitelist after the validator has been inactive for too long
     */
    function removeWhitelistAfterValidatorAfk() external {
        require(!validatorWhitelistDisabled, "WHITELIST_DISABLED");
        require(_validatorIsAfk(), "VALIDATOR_NOT_AFK");
        validatorWhitelistDisabled = true;
    }

    function isERC20Enabled() public view override returns (bool) {
        return stakeToken != address(0);
    }

    /**
     * @notice Confirm a unresolved assertion
     * @param confirmState The state to confirm
     * @param winningEdgeId The winning edge if a challenge is started
     */
    function confirmAssertion(
        bytes32 assertionHash,
        ExecutionState calldata confirmState,
        bytes32 winningEdgeId,
        BeforeStateData calldata beforeStateData
    ) external onlyValidator whenNotPaused {
        /*
        * To confirm an assertion, the following must be true:
        * 1. The assertion must be pending
        * 2. The assertion's deadline must have passed
        * 3. The assertion's prev must be latest confirmed
        * 4. The assertion's prev's child confirm deadline must have passed
        * 5. If the assertion's prev has more than 1 child, the assertion must be the winner of the challenge
        *
        * Note that we do not need to ever reject invalid assertion because they can never confirm
        *      and the stake on them is swept to the loserStakeEscrow as soon as the leaf is created
        */

        AssertionNode storage assertion = getAssertionStorage(assertionHash);
        // The assertion's must exists and be pending and will be checked in RollupCore

        // Check that deadline has passed
        // TODO: HN: do we need to check this? can we simply relies on the prev's ChildConfirmDeadline?
        //           ChildConfirmDeadline is set to 1 confirmPeriod after first child is created
        assertion.requirePastDeadline();

        // Check that prev is latest confirmed
        assert(assertion.prevId == latestConfirmed());

        AssertionNode storage prevAssertion = getAssertionStorage(assertion.prevId);
        // Check that prev's child confirm deadline has passed
        prevAssertion.requirePastChildConfirmDeadline();

        if (prevAssertion.secondChildBlock > 0) {
            // if the prev has more than 1 child, check if this assertion is the challenge winner
            RollupLib.validateConfigHash(beforeStateData, prevAssertion.configHash);
            ChallengeEdge memory winningEdge = challengeManager.getEdge(winningEdgeId);
            require(winningEdge.claimId == assertionHash, "NOT_WINNER");
            require(winningEdge.status == EdgeStatus.Confirmed, "EDGE_NOT_CONFIRMED");
        }

        confirmAssertionInternal(assertionHash, assertion.prevId, confirmState, beforeStateData.sequencerBatchAcc);
    }

    /**
     * @notice Create a new stake
     * @param depositAmount The amount of either eth or tokens staked
     */
    function _newStake(uint256 depositAmount) internal onlyValidator whenNotPaused {
        // Verify that sender is not already a staker
        require(!isStaked(msg.sender), "ALREADY_STAKED");
        // amount will be checked when creating an assertion
        createNewStake(msg.sender, depositAmount);
    }

    /**
     * @notice Create a new assertion and move stake onto it
     * @param assertion The assertion data
     * @param expectedAssertionHash The hash of the assertion being created (protects against reorgs)
     */
    function stakeOnNewAssertion(AssertionInputs calldata assertion, bytes32 expectedAssertionHash)
        public
        onlyValidator
        whenNotPaused
    {
        // Early revert on duplicated assertion if expectedAssertionHash is set
        require(
            expectedAssertionHash == bytes32(0)
                || getAssertionStorage(expectedAssertionHash).status == AssertionStatus.NoAssertion,
            "EXPECTED_ASSERTION_SEEN"
        );

        require(isStaked(msg.sender), "NOT_STAKED");

        // requiredStake is user supplied, will be verified against configHash later
        // the prev's requiredStake is used to make sure all children have the same stake
        // the staker may have more than enough stake, and the entire stake will be locked
        // we cannot do a refund here because the staker may be staker on an unconfirmed ancestor that requires more stake
        // excess stake can be removed by calling reduceDeposit when the staker is inactive
        require(amountStaked(msg.sender) >= assertion.beforeStateData.requiredStake, "INSUFFICIENT_STAKE");

        bytes32 prevAssertion = RollupLib.assertionHash(
            assertion.beforeStateData.prevPrevAssertionHash,
            assertion.beforeState,
            assertion.beforeStateData.sequencerBatchAcc
        );
        getAssertionStorage(prevAssertion).requireExists();

        // Staker can create new assertion only if
        // a) its last staked assertion is the prev; or
        // b) its last staked assertion have a child
        bytes32 lastAssertion = latestStakedAssertion(msg.sender);
        require(
            lastAssertion == prevAssertion || getAssertionStorage(lastAssertion).firstChildBlock > 0,
            "STAKED_ON_ANOTHER_BRANCH"
        );

        // Validate the config hash
        RollupLib.validateConfigHash(assertion.beforeStateData, getAssertionStorage(prevAssertion).configHash);

        uint256 timeSincePrev = block.number - getAssertionStorage(prevAssertion).createdAtBlock;
        // Verify that assertion meets the minimum Delta time requirement
        require(timeSincePrev >= minimumAssertionPeriod, "TIME_DELTA");

        bytes32 newAssertionHash = createNewAssertion(
            assertion, prevAssertion, assertion.beforeStateData.confirmPeriodBlocks, expectedAssertionHash
        );
        _stakerMap[msg.sender].latestStakedAssertion = newAssertionHash;

        if (!getAssertionStorage(newAssertionHash).isFirstChild) {
            // only 1 of the children can be confirmed and get their stake refunded
            // so we send the other children's stake to the loserStakeEscrow
            // NOTE: if the losing staker have staked more than requiredStake, the excess stake will be stuck
            increaseWithdrawableFunds(loserStakeEscrow, assertion.beforeStateData.requiredStake);
        }
    }

    /**
     * @notice Refund a staker that is currently staked on or before the latest confirmed assertion
     */
    function returnOldDeposit() external override onlyValidator whenNotPaused {
        requireInactiveStaker(msg.sender);
        withdrawStaker(msg.sender);
    }

    /**
     * @notice Increase the amount staked for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     * @param depositAmount The amount of either eth or tokens deposited
     */
    function _addToDeposit(address stakerAddress, uint256 depositAmount) internal onlyValidator whenNotPaused {
        require(isStaked(stakerAddress), "NOT_STAKED");
        increaseStakeBy(stakerAddress, depositAmount);
    }

    /**
     * @notice Reduce the amount staked for the sender (difference between initial amount staked and target is creditted back to the sender).
     * @param target Target amount of stake for the staker.
     */
    function reduceDeposit(uint256 target) external onlyValidator whenNotPaused {
        requireInactiveStaker(msg.sender);
        // amount will be checked when creating an assertion
        reduceStakeTo(msg.sender, target);
    }

    function owner() external view returns (address) {
        return _getAdmin();
    }
}

// TODO: Consider remove this and use WETH with ERC20RollupUserLogic
contract RollupUserLogic is AbsRollupUserLogic, IRollupUser {
    /// @dev the user logic just validated configuration and shouldn't write to state during init
    /// this allows the admin logic to ensure consistency on parameters.
    function initialize(address _stakeToken) external view override onlyProxy {
        require(_stakeToken == address(0), "NO_TOKEN_ALLOWED");
        require(!isERC20Enabled(), "FACET_NOT_ERC20");
    }

    /**
     * @notice Create a new stake on a new assertion
     * @param assertion Assertion describing the state change between the old assertion and the new one
     * @param expectedAssertionHash Assertion hash of the assertion that will be created
     */
    function newStakeOnNewAssertion(AssertionInputs calldata assertion, bytes32 expectedAssertionHash)
        external
        payable
        override
    {
        /**
         * Validators can create a stake by calling this function (or the ERC20 version).
         * Each validator can only create one stake, and they can increase or decrease it when the stake is inactive.
         *   A staker is considered inactive if:
         *       a) their last staked assertion is the latest confirmed assertion
         *       b) their last staked assertion has a child (where the staking responsibility is passed to the child)
         *
         * If the assertion is the 2nd child or later, since only one of the children can be confirmed and we know the contract
         * already have 1 stake from the 1st child to refund the winner, we send the other children's stake to the loserStakeEscrow.
         *
         * Stake can be withdrawn by calling `returnOldDeposit` followed by `withdrawStakerFunds` when the staker is inactive.
         */
        _newStake(msg.value);
        stakeOnNewAssertion(assertion, expectedAssertionHash);
    }

    /**
     * @notice Increase the amount staked eth for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     */
    function addToDeposit(address stakerAddress) external payable override onlyValidator whenNotPaused {
        _addToDeposit(stakerAddress, msg.value);
    }

    /**
     * @notice Withdraw uncommitted funds owned by sender from the rollup chain
     */
    function withdrawStakerFunds() external override whenNotPaused returns (uint256) {
        uint256 amount = withdrawFunds(msg.sender);
        require(amount > 0, "NO_FUNDS_TO_WITHDRAW");
        // This is safe because it occurs after all checks and effects
        // solhint-disable-next-line avoid-low-level-calls
        (bool success,) = msg.sender.call{value: amount}("");
        require(success, "TRANSFER_FAILED");
        return amount;
    }
}

contract ERC20RollupUserLogic is AbsRollupUserLogic, IRollupUserERC20 {
    /// @dev the user logic just validated configuration and shouldn't write to state during init
    /// this allows the admin logic to ensure consistency on parameters.
    function initialize(address _stakeToken) external view override onlyProxy {
        require(_stakeToken != address(0), "NEED_STAKE_TOKEN");
        require(isERC20Enabled(), "FACET_NOT_ERC20");
    }

    /**
     * @notice Create a new stake on a new assertion
     * @param tokenAmount Amount of the rollups staking token to stake
     * @param assertion Assertion describing the state change between the old assertion and the new one
     * @param expectedAssertionHash Assertion hash of the assertion that will be created
     */
    function newStakeOnNewAssertion(
        uint256 tokenAmount,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash
    ) external override {
        _newStake(tokenAmount);
        stakeOnNewAssertion(assertion, expectedAssertionHash);
        /// @dev This is an external call, safe because it's at the end of the function
        receiveTokens(tokenAmount);
    }

    /**
     * @notice Increase the amount staked tokens for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     * @param tokenAmount the amount of tokens staked
     */
    function addToDeposit(address stakerAddress, uint256 tokenAmount) external onlyValidator whenNotPaused {
        _addToDeposit(stakerAddress, tokenAmount);
        /// @dev This is an external call, safe because it's at the end of the function
        receiveTokens(tokenAmount);
    }

    /**
     * @notice Withdraw uncommitted funds owned by sender from the rollup chain
     */
    function withdrawStakerFunds() external override whenNotPaused returns (uint256) {
        uint256 amount = withdrawFunds(msg.sender);
        // This is safe because it occurs after all checks and effects
        require(IERC20Upgradeable(stakeToken).transfer(msg.sender, amount), "TRANSFER_FAILED");
        return amount;
    }

    function receiveTokens(uint256 tokenAmount) private {
        require(IERC20Upgradeable(stakeToken).transferFrom(msg.sender, address(this), tokenAmount), "TRANSFER_FAIL");
    }
}
