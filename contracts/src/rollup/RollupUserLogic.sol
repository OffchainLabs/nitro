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

abstract contract AbsRollupUserLogic is
    RollupCore,
    UUPSNotUpgradeable,
    IRollupUserAbs,
    IOldChallengeResultReceiver
{
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
     * @notice Extra number of blocks the validator can remain inactive before considered inactive
     *         This is 7 days assuming a 13.2 seconds block time
     */
    uint256 public constant VALIDATOR_AFK_BLOCKS = 45818;

    function _validatorIsAfk() internal view returns (bool) {
        AssertionNode memory latestAssertion = getAssertionStorage(latestAssertionCreated());
        if (latestAssertion.createdAtBlock == 0) return false;
        if (latestAssertion.createdAtBlock + confirmPeriodBlocks + VALIDATOR_AFK_BLOCKS < block.number) {
            return true;
        }
        return false;
    }

    function removeWhitelistAfterFork() external {
        require(!validatorWhitelistDisabled, "WHITELIST_DISABLED");
        require(_chainIdChanged(), "CHAIN_ID_NOT_CHANGED");
        validatorWhitelistDisabled = true;
    }

    function removeWhitelistAfterValidatorAfk() external {
        require(!validatorWhitelistDisabled, "WHITELIST_DISABLED");
        require(_validatorIsAfk(), "VALIDATOR_NOT_AFK");
        validatorWhitelistDisabled = true;
    }

    function isERC20Enabled() public view override returns (bool) {
        return stakeToken != address(0);
    }

    /**
     * @notice Reject the next unresolved assertion
     * @param stakerAddress Example staker staked on sibling, used to prove a assertion is on an unconfirmable branch and can be rejected
     */
    function rejectNextAssertion(address stakerAddress) external onlyValidator whenNotPaused {
        requireUnresolvedExists();
        uint64 latestConfirmedAssertionNum = latestConfirmed();
        uint64 firstUnresolvedAssertionNum = firstUnresolvedAssertion();
        AssertionNode storage firstUnresolvedAssertion_ = getAssertionStorage(firstUnresolvedAssertionNum);

        if (firstUnresolvedAssertion_.prevNum == latestConfirmedAssertionNum) {
            /**If the first unresolved assertion is a child of the latest confirmed assertion, to prove it can be rejected, we show:
             * a) Its deadline has expired
             * b) *Some* staker is staked on a sibling

             * The following three checks are sufficient to prove b:
            */

            // 1.  StakerAddress is indeed a staker
            require(isStakedOnLatestConfirmed(stakerAddress), "NOT_STAKED");

            // 2. Staker's latest staked assertion hasn't been resolved; this proves that staker's latest staked assertion can't be a parent of firstUnresolvedAssertion
            requireUnresolved(latestStakedAssertion(stakerAddress));

            // 3. staker isn't staked on first unresolved assertion; this proves staker's latest staked can't be a child of firstUnresolvedAssertion (recall staking on assertion requires staking on all of its parents)
            require(!assertionHasStaker(firstUnresolvedAssertionNum, stakerAddress), "STAKED_ON_TARGET");
            // If a staker is staked on a assertion that is neither a child nor a parent of firstUnresolvedAssertion, it must be a sibling, QED

            // // Verify the block's deadline has passed
            // firstUnresolvedAssertion_.requirePastDeadline();

            getAssertionStorage(latestConfirmedAssertionNum).requirePastChildConfirmDeadline();

            removeOldZombies(0);

            // HN: TODO: do we need this logic here?
            // // Verify that no staker is staked on this assertion
            // require(
            //     firstUnresolvedAssertion_.stakerCount == countStakedZombies(firstUnresolvedAssertionNum),
            //     "HAS_STAKERS"
            // );
        }
        // Simpler case: if the first unreseolved assertion doesn't point to the last confirmed assertion, another branch was confirmed and can simply reject it outright
        _rejectNextAssertion();

        emit AssertionRejected(firstUnresolvedAssertionNum);
    }

    /**
     * @notice Confirm the next unresolved assertion
     * @param blockHash The block hash at the end of the assertion
     * @param sendRoot The send root at the end of the assertion
     * @param winningEdge The winning edge if a challenge is started
     */
    function confirmNextAssertion(bytes32 blockHash, bytes32 sendRoot, bytes32 winningEdge)
        external
        onlyValidator
        whenNotPaused
    {
        requireUnresolvedExists();

        uint64 assertionNum = firstUnresolvedAssertion();
        AssertionNode storage assertion = getAssertionStorage(assertionNum);

        // // Verify the block's deadline has passed
        // assertion.requirePastDeadline();

        // Check that prev is latest confirmed
        assert(assertion.prevNum == latestConfirmed());

        AssertionNode storage prevAssertion = getAssertionStorage(assertion.prevNum);
        prevAssertion.requirePastChildConfirmDeadline();

        // HN: TODO: Do we need the zombie logic here?
        // removeOldZombies(0);
        //
        // // Require only zombies are staked on siblings to this assertion, and there's at least one non-zombie staked on this assertion
        // uint256 stakedZombies = countStakedZombies(assertionNum);
        // uint256 zombiesStakedOnOtherChildren = countZombiesStakedOnChildren(assertion.prevNum) -
        //     stakedZombies;
        // require(assertion.stakerCount > stakedZombies, "NO_STAKERS");
        // require(
        //     prevAssertion.childStakerCount == assertion.stakerCount + zombiesStakedOnOtherChildren,
        //     "NOT_ALL_STAKED"
        // );
        
        if(prevAssertion.secondChildBlock > 0) {
            // check if assertion is the challenge winner
            ChallengeEdge memory _winningEdge = challengeManager.getEdge(winningEdge);
            require(getAssertionNum(_winningEdge.claimId) == assertionNum, "NOT_WINNER");
            require(_winningEdge.status == EdgeStatus.Confirmed, "EDGE_NOT_CONFIRMED");
        }

        confirmAssertion(assertionNum, blockHash, sendRoot);
    }

    /**
     * @notice Create a new stake
     * @param depositAmount The amount of either eth or tokens staked
     */
    function _newStake(uint256 depositAmount) internal onlyValidator whenNotPaused {
        // Verify that sender is not already a staker
        require(!isStaked(msg.sender), "ALREADY_STAKED");
        require(!isZombie(msg.sender), "STAKER_IS_ZOMBIE");
        require(depositAmount >= currentRequiredStake(), "NOT_ENOUGH_STAKE");

        createNewStake(msg.sender, depositAmount);
    }

    /**
     * @notice Move stake onto existing child assertion
     * @param assertionNum Index of the assertion to move stake to. This must by a child of the assertion the staker is currently staked on
     * @param assertionHash Assertion hash of assertionNum (protects against reorgs)
     */
    function stakeOnExistingAssertion(uint64 assertionNum, bytes32 assertionHash)
        public
        onlyValidator
        whenNotPaused
    {
        require(isStakedOnLatestConfirmed(msg.sender), "NOT_STAKED");

        require(
            assertionNum >= firstUnresolvedAssertion() && assertionNum <= latestAssertionCreated(),
            "NODE_NUM_OUT_OF_RANGE"
        );
        AssertionNode storage assertion = getAssertionStorage(assertionNum);
        require(assertion.assertionHash == assertionHash, "NODE_REORG");
        require(latestStakedAssertion(msg.sender) == assertion.prevNum, "NOT_STAKED_PREV");
        stakeOnAssertion(msg.sender, assertionNum);
    }

    /**
     * @notice Create a new assertion and move stake onto it
     * @param assertion The assertion data
     * @param expectedAssertionHash The hash of the assertion being created (protects against reorgs)
     */
    function stakeOnNewAssertion(
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) public onlyValidator whenNotPaused {
        require(isStakedOnLatestConfirmed(msg.sender), "NOT_STAKED");
        // Ensure staker is staked on the previous assertion
        uint64 prevAssertion = latestStakedAssertion(msg.sender);

        {
            uint256 timeSinceLastAssertion = block.number - getAssertion(prevAssertion).createdAtBlock;
            // Verify that assertion meets the minimum Delta time requirement
            require(timeSinceLastAssertion >= minimumAssertionPeriod, "TIME_DELTA");

            bool isBeforeConsumedAllMsg = assertion.beforeState.globalState.getInboxPosition() == prevAssertionInboxMaxCount;

            // Minimum size requirement: any assertion must consume exactly all inbox messages
            // put into L1 inbox before the prev node’s L1 blocknum.
            // We make an exception if the machine enters the errored state,
            // as it can't consume future batches.
            require(
                assertion.afterState.machineStatus == MachineStatus.ERRORED ||
                    assertion.afterState.globalState.getInboxPosition() == prevAssertionInboxMaxCount,
                "WRONG_INBOX_POS"
            );

            // Minimum size requirement: any assertion must contain at least one block
            require(assertion.numBlocks > 0, "EMPTY_ASSERTION");

            // The rollup cannot advance normally from an errored state
            require(
                assertion.beforeState.machineStatus == MachineStatus.FINISHED,
                "BAD_PREV_STATUS"
            );
        }
        createNewAssertion(assertion, prevAssertion, prevAssertionInboxMaxCount, expectedAssertionHash);

        stakeOnAssertion(msg.sender, latestAssertionCreated());
    }

    /**
     * @notice Refund a staker that is currently staked on or before the latest confirmed assertion
     * @dev Since a staker is initially placed in the latest confirmed assertion, if they don't move it
     * a griefer can remove their stake. It is recomended to batch together the txs to place a stake
     * and move it to the desired assertion.
     * @param stakerAddress Address of the staker whose stake is refunded
     */
    function returnOldDeposit(address stakerAddress) external override onlyValidator whenNotPaused {
        require(latestStakedAssertion(stakerAddress) <= latestConfirmed(), "TOO_RECENT");
        requireUnchallengedStaker(stakerAddress);
        withdrawStaker(stakerAddress);
    }

    /**
     * @notice Increase the amount staked for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     * @param depositAmount The amount of either eth or tokens deposited
     */
    function _addToDeposit(address stakerAddress, uint256 depositAmount)
        internal
        onlyValidator
        whenNotPaused
    {
        requireUnchallengedStaker(stakerAddress);
        increaseStakeBy(stakerAddress, depositAmount);
    }

    /**
     * @notice Reduce the amount staked for the sender (difference between initial amount staked and target is creditted back to the sender).
     * @param target Target amount of stake for the staker. If this is below the current minimum, it will be set to minimum instead
     */
    function reduceDeposit(uint256 target) external onlyValidator whenNotPaused {
        requireUnchallengedStaker(msg.sender);
        uint256 currentRequired = currentRequiredStake();
        if (target < currentRequired) {
            target = currentRequired;
        }
        reduceStakeTo(msg.sender, target);
    }

    function createChallenge(
        uint64 assertionNum
    ) external onlyValidator whenNotPaused returns(bytes32) {
        revert("DEPRECATED");
    }

    /**
     * @notice Inform the rollup that the challenge between the given stakers is completed
     * @param winningStaker Address of the winning staker
     * @param losingStaker Address of the losing staker
     */
    function completeChallenge(
        uint256 challengeIndex,
        address winningStaker,
        address losingStaker
    ) external override whenNotPaused {
        revert("DEPRECATED");
    }

    /**
     * @notice Remove the given zombie from assertions it is staked on, moving backwords from the latest assertion it is staked on
     * @param zombieNum Index of the zombie to remove
     * @param maxAssertions Maximum number of assertions to remove the zombie from (to limit the cost of this transaction)
     */
    function removeZombie(uint256 zombieNum, uint256 maxAssertions)
        external
        onlyValidator
        whenNotPaused
    {
        require(zombieNum < zombieCount(), "NO_SUCH_ZOMBIE");
        address zombieStakerAddress = zombieAddress(zombieNum);
        uint64 latestAssertionStaked = zombieLatestStakedAssertion(zombieNum);
        uint256 assertionsRemoved = 0;
        uint256 latestConfirmedNum = latestConfirmed();
        while (latestAssertionStaked >= latestConfirmedNum && assertionsRemoved < maxAssertions) {
            AssertionNode storage assertion = getAssertionStorage(latestAssertionStaked);
            removeStaker(latestAssertionStaked, zombieStakerAddress);
            latestAssertionStaked = assertion.prevNum;
            assertionsRemoved++;
        }
        if (latestAssertionStaked < latestConfirmedNum) {
            removeZombie(zombieNum);
        } else {
            zombieUpdateLatestStakedAssertion(zombieNum, latestAssertionStaked);
        }
    }

    /**
     * @notice Remove any zombies whose latest stake is earlier than the latest confirmed assertion
     * @param startIndex Index in the zombie list to start removing zombies from (to limit the cost of this transaction)
     */
    function removeOldZombies(uint256 startIndex) public onlyValidator whenNotPaused {
        uint256 currentZombieCount = zombieCount();
        uint256 latestConfirmedNum = latestConfirmed();
        for (uint256 i = startIndex; i < currentZombieCount; i++) {
            while (zombieLatestStakedAssertion(i) < latestConfirmedNum) {
                removeZombie(i);
                currentZombieCount--;
                if (i >= currentZombieCount) {
                    return;
                }
            }
        }
    }

    /**
     * @notice Calculate the current amount of funds required to place a new stake in the rollup
     * @dev If the stake requirement get's too high, this function may start reverting due to overflow, but
     * that only blocks operations that should be blocked anyway
     * @return The current minimum stake requirement
     */
    function currentRequiredStake(
        uint256 _blockNumber,
        uint64 _firstUnresolvedAssertionNum,
        uint256 _latestCreatedAssertion
    ) internal view returns (uint256) {
        // If there are no unresolved assertions, then you can use the base stake
        if (_firstUnresolvedAssertionNum - 1 == _latestCreatedAssertion) {
            return baseStake;
        }
        uint256 firstUnresolvedDeadline = getAssertionStorage(_firstUnresolvedAssertionNum).deadlineBlock;
        if (_blockNumber < firstUnresolvedDeadline) {
            return baseStake;
        }
        uint24[10] memory numerators = [
            1,
            122971,
            128977,
            80017,
            207329,
            114243,
            314252,
            129988,
            224562,
            162163
        ];
        uint24[10] memory denominators = [
            1,
            114736,
            112281,
            64994,
            157126,
            80782,
            207329,
            80017,
            128977,
            86901
        ];
        uint256 firstUnresolvedAge = _blockNumber - firstUnresolvedDeadline;
        uint256 periodsPassed = (firstUnresolvedAge * 10) / confirmPeriodBlocks;
        uint256 baseMultiplier = 2**(periodsPassed / 10);
        uint256 withNumerator = baseMultiplier * numerators[periodsPassed % 10];
        uint256 multiplier = withNumerator / denominators[periodsPassed % 10];
        if (multiplier == 0) {
            multiplier = 1;
        }
        return baseStake * multiplier;
    }

    /**
     * @notice Calculate the current amount of funds required to place a new stake in the rollup
     * @dev If the stake requirement get's too high, this function may start reverting due to overflow, but
     * that only blocks operations that should be blocked anyway
     * @return The current minimum stake requirement
     */
    function requiredStake(
        uint256 blockNumber,
        uint64 firstUnresolvedAssertionNum,
        uint64 latestCreatedAssertion
    ) external view returns (uint256) {
        return currentRequiredStake(blockNumber, firstUnresolvedAssertionNum, latestCreatedAssertion);
    }

    function owner() external view returns (address) {
        return _getAdmin();
    }

    function currentRequiredStake() public view returns (uint256) {
        uint64 firstUnresolvedAssertionNum = firstUnresolvedAssertion();

        return currentRequiredStake(block.number, firstUnresolvedAssertionNum, latestAssertionCreated());
    }

    /**
     * @notice Calculate the number of zombies staked on the given assertion
     *
     * @dev This function could be uncallable if there are too many zombies. However,
     * removeZombie and removeOldZombies can be used to remove any zombies that exist
     * so that this will then be callable
     *
     * @param assertionNum The assertion on which to count staked zombies
     * @return The number of zombies staked on the assertion
     */
    function countStakedZombies(uint64 assertionNum) public view override returns (uint256) {
        uint256 currentZombieCount = zombieCount();
        uint256 stakedZombieCount = 0;
        for (uint256 i = 0; i < currentZombieCount; i++) {
            if (assertionHasStaker(assertionNum, zombieAddress(i))) {
                stakedZombieCount++;
            }
        }
        return stakedZombieCount;
    }

    /**
     * @notice Calculate the number of zombies staked on a child of the given assertion
     *
     * @dev This function could be uncallable if there are too many zombies. However,
     * removeZombie and removeOldZombies can be used to remove any zombies that exist
     * so that this will then be callable
     *
     * @param assertionNum The parent assertion on which to count zombies staked on children
     * @return The number of zombies staked on children of the assertion
     */
    function countZombiesStakedOnChildren(uint64 assertionNum) public view override returns (uint256) {
        uint256 currentZombieCount = zombieCount();
        uint256 stakedZombieCount = 0;
        for (uint256 i = 0; i < currentZombieCount; i++) {
            Zombie storage zombie = getZombieStorage(i);
            // If this zombie is staked on this assertion, but its _latest_ staked assertion isn't this assertion,
            // then it must be staked on a child of this assertion.
            if (
                zombie.latestStakedAssertion != assertionNum && assertionHasStaker(assertionNum, zombie.stakerAddress)
            ) {
                stakedZombieCount++;
            }
        }
        return stakedZombieCount;
    }

    /**
     * @notice Verify that there are some number of assertions still unresolved
     */
    function requireUnresolvedExists() public view override {
        uint256 firstUnresolved = firstUnresolvedAssertion();
        require(
            firstUnresolved > latestConfirmed() && firstUnresolved <= latestAssertionCreated(),
            "NO_UNRESOLVED"
        );
    }

    function requireUnresolved(uint256 assertionNum) public view override {
        require(assertionNum >= firstUnresolvedAssertion(), "ALREADY_DECIDED");
        require(assertionNum <= latestAssertionCreated(), "DOESNT_EXIST");
    }

    /**
     * @notice Verify that the given address is staked and not actively in a challenge
     * @param stakerAddress Address to check
     */
    function requireUnchallengedStaker(address stakerAddress) private view {
        require(isStaked(stakerAddress), "NOT_STAKED");
        require(currentChallenge(stakerAddress) == NO_CHAL_INDEX, "IN_CHAL");
    }
}

