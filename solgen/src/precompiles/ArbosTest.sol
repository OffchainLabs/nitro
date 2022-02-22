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

/// @title Deprecated - Provides a method of burning arbitrary amounts of gas,
/// @notice This exists for historical reasons. Pre-Nitro, `ArbosTest` had additional methods only the zero address could call. 
/// These have been removed since users don't use them and calls to missing methods revert.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000069.
interface ArbosTest {
    /// @notice Unproductively burns the amount of L2 ArbGas
    function burnArbGas(uint gasAmount) external pure;
}
