// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
/// @notice Precompiled contract that exists in every Arbitrum chain at 0x000000000000000000000000000000000000006d
interface ArbAggregator {
    /// @notice Get the preferred aggregator for an address.
    /// @param addr The address to fetch aggregator for
    /// @return (preferredAggregatorAddress, isDefault)
    ///     isDefault is true if addr is set to prefer the default aggregator
    function getPreferredAggregator(address addr) external view returns (address, bool);

    /// @notice Get default aggregator.
    function getDefaultAggregator() external view returns (address);

    /// @notice Set the preferred aggregator.
    /// This reverts unless called by the aggregator, its fee collector, or a chain owner
    /// @param newDefault New default aggregator
    function setDefaultAggregator(address newDefault) external;

    /// @notice Get the address where fees to aggregator are sent.
    /// @param aggregator The aggregator to get the fee collector for
    /// @return The fee collectors address. This will often but not always be the same as the aggregator's address.
    function getFeeCollector(address aggregator) external view returns (address);

    /// @notice Set the address where fees to aggregator are sent.
    /// This reverts unless called by the aggregator, its fee collector, or a chain owner
    /// @param aggregator The aggregator to set the fee collector for
    /// @param newFeeCollector The new fee collector to set
    function setFeeCollector(address aggregator, address newFeeCollector) external;

    /// @notice Deprecated, always returns zero
    /// @notice Get the tx base fee (in approximate L1 gas) for aggregator
    /// @param aggregator The aggregator to get the base fee for
    function getTxBaseFee(address aggregator) external view returns (uint256);

    /// @notice Deprecated, is now a no-op
    /// @notice Set the tx base fee (in approximate L1 gas) for aggregator
    /// Revert unless called by aggregator or the chain owner
    /// Revert if feeInL1Gas is outside the chain's allowed bounds
    /// @param aggregator The aggregator to set the fee for
    /// @param feeInL1Gas The base fee in L1 gas
    function setTxBaseFee(address aggregator, uint256 feeInL1Gas) external;
}
