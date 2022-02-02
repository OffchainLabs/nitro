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

import "../challenge/ChallengeLib.sol";
import "../state/GlobalStates.sol";

library RollupLib {
    struct Config {
        uint256 confirmPeriodBlocks;
        uint256 extraChallengeTimeBlocks;
        address stakeToken;
        uint256 baseStake;
        bytes32 wasmModuleRoot;
        address owner;
        uint256 chainId;
        // maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds
        uint256[4] sequencerInboxParams;
    }

    struct ExecutionState {
        GlobalState globalState;
        uint256 inboxMaxCount;
        MachineStatus machineStatus;
    }

    function stateHash(ExecutionState memory execState)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encodePacked(
                    GlobalStates.hash(execState.globalState),
                    execState.inboxMaxCount
                )
            );
    }

    struct Assertion {
        ExecutionState beforeState;
        ExecutionState afterState;
        uint64 numBlocks;
    }

    function decodeExecutionState(
        bytes32[2] memory bytes32Fields,
        uint64[3] memory intFields,
        uint256 inboxMaxCount
    ) internal pure returns (ExecutionState memory) {
        require(intFields[2] == uint64(MachineStatus.FINISHED) || intFields[2] == uint64(MachineStatus.ERRORED), "BAD_STATUS");
        MachineStatus machineStatus = MachineStatus(intFields[2]);
        uint64[2] memory gsIntFields;
        gsIntFields[0] = intFields[0];
        gsIntFields[1] = intFields[1];
        return
            ExecutionState(
                GlobalState(bytes32Fields, gsIntFields),
                inboxMaxCount,
                machineStatus
            );
    }

    function decodeAssertion(
        bytes32[2][2] memory bytes32Fields,
        uint64[3][2] memory intFields,
        uint256 beforeInboxMaxCount,
        uint256 inboxMaxCount,
        uint64 numBlocks
    ) internal pure returns (Assertion memory) {
        return
            Assertion(
                decodeExecutionState(
                    bytes32Fields[0],
                    intFields[0],
                    beforeInboxMaxCount
                ),
                decodeExecutionState(
                    bytes32Fields[1],
                    intFields[1],
                    inboxMaxCount
                ),
                numBlocks
            );
    }

    function executionHash(Assertion memory assertion)
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
            GlobalStates.hash(globalStates[0])
        );
        segments[1] = ChallengeLib.blockStateHash(
            statuses[1],
            GlobalStates.hash(globalStates[1])
        );
        return
            ChallengeLib.hashChallengeState(0, numBlocks, segments);
    }

    function challengeRoot(
        Assertion memory assertion,
        bytes32 assertionExecHash,
        uint256 blockProposed,
        bytes32 wasmModuleRoot
    ) internal pure returns (bytes32) {
        return
            challengeRootHash(
                assertionExecHash,
                blockProposed,
                wasmModuleRoot
            );
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

    function confirmHash(Assertion memory assertion)
        internal
        pure
        returns (bytes32)
    {
        return
            confirmHash(
                GlobalStates.getBlockHash(assertion.afterState.globalState),
                GlobalStates.getSendRoot(assertion.afterState.globalState)
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
