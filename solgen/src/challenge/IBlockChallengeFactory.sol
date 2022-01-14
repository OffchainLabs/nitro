//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "./IChallenge.sol";
import "./IChallengeResultReceiver.sol";

interface IBlockChallengeFactory {
    // contractAddresses = [ resultReceiver, sequencerInbox, delayedBridge ]
    function createChallenge(
        address[3] calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (IChallenge);
}
