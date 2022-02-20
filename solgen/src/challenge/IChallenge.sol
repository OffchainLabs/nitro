//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

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

    function asserter() external view returns (address);
    function challenger() external view returns (address);
    function lastMoveTimestamp() external view returns (uint256);
    function currentResponderTimeLeft() external view returns (uint256);

    function clearChallenge() external;
    function timeout() external;
}
