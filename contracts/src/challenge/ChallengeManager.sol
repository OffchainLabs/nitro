// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

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
    using ChallengeLib for ChallengeLib.Challenge;

    enum ChallengeModeRequirement {
        ANY,
        BLOCK,
        EXECUTION
    }

    string private constant NO_CHAL = "NO_CHAL";
    uint256 private constant MAX_CHALLENGE_DEGREE = 40;

    uint64 public totalChallengesCreated;
    mapping(uint256 => ChallengeLib.Challenge) public challenges;

    IChallengeResultReceiver public resultReceiver;

    ISequencerInbox public sequencerInbox;
    IBridge public bridge;
    IOneStepProofEntry public osp;

    function challengeInfo(uint64 challengeIndex)
        external
        view
        override
        returns (ChallengeLib.Challenge memory)
    {
        return challenges[challengeIndex];
    }

    modifier takeTurn(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        ChallengeModeRequirement expectedMode
    ) {
        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        require(msg.sender == currentResponder(challengeIndex), "CHAL_SENDER");
        require(!isTimedOut(challengeIndex), "CHAL_DEADLINE");

        if (expectedMode == ChallengeModeRequirement.ANY) {
            require(challenge.mode != ChallengeLib.ChallengeMode.NONE, NO_CHAL);
        } else if (expectedMode == ChallengeModeRequirement.BLOCK) {
            require(challenge.mode == ChallengeLib.ChallengeMode.BLOCK, "CHAL_NOT_BLOCK");
        } else if (expectedMode == ChallengeModeRequirement.EXECUTION) {
            require(challenge.mode == ChallengeLib.ChallengeMode.EXECUTION, "CHAL_NOT_EXECUTION");
        } else {
            assert(false);
        }

        require(
            challenge.challengeStateHash ==
                ChallengeLib.hashChallengeState(
                    selection.oldSegmentsStart,
                    selection.oldSegmentsLength,
                    selection.oldSegments
                ),
            "BIS_STATE"
        );
        if (
            selection.oldSegments.length < 2 ||
            selection.challengePosition >= selection.oldSegments.length - 1
        ) {
            revert("BAD_CHALLENGE_POS");
        }

        _;

        if (challenge.mode == ChallengeLib.ChallengeMode.NONE) {
            // Early return since challenge must have terminated
            return;
        }

        ChallengeLib.Participant memory current = challenge.current;
        current.timeLeft -= block.timestamp - challenge.lastMoveTimestamp;

        challenge.current = challenge.next;
        challenge.next = current;

        challenge.lastMoveTimestamp = block.timestamp;
    }

    function initialize(
        IChallengeResultReceiver resultReceiver_,
        ISequencerInbox sequencerInbox_,
        IBridge bridge_,
        IOneStepProofEntry osp_
    ) external override onlyDelegated {
        require(address(resultReceiver) == address(0), "ALREADY_INIT");
        require(address(resultReceiver_) != address(0), "NO_RESULT_RECEIVER");
        resultReceiver = resultReceiver_;
        sequencerInbox = sequencerInbox_;
        bridge = bridge_;
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
        segments[0] = ChallengeLib.blockStateHash(
            startAndEndMachineStatuses_[0],
            startAndEndGlobalStates_[0].hash()
        );
        segments[1] = ChallengeLib.blockStateHash(
            startAndEndMachineStatuses_[1],
            startAndEndGlobalStates_[1].hash()
        );

        uint64 challengeIndex = ++totalChallengesCreated;
        // The following is an assertion since it should never be possible, but it's an important invariant
        assert(challengeIndex != NO_CHAL_INDEX);
        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        challenge.wasmModuleRoot = wasmModuleRoot_;

        // See validator/assertion.go ExecutionState RequiredBatches() for reasoning
        uint64 maxInboxMessagesRead = startAndEndGlobalStates_[1].getInboxPosition();
        if (
            startAndEndMachineStatuses_[1] == MachineStatus.ERRORED ||
            startAndEndGlobalStates_[1].getPositionInMessage() > 0
        ) {
            maxInboxMessagesRead++;
        }
        challenge.maxInboxMessages = maxInboxMessagesRead;
        challenge.next = ChallengeLib.Participant({addr: asserter_, timeLeft: asserterTimeLeft_});
        challenge.current = ChallengeLib.Participant({
            addr: challenger_,
            timeLeft: challengerTimeLeft_
        });
        challenge.lastMoveTimestamp = block.timestamp;
        challenge.mode = ChallengeLib.ChallengeMode.BLOCK;

        emit InitiatedChallenge(
            challengeIndex,
            startAndEndGlobalStates_[0],
            startAndEndGlobalStates_[1]
        );
        completeBisection(challengeIndex, 0, numBlocks, segments);
        return challengeIndex;
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
    ) external takeTurn(challengeIndex, selection, ChallengeModeRequirement.ANY) {
        (uint256 challengeStart, uint256 challengeLength) = ChallengeLib.extractChallengeSegment(
            selection
        );
        require(challengeLength > 1, "TOO_SHORT");
        {
            uint256 expectedDegree = challengeLength;
            if (expectedDegree > MAX_CHALLENGE_DEGREE) {
                expectedDegree = MAX_CHALLENGE_DEGREE;
            }
            require(newSegments.length == expectedDegree + 1, "WRONG_DEGREE");
        }

        requireValidBisection(selection, newSegments[0], newSegments[newSegments.length - 1]);

        completeBisection(challengeIndex, challengeStart, challengeLength, newSegments);
    }

    function challengeExecution(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) external takeTurn(challengeIndex, selection, ChallengeModeRequirement.BLOCK) {
        require(numSteps >= 1, "CHALLENGE_TOO_SHORT");
        require(numSteps <= OneStepProofEntryLib.MAX_STEPS, "CHALLENGE_TOO_LONG");
        requireValidBisection(
            selection,
            ChallengeLib.blockStateHash(machineStatuses[0], globalStateHashes[0]),
            ChallengeLib.blockStateHash(machineStatuses[1], globalStateHashes[1])
        );

        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        (uint256 executionChallengeAtSteps, uint256 challengeLength) = ChallengeLib
            .extractChallengeSegment(selection);
        require(challengeLength == 1, "TOO_LONG");

        if (machineStatuses[0] != MachineStatus.FINISHED) {
            // If the machine is in a halted state, it can't change
            require(
                machineStatuses[0] == machineStatuses[1] &&
                    globalStateHashes[0] == globalStateHashes[1],
                "HALTED_CHANGE"
            );
            _currentWin(challengeIndex, ChallengeTerminationType.BLOCK_PROOF);
            return;
        }

        if (machineStatuses[1] == MachineStatus.ERRORED) {
            // If the machine errors, it must return to the previous global state
            require(globalStateHashes[0] == globalStateHashes[1], "ERROR_CHANGE");
        }

        bytes32[] memory segments = new bytes32[](2);
        segments[0] = ChallengeLib.getStartMachineHash(
            globalStateHashes[0],
            challenge.wasmModuleRoot
        );
        segments[1] = ChallengeLib.getEndMachineHash(machineStatuses[1], globalStateHashes[1]);

        challenge.mode = ChallengeLib.ChallengeMode.EXECUTION;

        completeBisection(challengeIndex, 0, numSteps, segments);

        emit ExecutionChallengeBegun(challengeIndex, executionChallengeAtSteps);
    }

    function oneStepProveExecution(
        uint64 challengeIndex,
        ChallengeLib.SegmentSelection calldata selection,
        bytes calldata proof
    ) external takeTurn(challengeIndex, selection, ChallengeModeRequirement.EXECUTION) {
        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        uint256 challengeStart;
        {
            uint256 challengeLength;
            (challengeStart, challengeLength) = ChallengeLib.extractChallengeSegment(selection);
            require(challengeLength == 1, "TOO_LONG");
        }

        bytes32 afterHash = osp.proveOneStep(
            ExecutionContext({maxInboxMessagesRead: challenge.maxInboxMessages, bridge: bridge}),
            challengeStart,
            selection.oldSegments[selection.challengePosition],
            proof
        );
        require(
            afterHash != selection.oldSegments[selection.challengePosition + 1],
            "SAME_OSP_END"
        );

        emit OneStepProofCompleted(challengeIndex);
        _currentWin(challengeIndex, ChallengeTerminationType.EXECUTION_PROOF);
    }

    function timeout(uint64 challengeIndex) external override {
        require(challenges[challengeIndex].mode != ChallengeLib.ChallengeMode.NONE, NO_CHAL);
        require(isTimedOut(challengeIndex), "TIMEOUT_DEADLINE");
        _nextWin(challengeIndex, ChallengeTerminationType.TIMEOUT);
    }

    function clearChallenge(uint64 challengeIndex) external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        require(challenges[challengeIndex].mode != ChallengeLib.ChallengeMode.NONE, NO_CHAL);
        delete challenges[challengeIndex];
        emit ChallengeEnded(challengeIndex, ChallengeTerminationType.CLEARED);
    }

    function currentResponder(uint64 challengeIndex) public view override returns (address) {
        return challenges[challengeIndex].current.addr;
    }

    function isTimedOut(uint64 challengeIndex) public view override returns (bool) {
        return challenges[challengeIndex].isTimedOut();
    }

    function requireValidBisection(
        ChallengeLib.SegmentSelection calldata selection,
        bytes32 startHash,
        bytes32 endHash
    ) private pure {
        require(selection.oldSegments[selection.challengePosition] == startHash, "WRONG_START");
        require(selection.oldSegments[selection.challengePosition + 1] != endHash, "SAME_END");
    }

    function completeBisection(
        uint64 challengeIndex,
        uint256 challengeStart,
        uint256 challengeLength,
        bytes32[] memory newSegments
    ) private {
        assert(challengeLength >= 1);
        assert(newSegments.length >= 2);

        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(
            challengeStart,
            challengeLength,
            newSegments
        );
        challenges[challengeIndex].challengeStateHash = challengeStateHash;

        emit Bisected(
            challengeIndex,
            challengeStateHash,
            challengeStart,
            challengeLength,
            newSegments
        );
    }

    /// @dev This function causes the mode of the challenge to be set to NONE by deleting the challenge
    function _nextWin(uint64 challengeIndex, ChallengeTerminationType reason) private {
        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        address next = challenge.next.addr;
        address current = challenge.current.addr;
        delete challenges[challengeIndex];
        resultReceiver.completeChallenge(challengeIndex, next, current);
        emit ChallengeEnded(challengeIndex, reason);
    }

    /**
     * @dev this currently sets a challenge hash of 0 - no move is possible for the next participant to progress the
     * state. It is assumed that wherever this function is consumed, the turn is then adjusted for the opposite party
     * to timeout. This is done as a safety measure so challenges can only be resolved by timeouts during mainnet beta.
     */
    function _currentWin(
        uint64 challengeIndex,
        ChallengeTerminationType /* reason */
    ) private {
        ChallengeLib.Challenge storage challenge = challenges[challengeIndex];
        challenge.challengeStateHash = bytes32(0);

        //        address next = challenge.next.addr;
        //        address current = challenge.current.addr;
        //        delete challenges[challengeIndex];
        //        resultReceiver.completeChallenge(challengeIndex, current, next);
        //        emit ChallengeEnded(challengeIndex, reason);
    }
}
