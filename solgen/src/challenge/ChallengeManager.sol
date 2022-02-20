//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/DelegateCallAware.sol";
import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./IChallengeManager.sol";

import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

contract ChallengeManager is DelegateCallAware, IChallengeManager {
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    string constant NO_TURN = "NO_TURN";
    uint256 constant MAX_CHALLENGE_DEGREE = 40;

    uint64 public totalChallengesCreated;
    mapping (uint256 => Challenge) public challenges;

    IChallengeResultReceiver public resultReceiver;

    ISequencerInbox public sequencerInbox;
    IBridge public delayedBridge;
    IOneStepProofEntry public osp;

    function challengeInfo(uint64 challengeIndex) external view override returns (Challenge memory) {
        return challenges[challengeIndex];
    }

    modifier takeTurn(uint64 challengeIndex) {
        Challenge storage challenge = challenges[challengeIndex];
        require(msg.sender == currentResponder(challengeIndex), "BIS_SENDER");
        require(
            block.timestamp - challenge.lastMoveTimestamp <= currentResponderTimeLeft(challengeIndex),
            "BIS_DEADLINE"
        );

        _;

        if (challenge.turn == Turn.CHALLENGER) {
            challenge.challengerTimeLeft -= block.timestamp - challenge.lastMoveTimestamp;
            challenge.turn = Turn.ASSERTER;
        } else {
            challenge.asserterTimeLeft -= block.timestamp - challenge.lastMoveTimestamp;
            challenge.turn = Turn.CHALLENGER;
        }
        challenge.lastMoveTimestamp = block.timestamp;
    }

    function initialize(
        IChallengeResultReceiver resultReceiver_,
        ISequencerInbox sequencerInbox_,
        IBridge delayedBridge_,
        IOneStepProofEntry osp_
    ) external override onlyDelegated {
        require(address(resultReceiver) == address(0), "ALREADY_INIT");
        require(address(resultReceiver_) != address(0), "NO_RESULT_RECEIVER");
        resultReceiver = resultReceiver_;
        sequencerInbox = sequencerInbox_;
        delayedBridge = delayedBridge_;
        osp = osp_;
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
    ) external override returns (uint64) {
        require(msg.sender == address(resultReceiver), "ONLY_ROLLUP_CHAL");
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[0], startAndEndGlobalStates_[0].hash());
        segments[1] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[1], startAndEndGlobalStates_[1].hash());
        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(0, numBlocks, segments);

        uint64 challengeIndex = ++totalChallengesCreated;
        // The following is an assertion since it should never be possible, but it's an important invariant
        assert(challengeIndex != NO_CHAL_INDEX);
        Challenge storage challenge = challenges[challengeIndex];
        challenge.wasmModuleRoot = wasmModuleRoot_;
        // No need to set maxInboxMessages until execution challenge
        challenge.startAndEndGlobalStates[0] = startAndEndGlobalStates_[0];
        challenge.startAndEndGlobalStates[1] = startAndEndGlobalStates_[1];
        challenge.asserter = asserter_;
        challenge.challenger = challenger_;
        challenge.asserterTimeLeft = asserterTimeLeft_;
        challenge.challengerTimeLeft = challengerTimeLeft_;
        challenge.lastMoveTimestamp = block.timestamp;
        challenge.turn = Turn.CHALLENGER;
        challenge.mode = ChallengeMode.BLOCK;
        challenge.challengeStateHash = challengeStateHash;

        emit InitiatedChallenge(challengeIndex);
        emit Bisected(
            challengeIndex,
            challengeStateHash,
            0,
            numBlocks,
            segments
        );
        return challengeIndex;
    }

    function getStartGlobalState(uint64 challengeIndex) external view returns (GlobalState memory) {
        Challenge storage challenge = challenges[challengeIndex];
        return challenge.startAndEndGlobalStates[0];
    }

    function getEndGlobalState(uint64 challengeIndex) external view returns (GlobalState memory) {
        Challenge storage challenge = challenges[challengeIndex];
        return challenge.startAndEndGlobalStates[1];
    }

    /**
     * @notice Initiate the next round in the bisection by objecting to execution correctness with a bisection
     * of an execution segment with the same length but a different endpoint. This is either the initial move
     * or follows another execution objection
     */
    function bisectExecution(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        bytes32[] calldata newSegments
    ) external takeTurn(challengeIndex) {
        Challenge storage challenge = challenges[challengeIndex];
        (uint256 challengeStart, uint256 challengeLength) = ChallengeLib.extractChallengeSegment(challenge.challengeStateHash, selection);
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
            selection.oldSegments[selection.challengePosition + 1],
            "SAME_END"
        );

        require(selection.oldSegments[selection.challengePosition] == newSegments[0], "DIFF_START");

        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(
            challengeStart,
            challengeLength,
            newSegments
        );
        challenge.challengeStateHash = challengeStateHash;

        emit Bisected(
            challengeIndex,
            challengeStateHash,
            challengeStart,
            challengeLength,
            newSegments
        );
    }

    function challengeExecution(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) external {
        Challenge storage challenge = challenges[challengeIndex];
        require(msg.sender == currentResponder(challengeIndex), "EXEC_SENDER");
        require(
            block.timestamp - challenge.lastMoveTimestamp <= currentResponderTimeLeft(challengeIndex),
            "EXEC_DEADLINE"
        );

        (uint256 executionChallengeAtSteps, uint256 challengeLength) = ChallengeLib.extractChallengeSegment(challenge.challengeStateHash, selection);
        require(challengeLength == 1, "TOO_LONG");

        require(
            selection.oldSegments[selection.challengePosition] ==
                ChallengeLib.blockStateHash(
                    machineStatuses[0],
                    globalStateHashes[0]
                ),
            "WRONG_START"
        );
        require(
            selection.oldSegments[selection.challengePosition + 1] !=
                ChallengeLib.blockStateHash(
                    machineStatuses[1],
                    globalStateHashes[1]
                ),
            "SAME_END"
        );

        if (machineStatuses[0] != MachineStatus.FINISHED) {
            // If the machine is in a halted state, it can't change
            require(
                machineStatuses[0] == machineStatuses[1] &&
                    globalStateHashes[0] == globalStateHashes[1],
                "HALTED_CHANGE"
            );
            _currentWin(challenge);
            return;
        }

        if (machineStatuses[1] == MachineStatus.ERRORED) {
            // If the machine errors, it must return to the previous global state
            require(globalStateHashes[0] == globalStateHashes[1], "ERROR_CHANGE");
        }

        bytes32[2] memory startAndEndHashes;
        startAndEndHashes[0] = ChallengeLib.getStartMachineHash(
            globalStateHashes[0],
            challenge.wasmModuleRoot
        );
        startAndEndHashes[1] = ChallengeLib.getEndMachineHash(
            machineStatuses[1],
            globalStateHashes[1]
        );

        uint256 maxInboxMessagesRead = challenge.startAndEndGlobalStates[1].getInboxPosition();
        if (machineStatuses[1] == MachineStatus.ERRORED || challenge.startAndEndGlobalStates[1].getPositionInMessage() > 0) {
            maxInboxMessagesRead++;
        }

        if (challenge.turn == Turn.CHALLENGER) {
            (challenge.asserter, challenge.challenger) = (challenge.challenger, challenge.asserter);
            (
                challenge.asserterTimeLeft,
                challenge.challengerTimeLeft
            ) =  (
                challenge.challengerTimeLeft,
                challenge.asserterTimeLeft
            );
        } else if (challenge.turn != Turn.ASSERTER) {
            revert(NO_TURN);
        }

        require(numSteps <= OneStepProofEntryLib.MAX_STEPS, "CHALLENGE_TOO_LONG");
        challenge.maxInboxMessages = challenge.maxInboxMessages;
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = startAndEndHashes[0];
        segments[1] = startAndEndHashes[1];
        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(0, numSteps, segments);
        challenge.challengeStateHash = challengeStateHash;
        challenge.lastMoveTimestamp = block.timestamp;
        challenge.turn = Turn.CHALLENGER;
        challenge.mode = ChallengeMode.EXECUTION;

        emit Bisected(
            challengeIndex,
            challengeStateHash,
            0,
            numSteps,
            segments
        );

        emit ExecutionChallengeBegun(challengeIndex, executionChallengeAtSteps);
    }

    function oneStepProveExecution(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        bytes calldata proof
    ) external takeTurn(challengeIndex) {
        Challenge storage challenge = challenges[challengeIndex];
        (uint256 challengeStart, uint256 challengeLength) = ChallengeLib.extractChallengeSegment(challenge.challengeStateHash, selection);
        require(challengeLength == 1, "TOO_LONG");

        bytes32 afterHash = osp.proveOneStep(
            ExecutionContext({
                maxInboxMessagesRead: challenge.maxInboxMessages,
                sequencerInbox: sequencerInbox,
                delayedBridge: delayedBridge
            }),
            challengeStart,
                selection.oldSegments[selection.challengePosition],
            proof
        );
        require(
            afterHash != selection.oldSegments[selection.challengePosition + 1],
            "SAME_OSP_END"
        );

        emit OneStepProofCompleted(challengeIndex);
        _currentWin(challenge);
    }

    function timeout(uint64 challengeIndex) external override {
        Challenge storage challenge = challenges[challengeIndex];
        uint256 timeSinceLastMove = block.timestamp - challenge.lastMoveTimestamp;
        require(
            timeSinceLastMove > currentResponderTimeLeft(challengeIndex),
            "TIMEOUT_DEADLINE"
        );

        if (challenge.turn == Turn.ASSERTER) {
            emit AsserterTimedOut(challengeIndex);
            _challengerWin(challenge);
        } else if (challenge.turn == Turn.CHALLENGER) {
            emit ChallengerTimedOut(challengeIndex);
            _asserterWin(challenge);
        } else {
            revert(NO_TURN);
        }
    }

    function clearChallenge(uint64 challengeIndex) external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        Challenge storage challenge = challenges[challengeIndex];
        challenge.turn = Turn.NO_CHALLENGE;
    }

    function currentResponder(uint64 challengeIndex) public view returns (address) {
        Challenge storage challenge = challenges[challengeIndex];
        if (challenge.turn == Turn.ASSERTER) {
            return challenge.asserter;
        } else if (challenge.turn == Turn.CHALLENGER) {
            return challenge.challenger;
        } else {
            revert(NO_TURN);
        }
    }

    function currentResponderTimeLeft(uint64 challengeIndex) public override view returns (uint256) {
        Challenge storage challenge = challenges[challengeIndex];
        if (challenge.turn == Turn.ASSERTER) {
            return challenge.asserterTimeLeft;
        } else if (challenge.turn == Turn.CHALLENGER) {
            return challenge.challengerTimeLeft;
        } else {
            revert(NO_TURN);
        }
    }

    function _asserterWin(Challenge storage challenge) private {
        challenge.turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(challenge.asserter, challenge.challenger);
    }

    function _challengerWin(Challenge storage challenge) private {
        challenge.turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(challenge.challenger, challenge.asserter);
    }

    function _currentWin(Challenge storage challenge) private {
        // As a safety measure, challenges can only be resolved by timeouts during mainnet beta.
        // As state is 0, no move is possible. The other party will lose via timeout
        challenge.challengeStateHash = bytes32(0);

        // if (turn == Turn.ASSERTER) {
        //     _asserterWin();
        // } else if (turn == Turn.CHALLENGER) {
        //     _challengerWin();
        // } else {
        // 	   revert(NO_TURN);
        // }
    }
}
