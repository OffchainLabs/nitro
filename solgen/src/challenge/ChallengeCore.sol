//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ChallengeLib.sol";
import "./IChallengeResultReceiver.sol";

enum ChallengeWinner {
    NoWinner,
    AsserterWin,
    ChallengerWin
}

enum Turn {
    NO_CHALLENGE,
    ASSERTER,
    CHALLENGER
}

struct BisectableChallengeState {
    address asserter;
    address challenger;
    uint256 asserterTimeLeft;
    uint256 challengerTimeLeft;
    uint256 lastMoveTimestamp;
    Turn turn;
    bytes32 challengeStateHash;
}

string constant NO_TURN = "NO_TURN";
uint256 constant MAX_CHALLENGE_DEGREE = 40;

// TODO: use clearer name to differentiate from ChallengeLib.sol that is used for utils
library ChallengeCoreLib {
    using ChallengeCoreLib for BisectableChallengeState;

    event InitiatedChallenge();
    event Bisected(
        bytes32 indexed challengeRoot,
        uint256 challengedSegmentStart,
        uint256 challengedSegmentLength,
        bytes32[] chainHashes
    );
    event AsserterTimedOut();
    event ChallengerTimedOut();

    function createBisectableChallenge(
        address _asserter,
        address _challenger,
        uint256 _asserterTimeLeft,
        uint256 _challengerTimeLeft,
        uint256 _lastMoveTimestamp,
        Turn _turn,
        bytes32 _challengeStateHash
    ) internal pure returns (BisectableChallengeState memory) {
        return BisectableChallengeState({
            asserter: _asserter,
            challenger: _challenger,
            asserterTimeLeft: _asserterTimeLeft,
            challengerTimeLeft: _challengerTimeLeft,
            lastMoveTimestamp: _lastMoveTimestamp,
            turn: _turn,
            challengeStateHash: _challengeStateHash
        });
    }

    function beforeTurn(BisectableChallengeState storage currChallenge) internal view {
        require(msg.sender == currChallenge.currentResponder(), "BIS_SENDER");
        require(
            block.timestamp - currChallenge.lastMoveTimestamp <= currChallenge.currentResponderTimeLeft(),
            "BIS_DEADLINE"
        );
    }

    function afterTurn(BisectableChallengeState storage currChallenge) internal {
        if (currChallenge.turn == Turn.CHALLENGER) {
            currChallenge.challengerTimeLeft -= block.timestamp - currChallenge.lastMoveTimestamp;
            currChallenge.turn = Turn.ASSERTER;
        } else {
            currChallenge.asserterTimeLeft -= block.timestamp - currChallenge.lastMoveTimestamp;
            currChallenge.turn = Turn.CHALLENGER;
        }
        currChallenge.lastMoveTimestamp = block.timestamp;
    }

    modifier takeTurn(BisectableChallengeState storage currChallenge) {
        currChallenge.beforeTurn();
        _;
        currChallenge.afterTurn();
    }

    function currentResponder(BisectableChallengeState memory currChallenge) internal pure returns (address) {
        if (currChallenge.turn == Turn.ASSERTER) {
            return currChallenge.asserter;
        } else if (currChallenge.turn == Turn.CHALLENGER) {
            return currChallenge.challenger;
        } else {
            revert(NO_TURN);
        }
    }

    function currentResponderTimeLeft(BisectableChallengeState memory currChallenge) internal pure returns (uint256) {
        if (currChallenge.turn == Turn.ASSERTER) {
            return currChallenge.asserterTimeLeft;
        } else if (currChallenge.turn == Turn.CHALLENGER) {
            return currChallenge.challengerTimeLeft;
        } else {
            revert(NO_TURN);
        }
    }

    function extractChallengeSegment(
        BisectableChallengeState memory currChallenge,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition
    ) internal pure returns (uint256 segmentStart, uint256 segmentLength) {
        require(
            currChallenge.challengeStateHash ==
                ChallengeLib.hashChallengeState(
                    oldSegmentsStart,
                    oldSegmentsLength,
                    oldSegments
                ),
            "BIS_STATE"
        );
        if (
            oldSegments.length < 2 ||
            challengePosition >= oldSegments.length - 1
        ) {
            revert("BAD_CHALLENGE_POS");
        }
        uint256 oldChallengeDegree = oldSegments.length - 1;
        segmentLength = oldSegmentsLength / oldChallengeDegree;
        // Intentionally done before challengeLength is potentially added to for the final segment
        segmentStart = oldSegmentsStart + segmentLength * challengePosition;
        if (challengePosition == oldSegments.length - 2) {
            segmentLength += oldSegmentsLength % oldChallengeDegree;
        }
    }

    /**
     * @notice Initiate the next round in the bisection by objecting to execution correctness with a bisection
     * of an execution segment with the same length but a different endpoint. This is either the initial move
     * or follows another execution objection
     */
    function bisectExecution(
        BisectableChallengeState storage currChallenge,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes32[] calldata newSegments
    ) internal takeTurn(currChallenge) {
        (
            uint256 challengeStart,
            uint256 challengeLength
        ) = ChallengeCoreLib.extractChallengeSegment(
                currChallenge,
                oldSegmentsStart,
                oldSegmentsLength,
                oldSegments,
                challengePosition
            );
		require(challengeLength > 1, "TOO_SHORT");
        {
            uint256 expectedDegree = challengeLength;
            if (expectedDegree > MAX_CHALLENGE_DEGREE) {
                expectedDegree = MAX_CHALLENGE_DEGREE;
            }
            require(newSegments.length == expectedDegree + 1, "WRONG_DEGREE");
        }
        require(
            newSegments[newSegments.length - 1] !=
                oldSegments[challengePosition + 1],
            "SAME_END"
        );

        require(oldSegments[challengePosition] == newSegments[0], "DIFF_START");

        bytes32 newChallengeStateHash = ChallengeLib.hashChallengeState(
            challengeStart,
            challengeLength,
            newSegments
        );
        currChallenge.challengeStateHash = newChallengeStateHash;

        emit Bisected(
            newChallengeStateHash,
            challengeStart,
            challengeLength,
            newSegments
        );
    }

    function timeout(BisectableChallengeState storage currChallenge) internal returns (ChallengeWinner winner) {
        uint256 timeSinceLastMove = block.timestamp - currChallenge.lastMoveTimestamp;
        require(
            timeSinceLastMove > currChallenge.currentResponderTimeLeft(),
            "TIMEOUT_DEADLINE"
        );
        if (currChallenge.turn == Turn.ASSERTER) {
            emit AsserterTimedOut();
            return ChallengeWinner.ChallengerWin;
        } else if (currChallenge.turn == Turn.CHALLENGER) {
            emit ChallengerTimedOut();
            return ChallengeWinner.AsserterWin;
        } else {
            revert(NO_TURN);
        }
    }
}
