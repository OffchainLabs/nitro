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
abstract contract EthCallAware {
    /// @dev Tries to determine if the current execution is a transaction
    /// or a call. Allows execution to continue if the execution is a transaction
    /// and reverts with the provided data if the execution is a call
    modifier revertOnCall() {
        if(isCall()) revert CallAwareData(msg.data);
        _;
    }

    function isCall() internal view returns(bool) {
        // because of the base fee, the gas price should 
        // never be this low in a production environment
        // remix sets a gasprice of 1, whereas ethersjs uses 0
        return tx.gasprice <= 1;
    }
}


