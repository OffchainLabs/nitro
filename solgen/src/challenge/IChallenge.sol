//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/GlobalState.sol";
import "./Challenge.sol";

interface IChallenge {
    enum Turn {
        NO_CHALLENGE,
        ASSERTER,
        CHALLENGER
    }

    enum ChallengeMode {
        NONE,
        BLOCK,
        EXECUTION
    }

    struct ChallengeData {
        address asserter;
        address challenger;

        uint256 asserterTimeLeft;
        uint256 challengerTimeLeft;
        uint256 lastMoveTimestamp;

        bytes32 wasmModuleRoot;
        uint256 maxInboxMessages;
        GlobalState[2] startAndEndGlobalStates;

        bytes32 challengeStateHash;

        Turn turn;
        ChallengeMode mode;
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

    event ExecutionChallengeBegun(uint256 blockSteps);
    event OneStepProofCompleted();

    function challengeInfo() external view returns (ChallengeData memory);

//    function asserter() external view returns (address);
//    function challenger() external view returns (address);
//    function lastMoveTimestamp() external view returns (uint256);
    function currentResponderTimeLeft() external view returns (uint256);

    function clearChallenge() external;
    function timeout() external;
}
