//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/GlobalState.sol";
import "../state/Machine.sol";
import "../bridge/IBridge.sol";
import "../bridge/ISequencerInbox.sol";
import "../osp/IOneStepProofEntry.sol";

import "./IChallengeResultReceiver.sol";

interface IChallengeManager {
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

    enum ChallengeWinner {
        NONE,
        ASSERTER,
        CHALLENGER
    }

    enum ChallengeTerminationType {
        TIMEOUT,
        CHALLENGER_TIMED_OUT
    }

    struct Challenge {
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

    event ChallengedEnded(uint64 indexed challengeIndex, ChallengeWinner winner, ChallengeTerminationType kind);

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

    function challengeInfo(uint64 challengeIndex_) external view returns (Challenge memory);

//    function asserter() external view returns (address);
//    function challenger() external view returns (address);
//    function lastMoveTimestamp() external view returns (uint256);
    function currentResponderTimeLeft(uint64 challengeIndex_) external view returns (uint256);

    function clearChallenge(uint64 challengeIndex_) external;
    function timeout(uint64 challengeIndex_) external;
}
