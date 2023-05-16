// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import "./Assertion.sol";
import "./RollupLib.sol";
import "./IRollupEventInbox.sol";
import "./IRollupCore.sol";

import "../challenge/IOldChallengeManager.sol";
import "../state/Machine.sol";

import "../bridge/ISequencerInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../challengeV2/EdgeChallengeManager.sol";
import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

abstract contract RollupCore is IRollupCore, PausableUpgradeable {
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
    // HN: TODO: decide if we want index or hash based mapping
    mapping(bytes32 => uint64) private _assertionHashToNum;

    address[] private _stakerList;
    mapping(address => Staker) public _stakerMap;

    Zombie[] private _zombies;

    mapping(address => uint256) private _withdrawableFunds;
    uint256 public totalWithdrawableFunds;
    uint256 public rollupDeploymentBlock;

    // The assertion number of the initial assertion
    uint64 internal constant GENESIS_NODE = 1;

    bool public validatorWhitelistDisabled;

    IEdgeChallengeManager public challengeManager;

    /**
     * @notice Get a storage reference to the Assertion for the given assertion index
     * @param assertionNum Index of the assertion
     * @return Assertion struct
     */
    function getAssertionStorage(uint64 assertionNum) internal view returns (AssertionNode storage) {
        require(assertionNum != 0, "ASSERTION_NUM_CANNOT_BE_ZERO");
        return _assertions[assertionNum];
    }

    /**
     * @notice Get the Assertion for the given index.
     */
    function getAssertion(uint64 assertionNum) public view override returns (AssertionNode memory) {
        return getAssertionStorage(assertionNum);
    }

    /**
     * @notice Get the total number of assertions
     */
    function numAssertions() public view returns (uint64) {
        return _latestAssertionCreated + 1;
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

    /// @return Genesis end state hash, assertion hash, and wasm module root
    function genesisAssertionHashes() public view override returns (bytes32, bytes32, bytes32) {
        GlobalState memory emptyGlobalState;
        ExecutionState memory emptyExecutionState = ExecutionState(emptyGlobalState, MachineStatus.FINISHED);
        bytes32 afterStateHash = RollupLib.executionStateHash(emptyExecutionState);
        bytes32 genesisHash = RollupLib.assertionHash({
            parentAssertionHash: bytes32(0),
            afterStateHash: afterStateHash,
            inboxAcc: bytes32(0),
            wasmModuleRoot: bytes32(0)
        });
        return (afterStateHash, genesisHash, wasmModuleRoot);
    }

    /**
     * @notice Initialize the core with an initial assertion
     * @param initialAssertion Initial assertion to start the chain with
     */
    function initializeCore(AssertionNode memory initialAssertion) internal {
        __Pausable_init();
        // TODO: HN: prolly should use the internal function to create genesis
        _assertions[GENESIS_NODE] = initialAssertion;
        _latestConfirmed = GENESIS_NODE;
        _latestAssertionCreated = GENESIS_NODE;
        _firstUnresolvedAssertion = GENESIS_NODE + 1;
        _assertionHashToNum[initialAssertion.assertionHash] = GENESIS_NODE;
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

    function confirmAssertion(uint64 assertionNum, bytes32 blockHash, bytes32 sendRoot) internal {
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
    function inChallenge(address stakerAddress1, address stakerAddress2) internal view returns (uint64) {
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
    function challengeStarted(address staker1, address staker2, uint64 challenge) internal {
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
        bytes32 stateHash;
        AssertionNode prevAssertion;
        bytes32 lastHash;
        bool hasSibling;
        uint64 deadlineBlock;
        bytes32 sequencerBatchAcc;
    }

    function createNewAssertion(
        AssertionInputs calldata assertion,
        uint64 prevAssertionNum,
        bytes32 expectedAssertionHash
    ) internal returns (bytes32) {
        require(
            assertion.afterState.machineStatus == MachineStatus.FINISHED
                || assertion.afterState.machineStatus == MachineStatus.ERRORED,
            "BAD_AFTER_STATUS"
        );

        AssertionNode storage prevAssertion = getAssertionStorage(prevAssertionNum);
        bytes32 prevAssertionHash = prevAssertion.assertionHash;
        // validate the before state
        require(
            RollupLib.assertionHash(
                assertion.beforeStateData.prevAssertionHash,
                assertion.beforeState,
                assertion.beforeStateData.sequencerBatchAcc,
                assertion.beforeStateData.wasmRoot
            ) == prevAssertionHash,
            "INVALID_BEFORE_STATE"
        );

        uint256 nextInboxPosition;
        bytes32 sequencerBatchAcc;
        {
            // Validate the inbox positions
            uint64 afterInboxCount = assertion.afterState.globalState.getInboxPosition();
            uint64 prevInboxPosition = assertion.beforeState.globalState.getInboxPosition();
            require(afterInboxCount >= prevInboxPosition, "INBOX_BACKWARDS");
            if (afterInboxCount == prevInboxPosition) {
                require(
                    assertion.afterState.globalState.getPositionInMessage()
                        >= assertion.beforeState.globalState.getPositionInMessage(),
                    "INBOX_POS_IN_MSG_BACKWARDS"
                );
            }

            // See validator/assertion.go ExecutionState RequiredBatches() for reasoning
            if (
                assertion.afterState.machineStatus == MachineStatus.ERRORED
                    || assertion.afterState.globalState.getPositionInMessage() > 0
            ) {
                // The current inbox message was read
                afterInboxCount++;
            }
            // Cannot read more messages than currently exist
            uint256 currentInboxPosition = bridge.sequencerMessageCount();
            require(afterInboxCount <= currentInboxPosition, "INBOX_PAST_END");

            if (assertion.afterState.globalState.getInboxPosition() == currentInboxPosition) {
                // assertions must consume exactly up to the message count that was in the inbox
                // when the prev assertion was made. However if no new messages are sent, the next assertion
                // would need to consume the same number of messages as the prev, meaning the chain
                // would be unable to make progress. To avoid this we say that if no new messages have been
                // made between the prev and now, then the next assertion should consume one message
                nextInboxPosition = currentInboxPosition + 1;
            } else {
                nextInboxPosition = currentInboxPosition;
            }

            // we don't create an assertion until messages are added to the inbox
            require(afterInboxCount != 0, "EMPTY_INBOX_COUNT");

            // This gives replay protection against the state of the inbox
            sequencerBatchAcc = bridge.sequencerInboxAccs(afterInboxCount - 1);
        }

        bytes32 newAssertionHash = RollupLib.assertionHash(
            // HN: TODO: is this ok?
            prevAssertionHash,
            assertion.afterState,
            sequencerBatchAcc,
            wasmModuleRoot // HN: TODO: should we include this in assertion hash?
        );
        require(
            newAssertionHash == expectedAssertionHash || expectedAssertionHash == bytes32(0), "UNEXPECTED_NODE_HASH"
        );

        require(_assertionHashToNum[newAssertionHash] == 0, "ASSERTION_SEEN");

        AssertionNode memory newAssertion = AssertionNodeLib.createAssertion(
            uint64(nextInboxPosition),
            RollupLib.confirmHash(assertion),
            prevAssertionNum,
            uint64(block.number) + confirmPeriodBlocks,
            newAssertionHash,
            prevAssertion.firstChildBlock == 0 // assume block 0 is impossible
        );

        {
            uint64 assertionNum = latestAssertionCreated() + 1;
            _assertionHashToNum[newAssertionHash] = assertionNum;

            // Fetch a storage reference to prevAssertion since we copied our other one into memory
            // and we don't have enough stack available to keep to keep the previous storage reference around
            prevAssertion.childCreated(assertionNum, confirmPeriodBlocks);
            assertionCreated(newAssertion);
        }

        emit AssertionCreated(
            latestAssertionCreated(),
            prevAssertionHash,
            newAssertionHash,
            assertion,
            sequencerBatchAcc,
            wasmModuleRoot,
            nextInboxPosition
        );

        return newAssertionHash;
    }

    function getPredecessorId(bytes32 assertionId) external view returns (bytes32) {
        uint64 prevNum = getAssertionStorage(getAssertionNum(assertionId)).prevNum;
        return getAssertionId(prevNum);
    }

    function getHeight(bytes32 assertionId) external view returns (uint256) {
        revert("DEPRECATED");
    }

    function proveExecutionState(bytes32 assertionId, ExecutionState memory state, bytes memory proof)
        external
        view
        returns (ExecutionState memory)
    {
        (bytes32 parentAssertionHash, bytes32 inboxAcc, bytes32 wasmModuleRootInner) =
            abi.decode(proof, (bytes32, bytes32, bytes32));

        require(
            getAssertionStorage(getAssertionNum(assertionId)).assertionHash
                == RollupLib.assertionHash(parentAssertionHash, state, inboxAcc, wasmModuleRootInner),
            "Invalid assertion hash"
        );

        return state;
    }

    function getNextInboxPosition(bytes32 assertionId) external view returns (uint64) {
        return getAssertionStorage(getAssertionNum(assertionId)).nextInboxPosition;
    }

    function hasSibling(bytes32 assertionId) external view returns (bool) {
        return getAssertionStorage(getAssertionStorage(getAssertionNum(assertionId)).prevNum).secondChildBlock != 0;
    }

    // HN: TODO: use block or timestamp?
    function getFirstChildCreationBlock(bytes32 assertionId) external view returns (uint256) {
        return getAssertionStorage(getAssertionNum(assertionId)).firstChildBlock;
    }

    function getSecondChildCreationBlock(bytes32 assertionId) external view returns (uint256) {
        return getAssertionStorage(getAssertionNum(assertionId)).secondChildBlock;
    }

    function proveWasmModuleRoot(bytes32 assertionId, bytes32 root, bytes memory proof)
        external
        view
        returns (bytes32)
    {
        (bytes32 parentAssertionHash, bytes32 afterStateHash, bytes32 inboxAcc) =
            abi.decode(proof, (bytes32, bytes32, bytes32));
        require(
            RollupLib.assertionHash({
                parentAssertionHash: parentAssertionHash,
                afterStateHash: afterStateHash,
                inboxAcc: inboxAcc,
                wasmModuleRoot: root
            }) == assertionId,
            "BAD_WASM_MODULE_ROOT_PROOF"
        );
        return root;
    }

    function isFirstChild(bytes32 assertionId) external view returns (bool) {
        return getAssertionStorage(getAssertionNum(assertionId)).isFirstChild;
    }

    function isPending(bytes32 assertionId) external view returns (bool) {
        return getAssertionNum(assertionId) >= _firstUnresolvedAssertion;
    }

    // HN: TODO: decide to keep using index or hash
    function getAssertionNum(bytes32 id) public view returns (uint64) {
        uint64 num = _assertionHashToNum[id];
        require(num > 0, "ASSERTION_NOT_EXIST");
        return uint64(num);
    }

    function getAssertionId(uint64 num) public view returns (bytes32) {
        require(num <= latestAssertionCreated(), "INVALID_ASSERTION_NUM");
        return getAssertionStorage(num).assertionHash;
    }
}
