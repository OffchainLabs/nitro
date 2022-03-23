// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Provides a registry of BLS public keys for accounts.
/// @notice Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000067.
interface ArbBLS {
    /// @notice Deprecated -- equivalent to registerAltBN128
    function register(
        uint256 x0,
        uint256 x1,
        uint256 y0,
        uint256 y1
    ) external; // DEPRECATED

    /// @notice Deprecated -- equivalent to getAltBN128
    function getPublicKey(address addr)
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256
        ); // DEPRECATED

    /// @notice Associate an AltBN128 public key with the caller's address
    function registerAltBN128(
        uint256 x0,
        uint256 x1,
        uint256 y0,
        uint256 y1
    ) external;

    /// @notice Get the AltBN128 public key associated with an address (revert if there isn't one)
    function getAltBN128(address addr)
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256
        );

    /// @notice Associate a BLS 12-381 public key with the caller's address
    function registerBLS12381(bytes calldata key) external;

    /// @notice Get the BLS 12-381 public key associated with an address (revert if there isn't one)
    function getBLS12381(address addr) external view returns (bytes memory);
}
