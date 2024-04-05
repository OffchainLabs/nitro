// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Provides aggregators and their users methods for configuring how they participate in L1 aggregation.
/// @notice Precompiled contract that exists in every Arbitrum chain at 0x000000000000000000000000000000000000006d
interface ArbAggregator {
    /// @notice Deprecated, customization of preferred aggregator is no longer supported
    /// @notice Get the address of an arbitrarily chosen batch poster.
    /// @param addr ignored
    /// @return (batchPosterAddress, true)
    function getPreferredAggregator(address addr) external view returns (address, bool);

    /// @notice Deprecated, there is no longer a single preferred aggregator, use getBatchPosters instead
    /// @notice Get default aggregator.
    function getDefaultAggregator() external view returns (address);

    /// @notice Get a list of all current batch posters
    /// @return Batch poster addresses
    function getBatchPosters() external view returns (address[] memory);

    /// @notice Adds newBatchPoster as a batch poster
    /// This reverts unless called by a chain owner
    /// @param newBatchPoster New batch poster
    function addBatchPoster(address newBatchPoster) external;

    /// @notice Get the address where fees to batchPoster are sent.
    /// @param batchPoster The batch poster to get the fee collector for
    /// @return The fee collectors address. This will sometimes but not always be the same as the batch poster's address.
    function getFeeCollector(address batchPoster) external view returns (address);

    /// @notice Set the address where fees to batchPoster are sent.
    /// This reverts unless called by the batch poster, its fee collector, or a chain owner
    /// @param batchPoster The batch poster to set the fee collector for
    /// @param newFeeCollector The new fee collector to set
    function setFeeCollector(address batchPoster, address newFeeCollector) external;

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
