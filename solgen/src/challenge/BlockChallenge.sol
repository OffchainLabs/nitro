//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./IChallenge.sol";
import "./ExecutionChallenge.sol";
import "./ChallengeManager.sol";
import { ChallengeContracts } from "./IChallengeManager.sol";

struct BlockChallengeState {
    BisectableChallengeState bisectionState;
    bytes32 wasmModuleRoot;
    GlobalState[2] startAndEndGlobalStates;
    uint256 executionChallengeAtSteps;
    ISequencerInbox sequencerInbox;
    IBridge delayedBridge;
}

library BlockChallengeLib {
    using ChallengeCoreLib for BisectableChallengeState;
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    event ExecutionChallengeBegun(uint256 blockSteps);


    function createBlockChallenge(
        BlockChallengeState storage currChallenge,
        ChallengeContracts memory contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) internal {
        bytes32 challengeStateHash;
        {
            bytes32[] memory segments = new bytes32[](2);
            segments[0] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[0], startAndEndGlobalStates_[0].hash());
            segments[1] = ChallengeLib.blockStateHash(startAndEndMachineStatuses_[1], startAndEndGlobalStates_[1].hash());
            challengeStateHash = ChallengeLib.hashChallengeState(0, numBlocks, segments);
            
            emit ChallengeCoreLib.InitiatedChallenge();
            emit ChallengeCoreLib.Bisected(
                challengeStateHash,
                0,
                numBlocks,
                segments
            );
        }

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
        currChallenge.wasmModuleRoot = wasmModuleRoot_;
        currChallenge.startAndEndGlobalStates[0] = startAndEndGlobalStates_[0];
        currChallenge.startAndEndGlobalStates[1] = startAndEndGlobalStates_[1];
        currChallenge.executionChallengeAtSteps = 0;
        currChallenge.sequencerInbox = ISequencerInbox(contractAddresses.sequencerInbox);
        currChallenge.delayedBridge = IBridge(contractAddresses.delayedBridge);
    }

    function challengeExecution(
        mapping(uint256 => ChallengeManager.ChallengeTracker) storage challenges,
        uint256 challengeId,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) internal {
        // we pass the mapping and key instead of the struct since this uses less of the stack
        ChallengeManager.ChallengeTracker storage currTrckr = challenges[challengeId];
        require(
            currTrckr.trackerState ==
            ChallengeManager.ChallengeTrackerState.PendingBlockChallenge,
            "NOT_BLOCK_CHALL"
        );
        BlockChallengeState storage currChallenge = currTrckr.blockChallState;

        require(msg.sender == currChallenge.bisectionState.currentResponder(), "EXEC_SENDER");
        require(
            block.timestamp - currChallenge.bisectionState.lastMoveTimestamp <= currChallenge.bisectionState.currentResponderTimeLeft(),
            "EXEC_DEADLINE"
        );

        uint256 challengeLength;
        (currChallenge.executionChallengeAtSteps, challengeLength) = currChallenge.bisectionState.extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        address newAsserter = currChallenge.bisectionState.asserter;
        address newChallenger = currChallenge.bisectionState.challenger;
        uint256 newAsserterTimeLeft = currChallenge.bisectionState.asserterTimeLeft;
        uint256 newChallengerTimeLeft = currChallenge.bisectionState.challengerTimeLeft;

        if (currChallenge.bisectionState.turn == Turn.CHALLENGER) {
            (newAsserter, newChallenger) = (newChallenger, newAsserter);
            (newAsserterTimeLeft, newChallengerTimeLeft) = (
                newChallengerTimeLeft,
                newAsserterTimeLeft
            );
        } else if (currChallenge.bisectionState.turn != Turn.ASSERTER) {
            revert(NO_TURN);
        }

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
            _currentWin(currChallenge);
            return;
        }

        if (machineStatuses[1] == MachineStatus.ERRORED) {
            // If the machine errors, it must return to the previous global state
            require(globalStateHashes[0] == globalStateHashes[1], "ERROR_CHANGE");
        }

        bytes32[2] memory startAndEndHashes;
        startAndEndHashes[0] = getStartMachineHash(
            globalStateHashes[0],
            currChallenge.wasmModuleRoot
        );
        startAndEndHashes[1] = getEndMachineHash(
            machineStatuses[1],
            globalStateHashes[1]
        );

        ExecutionContext memory execCtx = ExecutionContext({
            maxInboxMessagesRead: currChallenge.startAndEndGlobalStates[1].getInboxPosition(),
            sequencerInbox: currChallenge.sequencerInbox,
            delayedBridge: currChallenge.delayedBridge
        });

        // block bisection is now complete, so we create an execution challenge
        currChallenge.bisectionState.turn = Turn.NO_CHALLENGE;
        currTrckr.trackerState = ChallengeManager.ChallengeTrackerState.PendingExecutionChallenge;
        ExecutionChallengeLib.createExecutionChallenge(
            currTrckr.execChallState,
            execCtx,
            startAndEndHashes,
            numSteps,
            newAsserter,
            newChallenger,
            newAsserterTimeLeft,
            newChallengerTimeLeft
        );
        // TODO: should we emit the challenge id here? could be useful, but validator should already have it laying around
        emit ExecutionChallengeBegun(currChallenge.executionChallengeAtSteps);
    }

    function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot)
        internal
        pure
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

    function _currentWin(BlockChallengeState memory blockChallengeState) private pure {
        // As a safety measure, challenges can only be resolved by timeouts during mainnet beta.
        // As state is 0, no move is possible. The other party will lose via timeout
        blockChallengeState.bisectionState.challengeStateHash = bytes32(0);

        // if (turn == Turn.ASSERTER) {
        //     _asserterWin();
        // } else if (turn == Turn.CHALLENGER) {
        //     _challengerWin();
        // } else {
        // 	   revert(NO_TURN);
        // }
    }
}
