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

/// @title Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
/// @notice Precompiled contract that exists in every Arbitrum chain at 0x000000000000000000000000000000000000006d
interface ArbAggregator {
    /// @notice Get the preferred aggregator for an address.
    /// @param addr The address to fetch aggregator for
    /// @return (preferredAggregatorAddress, isDefault)
    ///     isDefault is true if addr is set to prefer the default aggregator
    function getPreferredAggregator(address addr) external view returns (address, bool);

    /// @notice Set the caller's preferred aggregator.
    /// @param prefAgg If prefAgg is zero, this sets the caller to prefer the default aggregator
    function setPreferredAggregator(address prefAgg) external;

    /// @notice Get default aggregator.
    function getDefaultAggregator() external view returns (address);

    /// @notice Set the preferred aggregator.
    /// This reverts unless called by the aggregator, its fee collector, or a chain owner
    /// @param newDefault New default aggregator
    function setDefaultAggregator(address newDefault) external;

    /// @notice Get the aggregator's compression ratio
    /// @param aggregator The aggregator to fetch the compression ratio for
    /// @return The compression ratio, measured in basis points
    function getCompressionRatio(address aggregator) external view returns (uint64);

    /// @notice Set the aggregator's compression ratio
    /// This reverts unless called by the aggregator, its fee collector, or a chain owner
    /// @param aggregator The aggregator to set the compression ratio for
    /// @param ratio The compression ratio, measured in basis points
    function setCompressionRatio(address aggregator, uint64 ratio) external;

    /// @notice Get the address where fees to aggregator are sent.
    /// @param aggregator The aggregator to get the fee collector for
    /// @return The fee collectors address. This will often but not always be the same as the aggregator's address.
    function getFeeCollector(address aggregator) external view returns (address);

    /// @notice Set the address where fees to aggregator are sent.
    /// This reverts unless called by the aggregator, its fee collector, or a chain owner
    /// @param aggregator The aggregator to set the fee collector for
    /// @param newFeeCollector The new fee collector to set
    function setFeeCollector(address aggregator, address newFeeCollector) external;

    /// @notice Get the tx base fee (in approximate L1 gas) for aggregator
    /// @param aggregator The aggregator to get the base fee for
    function getTxBaseFee(address aggregator) external view returns (uint256);

    /// @notice Set the tx base fee (in approximate L1 gas) for aggregator
    /// Revert unless called by aggregator or the chain owner
    /// Revert if feeInL1Gas is outside the chain's allowed bounds
    /// @param aggregator The aggregator to set the fee for
    /// @param feeInL1Gas The base fee in L1 gas
    function setTxBaseFee(address aggregator, uint256 feeInL1Gas) external;
}
