// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pragma solidity ^0.8.0;

pragma experimental ABIEncoderV2;

import "../rollup/IRollupCore.sol";
import "../rollup/IRollupLogic.sol";
import "../challenge/IChallengeManager.sol";

import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

contract ValidatorUtils {
    using NodeLib for Node;

    enum ConfirmType {
        NONE,
        VALID,
        INVALID
    }

    enum NodeConflict {
        NONE,
        FOUND,
        INDETERMINATE,
        INCOMPLETE
    }

    function stakerInfo(IRollupCore rollup, address stakerAddress)
        external
        view
        returns (
            bool isStaked,
            uint64 latestStakedNode,
            uint256 amountStaked,
            uint64 currentChallenge
        )
    {
        return (
            rollup.isStaked(stakerAddress),
            rollup.latestStakedNode(stakerAddress),
            rollup.amountStaked(stakerAddress),
            rollup.currentChallenge(stakerAddress)
        );
    }

    function findStakerConflict(
        IRollupCore rollup,
        address staker1,
        address staker2,
        uint256 maxDepth
    )
        external
        view
        returns (
            NodeConflict,
            uint64,
            uint64
        )
    {
        uint64 staker1NodeNum = rollup.latestStakedNode(staker1);
        uint64 staker2NodeNum = rollup.latestStakedNode(staker2);
        return findNodeConflict(rollup, staker1NodeNum, staker2NodeNum, maxDepth);
    }

    function checkDecidableNextNode(IRollupCore rollup) external view returns (ConfirmType) {
        try ValidatorUtils(address(this)).requireConfirmable(rollup) {
            return ConfirmType.VALID;
        } catch {}

        try ValidatorUtils(address(this)).requireRejectable(rollup) {
            return ConfirmType.INVALID;
        } catch {
            return ConfirmType.NONE;
        }
    }

    function requireRejectable(IRollupCore rollup) external view returns (bool) {
        IRollupUser(address(rollup)).requireUnresolvedExists();
        uint64 firstUnresolvedNode = rollup.firstUnresolvedNode();
        Node memory node = rollup.getNode(firstUnresolvedNode);
        bool inOrder = node.prevNum == rollup.latestConfirmed();
        if (inOrder) {
            // Verify the block's deadline has passed
            require(block.number >= node.deadlineBlock, "BEFORE_DEADLINE");
            rollup.getNode(node.prevNum).requirePastChildConfirmDeadline();

            // Verify that no staker is staked on this node
            require(
                node.stakerCount == IRollupUser(address(rollup)).countStakedZombies(firstUnresolvedNode),
                "HAS_STAKERS"
            );
        }
        return inOrder;
    }

    function requireConfirmable(IRollupCore rollup) external view {
        IRollupUser(address(rollup)).requireUnresolvedExists();

        uint256 stakerCount = rollup.stakerCount();
        // There is at least one non-zombie staker
        require(stakerCount > 0, "NO_STAKERS");

        uint64 firstUnresolved = rollup.firstUnresolvedNode();
        Node memory node = rollup.getNode(firstUnresolved);

        // Verify the block's deadline has passed
        node.requirePastDeadline();
        rollup.getNode(node.prevNum).requirePastChildConfirmDeadline();

        // Check that prev is latest confirmed
        require(node.prevNum == rollup.latestConfirmed(), "INVALID_PREV");
        require(
            node.stakerCount ==
                stakerCount + IRollupUser(address(rollup)).countStakedZombies(firstUnresolved),
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
            uint256 latestStakedNode = rollup.latestStakedNode(staker);
            if (
                latestStakedNode <= latestConfirmed && rollup.currentChallenge(staker) == 0
            ) {
                stakers[index] = staker;
                index++;
            }
        }
        assembly {
            mstore(stakers, index)
        }
        return stakers;
    }

    function latestStaked(IRollupCore rollup, address staker) external view returns (uint64, bytes32) {
        uint64 node = rollup.latestStakedNode(staker);
        if (node == 0) {
            node = rollup.latestConfirmed();
        }
        bytes32 acc = rollup.getNode(node).nodeHash;
        return (node, acc);
    }

    function stakedNodes(IRollupCore rollup, address staker) external view returns (uint64[] memory) {
        uint64[] memory nodes = new uint64[](100000);
        uint256 index = 0;
        for (uint64 i = rollup.latestConfirmed(); i <= rollup.latestNodeCreated(); i++) {
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

    function findNodeConflict(
        IRollupCore rollup,
        uint64 node1,
        uint64 node2,
        uint256 maxDepth
    )
        public
        view
        returns (
            NodeConflict,
            uint64,
            uint64
        )
    {
        uint64 firstUnresolvedNode = rollup.firstUnresolvedNode();
        uint64 node1Prev = rollup.getNode(node1).prevNum;
        uint64 node2Prev = rollup.getNode(node2).prevNum;

        for (uint256 i = 0; i < maxDepth; i++) {
            if (node1 == node2) {
                return (NodeConflict.NONE, node1, node2);
            }
            if (node1Prev == node2Prev) {
                return (NodeConflict.FOUND, node1, node2);
            }
            if (node1Prev < firstUnresolvedNode && node2Prev < firstUnresolvedNode) {
                return (NodeConflict.INDETERMINATE, 0, 0);
            }
            if (node1Prev < node2Prev) {
                node2 = node2Prev;
                node2Prev = rollup.getNode(node2).prevNum;
            } else {
                node1 = node1Prev;
                node1Prev = rollup.getNode(node1).prevNum;
            }
        }
        return (NodeConflict.INCOMPLETE, node1, node2);
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
            if (challengeIndex != NO_CHAL_INDEX &&
                challengeManager.isTimedOut(challengeIndex) &&
                challengeManager.currentResponder(challengeIndex) == staker) {
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
    function areUnresolvedNodesLinear(IRollupCore rollup) external view returns (bool) {
        uint256 end = rollup.latestNodeCreated();
        for (uint64 i = rollup.firstUnresolvedNode(); i <= end; i++) {
            if (i > 0 && rollup.getNode(i).prevNum != i - 1) {
                return false;
            }
        }
        return true;
    }
}
