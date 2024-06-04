// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import {IRollupUser} from "./IRollupLogic.sol";
import "../libraries/UUPSNotUpgradeable.sol";
import "./RollupCore.sol";
import "./IRollupLogic.sol";
import {ETH_POS_BLOCK_TIME} from "../libraries/Constants.sol";

contract RollupUserLogic is RollupCore, UUPSNotUpgradeable, IRollupUser {
    using AssertionNodeLib for AssertionNode;
    using GlobalStateLib for GlobalState;
    using SafeERC20 for IERC20;
    using EnumerableSetUpgradeable for EnumerableSetUpgradeable.AddressSet;

    modifier onlyValidator() {
        require(validators.contains(msg.sender) || validatorWhitelistDisabled, "NOT_VALIDATOR");
        _;
    }

    /// @dev the user logic just validated configuration and shouldn't write to state during init
    /// this allows the admin logic to ensure consistency on parameters.
    function initialize(address _stakeToken) external view override onlyProxy {
        require(_stakeToken != address(0), "NEED_STAKE_TOKEN");
    }

    uint256 internal immutable deployTimeChainId = block.chainid;

    function _chainIdChanged() internal view returns (bool) {
        return deployTimeChainId != block.chainid;
    }

    /**
     * @notice Number of blocks since the last confirmed assertion before the validator whitelist is removed
     *         This is 28 days assuming a 12 seconds block time. Since it can take 14 days under normal
     *         circumstances to confirm an assertion, this means that validators will have been inactive for
     *         a further 14 days before the validator whitelist is removed.
     *
     *         It's important that this time is greater than the max amount of time it can take to
     *         to confirm an assertion via the normal method. Therefore we need it to be greater
     *         than max(2* confirmPeriod, 2 * challengePeriod). With some additional margin
     *         It's expected that initially confirm and challenge periods are set to 1 week, so 4 weeks
     *         should give a two weeks of margin before the validators are considered afk.
     */
    uint256 public constant VALIDATOR_AFK_BLOCKS = 201600;

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

    /**
     * @notice Confirm a unresolved assertion
     * @param confirmState The state to confirm
     * @param winningEdgeId The winning edge if a challenge is started
     */
    function confirmAssertion(
        bytes32 assertionHash,
        bytes32 prevAssertionHash,
        AssertionState calldata confirmState,
        bytes32 winningEdgeId,
        ConfigData calldata prevConfig,
        bytes32 inboxAcc
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

        // The assertion's must exists and be pending and will be validated in RollupCore.confirmAssertionInternal
        AssertionNode storage assertion = getAssertionStorage(assertionHash);

        // prevAssertionHash is user supplied, but will be validated in RollupCore.confirmAssertionInternal
        AssertionNode storage prevAssertion = getAssertionStorage(prevAssertionHash);
        RollupLib.validateConfigHash(prevConfig, prevAssertion.configHash);

        // Check that deadline has passed
        require(block.number >= assertion.createdAtBlock + prevConfig.confirmPeriodBlocks, "BEFORE_DEADLINE");

        // Check that prev is latest confirmed
        require(prevAssertionHash == latestConfirmed(), "PREV_NOT_LATEST_CONFIRMED");

        if (prevAssertion.secondChildBlock > 0) {
            // if the prev has more than 1 child, check if this assertion is the challenge winner
            ChallengeEdge memory winningEdge = IEdgeChallengeManager(prevConfig.challengeManager).getEdge(winningEdgeId);
            require(winningEdge.claimId == assertionHash, "NOT_WINNER");
            require(winningEdge.status == EdgeStatus.Confirmed, "EDGE_NOT_CONFIRMED");
            require(winningEdge.confirmedAtBlock != 0, "ZERO_CONFIRMED_AT_BLOCK");
            // an additional number of blocks is added to ensure that the result of the challenge is widely
            // observable before it causes an assertion to be confirmed. After a winning edge is found, it will
            // always be challengeGracePeriodBlocks before an assertion can be confirmed
            require(
                block.number >= winningEdge.confirmedAtBlock + challengeGracePeriodBlocks,
                "CHALLENGE_GRACE_PERIOD_NOT_PASSED"
            );
        }

        confirmAssertionInternal(assertionHash, prevAssertionHash, confirmState, inboxAcc);
    }

    /**
     * @notice Create a new stake
     * @param depositAmount The amount of either eth or tokens staked
     */
    function _newStake(uint256 depositAmount, address withdrawalAddress) internal onlyValidator whenNotPaused {
        // Verify that sender is not already a staker
        require(!isStaked(msg.sender), "ALREADY_STAKED");
        // amount will be checked when creating an assertion
        createNewStake(msg.sender, depositAmount, withdrawalAddress);
    }

    /**
     * @notice Computes the hash of an assertion
     * @param state The execution state for the assertion
     * @param prevAssertionHash The hash of the assertion's parent
     * @param inboxAcc The inbox batch accumulator
     */
    function computeAssertionHash(bytes32 prevAssertionHash, AssertionState calldata state, bytes32 inboxAcc)
        external
        pure
        returns (bytes32)
    {
        return RollupLib.assertionHash(prevAssertionHash, state, inboxAcc);
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
        require(amountStaked(msg.sender) >= assertion.beforeStateData.configData.requiredStake, "INSUFFICIENT_STAKE");

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

        (bytes32 newAssertionHash, bool overflowAssertion) =
            createNewAssertion(assertion, prevAssertion, expectedAssertionHash);
        _stakerMap[msg.sender].latestStakedAssertion = newAssertionHash;

        if (!overflowAssertion) {
            uint256 timeSincePrev = block.number - getAssertionStorage(prevAssertion).createdAtBlock;
            // Verify that assertion meets the minimum Delta time requirement
            require(timeSincePrev >= minimumAssertionPeriod, "TIME_DELTA");
        }

        if (!getAssertionStorage(newAssertionHash).isFirstChild) {
            // We assume assertion.beforeStateData is valid here as it will be validated in createNewAssertion
            // only 1 of the children can be confirmed and get their stake refunded
            // so we send the other children's stake to the loserStakeEscrow
            // NOTE: if the losing staker have staked more than requiredStake, the excess stake will be stuck
            IERC20(stakeToken).safeTransfer(loserStakeEscrow, assertion.beforeStateData.configData.requiredStake);
        }
    }

    /**
     * @notice Refund a staker that is currently staked on an assertion that either has a chlid assertion or is the latest confirmed assertion.
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

    /**
     * @notice This allow the anyTrustFastConfirmer to force confirm any pending assertion
     *         the anyTrustFastConfirmer is supposed to be set only on an AnyTrust chain to
     *         a contract that can call this function when received sufficient signatures
     */
    function fastConfirmAssertion(
        bytes32 assertionHash,
        bytes32 parentAssertionHash,
        AssertionState calldata confirmState,
        bytes32 inboxAcc
    ) public whenNotPaused {
        require(msg.sender == anyTrustFastConfirmer, "NOT_FAST_CONFIRMER");
        // this skip deadline, prev, challenge validations
        confirmAssertionInternal(assertionHash, parentAssertionHash, confirmState, inboxAcc);
    }

    /**
     * @notice This allow the anyTrustFastConfirmer to immediately create and confirm an assertion
     *         the anyTrustFastConfirmer is supposed to be set only on an AnyTrust chain to
     *         a contract that can call this function when received sufficient signatures
     *         The logic in this function is similar to stakeOnNewAssertion, but without staker checks
     *
     *         We trust the anyTrustFastConfirmer to not call this function multiple times on the same prev,
     *         as doing so would result in incorrect accounting of withdrawable funds in the loserStakeEscrow.
     *         This is because the protocol assume there is only 1 unique confirmable child assertion.
     */
    function fastConfirmNewAssertion(AssertionInputs calldata assertion, bytes32 expectedAssertionHash)
        external
        whenNotPaused
    {
        // Must supply expectedAssertionHash to fastConfirmNewAssertion
        require(expectedAssertionHash != bytes32(0), "EXPECTED_ASSERTION_HASH");
        AssertionStatus status = getAssertionStorage(expectedAssertionHash).status;

        bytes32 prevAssertion = RollupLib.assertionHash(
            assertion.beforeStateData.prevPrevAssertionHash,
            assertion.beforeState,
            assertion.beforeStateData.sequencerBatchAcc
        );
        getAssertionStorage(prevAssertion).requireExists();

        if (status == AssertionStatus.NoAssertion) {
            // If not exists, we create the new assertion
            (bytes32 newAssertionHash,) = createNewAssertion(assertion, prevAssertion, expectedAssertionHash);
            if (!getAssertionStorage(newAssertionHash).isFirstChild) {
                // only 1 of the children can be confirmed and get their stake refunded
                // so we send the other children's stake to the loserStakeEscrow
                // NOTE: if the losing staker have staked more than requiredStake, the excess stake will be stuck
                IERC20(stakeToken).safeTransfer(loserStakeEscrow, assertion.beforeStateData.configData.requiredStake);
            }
        }

        // This would revert if the assertion is already confirmed
        fastConfirmAssertion(
            expectedAssertionHash,
            prevAssertion,
            assertion.afterState,
            bridge.sequencerInboxAccs(assertion.afterState.globalState.getInboxPosition() - 1)
        );
    }

    function owner() external view returns (address) {
        return _getAdmin();
    }

    /**
     * @notice Deprecated, use the function with `withdrawalAddress` instead
     *         Using this default `withdrawalAddress` to msg.sender
     */
    function newStakeOnNewAssertion(
        uint256 tokenAmount,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash
    ) external {
        newStakeOnNewAssertion(tokenAmount, assertion, expectedAssertionHash, msg.sender);
    }

    /**
     * @notice Create a new stake on a new assertion
     * @param tokenAmount Amount of the rollups staking token to stake
     * @param assertion Assertion describing the state change between the old assertion and the new one
     * @param expectedAssertionHash Assertion hash of the assertion that will be created
     * @param withdrawalAddress The address the send the stake back upon withdrawal
     */
    function newStakeOnNewAssertion(
        uint256 tokenAmount,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        address withdrawalAddress
    ) public {
        require(withdrawalAddress != address(0), "EMPTY_WITHDRAWAL_ADDRESS");
        _newStake(tokenAmount, withdrawalAddress);
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
        require(amount > 0, "NO_FUNDS_TO_WITHDRAW");
        // This is safe because it occurs after all checks and effects
        IERC20(stakeToken).safeTransfer(msg.sender, amount);
        return amount;
    }

    function receiveTokens(uint256 tokenAmount) private {
        IERC20(stakeToken).safeTransferFrom(msg.sender, address(this), tokenAmount);
    }
}
