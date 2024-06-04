// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/structs/EnumerableSetUpgradeable.sol";

import "./Assertion.sol";
import "./RollupLib.sol";
import "./IRollupEventInbox.sol";
import "./IRollupCore.sol";

import "../state/Machine.sol";

import "../bridge/ISequencerInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../challengeV2/EdgeChallengeManager.sol";
import "../libraries/ArbitrumChecker.sol";

abstract contract RollupCore is IRollupCore, PausableUpgradeable {
    using AssertionNodeLib for AssertionNode;
    using GlobalStateLib for GlobalState;
    using EnumerableSetUpgradeable for EnumerableSetUpgradeable.AddressSet;

    // Rollup Config
    uint256 public chainId;

    // These 4 config should be stored into the prev and not used directly
    // An assertion can be confirmed after confirmPeriodBlocks when it is unchallenged
    uint64 public confirmPeriodBlocks;

    // ------------------------------
    // STAKING
    // ------------------------------

    // Overall
    // ------------------------------
    // In order to create a new assertion the validator creating it must be staked. Only one stake
    // is needed per consistent lineage of assertions, so additional stakes must be placed when
    // lineages diverge.
    // As an example, for the following chain only one stake would be locked up in the C assertion
    // A -- B -- C
    // However for the following chain 2 stakes would be locked up, in C and in D
    // A -- B -- C
    //       \-- D
    // Since we know that only one assertion chain can be correct, we only need one stake available
    // to be refunded at any one time, and any more than one stake can be immediately confiscated.
    // So in the above situation although 2 stakes are not available to be withdrawn as they are locked
    // by C and D, only 1 stake needs to remain in the contract since one of the stakes will eventually
    // be confiscated anyway.
    // In practice, what we do here is increase the withdrawable amount of an escrow address that is
    // expected to be controlled by the rollup owner, whenever the lineage forks.

    // Moving stake
    // ------------------------------
    // Since we only need one stake per lineage we can lock the stake of the validator that last extended that
    // lineage. All other stakes within that lineage are then free to be moved to other lineages, or be withdrawn.
    // Additionally, it's inconsistent for a validator to stake on two different lineages, and as a validator
    // should only need to have one stake in the system at any one time.
    // In order to create a new assertion a validator needs to have free stake. Since stake is freed from an assertion
    // when another assertion builds on it, we know that if the assertion that was last staked on by a validator
    // has children, then that validator has free stake. Likewise, if the last staked assertion does not have children
    // but it is the parent of the assertion the validator is trying to create, then we know that by the time the assertion
    // is created it will have children, so we can allow this condition as well.

    // Updating stake amount
    // ------------------------------
    // The stake required to create an assertion can be updated by the rollup owner. A required stake value is stored on each
    // assertion, and shows how much stake is required to create the next assertion. Since we only store the last
    // assertion made by a validator, we don't know if it has previously staked on lower/higher amounts and
    // therefore offer partial withdrawals due to this difference. Instead we enforce that either all of the
    // validators stake is locked, or none of it.
    uint256 public baseStake;

    bytes32 public wasmModuleRoot;
    // When there is a challenge, we trust the challenge manager to determine the winner
    IEdgeChallengeManager public challengeManager;

    // If an assertion was challenged we leave an additional period after it could have completed
    // so that the result of a challenge is observable widely before it causes an assertion to be confirmed
    uint64 public challengeGracePeriodBlocks;

    IInboxBase public inbox;
    IBridge public bridge;
    IOutbox public outbox;
    IRollupEventInbox public rollupEventInbox;

    address public validatorWalletCreator;

    // only 1 child can be confirmed, the excess/loser stake will be sent to this address
    address public loserStakeEscrow;
    address public stakeToken;
    uint256 public minimumAssertionPeriod;

    EnumerableSetUpgradeable.AddressSet validators;

    bytes32 private _latestConfirmed;
    mapping(bytes32 => AssertionNode) private _assertions;

    address[] private _stakerList;
    mapping(address => Staker) public _stakerMap;

    mapping(address => uint256) private _withdrawableFunds;
    uint256 public totalWithdrawableFunds;
    uint256 public rollupDeploymentBlock;

    bool public validatorWhitelistDisabled;
    address public anyTrustFastConfirmer;

    // If the chain this RollupCore is deployed on is an Arbitrum chain.
    bool internal immutable _hostChainIsArbitrum = ArbitrumChecker.runningOnArbitrum();
    // If the chain RollupCore is deployed on, this will contain the ArbSys.blockNumber() at each node's creation.
    mapping(bytes32 => uint256) internal _assertionCreatedAtArbSysBlock;

    function sequencerInbox() public view virtual returns (ISequencerInbox) {
        return ISequencerInbox(bridge.sequencerInbox());
    }

    /**
     * @notice Get a storage reference to the Assertion for the given assertion hash
     * @dev The assertion may not exists
     * @param assertionHash Id of the assertion
     * @return Assertion struct
     */
    function getAssertionStorage(bytes32 assertionHash) internal view returns (AssertionNode storage) {
        require(assertionHash != bytes32(0), "ASSERTION_ID_CANNOT_BE_ZERO");
        return _assertions[assertionHash];
    }

    /**
     * @notice Get the Assertion for the given index.
     */
    function getAssertion(bytes32 assertionHash) public view override returns (AssertionNode memory) {
        return getAssertionStorage(assertionHash);
    }

    /**
     * @notice Returns the block in which the given assertion was created for looking up its creation event.
     * Unlike the assertion's createdAtBlock field, this will be the ArbSys blockNumber if the host chain is an Arbitrum chain.
     * That means that the block number returned for this is usable for event queries.
     * This function will revert if the given assertion hash does not exist.
     * @dev This function is meant for internal use only and has no stability guarantees.
     */
    function getAssertionCreationBlockForLogLookup(bytes32 assertionHash) external view override returns (uint256) {
        if (_hostChainIsArbitrum) {
            uint256 blockNum = _assertionCreatedAtArbSysBlock[assertionHash];
            require(blockNum > 0, "NO_ASSERTION");
            return blockNum;
        } else {
            AssertionNode storage assertion = getAssertionStorage(assertionHash);
            assertion.requireExists();
            return assertion.createdAtBlock;
        }
    }

    /**
     * @notice Get the address of the staker at the given index
     * @param stakerNum Index of the staker
     * @return Address of the staker
     */
    function getStakerAddress(uint64 stakerNum) external view override returns (address) {
        return _stakerList[stakerNum];
    }

    /**
     * @notice Check whether the given staker is staked
     * @param staker Staker address to check
     * @return True or False for whether the staker was staked
     */
    function isStaked(address staker) public view override returns (bool) {
        return _stakerMap[staker].isStaked;
    }

    /**
     * @notice Get the latest staked assertion of the given staker
     * @param staker Staker address to lookup
     * @return Latest assertion staked of the staker
     */
    function latestStakedAssertion(address staker) public view override returns (bytes32) {
        return _stakerMap[staker].latestStakedAssertion;
    }

    /**
     * @notice Get the amount staked of the given staker
     * @param staker Staker address to lookup
     * @return Amount staked of the staker
     */
    function amountStaked(address staker) public view override returns (uint256) {
        return _stakerMap[staker].amountStaked;
    }

    /**
     * @notice Get the withdrawal address of the given staker
     * @param staker Staker address to lookup
     * @return Withdrawal address of the staker
     */
    function withdrawalAddress(address staker) public view override returns (address) {
        return _stakerMap[staker].withdrawalAddress;
    }

    /**
     * @notice Retrieves stored information about a requested staker
     * @param staker Staker address to retrieve
     * @return A structure with information about the requested staker
     */
    function getStaker(address staker) external view override returns (Staker memory) {
        return _stakerMap[staker];
    }

    /**
     * @notice Get the amount of funds withdrawable by the given address
     * @param user Address to check the funds of
     * @return Amount of funds withdrawable by user
     */
    function withdrawableFunds(address user) external view override returns (uint256) {
        return _withdrawableFunds[user];
    }

    /// @return Index of the latest confirmed assertion
    function latestConfirmed() public view override returns (bytes32) {
        return _latestConfirmed;
    }

    /// @return Number of active stakers currently staked
    function stakerCount() public view override returns (uint64) {
        return uint64(_stakerList.length);
    }

    /**
     * @notice Initialize the core with an initial assertion
     * @param initialAssertion Initial assertion to start the chain with
     */
    function initializeCore(AssertionNode memory initialAssertion, bytes32 assertionHash) internal {
        __Pausable_init();
        initialAssertion.status = AssertionStatus.Confirmed;
        _assertions[assertionHash] = initialAssertion;
        _latestConfirmed = assertionHash;
    }

    /**
     * @dev This function will validate the parentAssertionHash, confirmState and inboxAcc against the assertionHash
     *          and check if the assertionHash is currently pending. If all checks pass, the assertion will be confirmed.
     */
    function confirmAssertionInternal(
        bytes32 assertionHash,
        bytes32 parentAssertionHash,
        AssertionState calldata confirmState,
        bytes32 inboxAcc
    ) internal {
        AssertionNode storage assertion = getAssertionStorage(assertionHash);
        // Check that assertion is pending, this also checks that assertion exists
        require(assertion.status == AssertionStatus.Pending, "NOT_PENDING");

        // Authenticate data against assertionHash pre-image
        require(
            assertionHash
                == RollupLib.assertionHash({
                    parentAssertionHash: parentAssertionHash,
                    afterState: confirmState,
                    inboxAcc: inboxAcc
                }),
            "CONFIRM_DATA"
        );

        bytes32 blockHash = confirmState.globalState.getBlockHash();
        bytes32 sendRoot = confirmState.globalState.getSendRoot();

        // trusted external call to outbox
        outbox.updateSendRoot(sendRoot, blockHash);

        _latestConfirmed = assertionHash;
        assertion.status = AssertionStatus.Confirmed;

        emit AssertionConfirmed(assertionHash, blockHash, sendRoot);
    }

    /**
     * @notice Create a new stake at latest confirmed assertion
     * @param stakerAddress Address of the new staker
     * @param depositAmount Stake amount of the new staker
     */
    function createNewStake(address stakerAddress, uint256 depositAmount, address _withdrawalAddress) internal {
        uint64 stakerIndex = uint64(_stakerList.length);
        _stakerList.push(stakerAddress);
        _stakerMap[stakerAddress] = Staker(depositAmount, _latestConfirmed, stakerIndex, true, _withdrawalAddress);
        emit UserStakeUpdated(stakerAddress, _withdrawalAddress, 0, depositAmount);
    }

    /**
     * @notice Add to the stake of the given staker by the given amount
     * @param stakerAddress Address of the staker to increase the stake of
     * @param amountAdded Amount of stake to add to the staker
     */
    function increaseStakeBy(address stakerAddress, uint256 amountAdded) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        uint256 initialStaked = staker.amountStaked;
        uint256 finalStaked = initialStaked + amountAdded;
        staker.amountStaked = finalStaked;
        emit UserStakeUpdated(stakerAddress, staker.withdrawalAddress, initialStaked, finalStaked);
    }

    /**
     * @notice Reduce the stake of the given staker to the given target
     * @param stakerAddress Address of the staker to reduce the stake of
     * @param target Amount of stake to leave with the staker
     * @return Amount of value released from the stake
     */
    function reduceStakeTo(address stakerAddress, uint256 target) internal returns (uint256) {
        Staker storage staker = _stakerMap[stakerAddress];
        address _withdrawalAddress = staker.withdrawalAddress;
        uint256 current = staker.amountStaked;
        require(target <= current, "TOO_LITTLE_STAKE");
        uint256 amountWithdrawn = current - target;
        staker.amountStaked = target;
        increaseWithdrawableFunds(_withdrawalAddress, amountWithdrawn);
        emit UserStakeUpdated(stakerAddress, _withdrawalAddress, current, target);
        return amountWithdrawn;
    }

    /**
     * @notice Remove the given staker and return their stake
     * This should only be called when the staker is inactive
     * @param stakerAddress Address of the staker withdrawing their stake
     */
    function withdrawStaker(address stakerAddress) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        address _withdrawalAddress = staker.withdrawalAddress;
        uint256 initialStaked = staker.amountStaked;
        increaseWithdrawableFunds(_withdrawalAddress, initialStaked);
        deleteStaker(stakerAddress);
        emit UserStakeUpdated(stakerAddress, _withdrawalAddress, initialStaked, 0);
    }

    /**
     * @notice Clear the withdrawable funds for the given address
     * @param account Address of the account to remove funds from
     * @return Amount of funds removed from account
     */
    function withdrawFunds(address account) internal returns (uint256) {
        uint256 amount = _withdrawableFunds[account];
        _withdrawableFunds[account] = 0;
        totalWithdrawableFunds -= amount;
        emit UserWithdrawableFundsUpdated(account, amount, 0);
        return amount;
    }

    /**
     * @notice Increase the withdrawable funds for the given address
     * @param account Address of the account to add withdrawable funds to
     */
    function increaseWithdrawableFunds(address account, uint256 amount) internal {
        uint256 initialWithdrawable = _withdrawableFunds[account];
        uint256 finalWithdrawable = initialWithdrawable + amount;
        _withdrawableFunds[account] = finalWithdrawable;
        totalWithdrawableFunds += amount;
        emit UserWithdrawableFundsUpdated(account, initialWithdrawable, finalWithdrawable);
    }

    /**
     * @notice Remove the given staker
     * @param stakerAddress Address of the staker to remove
     */
    function deleteStaker(address stakerAddress) private {
        Staker storage staker = _stakerMap[stakerAddress];
        require(staker.isStaked, "NOT_STAKED");
        uint64 stakerIndex = staker.index;
        _stakerList[stakerIndex] = _stakerList[_stakerList.length - 1];
        _stakerMap[_stakerList[stakerIndex]].index = stakerIndex;
        _stakerList.pop();
        delete _stakerMap[stakerAddress];
    }

    function createNewAssertion(
        AssertionInputs calldata assertion,
        bytes32 prevAssertionHash,
        bytes32 expectedAssertionHash
    ) internal returns (bytes32 newAssertionHash, bool overflowAssertion) {
        // Validate the config hash
        RollupLib.validateConfigHash(
            assertion.beforeStateData.configData, getAssertionStorage(prevAssertionHash).configHash
        );

        // reading inbox messages always terminates in either a finished or errored state
        // although the challenge protocol that any invalid terminal state will be proven incorrect
        // we can do a quick sanity check here
        require(
            assertion.afterState.machineStatus == MachineStatus.FINISHED
                || assertion.afterState.machineStatus == MachineStatus.ERRORED,
            "BAD_AFTER_STATUS"
        );

        // validate the provided before state is correct by checking that it's part of the prev assertion hash
        require(
            RollupLib.assertionHash(
                assertion.beforeStateData.prevPrevAssertionHash,
                assertion.beforeState,
                assertion.beforeStateData.sequencerBatchAcc
            ) == prevAssertionHash,
            "INVALID_BEFORE_STATE"
        );

        // The rollup cannot advance from an errored state
        // If it reaches an errored state it must be corrected by an administrator
        // This will involve updating the wasm root and creating an alternative assertion
        // that consumes the correct number of inbox messages, and correctly transitions to the
        // FINISHED state so that normal progress can continue
        require(assertion.beforeState.machineStatus == MachineStatus.FINISHED, "BAD_PREV_STATUS");

        AssertionNode storage prevAssertion = getAssertionStorage(prevAssertionHash);
        // Required inbox position through which the next assertion (the one after this new assertion) must consume
        uint256 nextInboxPosition;
        bytes32 sequencerBatchAcc;
        {
            // This new assertion consumes the messages from prevInboxPosition to afterInboxPosition
            GlobalState calldata afterGS = assertion.afterState.globalState;
            GlobalState calldata beforeGS = assertion.beforeState.globalState;

            // there are 3 kinds of assertions that can be made. Assertions must be made when they fill the maximum number
            // of blocks, or when they process all messages up to prev.nextInboxPosition. When they fill the max
            // blocks, but dont manage to process all messages, we call this an "overflow" assertion.
            // 1. ERRORED assertion
            //    The machine finished in an ERRORED state. This can happen with processing any
            //    messages, or moving the position in the message.
            // 2. FINISHED assertion that did not overflow
            //    The machine finished as normal, and fully processed all the messages up to prev.nextInboxPosition.
            //    In this case the inbox position must equal prev.nextInboxPosition and position in message must be 0
            // 3. FINISHED assertion that did overflow
            //    The machine finished as normal, but didn't process all messages in the inbox.
            //    The inbox can be anywhere between the previous assertion's position and the nextInboxPosition, exclusive.

            //    All types of assertion must have inbox position in the range prev.inboxPosition <= x <= prev.nextInboxPosition
            require(afterGS.comparePositions(beforeGS) >= 0, "INBOX_BACKWARDS");
            int256 afterStateCmpMaxInbox =
                afterGS.comparePositionsAgainstStartOfBatch(assertion.beforeStateData.configData.nextInboxPosition);
            require(afterStateCmpMaxInbox <= 0, "INBOX_TOO_FAR");

            if (assertion.afterState.machineStatus != MachineStatus.ERRORED && afterStateCmpMaxInbox < 0) {
                // If we didn't reach the target next inbox position, this is an overflow assertion.
                overflowAssertion = true;
                // This shouldn't be necessary, but might as well constrain the assertion to be non-empty
                require(afterGS.comparePositions(beforeGS) > 0, "OVERFLOW_STANDSTILL");
            }
            // Inbox position at the time of this assertion being created
            uint256 currentInboxPosition = bridge.sequencerMessageCount();
            // Cannot read more messages than currently exist in the inbox
            require(afterGS.comparePositionsAgainstStartOfBatch(currentInboxPosition) <= 0, "INBOX_PAST_END");

            // under normal circumstances prev.nextInboxPosition is guaranteed to exist
            // because we populate it from bridge.sequencerMessageCount(). However, when
            // the inbox message count doesnt change we artificially increase it by 1 as explained below
            // in this case we need to ensure when the assertion is made the inbox messages are available
            // to ensure that a valid assertion can actually be made.
            require(
                assertion.beforeStateData.configData.nextInboxPosition <= currentInboxPosition, "INBOX_NOT_POPULATED"
            );

            // The next assertion must consume all the messages that are currently found in the inbox
            uint256 afterInboxPosition = afterGS.getInboxPosition();
            if (afterInboxPosition == currentInboxPosition) {
                // No new messages have been added to the inbox since the last assertion
                // In this case if we set the next inbox position to the current one we would be insisting that
                // the next assertion process no messages. So instead we increment the next inbox position to current
                // plus one, so that the next assertion will process exactly one message.
                // Thus, no assertion can be empty (except the genesis assertion, which is created
                // via a different codepath).
                nextInboxPosition = currentInboxPosition + 1;
            } else {
                nextInboxPosition = currentInboxPosition;
            }

            // only the genesis assertion processes no messages, and that assertion is created
            // when we initialize this contract. Therefore, all assertions created here should have a non
            // zero inbox position.
            require(afterInboxPosition != 0, "EMPTY_INBOX_COUNT");

            // Fetch the inbox accumulator for this message count. Fetching this and checking against it
            // allows the assertion creator to ensure they're creating an assertion against the expected
            // inbox messages
            sequencerBatchAcc = bridge.sequencerInboxAccs(afterInboxPosition - 1);
        }

        newAssertionHash = RollupLib.assertionHash(prevAssertionHash, assertion.afterState, sequencerBatchAcc);

        // allow an assertion creator to ensure that they're creating their assertion against the expected state
        require(
            newAssertionHash == expectedAssertionHash || expectedAssertionHash == bytes32(0),
            "UNEXPECTED_ASSERTION_HASH"
        );

        // the assertion hash is unique - it's only possible to have one correct assertion hash
        // per assertion. Therefore we can check if this assertion has already been made, and if so
        // we can revert
        require(getAssertionStorage(newAssertionHash).status == AssertionStatus.NoAssertion, "ASSERTION_SEEN");

        // state updates
        AssertionNode memory newAssertion = AssertionNodeLib.createAssertion(
            prevAssertion.firstChildBlock == 0, // assumes block 0 is impossible
            RollupLib.configHash({
                wasmModuleRoot: wasmModuleRoot,
                requiredStake: baseStake,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: confirmPeriodBlocks,
                nextInboxPosition: uint64(nextInboxPosition)
            })
        );

        // Fetch a storage reference to prevAssertion since we copied our other one into memory
        // and we don't have enough stack available to keep to keep the previous storage reference around
        prevAssertion.childCreated();
        _assertions[newAssertionHash] = newAssertion;

        emit AssertionCreated(
            newAssertionHash,
            prevAssertionHash,
            assertion,
            sequencerBatchAcc,
            nextInboxPosition,
            wasmModuleRoot,
            baseStake,
            address(challengeManager),
            confirmPeriodBlocks
        );
        if (_hostChainIsArbitrum) {
            _assertionCreatedAtArbSysBlock[newAssertionHash] = ArbSys(address(100)).arbBlockNumber();
        }
    }

    function genesisAssertionHash() external pure returns (bytes32) {
        GlobalState memory emptyGlobalState;
        AssertionState memory emptyAssertionState = AssertionState(emptyGlobalState, MachineStatus.FINISHED, bytes32(0));
        bytes32 parentAssertionHash = bytes32(0);
        bytes32 inboxAcc = bytes32(0);
        return RollupLib.assertionHash({
            parentAssertionHash: parentAssertionHash,
            afterState: emptyAssertionState,
            inboxAcc: inboxAcc
        });
    }

    function getFirstChildCreationBlock(bytes32 assertionHash) external view returns (uint64) {
        return getAssertionStorage(assertionHash).firstChildBlock;
    }

    function getSecondChildCreationBlock(bytes32 assertionHash) external view returns (uint64) {
        return getAssertionStorage(assertionHash).secondChildBlock;
    }

    function validateAssertionHash(
        bytes32 assertionHash,
        AssertionState calldata state,
        bytes32 prevAssertionHash,
        bytes32 inboxAcc
    ) external pure {
        require(assertionHash == RollupLib.assertionHash(prevAssertionHash, state, inboxAcc), "INVALID_ASSERTION_HASH");
    }

    function validateConfig(bytes32 assertionHash, ConfigData calldata configData) external view {
        RollupLib.validateConfigHash(configData, getAssertionStorage(assertionHash).configHash);
    }

    function isFirstChild(bytes32 assertionHash) external view returns (bool) {
        return getAssertionStorage(assertionHash).isFirstChild;
    }

    function isPending(bytes32 assertionHash) external view returns (bool) {
        return getAssertionStorage(assertionHash).status == AssertionStatus.Pending;
    }

    function getValidators() external view returns (address[] memory) {
        return validators.values();
    }

    function isValidator(address validator) external view returns (bool) {
        return validators.contains(validator);
    }

    /**
     * @notice Verify that the given staker is not active
     * @param stakerAddress Address to check
     */
    function requireInactiveStaker(address stakerAddress) internal view {
        require(isStaked(stakerAddress), "NOT_STAKED");
        // A staker is inactive if
        // a) their last staked assertion is the latest confirmed assertion
        // b) their last staked assertion have a child
        bytes32 lastestAssertion = latestStakedAssertion(stakerAddress);
        bool isLatestConfirmed = lastestAssertion == latestConfirmed();
        bool haveChild = getAssertionStorage(lastestAssertion).firstChildBlock > 0;
        require(isLatestConfirmed || haveChild, "STAKE_ACTIVE");
    }
}
