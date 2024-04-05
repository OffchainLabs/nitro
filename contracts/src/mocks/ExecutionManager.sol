// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/OldChallengeManager.sol";

contract SingleExecutionChallenge is OldChallengeManager {
    constructor(
        IOneStepProofEntry osp_,
        IOldChallengeResultReceiver resultReceiver_,
        uint64 maxInboxMessagesRead_,
        bytes32[2] memory startAndEndHashes,
        uint256 numSteps_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) {
        osp = osp_;
        resultReceiver = resultReceiver_;
        uint64 challengeIndex = ++totalChallengesCreated;
        OldChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        challenge.maxInboxMessages = maxInboxMessagesRead_;
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = startAndEndHashes[0];
        segments[1] = startAndEndHashes[1];
        bytes32 challengeStateHash = OldChallengeLib.hashChallengeState(0, numSteps_, segments);
        challenge.challengeStateHash = challengeStateHash;
        challenge.next = OldChallengeLib.Participant({addr: asserter_, timeLeft: asserterTimeLeft_});
        challenge.current = OldChallengeLib.Participant({
            addr: challenger_,
            timeLeft: challengerTimeLeft_
        });
        challenge.lastMoveTimestamp = block.timestamp;
        challenge.mode = OldChallengeLib.ChallengeMode.EXECUTION;

        emit Bisected(challengeIndex, challengeStateHash, 0, numSteps_, segments);
    }
}
