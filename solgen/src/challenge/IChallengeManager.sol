//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./IChallengeResultReceiver.sol";
import "../osp/IOneStepProofEntry.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/IBridge.sol";

struct ChallengeContracts {
    IChallengeResultReceiver resultReceiver;
    ISequencerInbox sequencerInbox;
    IBridge delayedBridge;
}

interface IChallengeManager {
    function osp() external returns (IOneStepProofEntry);

    function createChallenge(
        ChallengeContracts calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (uint256);

    /// @dev only callable by the result receiver
    function clearChallenge(uint256 challengeId) external;

    function challengeWinner(uint256 challengeId) external view returns (address);
}
