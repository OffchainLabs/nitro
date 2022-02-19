//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/DelegateCallAware.sol";
import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./IChallenge.sol";
import "./ExecutionChallenge.sol";
import "./IBlockChallengeFactory.sol";


struct BlockChallengeState {
    BisectableChallengeState bisectionState;
    ExecutionChallengeState executionChallenge;
    bytes32 wasmModuleRoot;
    GlobalState[2] startAndEndGlobalStates;
    uint256 executionChallengeAtSteps;
    ISequencerInbox sequencerInbox;
    IBridge delayedBridge;
}


library BlockChallengeLib {
    using ExecutionChallengeLib for ExecutionChallengeState;
    using BlockChallengeLib for BlockChallengeState;
    using ChallengeCoreLib for BisectableChallengeState;
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    event ExecutionChallengeBegun(uint256 challengeId, uint256 blockSteps);


    function createBlockChallenge(
        BlockChallengeState storage storagePointer,
        IBlockChallengeFactory.ChallengeContracts memory contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) internal {
        // We need to use a storagePointer since solidity can't copy startAndEndGlobalStates_ from mem to state
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

        storagePointer.bisectionState = bisectionState;
        storagePointer.wasmModuleRoot = wasmModuleRoot_;
        storagePointer.startAndEndGlobalStates[0] = startAndEndGlobalStates_[0];
        storagePointer.startAndEndGlobalStates[1] = startAndEndGlobalStates_[1];
        storagePointer.executionChallenge = ExecutionChallengeLib.emptyExecutionState();
        // TODO: validate this value before using
        storagePointer.executionChallengeAtSteps = 0;
        storagePointer.sequencerInbox = ISequencerInbox(contractAddresses.sequencerInbox);
        storagePointer.delayedBridge = IBridge(contractAddresses.delayedBridge);
    }

    function challengeExecution(
        BlockChallengeState memory currChallenge,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) internal {
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

        // TODO: read OSP from manager?
        IOneStepProofEntry temp = IOneStepProofEntry(address(0));
        currChallenge.executionChallenge = ExecutionChallengeLib.createExecutionChallenge(
            temp,
            execCtx,
            startAndEndHashes,
            numSteps,
            newAsserter,
            newChallenger,
            newAsserterTimeLeft,
            newChallengerTimeLeft
        );
        currChallenge.bisectionState.turn = Turn.NO_CHALLENGE;
        // TODO: create Id system for exec/block challenges
        uint256 execChallId = 0;
        emit ExecutionChallengeBegun(execChallId, currChallenge.executionChallengeAtSteps);
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

    // TODO: think through flow of clearing challenges, involves exec and block called by manager
    function clearChallenge(BlockChallengeState memory blockChallengeState) internal pure {
        // require(msg.sender == address(blockChallengeState.bisectionState.resultReceiver), "NOT_RES_RECEIVER");
        blockChallengeState.bisectionState.turn = Turn.NO_CHALLENGE;
        if (blockChallengeState.executionChallenge.isEmpty()) {
            blockChallengeState.executionChallenge.clearChallenge();
        }
    }

    // TODO: this is a callback from execution challenge, manager needs to stich together
    function completeChallenge(
        BlockChallengeState memory currChallenge,
        address /* winner */,
        address /* loser */
    )
        internal pure
    {
        // TODO: this validation is now down by ChallengeManager
        // require(msg.sender == address(currChallenge.bisectionState.executionChallenge), "NOT_EXEC_CHAL");
        // since this is being called by the execution challenge, 
        // and since we transition to NO_CHALLENGE when we create 
        // an execution challenge, that must mean the state is 
        // already NO_CHALLENGE. So we dont technically need to set that here.
        // However to guard against a possible future missed refactoring
        // it's probably safest to set it here anyway
        if(currChallenge.bisectionState.turn != Turn.NO_CHALLENGE) currChallenge.bisectionState.turn = Turn.NO_CHALLENGE;
        // TODO: flatten this out, since exec calls this, which then calls rollup
        // currChallenge.bisectionState.resultReceiver.completeChallenge(winner, loser);
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
