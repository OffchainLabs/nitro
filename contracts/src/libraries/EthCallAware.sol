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

pragma solidity ^0.8.4;

/// @dev Thrown when the execution context detected to be an eth_call
/// @param version identifies how data should be encoded
/// @param data hex bytes of data that can be decoded according to the version
error CallAwareData(uint256 version, bytes data);

/// @dev Tools for inferring whether a transaction was made in the context of an eth_call
library EthCallAware {
    /// @dev this is equivalent to 0.000051729 gwei
    uint256 public constant MAGIC_GAS = uint256(0xcA11);
    /// @dev the address gets padded to `0x000000000000000000000000000000000000cA11`
    address public constant MAGIC_ORIGIN = address(uint160(MAGIC_GAS));

    /// @dev Tries to determine if the current execution is a transaction or a call
    function isCall() internal view returns (bool) {
        // when making eth_calls many libraries leave empty, or allow arbitrary setting of, some
        // transaction fields such as 'from' and 'gasPrice'. Since it's impossible for a user to
        // sign a valid transaction from MAGIC_ORIGIN we know that if a transaction has that as its origin
        // then we must be in an eth_call. Likewise the base fee stops transactions being mined at 0 or 1 wei
        // gas prices, so those values are also indicators of an eth_call.
        // instead of relying on gasPrice of 0 or 1 we use a magic value so this flow can be opt-in
        // (ie some tools set basefee of 0 and may hit this codepath accidentally)
        // See https://twitter.com/0xkarmacoma/status/1493380279309717505 for more details.

        // we compare tx.origin and gas price to the magic number so this codepath isn't hit accidentally
        return tx.gasprice == MAGIC_GAS || tx.origin == MAGIC_ORIGIN;
    }
}
