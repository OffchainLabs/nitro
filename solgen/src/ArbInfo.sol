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

pragma solidity ^0.6.2;

contract ArbInfo {
    function getBalance(address account) external view returns (uint256) {
        return account.balance;
    }

    function getCode(address account) external view returns (bytes memory) {
        uint256 size;
        assembly {
            size := extcodesize(account)
        }
        bytes memory code = new bytes(size);
        assembly {
            extcodecopy(account, add(code, 0x20), 0, size)
        }
        return code;
    }
}

