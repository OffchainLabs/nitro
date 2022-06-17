// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2020, Offchain Labs, Inc.
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

pragma solidity >=0.4.21 <0.9.0;

/**
 * @title This precompile represents ArbOS's internal actions as calls it makes to itself
 * @notice Calling this precompile will always revert and should not be done.
 */
interface ArbosActs {
    /**
     * @notice ArbOS "calls" this when starting a block
     * @param l1BaseFee the L1 BaseFee
     * @param l1BlockNumber the L1 block number
     * @param timePassed number of seconds since the last block
     */
    function startBlock(
        uint256 l1BaseFee,
        uint64 l1BlockNumber,
        uint64 l2BlockNumber,
        uint64 timePassed
    ) external;

    function batchPostingReport(
        uint256 batchTimestamp,
        address batchPosterAddress,
        uint64 batchNumber,
        uint64 batchDataGas,
        uint256 l1BaseFeeWei
    ) external;

    error CallerNotArbOS();
}
