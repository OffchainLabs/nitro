//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ChallengeLib.sol";
import "./IChallengeResultReceiver.sol";

abstract contract ChallengeCore {
    event InitiatedChallenge();

    enum Turn {
        NO_CHALLENGE,
        ASSERTER,
        CHALLENGER
    }

    event Bisected(
        bytes32 indexed challengeRoot,
        uint256 challengedSegmentStart,
        uint256 challengedSegmentLength,
        bytes32[] chainHashes
    );
    event AsserterTimedOut();
    event ChallengerTimedOut();

    address public asserter;
    address public challenger;

    uint256 public asserterTimeLeft;
    uint256 public challengerTimeLeft;
    uint256 public lastMoveTimestamp;

    Turn public turn;
    bytes32 public challengeStateHash;

    string constant NO_TURN = "NO_TURN";
    uint256 constant MAX_CHALLENGE_DEGREE = 40;

    IChallengeResultReceiver public resultReceiver;

    modifier takeTurn() {
        require(msg.sender == currentResponder(), "BIS_SENDER");
        require(
            block.timestamp - lastMoveTimestamp <= currentResponderTimeLeft(),
            "BIS_DEADLINE"
        );

        _;

        if (turn == Turn.CHALLENGER) {
            challengerTimeLeft -= block.timestamp - lastMoveTimestamp;
            turn = Turn.ASSERTER;
        } else {
            asserterTimeLeft -= block.timestamp - lastMoveTimestamp;
            turn = Turn.CHALLENGER;
        }
        lastMoveTimestamp = block.timestamp;
    }

    function currentResponder() public view returns (address) {
        if (turn == Turn.ASSERTER) {
            return asserter;
        } else if (turn == Turn.CHALLENGER) {
            return challenger;
        } else {
            revert(NO_TURN);
        }
    }

    function currentResponderTimeLeft() public view returns (uint256) {
        if (turn == Turn.ASSERTER) {
            return asserterTimeLeft;
        } else if (turn == Turn.CHALLENGER) {
            return challengerTimeLeft;
        } else {
            revert(NO_TURN);
        }
    }

    function extractChallengeSegment(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition
    ) internal view returns (uint256 segmentStart, uint256 segmentLength) {
        require(
            challengeStateHash ==
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
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes32[] calldata newSegments
    ) external takeTurn {
        (
            uint256 challengeStart,
            uint256 challengeLength
        ) = extractChallengeSegment(
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

        challengeStateHash = ChallengeLib.hashChallengeState(
            challengeStart,
            challengeLength,
            newSegments
        );

        emit Bisected(
            challengeStateHash,
            challengeStart,
            challengeLength,
            newSegments
        );
    }

    function timeout() external {
        uint256 timeSinceLastMove = block.timestamp - lastMoveTimestamp;
        require(
            timeSinceLastMove > currentResponderTimeLeft(),
            "TIMEOUT_DEADLINE"
        );

        if (turn == Turn.ASSERTER) {
            emit AsserterTimedOut();
            _challengerWin();
        } else if (turn == Turn.CHALLENGER) {
            emit ChallengerTimedOut();
            _asserterWin();
        } else {
            revert(NO_TURN);
        }
    }

    function _asserterWin() private {
        turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(asserter, challenger);
    }

    function _challengerWin() private {
        turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(challenger, asserter);
    }
}
