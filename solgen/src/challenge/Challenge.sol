//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/DelegateCallAware.sol";
import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./IChallenge.sol";
import "./IChallengeFactory.sol";

contract Challenge is DelegateCallAware, IChallenge {
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    string constant NO_TURN = "NO_TURN";
    uint256 constant MAX_CHALLENGE_DEGREE = 40;

    ChallengeData public challenge;

    IChallengeResultReceiver public resultReceiver;

    ISequencerInbox public sequencerInbox;
    IBridge public delayedBridge;
    IOneStepProofEntry public osp;

    function challengeInfo() external view override returns (ChallengeData memory) {
        return challenge;
    }

    modifier takeTurn() {
        require(msg.sender == currentResponder(), "BIS_SENDER");
        require(
            block.timestamp - challenge.lastMoveTimestamp <= currentResponderTimeLeft(),
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

    // contractAddresses = [ resultReceiver, sequencerInbox, delayedBridge ]
    function initialize(
        IOneStepProofEntry osp_,
        IChallengeFactory.ChallengeContracts calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external onlyDelegated {
        require(address(resultReceiver) == address(0), "ALREADY_INIT");
        require(address(contractAddresses.resultReceiver) != address(0), "NO_RESULT_RECEIVER");
        resultReceiver = IChallengeResultReceiver(contractAddresses.resultReceiver);
        sequencerInbox = ISequencerInbox(contractAddresses.sequencerInbox);
        delayedBridge = IBridge(contractAddresses.delayedBridge);
        osp = osp_;

        bytes32[] memory segments = new bytes32[](2);
        segments[0] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[0], startAndEndGlobalStates_[0].hash());
        segments[1] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[1], startAndEndGlobalStates_[1].hash());
        bytes32 challengeStateHash = ChallengeLib.hashChallengeState(0, numBlocks, segments);

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

        emit InitiatedChallenge();
        emit Bisected(
            challengeStateHash,
            0,
            numBlocks,
            segments
        );
    }

    function getStartGlobalState() external view returns (GlobalState memory) {
        return challenge.startAndEndGlobalStates[0];
    }

    function getEndGlobalState() external view returns (GlobalState memory) {
        return challenge.startAndEndGlobalStates[1];
    }

    /**
     * @notice Initiate the next round in the bisection by objecting to execution correctness with a bisection
     * of an execution segment with the same length but a different endpoint. This is either the initial move
     * or follows another execution objection
     */
    function bisectExecution(
        SegmentSelection calldata selection,
        bytes32[] calldata newSegments
    ) external takeTurn {
        (uint256 challengeStart, uint256 challengeLength) = extractChallengeSegment(selection);
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
            challengeStateHash,
            challengeStart,
            challengeLength,
            newSegments
        );
    }

    function challengeExecution(
        SegmentSelection calldata selection,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) external {
        require(msg.sender == currentResponder(), "EXEC_SENDER");
        require(
            block.timestamp - challenge.lastMoveTimestamp <= currentResponderTimeLeft(),
            "EXEC_DEADLINE"
        );

        (uint256 executionChallengeAtSteps, uint256 challengeLength) = extractChallengeSegment(selection);
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
            _currentWin();
            return;
        }

        if (machineStatuses[1] == MachineStatus.ERRORED) {
            // If the machine errors, it must return to the previous global state
            require(globalStateHashes[0] == globalStateHashes[1], "ERROR_CHANGE");
        }

        bytes32[2] memory startAndEndHashes;
        startAndEndHashes[0] = getStartMachineHash(
            globalStateHashes[0]
        );
        startAndEndHashes[1] = getEndMachineHash(
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

        emit InitiatedChallenge();
        emit Bisected(
            challengeStateHash,
            0,
            numSteps,
            segments
        );

        emit ExecutionChallengeBegun(executionChallengeAtSteps);
    }

    function oneStepProveExecution(
        SegmentSelection calldata selection,
        bytes calldata proof
    ) external takeTurn {
        (uint256 challengeStart, uint256 challengeLength) = extractChallengeSegment(selection);
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

        emit OneStepProofCompleted();
        _currentWin();
    }

    function timeout() external override {
        uint256 timeSinceLastMove = block.timestamp - challenge.lastMoveTimestamp;
        require(
            timeSinceLastMove > currentResponderTimeLeft(),
            "TIMEOUT_DEADLINE"
        );

        if (challenge.turn == Turn.ASSERTER) {
            emit AsserterTimedOut();
            _challengerWin();
        } else if (challenge.turn == Turn.CHALLENGER) {
            emit ChallengerTimedOut();
            _asserterWin();
        } else {
            revert(NO_TURN);
        }
    }

    function clearChallenge() external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        challenge.turn = Turn.NO_CHALLENGE;
    }

    function currentResponder() public view returns (address) {
        if (challenge.turn == Turn.ASSERTER) {
            return challenge.asserter;
        } else if (challenge.turn == Turn.CHALLENGER) {
            return challenge.challenger;
        } else {
            revert(NO_TURN);
        }
    }

    function currentResponderTimeLeft() public override view returns (uint256) {
        if (challenge.turn == Turn.ASSERTER) {
            return challenge.asserterTimeLeft;
        } else if (challenge.turn == Turn.CHALLENGER) {
            return challenge.challengerTimeLeft;
        } else {
            revert(NO_TURN);
        }
    }

    function extractChallengeSegment(SegmentSelection calldata selection) internal view returns (uint256 segmentStart, uint256 segmentLength) {
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
        uint256 oldChallengeDegree = selection.oldSegments.length - 1;
        segmentLength = selection.oldSegmentsLength / oldChallengeDegree;
        // Intentionally done before challengeLength is potentially added to for the final segment
        segmentStart = selection.oldSegmentsStart + segmentLength * selection.challengePosition;
        if (selection.challengePosition == selection.oldSegments.length - 2) {
            segmentLength += selection.oldSegmentsLength % oldChallengeDegree;
        }
    }

    function getStartMachineHash(bytes32 globalStateHash)
        internal
        view
        returns (bytes32)
    {
        ValueStack memory values;
        {
            // Start the value stack with the function call ABI for the entrypoint
            Value[] memory startingValues = new Value[](3);
            startingValues[0] = ValueLib.newRefNull();
            startingValues[1] = ValueLib.newI32(0);
            startingValues[2] = ValueLib.newI32(0);
            ValueArray memory valuesArray = ValueArray({
                inner: startingValues
            });
            values = ValueStack({
                proved: valuesArray,
                remainingHash: 0
            });
        }
		ValueStack memory internalStack;
		PcStack memory blocks;
		StackFrameWindow memory frameStack;

		Machine memory mach = Machine({
			status: MachineStatus.RUNNING,
			valueStack: values,
			internalStack: internalStack,
			blockStack: blocks,
			frameStack: frameStack,
			globalStateHash: globalStateHash,
			moduleIdx: 0,
			functionIdx: 0,
			functionPc: 0,
			modulesRoot: challenge.wasmModuleRoot
		});
        return mach.hash();
    }

    function getEndMachineHash(MachineStatus status, bytes32 globalStateHash)
        internal
        pure
        returns (bytes32)
    {
        if (status == MachineStatus.FINISHED) {
            return
                keccak256(
                    abi.encodePacked("Machine finished:", globalStateHash)
                );
        } else if (status == MachineStatus.ERRORED) {
            return keccak256(abi.encodePacked("Machine errored:"));
        } else if (status == MachineStatus.TOO_FAR) {
            return keccak256(abi.encodePacked("Machine too far:"));
        } else {
            revert("BAD_BLOCK_STATUS");
        }
    }

    function _asserterWin() private {
        challenge.turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(challenge.asserter, challenge.challenger);
    }

    function _challengerWin() private {
        challenge.turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(challenge.challenger, challenge.asserter);
    }

    function _currentWin() private {
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
