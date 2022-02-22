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

/// @title Provides insight into the cost of using the chain. 
/// @notice These methods have been adjusted to account for Nitro's heavy use of calldata compression. 
/// Of note to end-users, we no longer make a distinction between non-zero and zero-valued calldata bytes.
/// Precompiled contract that exists in every Arbitrum chain at 0x000000000000000000000000000000000000006c.
interface ArbGasInfo {
    /// @notice Get gas prices for a provided aggregator
    /// @return return gas prices in wei
    ///        (
    ///            per L2 tx,
    ///            per L1 calldata unit, (a byte, non-zero or otherwise, is 16 units)
    ///            per storage allocation,
    ///            per ArbGas base,
    ///            per ArbGas congestion,
    ///            per ArbGas total
    ///        )
    function getPricesInWeiWithAggregator(address aggregator) external view returns (uint, uint, uint, uint, uint, uint);

    /// @notice Get gas prices. Uses the caller's preferred aggregator, or the default if the caller doesn't have a preferred one.
    /// @return return gas prices in wei
    ///        (
    ///            per L2 tx,
    ///            per L1 calldata unit, (a byte, non-zero or otherwise, is 16 units)
    ///            per storage allocation,
    ///            per ArbGas base,
    ///            per ArbGas congestion,
    ///            per ArbGas total
    ///        )
    function getPricesInWei() external view returns (uint, uint, uint, uint, uint, uint);

    /// @notice Get prices in ArbGas for the supplied aggregator
    /// @return (per L2 tx, per L1 calldata unit, per storage allocation)
    function getPricesInArbGasWithAggregator(address aggregator) external view returns (uint, uint, uint);

    /// @notice Get prices in ArbGas. Assumes the callers preferred validator, or the default if caller doesn't have a preferred one.
    /// @return (per L2 tx, per L1 calldata unit, per storage allocation)
    function getPricesInArbGas() external view returns (uint, uint, uint);

    /// @notice Get the gas accounting parameters
    /// @return (speedLimitPerSecond, gasPoolMax, maxTxGasLimit)
    function getGasAccountingParams() external view returns (uint, uint, uint);

    /// @notice Get the minimum gas price needed for a tx to succeed
    function getMinimumGasPrice() external view returns(uint);

    /// @notice Get the number of seconds worth of the speed limit the large gas pool contains
    function getGasPoolSeconds() external view returns(uint);

    /// @notice Get the number of seconds worth of the speed limit the small gas pool contains
    function getSmallGasPoolSeconds() external view returns(uint);

    /// @notice Get ArbOS's estimate of the L1 gas price in wei
    function getL1GasPriceEstimate() external view returns(uint);

    /// @notice Get L1 gas fees paid by the current transaction
    function getCurrentTxL1GasFees() external view returns(uint);
}
