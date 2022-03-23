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

/**
 * @title A test contract whose methods are only accessible in debug mode
 * @notice Precompiled contract that exists in every Arbitrum chain at 0x00000000000000000000000000000000000000ff.
 */
interface ArbDebug {
    /// @notice Caller becomes a chain owner
    function becomeChainOwner() external;

    /// @notice Emit events with values based on the args provided
    function events(bool flag, bytes32 value) external payable returns (address, uint256);

    // Events that exist for testing log creation and pricing
    event Basic(bool flag, bytes32 indexed value);
    event Mixed(
        bool indexed flag,
        bool not,
        bytes32 indexed value,
        address conn,
        address indexed caller
    );
    event Store(
        bool indexed flag,
        address indexed field,
        uint24 number,
        bytes32 value,
        bytes store
    );

    function customRevert(uint64 number) external pure;

    error Custom(uint64, string, bool);
}
