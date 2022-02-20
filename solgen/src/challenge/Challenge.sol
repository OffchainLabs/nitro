//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/DelegateCallAware.sol";
import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./ChallengeCore.sol";
import "./IChallenge.sol";
import "./IChallengeFactory.sol";

contract Challenge is ChallengeCore, DelegateCallAware {
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    enum ChallengeMode {
        NONE,
        BLOCK,
        EXECUTION
    }

    event ExecutionChallengeBegun(uint256 blockSteps);
    event OneStepProofCompleted();

    bytes32 public wasmModuleRoot;
    GlobalState[2] internal startAndEndGlobalStates;

    ISequencerInbox public sequencerInbox;
    IBridge public delayedBridge;
    IOneStepProofEntry public osp;

    ChallengeMode public mode;

    uint256 maxInboxMessages;

    // contractAddresses = [ resultReceiver, sequencerInbox, delayedBridge ]
    function initialize(
        IOneStepProofEntry osp_,
        IChallengeFactory.ChallengeContracts memory contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
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
        wasmModuleRoot = wasmModuleRoot_;
        startAndEndGlobalStates[0] = startAndEndGlobalStates_[0];
        startAndEndGlobalStates[1] = startAndEndGlobalStates_[1];
        asserter = asserter_;
        challenger = challenger_;
        asserterTimeLeft = asserterTimeLeft_;
        challengerTimeLeft = challengerTimeLeft_;
        lastMoveTimestamp = block.timestamp;
        turn = Turn.CHALLENGER;
        mode = ChallengeMode.BLOCK;

        bytes32[] memory segments = new bytes32[](2);
        segments[0] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[0], startAndEndGlobalStates_[0].hash());
        segments[1] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[1], startAndEndGlobalStates_[1].hash());
        challengeStateHash = ChallengeLib.hashChallengeState(0, numBlocks, segments);

        emit InitiatedChallenge();
        emit Bisected(
            challengeStateHash,
            0,
            numBlocks,
            segments
        );
    }

    function getStartGlobalState() external view returns (GlobalState memory) {
        return startAndEndGlobalStates[0];
    }

    function getEndGlobalState() external view returns (GlobalState memory) {
        return startAndEndGlobalStates[1];
    }

    function challengeExecution(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) external {
        require(msg.sender == currentResponder(), "EXEC_SENDER");
        require(
            block.timestamp - lastMoveTimestamp <= currentResponderTimeLeft(),
            "EXEC_DEADLINE"
        );

        (uint256 executionChallengeAtSteps, uint256 challengeLength) = extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        require(
            oldSegments[challengePosition] ==
                ChallengeLib.blockStateHash(
                    machineStatuses[0],
                    globalStateHashes[0]
                ),
            "WRONG_START"
        );
        require(
            oldSegments[challengePosition + 1] !=
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

        uint256 maxInboxMessagesRead = startAndEndGlobalStates[1].getInboxPosition();
        if (machineStatuses[1] == MachineStatus.ERRORED || startAndEndGlobalStates[1].getPositionInMessage() > 0) {
            maxInboxMessagesRead++;
        }


        if (turn == Turn.CHALLENGER) {
            (asserter, challenger) = (challenger, asserter);
            (asserterTimeLeft, challengerTimeLeft) = (challengerTimeLeft, asserterTimeLeft);
        } else if (turn != Turn.ASSERTER) {
            revert(NO_TURN);
        }

        require(numSteps <= OneStepProofEntryLib.MAX_STEPS, "CHALLENGE_TOO_LONG");
        maxInboxMessages = maxInboxMessages;
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = startAndEndHashes[0];
        segments[1] = startAndEndHashes[1];
        challengeStateHash = ChallengeLib.hashChallengeState(0, numSteps, segments);
        lastMoveTimestamp = block.timestamp;
        turn = Turn.CHALLENGER;
        mode = ChallengeMode.EXECUTION;

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
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes calldata proof
    ) external takeTurn {
        (uint256 challengeStart, uint256 challengeLength) = extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        bytes32 afterHash = osp.proveOneStep(
            ExecutionContext({
                maxInboxMessagesRead: maxInboxMessages,
                sequencerInbox: sequencerInbox,
                delayedBridge: delayedBridge
            }),
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
			modulesRoot: wasmModuleRoot
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

    function clearChallenge() external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        turn = Turn.NO_CHALLENGE;
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
}
