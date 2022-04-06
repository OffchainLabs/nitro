// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/IChallengeResultReceiver.sol";
import "../challenge/IChallengeManager.sol";

contract MockResultReceiver is IChallengeResultReceiver {
    IChallengeManager public manager;
    address public winner;
    address public loser;
    uint256 public challengeIndex;

    event ChallengeCompleted(
        uint256 indexed challengeIndex,
        address indexed winner,
        address indexed loser
    );

    constructor(IChallengeManager manager_) {
        manager = manager_;
    }

    function createChallenge(
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (uint64) {
        return
            manager.createChallenge(
                wasmModuleRoot_,
                startAndEndMachineStatuses_,
                startAndEndGlobalStates_,
                numBlocks,
                asserter_,
                challenger_,
                asserterTimeLeft_,
                challengerTimeLeft_
            );
    }

    function completeChallenge(
        uint256 challengeIndex_,
        address winner_,
        address loser_
    ) external override {
        winner = winner_;
        loser = loser_;
        challengeIndex = challengeIndex_;
        emit ChallengeCompleted(challengeIndex, winner_, loser_);
    }
}
