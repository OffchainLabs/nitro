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

import "./RollupCore.sol";
import "./RollupEventBridge.sol";
import "./RollupLib.sol";
import "./Node.sol";

import "../libraries/Cloneable.sol";
import "../libraries/ProxyUtil.sol";

/// @dev this is assumed to always be the first inherited contract. the initial storage slots are read by dispatch contract
abstract contract AAPStorage is Cloneable {
    address public owner;
    AAPStorage public adminLogic;
    AAPStorage public userLogic;

    struct ContractDependencies {
        IBridge delayedBridge;
        ISequencerInbox sequencerInbox;
        IOutbox outbox;
        RollupEventBridge rollupEventBridge;
        IBlockChallengeFactory blockChallengeFactory;

        AAPStorage rollupAdminLogic;
        AAPStorage rollupUserLogic;
    }

    // _rollupParams = [ confirmPeriodBlocks, extraChallengeTimeBlocks, chainId, baseStake ]
    // connectedContracts = [delayedBridge, sequencerInbox, outbox, rollupEventBridge, blockChallengeFactory]
    // sequencerInboxParams = [ maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds ]
    function initialize(
        RollupLib.Config calldata config,
        ContractDependencies calldata connectedContracts
    ) external virtual;
}


/// @dev The Proxy dispatch also inherits from the Logic in order to keep the same storage layout
contract AdminAwareProxy is Proxy, AAPStorage {
    using Address for address;

    function initialize(
        RollupLib.Config calldata config,
        ContractDependencies calldata connectedContracts
    ) external override {
        require(address(adminLogic) == address(0) && address(userLogic) == address(0), "ALREADY_INIT");
        require(!isMasterCopy, "NO_INIT_MASTER");
        // we don't check `owner == 0 && config.owner != 0` here since the rollup could have no owner

        require(address(connectedContracts.rollupAdminLogic).isContract(), "ADMIN_LOGIC_NOT_CONTRACT");
        require(address(connectedContracts.rollupUserLogic).isContract(), "USER_LOGIC_NOT_CONTRACT");
        adminLogic = connectedContracts.rollupAdminLogic;
        userLogic = connectedContracts.rollupUserLogic;
        owner = config.owner;

        (bool successAdmin, ) = address(connectedContracts.rollupUserLogic).delegatecall(
            abi.encodeWithSelector(AAPStorage.initialize.selector, config, connectedContracts)
        );
        require(successAdmin, "FAIL_INIT_ADMIN_LOGIC");
        
        (bool successUser, ) = address(connectedContracts.rollupUserLogic).delegatecall(
            abi.encodeWithSelector(AAPStorage.initialize.selector, config, connectedContracts)
        );
        require(successUser, "FAIL_INIT_USER_LOGIC");
    }

    function postUpgradeInit() external {
        // it is assumed the rollup contract is behind a Proxy controlled by a proxy admin
        // this function can only be called by the proxy admin contract
        address proxyAdmin = ProxyUtil.getProxyAdmin();
        require(msg.sender == proxyAdmin, "NOT_FROM_ADMIN");
    }

    /**
     * @dev This is a virtual function that should be overriden so it returns the address to which the fallback function
     * and {_fallback} should delegate.
     */
    function _implementation()
        internal
        view
        override
        returns (address)
    {
        require(msg.data.length >= 4, "NO_FUNC_SIG");
        address rollupOwner = owner;
        // if there is an owner and it is the sender, delegate to admin logic
        address target = rollupOwner != address(0) && rollupOwner == msg.sender
            ? address(adminLogic)
            : address(userLogic);
        require(target.isContract(), "TARGET_NOT_CONTRACT");
        return target;
    }
}
