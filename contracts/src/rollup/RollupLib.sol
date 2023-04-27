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

    // Not the same as a machine hash for a given execution state
    function executionStateHash(ExecutionState memory state) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(state.machineStatus, state.globalState.hash()));
    }

    // The `assertionHash` contains all the information needed to determine an assertion's validity.
    // This helps protect validators against reorgs by letting them bind their assertion to the current chain state.
    function assertionHash(
        bytes32 parentAssertionHash,
        ExecutionState memory afterState,
        bytes32 inboxAcc,
        bytes32 wasmModuleRoot
    ) internal pure returns (bytes32) {
        // we can no longer have `hasSibling` in the assertion hash as it would allow identical assertions
        // uint8 hasSiblingInt = hasSibling ? 1 : 0;
        return
            keccak256(
                abi.encodePacked(
                    parentAssertionHash,
                    executionStateHash(afterState),
                    inboxAcc,
                    wasmModuleRoot
                )
            );
    }

    // Takes in a hash of the afterState instead of the afterState itself
    function assertionHash(
        bytes32 parentAssertionHash,
        bytes32 afterStateHash,
        bytes32 inboxAcc,
        bytes32 wasmModuleRoot
    ) internal pure returns (bytes32) {
        // we can no longer have `hasSibling` in the assertion hash as it would allow identical assertions
        // uint8 hasSiblingInt = hasSibling ? 1 : 0;
        return
            keccak256(
                abi.encodePacked(
                    parentAssertionHash,
                    afterStateHash,
                    inboxAcc,
                    wasmModuleRoot
                )
            );
    }
}
