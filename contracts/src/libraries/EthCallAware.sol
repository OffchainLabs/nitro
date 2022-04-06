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

/// @dev Thrown when the execution context detected to be an eth_call.
/// @param data The msg.data of the current call
error CallAwareData(bytes data);

/// @dev Tools for inferring whether a transaction was made in the context of an eth_call
library EthCallAware {
    /// @dev Tries to determine if the current execution is a transaction
    /// or a call. Allows execution to continue if the execution is a transaction
    /// and reverts with the provided data if the execution is a call
    function revertOnCall(bytes memory data) internal view {
        if (isCall()) revert CallAwareData(data);
    }

    /// @dev Tries to determine if the current execution is a transaction or a call
    function isCall() internal view returns (bool) {
        // when making eth_calls many libraries leave empty, or allow arbitrary setting of, some
        // transaction fields such as 'from' and 'gasPrice'. Since it's impossible for a user to
        // sign a transaction from the 0x000.. address we know that if a transaction has that as its origin
        // then we must be in an eth_call. Likewise the base fee stops transactions being mined at 0 or 1 wei
        // gas prices, so those values are also indicators of an eth_call.
        // See https://twitter.com/0xkarmacoma/status/1493380279309717505 for more details.

        // remix sets a gasprice of 1, whereas ethersjs uses 0
        return tx.gasprice <= 1;
    }
}
