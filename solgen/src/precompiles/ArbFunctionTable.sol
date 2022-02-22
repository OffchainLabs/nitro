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

/// @title Deprecated - Provided aggregator's the ability to manage function tables, 
//  this enables one form of transaction compression. 
/// @notice The Nitro aggregator implementation does not use these, 
//  so these methods have been stubbed and their effects disabled. 
/// They are kept for backwards compatibility.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000068.
interface ArbFunctionTable {
    /// @notice Reverts since the table is empty
    function upload(bytes calldata buf) external;

    /// @notice Returns the empty table's size, which is 0
    function size(address addr) external view returns(uint);

    /// @notice No-op
    function get(address addr, uint index) external view returns(uint, bool, uint);
}
