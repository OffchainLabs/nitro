// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/**
 * @title A test contract whose methods are only accessible in debug mode
 * @notice Precompiled contract that exists in every Arbitrum chain at 0x00000000000000000000000000000000000000ff.
 */
interface ArbDebug {
    /// @notice Caller becomes a chain owner
    function becomeChainOwner() external;

    /// @notice Emit events with values based on the args provided
    function events(bool flag, bytes32 value) external payable returns (address, uint256);

    /// @notice Tries (and fails) to emit logs in a view context
    function eventsView() external view;

    // Events that exist for testing log creation and pricing
    event Basic(bool flag, bytes32 indexed value);
    event Mixed(
        bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller
    );
    event Store(
        bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store
    );

    function customRevert(
        uint64 number
    ) external pure;

    function panic() external;

    function legacyError() external pure;

    error Custom(uint64, string, bool);
    error Unused();
}
