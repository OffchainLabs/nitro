// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import "./Assertion.sol";
import "./IRollupCore.sol";
import "./RollupLib.sol";
import "./IRollupEventInbox.sol";
import "./IRollupCore.sol";

import "../challenge/IOldChallengeManager.sol";

import "../bridge/ISequencerInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../challengeV2/ChallengeManagerImpl.sol";
import "../challengeV2/DataEntities.sol";
import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

abstract contract RollupCore is IRollupCore, PausableUpgradeable, IAssertionChain {
    using AssertionNodeLib for AssertionNode;
    using GlobalStateLib for GlobalState;

    // Rollup Config
    uint64 public confirmPeriodBlocks;
    uint64 public extraChallengeTimeBlocks;
    uint256 public chainId;
    uint256 public baseStake;
    bytes32 public wasmModuleRoot;

    IInbox public inbox;
    IBridge public bridge;
    IOutbox public outbox;
    ISequencerInbox public sequencerInbox;
    IRollupEventInbox public rollupEventInbox;
    IOldChallengeManager public override oldChallengeManager;

    // misc useful contracts when interacting with the rollup
    address public validatorUtils;
    address public validatorWalletCreator;

    // when a staker loses a challenge, half of their funds get escrowed in this address
    address public loserStakeEscrow;
    address public stakeToken;
    uint256 public minimumAssertionPeriod;

    mapping(address => bool) public isValidator;

    // Stakers become Zombies after losing a challenge
    struct Zombie {
        address stakerAddress;
        uint64 latestStakedAssertion;
    }

    uint64 private _latestConfirmed;
    uint64 private _firstUnresolvedAssertion;
    uint64 private _latestAssertionCreated;
    uint64 private _lastStakeBlock;
    mapping(uint64 => AssertionNode) private _assertions;
    mapping(uint64 => mapping(address => bool)) private _assertionStakers;

    address[] private _stakerList;
    mapping(address => Staker) public _stakerMap;

    Zombie[] private _zombies;

    mapping(address => uint256) private _withdrawableFunds;
    uint256 public totalWithdrawableFunds;
    uint256 public rollupDeploymentBlock;

    // The assertion number of the initial assertion
    uint64 internal constant GENESIS_NODE = 0;

    bool public validatorWhitelistDisabled;

    IChallengeManager public challengeManager;

    /**
     * @notice Get a storage reference to the Assertion for the given assertion index
     * @param assertionNum Index of the assertion
     * @return Assertion struct
     */
    function getAssertionStorage(uint64 assertionNum) internal view returns (AssertionNode storage) {
        return _assertions[assertionNum];
    }

    /**
     * @notice Get the Assertion for the given index.
     */
    function getAssertion(uint64 assertionNum) public view override returns (AssertionNode memory) {
        return getAssertionStorage(assertionNum);
    }

    /**
     * @notice Check if the specified assertion has been staked on by the provided staker.
     * Only accurate at the latest confirmed assertion and afterwards.
     */
    function assertionHasStaker(uint64 assertionNum, address staker) public view override returns (bool) {
        return _assertionStakers[assertionNum][staker];
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
     * @notice Check whether the given staker is staked on the latest confirmed assertion,
     * which includes if the staker is staked on a descendent of the latest confirmed assertion.
     * @param staker Staker address to check
     * @return True or False for whether the staker was staked
     */
    function isStakedOnLatestConfirmed(address staker) public view returns (bool) {
        return _stakerMap[staker].isStaked && assertionHasStaker(_latestConfirmed, staker);
    }

    /**
     * @notice Get the latest staked assertion of the given staker
     * @param staker Staker address to lookup
     * @return Latest assertion staked of the staker
     */
    function latestStakedAssertion(address staker) public view override returns (uint64) {
        return _stakerMap[staker].latestStakedAssertion;
    }

    /**
     * @notice Get the current challenge of the given staker
     * @param staker Staker address to lookup
     * @return Current challenge of the staker
     */
    function currentChallenge(address staker) public view override returns (uint64) {
        return _stakerMap[staker].currentChallenge;
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
     * @notice Retrieves stored information about a requested staker
     * @param staker Staker address to retrieve
     * @return A structure with information about the requested staker
     */
    function getStaker(address staker) external view override returns (Staker memory) {
        return _stakerMap[staker];
    }

    /**
     * @notice Get the original staker address of the zombie at the given index
     * @param zombieNum Index of the zombie to lookup
     * @return Original staker address of the zombie
     */
    function zombieAddress(uint256 zombieNum) public view override returns (address) {
        return _zombies[zombieNum].stakerAddress;
    }

    /**
     * @notice Get Latest assertion that the given zombie at the given index is staked on
     * @param zombieNum Index of the zombie to lookup
     * @return Latest assertion that the given zombie is staked on
     */
    function zombieLatestStakedAssertion(uint256 zombieNum) public view override returns (uint64) {
        return _zombies[zombieNum].latestStakedAssertion;
    }

    /**
     * @notice Retrieves stored information about a requested zombie
     * @param zombieNum Index of the zombie to lookup
     * @return A structure with information about the requested staker
     */
    function getZombieStorage(uint256 zombieNum) internal view returns (Zombie storage) {
        return _zombies[zombieNum];
    }

    /// @return Current number of un-removed zombies
    function zombieCount() public view override returns (uint256) {
        return _zombies.length;
    }

    function isZombie(address staker) public view override returns (bool) {
        for (uint256 i = 0; i < _zombies.length; i++) {
            if (staker == _zombies[i].stakerAddress) {
                return true;
            }
        }
        return false;
    }

    /**
     * @notice Get the amount of funds withdrawable by the given address
     * @param user Address to check the funds of
     * @return Amount of funds withdrawable by user
     */
    function withdrawableFunds(address user) external view override returns (uint256) {
        return _withdrawableFunds[user];
    }

    /**
     * @return Index of the first unresolved assertion
     * @dev If all assertions have been resolved, this will be latestAssertionCreated + 1
     */
    function firstUnresolvedAssertion() public view override returns (uint64) {
        return _firstUnresolvedAssertion;
    }

    /// @return Index of the latest confirmed assertion
    function latestConfirmed() public view override returns (uint64) {
        return _latestConfirmed;
    }

    /// @return Index of the latest rollup assertion created
    function latestAssertionCreated() public view override returns (uint64) {
        return _latestAssertionCreated;
    }

    /// @return Ethereum block that the most recent stake was created
    function lastStakeBlock() external view override returns (uint64) {
        return _lastStakeBlock;
    }

    /// @return Number of active stakers currently staked
    function stakerCount() public view override returns (uint64) {
        return uint64(_stakerList.length);
    }

    /**
     * @notice Initialize the core with an initial assertion
     * @param initialAssertion Initial assertion to start the chain with
     */
    function initializeCore(AssertionNode memory initialAssertion) internal {
        __Pausable_init();
        _assertions[GENESIS_NODE] = initialAssertion;
        _firstUnresolvedAssertion = GENESIS_NODE + 1;
    }

    /**
     * @notice React to a new assertion being created by storing it an incrementing the latest assertion counter
     * @param assertion Assertion that was newly created
     */
    function assertionCreated(AssertionNode memory assertion) internal {
        _latestAssertionCreated++;
        _assertions[_latestAssertionCreated] = assertion;
    }

    /// @notice Reject the next unresolved assertion
    function _rejectNextAssertion() internal {
        _firstUnresolvedAssertion++;
    }

    function confirmAssertion(
        uint64 assertionNum,
        bytes32 blockHash,
        bytes32 sendRoot
    ) internal {
        AssertionNode storage assertion = getAssertionStorage(assertionNum);
        // Authenticate data against assertion's confirm data pre-image
        require(assertion.confirmData == RollupLib.confirmHash(blockHash, sendRoot), "CONFIRM_DATA");

        // trusted external call to outbox
        outbox.updateSendRoot(sendRoot, blockHash);

        _latestConfirmed = assertionNum;
        _firstUnresolvedAssertion = assertionNum + 1;

        emit AssertionConfirmed(assertionNum, blockHash, sendRoot);
    }

    /**
     * @notice Create a new stake at latest confirmed assertion
     * @param stakerAddress Address of the new staker
     * @param depositAmount Stake amount of the new staker
     */
    function createNewStake(address stakerAddress, uint256 depositAmount) internal {
        uint64 stakerIndex = uint64(_stakerList.length);
        _stakerList.push(stakerAddress);
        _stakerMap[stakerAddress] = Staker(
            depositAmount,
            stakerIndex,
            _latestConfirmed,
            NO_CHAL_INDEX, // new staker is not in challenge
            true
        );
        _assertionStakers[_latestConfirmed][stakerAddress] = true;
        _lastStakeBlock = uint64(block.number);
        emit UserStakeUpdated(stakerAddress, 0, depositAmount);
    }

    /**
     * @notice Check to see whether the two stakers are in the same challenge
     * @param stakerAddress1 Address of the first staker
     * @param stakerAddress2 Address of the second staker
     * @return Address of the challenge that the two stakers are in
     */
    function inChallenge(address stakerAddress1, address stakerAddress2)
        internal
        view
        returns (uint64)
    {
        Staker storage staker1 = _stakerMap[stakerAddress1];
        Staker storage staker2 = _stakerMap[stakerAddress2];
        uint64 challenge = staker1.currentChallenge;
        require(challenge != NO_CHAL_INDEX, "NO_CHAL");
        require(challenge == staker2.currentChallenge, "DIFF_IN_CHAL");
        return challenge;
    }

    /**
     * @notice Make the given staker as not being in a challenge
     * @param stakerAddress Address of the staker to remove from a challenge
     */
    function clearChallenge(address stakerAddress) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        staker.currentChallenge = NO_CHAL_INDEX;
    }

    /**
     * @notice Mark both the given stakers as engaged in the challenge
     * @param staker1 Address of the first staker
     * @param staker2 Address of the second staker
     * @param challenge Address of the challenge both stakers are now in
     */
    function challengeStarted(
        address staker1,
        address staker2,
        uint64 challenge
    ) internal {
        _stakerMap[staker1].currentChallenge = challenge;
        _stakerMap[staker2].currentChallenge = challenge;
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
        emit UserStakeUpdated(stakerAddress, initialStaked, finalStaked);
    }

    /**
     * @notice Reduce the stake of the given staker to the given target
     * @param stakerAddress Address of the staker to reduce the stake of
     * @param target Amount of stake to leave with the staker
     * @return Amount of value released from the stake
     */
    function reduceStakeTo(address stakerAddress, uint256 target) internal returns (uint256) {
        Staker storage staker = _stakerMap[stakerAddress];
        uint256 current = staker.amountStaked;
        require(target <= current, "TOO_LITTLE_STAKE");
        uint256 amountWithdrawn = current - target;
        staker.amountStaked = target;
        increaseWithdrawableFunds(stakerAddress, amountWithdrawn);
        emit UserStakeUpdated(stakerAddress, current, target);
        return amountWithdrawn;
    }

    /**
     * @notice Remove the given staker and turn them into a zombie
     * @param stakerAddress Address of the staker to remove
     */
    function turnIntoZombie(address stakerAddress) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        _zombies.push(Zombie(stakerAddress, staker.latestStakedAssertion));
        deleteStaker(stakerAddress);
    }

    /**
     * @notice Update the latest staked assertion of the zombie at the given index
     * @param zombieNum Index of the zombie to move
     * @param latest New latest assertion the zombie is staked on
     */
    function zombieUpdateLatestStakedAssertion(uint256 zombieNum, uint64 latest) internal {
        _zombies[zombieNum].latestStakedAssertion = latest;
    }

    /**
     * @notice Remove the zombie at the given index
     * @param zombieNum Index of the zombie to remove
     */
    function removeZombie(uint256 zombieNum) internal {
        _zombies[zombieNum] = _zombies[_zombies.length - 1];
        _zombies.pop();
    }

    /**
     * @notice Mark the given staker as staked on this assertion
     * @param staker Address of the staker to mark
     */
    function addStaker(uint64 assertionNum, address staker) internal {
        require(!_assertionStakers[assertionNum][staker], "ALREADY_STAKED");
        _assertionStakers[assertionNum][staker] = true;
        AssertionNode storage assertion = getAssertionStorage(assertionNum);
        require(assertion.deadlineBlock != 0, "NO_NODE");

        uint64 prevCount = assertion.stakerCount;
        assertion.stakerCount = prevCount + 1;

        if (assertionNum > GENESIS_NODE) {
            AssertionNode storage parent = getAssertionStorage(assertion.prevNum);
            parent.childStakerCount++;
            // if (prevCount == 0) {
            //     parent.newChildConfirmDeadline(uint64(block.number) + confirmPeriodBlocks);
            // }
        }
    }

    /**
     * @notice Remove the given staker from this assertion
     * @param staker Address of the staker to remove
     */
    function removeStaker(uint64 assertionNum, address staker) internal {
        require(_assertionStakers[assertionNum][staker], "NOT_STAKED");
        _assertionStakers[assertionNum][staker] = false;

        AssertionNode storage assertion = getAssertionStorage(assertionNum);
        assertion.stakerCount--;

        if (assertionNum > GENESIS_NODE) {
            getAssertionStorage(assertion.prevNum).childStakerCount--;
        }
    }

    /**
     * @notice Remove the given staker and return their stake
     * This should not be called if the staker is staked on a descendent of the latest confirmed assertion
     * @param stakerAddress Address of the staker withdrawing their stake
     */
    function withdrawStaker(address stakerAddress) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        uint64 latestConfirmedNum = latestConfirmed();
        if (assertionHasStaker(latestConfirmedNum, stakerAddress)) {
            // Withdrawing a staker whose latest staked assertion isn't resolved should be impossible
            assert(staker.latestStakedAssertion == latestConfirmedNum);
            removeStaker(latestConfirmedNum, stakerAddress);
        }
        uint256 initialStaked = staker.amountStaked;
        increaseWithdrawableFunds(stakerAddress, initialStaked);
        deleteStaker(stakerAddress);
        emit UserStakeUpdated(stakerAddress, initialStaked, 0);
    }

    /**
     * @notice Advance the given staker to the given assertion
     * @param stakerAddress Address of the staker adding their stake
     * @param assertionNum Index of the assertion to stake on
     */
    function stakeOnAssertion(address stakerAddress, uint64 assertionNum) internal {
        Staker storage staker = _stakerMap[stakerAddress];
        addStaker(assertionNum, stakerAddress);
        staker.latestStakedAssertion = assertionNum;
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

    struct StakeOnNewAssertionFrame {
        uint256 currentInboxSize;
        AssertionNode assertion;
        bytes32 executionHash;
        AssertionNode prevAssertion;
        bytes32 lastHash;
        bool hasSibling;
        uint64 deadlineBlock;
        bytes32 sequencerBatchAcc;
    }

    function createNewAssertion(
        AssertionInputs calldata assertion,
        uint64 prevAssertionNum,
        uint256 prevAssertionInboxMaxCount,
        bytes32 expectedAssertionHash
    ) internal returns (bytes32 newAssertionHash) {
        require(
            assertion.afterState.machineStatus == MachineStatus.FINISHED ||
                assertion.afterState.machineStatus == MachineStatus.ERRORED,
            "BAD_AFTER_STATUS"
        );

        StakeOnNewAssertionFrame memory memoryFrame;
        {
            // validate data
            memoryFrame.prevAssertion = getAssertion(prevAssertionNum);
            memoryFrame.currentInboxSize = bridge.sequencerMessageCount();

            // Make sure the previous state is correct against the assertion being built on
            require(
                RollupLib.stateHash(assertion.beforeState, prevAssertionInboxMaxCount) ==
                    memoryFrame.prevAssertion.stateHash,
                "PREV_STATE_HASH"
            );

            // Ensure that the assertion doesn't read past the end of the current inbox
            uint64 afterInboxCount = assertion.afterState.globalState.getInboxPosition();
            uint64 prevInboxPosition = assertion.beforeState.globalState.getInboxPosition();
            require(afterInboxCount >= prevInboxPosition, "INBOX_BACKWARDS");
            if (afterInboxCount == prevInboxPosition) {
                require(
                    assertion.afterState.globalState.getPositionInMessage() >=
                        assertion.beforeState.globalState.getPositionInMessage(),
                    "INBOX_POS_IN_MSG_BACKWARDS"
                );
            }
            // See validator/assertion.go ExecutionState RequiredBatches() for reasoning
            if (
                assertion.afterState.machineStatus == MachineStatus.ERRORED ||
                assertion.afterState.globalState.getPositionInMessage() > 0
            ) {
                // The current inbox message was read
                afterInboxCount++;
            }
            require(afterInboxCount <= memoryFrame.currentInboxSize, "INBOX_PAST_END");
            // This gives replay protection against the state of the inbox
            if (afterInboxCount > 0) {
                memoryFrame.sequencerBatchAcc = bridge.sequencerInboxAccs(afterInboxCount - 1);
            }
        }

        {
            memoryFrame.executionHash = RollupLib.executionHash(assertion);

            memoryFrame.deadlineBlock = uint64(block.number) + confirmPeriodBlocks;

            memoryFrame.hasSibling = memoryFrame.prevAssertion.firstChildBlock > 0;
            // here we don't use ternacy operator to remain compatible with slither
            // if (memoryFrame.hasSibling) {
            //     memoryFrame.lastHash = getAssertionStorage(memoryFrame.prevAssertion.latestChildNumber)
            //         .assertionHash;
            // } else {
            //     memoryFrame.lastHash = memoryFrame.prevAssertion.assertionHash;
            // }
            // HN: TODO: is this ok?
            memoryFrame.lastHash = memoryFrame.prevAssertion.assertionHash;

            newAssertionHash = RollupLib.assertionHash(
                memoryFrame.hasSibling,
                memoryFrame.lastHash,
                memoryFrame.executionHash,
                memoryFrame.sequencerBatchAcc,
                wasmModuleRoot
            );
            require(
                newAssertionHash == expectedAssertionHash || expectedAssertionHash == bytes32(0),
                "UNEXPECTED_NODE_HASH"
            );

            memoryFrame.assertion = AssertionNodeLib.createAssertion(
                RollupLib.stateHash(assertion.afterState, memoryFrame.currentInboxSize),
                RollupLib.challengeRootHash(
                    memoryFrame.executionHash,
                    block.number,
                    wasmModuleRoot
                ),
                RollupLib.confirmHash(assertion),
                prevAssertionNum,
                memoryFrame.deadlineBlock,
                newAssertionHash,
                assertion.numBlocks + memoryFrame.prevAssertion.height,
                memoryFrame.currentInboxSize,
                !memoryFrame.hasSibling
            );
        }

        {
            uint64 assertionNum = latestAssertionCreated() + 1;

            // Fetch a storage reference to prevAssertion since we copied our other one into memory
            // and we don't have enough stack available to keep to keep the previous storage reference around
            AssertionNode storage prevAssertion = getAssertionStorage(prevAssertionNum);
            prevAssertion.childCreated(assertionNum, confirmPeriodBlocks);

            assertionCreated(memoryFrame.assertion);
        }

        emit AssertionCreated(
            latestAssertionCreated(),
            memoryFrame.prevAssertion.assertionHash,
            newAssertionHash,
            memoryFrame.executionHash,
            assertion,
            memoryFrame.sequencerBatchAcc,
            wasmModuleRoot,
            memoryFrame.currentInboxSize
        );

        return newAssertionHash;
    }

    function getPredecessorId(bytes32 assertionId) external view returns (bytes32){
        uint64 prevNum = getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).prevNum;
        return AssertionNodeLib.AssertionNum2Id(prevNum);
    }

    function getHeight(bytes32 assertionId) external view returns (uint256){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).height;
    }

    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).inboxMsgCountSeen;
    }

    function getStateHash(bytes32 assertionId) external view returns (bytes32){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).stateHash;
    }

    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).successionChallenge;
    }

    // HN: TODO: use block or timestamp?
    function getFirstChildCreationBlock(bytes32 assertionId) external view returns (uint256){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).firstChildBlock;
    }

    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).firstChildTime;
    }

    function isFirstChild(bytes32 assertionId) external view returns (bool){
        return getAssertionStorage(AssertionNodeLib.AssertionId2Num(assertionId)).isFirstChild;
    }
}
