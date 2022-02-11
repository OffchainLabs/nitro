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

import "./AdminFallbackProxy.sol";
import { Config, ContractDependencies } from "../rollup/RollupLib.sol";

interface IArbitrumInit {
    function initialize(
        Config calldata config,
        ContractDependencies calldata connectedContracts
    ) external;
}

contract ArbitrumProxy is AdminFallbackProxy {
    using Address for address;

    // _rollupParams = [ confirmPeriodBlocks, extraChallengeTimeBlocks, chainId, baseStake ]
    // connectedContracts = [delayedBridge, sequencerInbox, outbox, rollupEventBridge, blockChallengeFactory]
    // sequencerInboxParams = [ maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds ]
    constructor(
        Config memory config,
        ContractDependencies memory connectedContracts
    ) AdminFallbackProxy(
        address(connectedContracts.rollupUserLogic),
        address(connectedContracts.rollupAdminLogic),
        config.owner
    ) {
        require(address(connectedContracts.rollupAdminLogic).isContract(), "ADMIN_LOGIC_NOT_CONTRACT");
        require(address(connectedContracts.rollupUserLogic).isContract(), "USER_LOGIC_NOT_CONTRACT");

        (bool successAdmin, ) = address(connectedContracts.rollupAdminLogic).delegatecall(
            abi.encodeWithSelector(IArbitrumInit.initialize.selector, config, connectedContracts)
        );
        require(successAdmin, "FAIL_INIT_ADMIN_LOGIC");

        (bool successUser, ) = address(connectedContracts.rollupUserLogic).delegatecall(
            abi.encodeWithSelector(IArbitrumInit.initialize.selector, config, connectedContracts)
        );
        require(successUser, "FAIL_INIT_USER_LOGIC");
    }
}
