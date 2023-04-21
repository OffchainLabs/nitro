// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/IOldChallengeManager.sol";
import "../challenge/OldChallengeLib.sol";
import "../state/GlobalState.sol";
import "../bridge/ISequencerInbox.sol";

import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../bridge/IInbox.sol";
import "./Assertion.sol";
import "./IRollupEventInbox.sol";
import "../challengeV2/EdgeChallengeManager.sol";

library RollupLib {
    using GlobalStateLib for GlobalState;

    function stateHash(ExecutionState calldata execState, uint256 inboxMaxCount)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encodePacked(
                    execState.globalState.hash(),
                    inboxMaxCount,
                    execState.machineStatus
                )
            );
    }

    /// @dev same as stateHash but expects execState in memory instead of calldata
    function stateHashMem(ExecutionState memory execState, uint256 inboxMaxCount)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encodePacked(
                    execState.globalState.hash(),
                    inboxMaxCount,
                    execState.machineStatus
                )
            );
    }

    function executionHash(AssertionInputs memory assertion) internal pure returns (bytes32) {
        MachineStatus[2] memory statuses;
        statuses[0] = assertion.beforeState.machineStatus;
        statuses[1] = assertion.afterState.machineStatus;
        GlobalState[2] memory globalStates;
        globalStates[0] = assertion.beforeState.globalState;
        globalStates[1] = assertion.afterState.globalState;
        // TODO: benchmark how much this abstraction adds of gas overhead
        return executionHash(statuses, globalStates, 0); // hardcoded numBlocks to 0 
        // TODO: remove numBlocks from executionHash as it is now a constant
    }

    function executionHash(
        MachineStatus[2] memory statuses,
        GlobalState[2] memory globalStates,
        uint64 numBlocks
    ) internal pure returns (bytes32) {
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = OldChallengeLib.blockStateHash(statuses[0], globalStates[0].hash());
        segments[1] = OldChallengeLib.blockStateHash(statuses[1], globalStates[1].hash());
        return OldChallengeLib.hashChallengeState(0, numBlocks, segments);
    }

    function confirmHash(AssertionInputs memory assertion) internal pure returns (bytes32) {
        return
            confirmHash(
                assertion.afterState.globalState.getBlockHash(),
                assertion.afterState.globalState.getSendRoot()
            );
    }

    function confirmHash(bytes32 blockHash, bytes32 sendRoot) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(blockHash, sendRoot));
    }

    function assertionHash(
        bytes32 lastHash,
        bytes32 assertionExecHash,
        bytes32 inboxAcc,
        bytes32 wasmModuleRoot
    ) internal pure returns (bytes32) {
        // we can no longer have `hasSibling` in the assertion hash as it would allow identical assertions
        // uint8 hasSiblingInt = hasSibling ? 1 : 0;
        return
            keccak256(
                abi.encodePacked(
                    lastHash,
                    assertionExecHash,
                    inboxAcc,
                    wasmModuleRoot
                )
            );
    }
}
