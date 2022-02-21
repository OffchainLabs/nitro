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
* @title Allows registering / retrieving addresses at uint indices, saving calldata.
* @notice Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000066.
*/
interface ArbAddressTable {
    /**
    * @notice Check whether an address exists in the address table
    * @param addr address to check for presence in table
    * @return true if address is in table
    */
    function addressExists(address addr) external view returns(bool);

    /**
    * @notice compress an address and return the result
    * @param addr address to compress
    * @return compressed address bytes
    */
    function compress(address addr) external returns(bytes memory);

    /**
    * @notice read a compressed address from a bytes buffer
    * @param buf bytes buffer containing an address
    * @param offset offset of target address
    * @return resulting address and updated offset into the buffer (revert if buffer is too short)
    */
    function decompress(bytes calldata buf, uint offset) external view returns(address, uint);

    /**
    * @param addr address to lookup
    * @return index of an address in the address table (revert if address isn't in the table)
    */
    function lookup(address addr) external view returns(uint);

    /**
    * @param index index to lookup address
    * @return address at a given index in address table (revert if index is beyond end of table)
    */
    function lookupIndex(uint index) external view returns(address);

    /**
    * @notice Register an address in the address table
    * @param addr address to register
    * @return index of the address (existing index, or newly created index if not already registered)
    */
    function register(address addr) external returns(uint);

    /**
    * @return size of address table (= first unused index)
     */
    function size() external view returns(uint);
}
