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

/// @title Provides owners with tools for managing the rollup.
/// @notice Calls by non-owners will always revert.
/// Most of Arbitrum Classic's owner methods have been removed since they no longer make sense in Nitro:
/// - What were once chain parameters are now parts of ArbOS's state, and those that remain are set at genesis.
/// - ArbOS upgrades happen with the rest of the system rather than being independent
/// - Exemptions to address aliasing are no longer offered. Exemptions were intended to support backward compatibility for contracts deployed before aliasing was introduced, but no exemptions were ever requested.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000070.
interface ArbOwner {
    /// @notice Add account as a chain owner
    function addChainOwner(address newOwner) external;

    /// @notice Remove account from the list of chain owners
    function removeChainOwner(address ownerToRemove) external;

    /// @notice See if the user is a chain owner
    function isChainOwner(address addr) external view returns (bool);

    /// @notice Retrieves the list of chain owners
    function getAllChainOwners() external view returns (address[] memory);

    /// @notice Sets the L1 gas price estimate directly, bypassing the autoregression
    function setL1GasPriceEstimate(uint256 priceInWei) external;

    /// @notice Sets the L2 gas price directly, bypassing the pool calculus
    function setL2GasPrice(uint256 priceInWei) external;

    /// @notice Sets the minimum gas price needed for a transaction to succeed
    function setMinimumGasPrice(uint256 priceInWei) external;

    /// @notice Sets the computational speed limit for the chain
    function setSpeedLimit(uint64 limit) external view;

    /// @notice Sets the number of seconds worth of the speed limit the large gas pool contains
    function setGasPoolSeconds(uint64 factor) external view;

    /// @notice Sets the maximum size a tx (and block) can be
    function setMaxTxGasLimit(uint64 limit) external view;

    /// @notice Gets the network fee collector
    function getNetworkFeeAccount() external view returns (address);

    /// @notice Sets the network fee collector
    function setNetworkFeeAccount(address newNetworkFeeAccount) external view;

    // Emitted when a successful call is made to this precompile
    event OwnerActs(bytes4 indexed method, address indexed owner, bytes data);
}
