//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/Cloneable.sol";
import "../osp/IOneStepProofEntry.sol";
import "../state/GlobalState.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./ChallengeCore.sol";
import "./IChallenge.sol";
import "./IExecutionChallengeFactory.sol";

contract BlockChallenge is ChallengeCore, IChallengeResultReceiver, IChallenge {
    using GlobalStateLib for GlobalState;
    using MachineLib for Machine;

    event ExecutionChallengeBegun(IChallenge indexed challenge, uint256 blockSteps);

    IExecutionChallengeFactory public executionChallengeFactory;

    bytes32 public wasmModuleRoot;
    GlobalState[2] internal startAndEndGlobalStates;

    IChallenge public executionChallenge;
    uint256 public executionChallengeAtSteps;

    ISequencerInbox public sequencerInbox;
    IBridge public delayedBridge;

    // contractAddresses = [ resultReceiver, sequencerInbox, delayedBridge ]
    function initialize(
        IExecutionChallengeFactory executionChallengeFactory_,
        address[3] memory contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external {
        executionChallengeFactory = executionChallengeFactory_;
        resultReceiver = IChallengeResultReceiver(contractAddresses[0]);
        sequencerInbox = ISequencerInbox(contractAddresses[1]);
        delayedBridge = IBridge(contractAddresses[2]);
        wasmModuleRoot = wasmModuleRoot_;
        startAndEndGlobalStates[0] = startAndEndGlobalStates_[0];
        startAndEndGlobalStates[1] = startAndEndGlobalStates_[1];
        asserter = asserter_;
        challenger = challenger_;
        asserterTimeLeft = asserterTimeLeft_;
        challengerTimeLeft = challengerTimeLeft_;
        lastMoveTimestamp = block.timestamp;
        turn = Turn.CHALLENGER;

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

        uint256 challengeLength;
        (executionChallengeAtSteps, challengeLength) = extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        address newAsserter = asserter;
        address newChallenger = challenger;
        uint256 newAsserterTimeLeft = asserterTimeLeft;
        uint256 newChallengerTimeLeft = challengerTimeLeft;

        if (turn == Turn.CHALLENGER) {
            (newAsserter, newChallenger) = (newChallenger, newAsserter);
            (newAsserterTimeLeft, newChallengerTimeLeft) = (
                newChallengerTimeLeft,
                newAsserterTimeLeft
            );
        } else if (turn != Turn.ASSERTER) {
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

        ExecutionContext memory execCtx = ExecutionContext({
            maxInboxMessagesRead: startAndEndGlobalStates[1].getInboxPosition(),
            sequencerInbox: sequencerInbox,
            delayedBridge: delayedBridge
        });

        executionChallenge = executionChallengeFactory.createChallenge(
            this,
            execCtx,
            startAndEndHashes,
            numSteps,
            newAsserter,
            newChallenger,
            newAsserterTimeLeft,
            newChallengerTimeLeft
        );
        turn = Turn.NO_CHALLENGE;

        emit ExecutionChallengeBegun(executionChallenge, executionChallengeAtSteps);
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
        if (address(executionChallenge) != address(0)) {
            executionChallenge.clearChallenge();
        }
    }

    function completeChallenge(address winner, address loser)
        external
        override 
    {
        require(msg.sender == address(executionChallenge), "NOT_EXEC_CHAL");
        // since this is being called by the execution challenge, 
        // and since we transition to NO_CHALLENGE when we create 
        // an execution challenge, that must mean the state is 
        // already NO_CHALLENGE. So we dont technically need to set that here.
        // However to guard against a possible future missed refactoring
        // it's probably safest to set it here anyway
        if(turn != Turn.NO_CHALLENGE) turn = Turn.NO_CHALLENGE;
        resultReceiver.completeChallenge(winner, loser);
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
