//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./ChallengeCore.sol";
import "./IChallenge.sol";
import "./ChallengeManager.sol";

struct ExecutionChallengeState {
    BisectableChallengeState bisectionState;
    ExecutionContext execCtx;
}

library ExecutionChallengeLib {
    using ChallengeCoreLib for BisectableChallengeState;

    event OneStepProofCompleted();

    function createExecutionChallenge(
        ExecutionChallengeState storage currChallenge,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 challenge_length,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) internal {
        require(challenge_length <= OneStepProofEntryLib.MAX_STEPS, "CHALLENGE_TOO_LONG");

        bytes32[] memory segments = new bytes32[](2);
        segments[0] = startAndEndHashes[0];
        segments[1] = startAndEndHashes[1];
        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(0, challenge_length, segments);

        emit ChallengeCoreLib.InitiatedChallenge();
        emit ChallengeCoreLib.Bisected(
            challengeStateHash,
            0,
            challenge_length,
            segments
        );

        BisectableChallengeState memory bisectionState = ChallengeCoreLib.createBisectableChallenge(
            asserter_,
            challenger_,
            asserterTimeLeft_,
            challengerTimeLeft_,
            block.timestamp,
            Turn.CHALLENGER,
            challengeStateHash
        );
        currChallenge.bisectionState = bisectionState;
        currChallenge.execCtx = execCtx_;
    }

    function oneStepProveExecution(
        mapping(uint256 => ChallengeManager.ChallengeTracker) storage challenges,
        uint256 challengeId,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes calldata proof
    ) internal {
        ChallengeManager.ChallengeTracker storage currTrckr = challenges[challengeId];
        require(
            currTrckr.trackerState ==
            ChallengeManager.ChallengeTrackerState.PendingExecutionChallenge,
            "NOT_EXEC_CHALL"
        );
        ExecutionChallengeState storage currChallenge = currTrckr.execChallState;

        // TODO: use takeTurn modifier if stack allows
        currChallenge.bisectionState.beforeTurn();

        (uint256 challengeStart, uint256 challengeLength) = currChallenge.bisectionState.extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        bytes32 afterHash = currTrckr.osp.proveOneStep(
            currChallenge.execCtx,
            challengeStart,
            oldSegments[challengePosition],
            proof
        );
        require(
            afterHash != oldSegments[challengePosition + 1],
            "SAME_OSP_END"
        );

        emit OneStepProofCompleted();
        _currentWin(currChallenge);
        currChallenge.bisectionState.afterTurn();
    }

    function clearChallenge(ExecutionChallengeState memory currChallenge) internal pure {
        // TODO: review this logic on how its triggered
        // require(msg.sender == address(currChallenge.bisectionState.resultReceiver), "NOT_RES_RECEIVER");
        currChallenge.bisectionState.turn = Turn.NO_CHALLENGE;
    }

    function _currentWin(ExecutionChallengeState memory currChallenge) private pure {
        // As a safety measure, challenges can only be resolved by timeouts during mainnet beta.
        // As state is 0, no move is possible. The other party will lose via timeout
        currChallenge.bisectionState.challengeStateHash = bytes32(0);

        // if (turn == Turn.ASSERTER) {
        //     _asserterWin();
        // } else if (turn == Turn.CHALLENGER) {
        //     _challengerWin();
        // } else {
        // 	   revert(NO_TURN);
        // }
    }
}
