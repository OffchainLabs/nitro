// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pragma solidity ^0.8.0;

import "../challenge/IChallengeManager.sol";
import "../challenge/ChallengeLib.sol";
import "../state/GlobalState.sol";
import "../bridge/ISequencerInbox.sol";

import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "./RollupEventBridge.sol";
import "./IRollupLogic.sol";

struct Config {
    uint64 confirmPeriodBlocks;
    uint64 extraChallengeTimeBlocks;
    address stakeToken;
    uint256 baseStake;
    bytes32 wasmModuleRoot;
    address owner;
    address loserStakeEscrow;
    uint256 chainId;
    ISequencerInbox.MaxTimeVariation sequencerInboxMaxTimeVariation;
}

struct ContractDependencies {
    IBridge delayedBridge;
    ISequencerInbox sequencerInbox;
    IOutbox outbox;
    RollupEventBridge rollupEventBridge;
    IChallengeManager challengeManager;

    IRollupAdmin rollupAdminLogic;
    IRollupUser rollupUserLogic;
}

library RollupLib {
    using GlobalStateLib for GlobalState;

    struct ExecutionState {
        GlobalState globalState;
        MachineStatus machineStatus;
    }

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

    struct Assertion {
        ExecutionState beforeState;
        ExecutionState afterState;
        uint64 numBlocks;
    }

    function executionHash(Assertion calldata assertion)
        internal
        pure
        returns (bytes32)
    {
        MachineStatus[2] memory statuses;
        statuses[0] = assertion.beforeState.machineStatus;
        statuses[1] = assertion.afterState.machineStatus;
        GlobalState[2] memory globalStates;
        globalStates[0] = assertion.beforeState.globalState;
        globalStates[1] = assertion.afterState.globalState;
        // TODO: benchmark how much this abstraction adds of gas overhead
        return executionHash(statuses, globalStates, assertion.numBlocks);
    }

    function executionHash(MachineStatus[2] memory statuses, GlobalState[2] memory globalStates, uint64 numBlocks)
        internal
        pure
        returns (bytes32)
    {
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = ChallengeLib.blockStateHash(
            statuses[0],
            globalStates[0].hash()
        );
        segments[1] = ChallengeLib.blockStateHash(
            statuses[1],
            globalStates[1].hash()
        );
        return
            ChallengeLib.hashChallengeState(0, numBlocks, segments);
    }

    function challengeRootHash(
        bytes32 execution,
        uint256 proposedTime,
        bytes32 wasmModuleRoot
    ) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(
                    execution,
                    proposedTime,
                    wasmModuleRoot
                )
            );
    }

    function confirmHash(Assertion calldata assertion)
        internal
        pure
        returns (bytes32)
    {
        return
            confirmHash(
                assertion.afterState.globalState.getBlockHash(),
                assertion.afterState.globalState.getSendRoot()
            );
    }

    function confirmHash(bytes32 blockHash, bytes32 sendRoot)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(blockHash, sendRoot));
    }

    function nodeHash(
        bool hasSibling,
        bytes32 lastHash,
        bytes32 assertionExecHash,
        bytes32 inboxAcc
    ) internal pure returns (bytes32) {
        uint8 hasSiblingInt = hasSibling ? 1 : 0;
        return
            keccak256(
                abi.encodePacked(
                    hasSiblingInt,
                    lastHash,
                    assertionExecHash,
                    inboxAcc
                )
            );
    }
}
