// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Deprecated - Info about the rollup just prior to the Nitro upgrade
/// @notice Precompiled contract in every Arbitrum chain for retryable transaction related data retrieval and interactions. Exists at 0x000000000000000000000000000000000000006f
interface ArbStatistics {
    /// @notice Get Arbitrum block number and other statistics as they were right before the Nitro upgrade.
    /// @return (
    ///      Number of accounts,
    ///      Total storage allocated (includes storage that was later deallocated),
    ///      Total ArbGas used,
    ///      Number of transaction receipt issued,
    ///      Number of contracts created,
    ///    )
    function getStats()
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256,
            uint256,
            uint256
        );
}
