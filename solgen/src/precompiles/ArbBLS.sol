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

/// @title Provides a registry of BLS public keys for accounts.
/// @notice Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000067.
interface ArbBLS {
    /// @notice Deprecated -- equivalent to registerAltBN128
    function register(uint x0, uint x1, uint y0, uint y1) external;  // DEPRECATED

    /// @notice Deprecated -- equivalent to getAltBN128
    function getPublicKey(address addr) external view returns (uint, uint, uint, uint);  // DEPRECATED

    /// @notice Associate an AltBN128 public key with the caller's address
    function registerAltBN128(uint x0, uint x1, uint y0, uint y1) external;

    /// @notice Get the AltBN128 public key associated with an address (revert if there isn't one)
    function getAltBN128(address addr) external view returns (uint, uint, uint, uint);

    /// @notice Associate a BLS 12-381 public key with the caller's address
    function registerBLS12381(bytes calldata key) external;

    /// @notice Get the BLS 12-381 public key associated with an address (revert if there isn't one)
    function getBLS12381(address addr) external view returns (bytes memory);
}
