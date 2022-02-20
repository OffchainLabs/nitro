//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Machine.sol";
import "../bridge/IBridge.sol";
import "../bridge/ISequencerInbox.sol";
import "../osp/IOneStepProofEntry.sol";

import "./IChallengeResultReceiver.sol";

import "./ChallengeLib.sol";

interface IChallengeManager {
    enum Turn {
        NO_CHALLENGE,
        ASSERTER,
        CHALLENGER
    }

    enum ChallengeWinner {
        NONE,
        ASSERTER,
        CHALLENGER
    }

    enum ChallengeTerminationType {
        TIMEOUT,
        CLEARED
    }

    event InitiatedChallenge(uint64 indexed challengeIndex);

    event Bisected(
        uint64 indexed challengeIndex,
        bytes32 indexed challengeRoot,
        uint256 challengedSegmentStart,
        uint256 challengedSegmentLength,
        bytes32[] chainHashes
    );

    event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps);
    event OneStepProofCompleted(uint64 indexed challengeIndex);

    event ChallengeEnded(uint64 indexed challengeIndex, ChallengeTerminationType kind);

    function initialize(
        IChallengeResultReceiver resultReceiver_,
        ISequencerInbox sequencerInbox_,
        IBridge delayedBridge_,
        IOneStepProofEntry osp_
    ) external;

    function createChallenge(
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (uint64);

    function challengeInfo(uint64 challengeIndex_) external view returns (ChallengeLib.Challenge memory);

//    function asserter() external view returns (address);
//    function challenger() external view returns (address);
//    function lastMoveTimestamp() external view returns (uint256);
    function currentResponder(uint64 challengeIndex) external view returns (address);
    function isTimedOut(uint64 challengeIndex) external view returns (bool);
    function currentResponderTimeLeft(uint64 challengeIndex_) external view returns (uint256);

    function clearChallenge(uint64 challengeIndex_) external;
    function timeout(uint64 challengeIndex_) external;
}