contract RollupUserLogic is AbsRollupUserLogic, IRollupUser {
    /// @dev the user logic just validated configuration and shouldn't write to state during init
    /// this allows the admin logic to ensure consistency on parameters.
    function initialize(address _stakeToken) external view override onlyProxy {
        require(_stakeToken == address(0), "NO_TOKEN_ALLOWED");
        require(!isERC20Enabled(), "FACET_NOT_ERC20");
    }

    /**
     * @notice Create a new stake on an existing assertion
     * @param assertionNum Number of the assertion your stake will be place one
     * @param assertionHash Assertion hash of the assertion with the given assertionNum
     */
    function newStakeOnExistingAssertion(uint64 assertionNum, bytes32 assertionHash) external payable override {
        _newStake(msg.value);
        stakeOnExistingAssertion(assertionNum, assertionHash);
    }

    /**
     * @notice Create a new stake on a new assertion
     * @param assertion Assertion describing the state change between the old assertion and the new one
     * @param expectedAssertionHash Assertion hash of the assertion that will be created
     * @param prevAssertionInboxMaxCount Total of messages in the inbox as of the previous assertion
     */
    function newStakeOnNewAssertion(
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) external payable override {
        _newStake(msg.value);
        stakeOnNewAssertion(assertion, expectedAssertionHash, prevAssertionInboxMaxCount);
    }

    /**
     * @notice Increase the amount staked eth for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     */
    function addToDeposit(address stakerAddress)
        external
        payable
        override
        onlyValidator
        whenNotPaused
    {
        _addToDeposit(stakerAddress, msg.value);
    }

    /**
     * @notice Withdraw uncommitted funds owned by sender from the rollup chain
     */
    function withdrawStakerFunds() external override onlyValidator whenNotPaused returns (uint256) {
        uint256 amount = withdrawFunds(msg.sender);
        // This is safe because it occurs after all checks and effects
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = msg.sender.call{value: amount}("");
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
     * @notice Create a new stake on an existing assertion
     * @param tokenAmount Amount of the rollups staking token to stake
     * @param assertionNum Number of the assertion your stake will be place one
     * @param assertionHash Assertion hash of the assertion with the given assertionNum
     */
    function newStakeOnExistingAssertion(
        uint256 tokenAmount,
        uint64 assertionNum,
        bytes32 assertionHash
    ) external override {
        _newStake(tokenAmount);
        stakeOnExistingAssertion(assertionNum, assertionHash);
        /// @dev This is an external call, safe because it's at the end of the function
        receiveTokens(tokenAmount);
    }

    /**
     * @notice Create a new stake on a new assertion
     * @param tokenAmount Amount of the rollups staking token to stake
     * @param assertion Assertion describing the state change between the old assertion and the new one
     * @param expectedAssertionHash Assertion hash of the assertion that will be created
     * @param prevAssertionInboxMaxCount Total of messages in the inbox as of the previous assertion
     */
    function newStakeOnNewAssertion(
        uint256 tokenAmount,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash,
        uint256 prevAssertionInboxMaxCount
    ) external override {
        _newStake(tokenAmount);
        stakeOnNewAssertion(assertion, expectedAssertionHash, prevAssertionInboxMaxCount);
        /// @dev This is an external call, safe because it's at the end of the function
        receiveTokens(tokenAmount);
    }

    /**
     * @notice Increase the amount staked tokens for the given staker
     * @param stakerAddress Address of the staker whose stake is increased
     * @param tokenAmount the amount of tokens staked
     */
    function addToDeposit(address stakerAddress, uint256 tokenAmount)
        external
        onlyValidator
        whenNotPaused
    {
        _addToDeposit(stakerAddress, tokenAmount);
        /// @dev This is an external call, safe because it's at the end of the function
        receiveTokens(tokenAmount);
    }

    /**
     * @notice Withdraw uncommitted funds owned by sender from the rollup chain
     */
    function withdrawStakerFunds() external override onlyValidator whenNotPaused returns (uint256) {
        uint256 amount = withdrawFunds(msg.sender);
        // This is safe because it occurs after all checks and effects
        require(IERC20Upgradeable(stakeToken).transfer(msg.sender, amount), "TRANSFER_FAILED");
        return amount;
    }

    function receiveTokens(uint256 tokenAmount) private {
        require(
            IERC20Upgradeable(stakeToken).transferFrom(msg.sender, address(this), tokenAmount),
            "TRANSFER_FAIL"
        );
    }
}
