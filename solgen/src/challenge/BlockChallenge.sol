//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./ChallengeCore.sol";
import "./IExecutionChallenge.sol";
import "./IExecutionChallengeFactory.sol";
import "./Cloneable.sol";

contract BlockChallenge is ChallengeCore, IChallengeResultReceiver, Cloneable {
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
    event ContinuedExecutionProven();

    uint256 constant MAX_CHALLENGE_DEGREE = 40;

    string constant NO_TURN = "NO_TURN";

    IExecutionChallengeFactory public executionChallengeFactory;
    IChallengeResultReceiver resultReceiver;

    ExecutionContext public execCtx;
    bytes32 public wasmModuleRoot;

    address public asserter;
    address public challenger;

    uint256 public asserterTimeLeft;
    uint256 public challengerTimeLeft;
    uint256 public lastMoveTimestamp;

    Turn public turn;

    IExecutionChallenge public executionChallenge;

    constructor(
        IExecutionChallengeFactory executionChallengeFactory_,
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32 wasmModuleRoot_,
        bytes32 challengeStateHash_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) {
        executionChallengeFactory = executionChallengeFactory_;
        resultReceiver = resultReceiver_;
        execCtx = execCtx_;
        wasmModuleRoot = wasmModuleRoot_;
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
        require(address(executionChallenge) == address(0), "BIS_EXEC");

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
        (uint256 challengeStart, uint256 challengeLength) = extractChallengeSegment(
                oldSegmentsStart,
                oldSegmentsLength,
                oldSegments,
                challengePosition
            );
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

    function challengeExecution(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes
    ) external {
        require(msg.sender == currentResponder(), "EXEC_SENDER");
        require(
            block.timestamp - lastMoveTimestamp <= currentResponderTimeLeft(),
            "EXEC_DEADLINE"
        );

        (, uint256 challengeLength) = extractChallengeSegment(
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

        bytes32 execChallengeStateHash;
        {
            bytes32 startMachineHash = getStartMachineHash(
                globalStateHashes[0]
            );
            bytes32 endMachineHash = getEndMachineHash(
                machineStatuses[1],
                globalStateHashes[1]
            );
            bytes32[] memory machineSegments = new bytes32[](2);
            machineSegments[0] = startMachineHash;
            machineSegments[1] = endMachineHash;
            execChallengeStateHash = ChallengeLib.hashChallengeState(
                0,
                ~uint64(0), // Constrain max machine steps to max uint64
                machineSegments
            );
        }

        executionChallenge = executionChallengeFactory.createChallenge(
            this,
            execCtx,
            execChallengeStateHash,
            newAsserter,
            newChallenger,
            newAsserterTimeLeft,
            newChallengerTimeLeft
        );
        turn = Turn.NO_CHALLENGE;
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
            startingValues[0] = Values.newRefNull();
            startingValues[1] = Values.newI32(0);
            startingValues[2] = Values.newI32(0);
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
        return Machines.hash(mach);
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

    function timeout() external {
        require(address(executionChallenge) != address(0), "TIMEOUT_EXEC");
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

    function clearChallenge() external {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        if (address(executionChallenge) != address(0)) {
            executionChallenge.clearChallenge();
        }
        safeSelfDestruct(payable(0));
    }

    function completeChallenge(address winner, address loser)
        external
        override
    {
        require(msg.sender == address(executionChallenge), "NOT_EXEC_CHAL");
        resultReceiver.completeChallenge(winner, loser);
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
