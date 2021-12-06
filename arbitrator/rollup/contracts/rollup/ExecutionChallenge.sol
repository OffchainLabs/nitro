//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./IExecutionChallenge.sol";
import "./Cloneable.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ExecutionChallenge is IExecutionChallenge, Cloneable {
    enum Turn {
        NO_CHALLENGE,
        ASSERTER,
        CHALLENGER
    }

    event InitiatedChallenge();
    event Bisected(
        bytes32 indexed challengeRoot,
        uint256 challengedSegmentStart,
        uint256 challengedSegmentLength,
        bytes32[] chainHashes
    );
    event AsserterTimedOut();
    event ChallengerTimedOut();
    event OneStepProofCompleted();
    event ContinuedExecutionProven();

    uint256 constant MAX_CHALLENGE_DEGREE = 40;
    uint256 constant MAX_STEPS = ~uint64(0) - 1;

    string constant NO_TURN = "NO_TURN";

    IOneStepProofEntry public osp;
    IChallengeResultReceiver resultReceiver;

    ExecutionContext public execCtx;

    bytes32 public challengeStateHash;

    address public asserter;
    address public challenger;

    uint256 public asserterTimeLeft;
    uint256 public challengerTimeLeft;
    uint256 public lastMoveTimestamp;

    Turn public turn;

    constructor(
        IOneStepProofEntry osp_,
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32 challengeStateHash_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) {
        osp = osp_;
        resultReceiver = resultReceiver_;
        execCtx = execCtx_;
        challengeStateHash = challengeStateHash_;
        asserter = asserter_;
        challenger = challenger_;
        asserterTimeLeft = asserterTimeLeft_;
        challengerTimeLeft = challengerTimeLeft_;
        lastMoveTimestamp = block.timestamp;
        turn = Turn.CHALLENGER;

        emit InitiatedChallenge();
    }

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
        ) = ChallengeLib.extractChallengeSegment(
				challengeStateHash,
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
            require(expectedDegree >= 1, "BAD_DEGREE");
            require(newSegments.length == expectedDegree + 1, "WRONG_DEGREE");
        }
        require(
            newSegments[newSegments.length - 1] !=
                oldSegments[oldSegments.length - 1],
            "SAME_END"
        );

        require(oldSegments[0] == newSegments[0], "DIFF_START");

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

    function oneStepProveExecution(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes calldata proof
    ) external takeTurn {
        (uint256 challengeStart, uint256 challengeLength) = ChallengeLib.extractChallengeSegment(
			challengeStateHash,
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        bytes32 afterHash = osp.proveOneStep(
            execCtx,
            challengeStart,
            oldSegments[challengePosition],
            proof
        );
        require(
            afterHash != oldSegments[challengePosition + 1],
            "SAME_OSP_END"
        );

        emit OneStepProofCompleted();
        _currentWin();
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

    function clearChallenge() external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        safeSelfDestruct(payable(0));
    }

    function _currentWin() private {
        // As a safety measure, challenges can only be resolved by timeouts during mainnet beta.
        // As state is 0, no move is possible. The other party will lose via timeout
        challengeStateHash = bytes32(0);

        // if (turn == Turn.ASSERTER) {
        //     _asserterWin();
        // } else if (turn == Turn.CHALLENGER) {
        //     _challengerWin();
        // } else {
        // 	   revert(NO_TURN);
        // }
    }

    function _asserterWin() private {
        resultReceiver.completeChallenge(asserter, challenger);
        safeSelfDestruct(payable(0));
    }

    function _challengerWin() private {
        resultReceiver.completeChallenge(challenger, asserter);
        safeSelfDestruct(payable(0));
    }
}
