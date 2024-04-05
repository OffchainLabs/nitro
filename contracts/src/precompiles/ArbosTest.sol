// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Deprecated - Provides a method of burning arbitrary amounts of gas,
/// @notice This exists for historical reasons. Pre-Nitro, `ArbosTest` had additional methods only the zero address could call.
/// These have been removed since users don't use them and calls to missing methods revert.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000069.
interface ArbosTest {
    /// @notice Unproductively burns the amount of L2 ArbGas
    function burnArbGas(uint256 gasAmount) external pure;
}
