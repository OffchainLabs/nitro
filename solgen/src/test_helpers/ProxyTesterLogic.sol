// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2021, Offchain Labs, Inc.
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

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/Proxy.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "../rollup/AdminAwareProxy.sol";
import "../rollup/RollupCore.sol";

contract ProxyTesterLogic is AAPStorage, RollupCore {
    function initialize(
        RollupLib.Config calldata config,
        ContractDependencies calldata /* connectedContracts */
    ) external pure override {
        require(config.owner != address(0), "OWNER_IS_ZERO");
    }

    function setOwner(address newOwner) external {
        owner = newOwner;
    }
}
