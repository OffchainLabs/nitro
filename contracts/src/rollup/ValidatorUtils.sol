// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

pragma experimental ABIEncoderV2;

import "../rollup/IRollupCore.sol";
import "../challenge/IChallengeManager.sol";

import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

contract ValidatorUtils {
    using AssertionLib for Assertion;

    enum ConfirmType {
        NONE,
        VALID,
        INVALID
    }

    enum AssertionConflictType {
        NONE,
        FOUND,
        INDETERMINATE,
        INCOMPLETE
    }

    struct AssertionConflict {
        AssertionConflictType ty;
        uint64 node1;
        uint64 node2;
    }

    function findStakerConflict(
        IRollupCore rollup,
        address staker1,
        address staker2,
        uint256 maxDepth
    ) external view returns (AssertionConflict memory) {
        uint64 staker1AssertionNum = rollup.latestStakedAssertion(staker1);
        uint64 staker2AssertionNum = rollup.latestStakedAssertion(staker2);
        return findAssertionConflict(rollup, staker1AssertionNum, staker2AssertionNum, maxDepth);
    }

    function checkDecidableNextAssertion(IRollupUserAbs rollup) external view returns (ConfirmType) {
        try ValidatorUtils(address(this)).requireConfirmable(rollup) {
            return ConfirmType.VALID;
        } catch {}

        try ValidatorUtils(address(this)).requireRejectable(rollup) {
            return ConfirmType.INVALID;
        } catch {
            return ConfirmType.NONE;
        }
    }

    function requireRejectable(IRollupCore rollup) external view {
        IRollupUser(address(rollup)).requireUnresolvedExists();
        uint64 firstUnresolvedAssertion = rollup.firstUnresolvedAssertion();
        Assertion memory node = rollup.getAssertion(firstUnresolvedAssertion);
        if (node.prevNum == rollup.latestConfirmed()) {
            // Verify the block's deadline has passed
            require(block.number >= node.deadlineBlock, "BEFORE_DEADLINE");
            rollup.getAssertion(node.prevNum).requirePastChildConfirmDeadline();

            // Verify that no staker is staked on this node
            require(
                node.stakerCount ==
                    IRollupUser(address(rollup)).countStakedZombies(firstUnresolvedAssertion),
                "HAS_STAKERS"
            );
        }
    }

    function requireConfirmable(IRollupUserAbs rollup) external view {
        rollup.requireUnresolvedExists();

        uint256 stakerCount = rollup.stakerCount();
        // There is at least one non-zombie staker
        require(stakerCount > 0, "NO_STAKERS");

        uint64 firstUnresolved = rollup.firstUnresolvedAssertion();
        Assertion memory node = rollup.getAssertion(firstUnresolved);

        // Verify the block's deadline has passed
        node.requirePastDeadline();

        // Check that prev is latest confirmed
        assert(node.prevNum == rollup.latestConfirmed());

        Assertion memory prevAssertion = rollup.getAssertion(node.prevNum);
        prevAssertion.requirePastChildConfirmDeadline();

        uint256 zombiesStakedOnOtherChildren = rollup.countZombiesStakedOnChildren(node.prevNum) -
            rollup.countStakedZombies(firstUnresolved);
        require(
            prevAssertion.childStakerCount == node.stakerCount + zombiesStakedOnOtherChildren,
            "NOT_ALL_STAKED"
        );
    }

    function refundableStakers(IRollupCore rollup) external view returns (address[] memory) {
        uint256 stakerCount = rollup.stakerCount();
        address[] memory stakers = new address[](stakerCount);
        uint256 latestConfirmed = rollup.latestConfirmed();
        uint256 index = 0;
        for (uint64 i = 0; i < stakerCount; i++) {
            address staker = rollup.getStakerAddress(i);
            uint256 latestStakedAssertion = rollup.latestStakedAssertion(staker);
            if (latestStakedAssertion <= latestConfirmed && rollup.currentChallenge(staker) == 0) {
                stakers[index] = staker;
                index++;
            }
        }
        assembly {
            mstore(stakers, index)
        }
        return stakers;
    }

    function latestStaked(IRollupCore rollup, address staker)
        external
        view
        returns (uint64, Assertion memory)
    {
        uint64 num = rollup.latestStakedAssertion(staker);
        if (num == 0) {
            num = rollup.latestConfirmed();
        }
        Assertion memory node = rollup.getAssertion(num);
        return (num, node);
    }

    function stakedAssertions(IRollupCore rollup, address staker)
        external
        view
        returns (uint64[] memory)
    {
        uint64[] memory nodes = new uint64[](100000);
        uint256 index = 0;
        for (uint64 i = rollup.latestConfirmed(); i <= rollup.latestAssertionCreated(); i++) {
            if (rollup.nodeHasStaker(i, staker)) {
                nodes[index] = i;
                index++;
            }
        }
        // Shrink array down to real size
        assembly {
            mstore(nodes, index)
        }
        return nodes;
    }

    function findAssertionConflict(
        IRollupCore rollup,
        uint64 node1,
        uint64 node2,
        uint256 maxDepth
    ) public view returns (AssertionConflict memory) {
        uint64 firstUnresolvedAssertion = rollup.firstUnresolvedAssertion();
        uint64 node1Prev = rollup.getAssertion(node1).prevNum;
        uint64 node2Prev = rollup.getAssertion(node2).prevNum;

        for (uint256 i = 0; i < maxDepth; i++) {
            if (node1 == node2) {
                return AssertionConflict(AssertionConflictType.NONE, node1, node2);
            }
            if (node1Prev == node2Prev) {
                return AssertionConflict(AssertionConflictType.FOUND, node1, node2);
            }
            if (node1Prev < firstUnresolvedAssertion && node2Prev < firstUnresolvedAssertion) {
                return AssertionConflict(AssertionConflictType.INDETERMINATE, 0, 0);
            }
            if (node1Prev < node2Prev) {
                node2 = node2Prev;
                node2Prev = rollup.getAssertion(node2).prevNum;
            } else {
                node1 = node1Prev;
                node1Prev = rollup.getAssertion(node1).prevNum;
            }
        }
        return AssertionConflict(AssertionConflictType.INCOMPLETE, 0, 0);
    }

    function getStakers(
        IRollupCore rollup,
        uint64 startIndex,
        uint64 max
    ) public view returns (address[] memory, bool hasMore) {
        uint256 maxStakers = rollup.stakerCount();
        if (startIndex + max <= maxStakers) {
            maxStakers = startIndex + max;
            hasMore = true;
        }

        address[] memory stakers = new address[](maxStakers);
        for (uint64 i = 0; i < maxStakers; i++) {
            stakers[i] = rollup.getStakerAddress(startIndex + i);
        }
        return (stakers, hasMore);
    }

    function timedOutChallenges(
        IRollupCore rollup,
        uint64 startIndex,
        uint64 max
    ) external view returns (uint64[] memory, bool hasMore) {
        (address[] memory stakers, bool hasMoreStakers) = getStakers(rollup, startIndex, max);
        uint64[] memory challenges = new uint64[](stakers.length);
        uint256 index = 0;
        IChallengeManager challengeManager = rollup.challengeManager();
        for (uint256 i = 0; i < stakers.length; i++) {
            address staker = stakers[i];
            uint64 challengeIndex = rollup.currentChallenge(staker);
            if (
                challengeIndex != NO_CHAL_INDEX &&
                challengeManager.isTimedOut(challengeIndex) &&
                challengeManager.currentResponder(challengeIndex) == staker
            ) {
                challenges[index++] = challengeIndex;
            }
        }
        // Shrink array down to real size
        assembly {
            mstore(challenges, index)
        }
        return (challenges, hasMoreStakers);
    }

    // Worst case runtime of O(depth), as it terminates if it switches paths.
    function areUnresolvedAssertionsLinear(IRollupCore rollup) external view returns (bool) {
        uint256 end = rollup.latestAssertionCreated();
        for (uint64 i = rollup.firstUnresolvedAssertion(); i <= end; i++) {
            if (i > 0 && rollup.getAssertion(i).prevNum != i - 1) {
                return false;
            }
        }
        return true;
    }
}
