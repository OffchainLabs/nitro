// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/**
 * @title Methods for managing Stylus caches
 * @notice Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000072.
 */
interface ArbWasmCache {
    /// @notice See if the user is a cache manager.
    function isCacheManager(
        address manager
    ) external view returns (bool);

    /// @notice Retrieve all address managers.
    /// @return managers the list of managers.
    function allCacheManagers() external view returns (address[] memory managers);

    /// @dev Deprecated, replaced with cacheProgram
    function cacheCodehash(
        bytes32 codehash
    ) external;

    /// @notice Caches all programs with a codehash equal to the given address.
    /// @notice Reverts if the programs have expired.
    /// @notice Caller must be a cache manager or chain owner.
    /// @notice If you're looking for how to bid for position, interact with the chain's cache manager contract.
    function cacheProgram(
        address addr
    ) external;

    /// @notice Evicts all programs with the given codehash.
    /// @notice Caller must be a cache manager or chain owner.
    function evictCodehash(
        bytes32 codehash
    ) external;

    /// @notice Gets whether a program is cached. Note that the program may be expired.
    function codehashIsCached(
        bytes32 codehash
    ) external view returns (bool);

    event UpdateProgramCache(address indexed manager, bytes32 indexed codehash, bool cached);
}
