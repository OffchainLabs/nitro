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

import "../bridge/Bridge.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/Inbox.sol";
import "../bridge/Outbox.sol";
import "./RollupEventBridge.sol";
import "./BridgeCreator.sol";

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

import "./AdminAwareProxy.sol";
import "./RollupUserLogic.sol";
import "./RollupAdminLogic.sol";
import "../bridge/IBridge.sol";

import "./RollupLib.sol";
import "../libraries/ICloneable.sol";

contract RollupCreator is Ownable {
    event RollupCreated(address indexed rollupAddress, address inboxAddress, address adminProxy, address sequencerInbox, address delayedBridge);
    event TemplatesUpdated();

    BridgeCreator public bridgeCreator;
    ICloneable public rollupTemplate;
    IBlockChallengeFactory public challengeFactory;
    IRollupAdmin public rollupAdminLogic;
    IRollupUser public rollupUserLogic;

    constructor() Ownable() {}

    function setTemplates(
        BridgeCreator _bridgeCreator,
        ICloneable _rollupTemplate,
        IBlockChallengeFactory  _challengeFactory,
        IRollupAdmin _rollupAdminLogic,
        IRollupUser _rollupUserLogic
    ) external onlyOwner {
        bridgeCreator = _bridgeCreator;
        rollupTemplate = _rollupTemplate;
        challengeFactory = _challengeFactory;
        rollupAdminLogic = _rollupAdminLogic;
        rollupUserLogic = _rollupUserLogic;
        emit TemplatesUpdated();
    }

    struct CreateRollupFrame {
        ProxyAdmin admin;
        Bridge delayedBridge;
        SequencerInbox sequencerInbox;
        Inbox inbox;
        RollupEventBridge rollupEventBridge;
        Outbox outbox;
        AdminAwareProxy rollup;
    }

    // After this setup:
    // Rollup should be the owner of bridge
    // RollupOwner should be the owner of Rollup's ProxyAdmin
    // RollupOwner should be the owner of Rollup
    // Bridge should have a single inbox and outbox
    function createRollup(RollupLib.Config memory config) external returns (address) {
        CreateRollupFrame memory frame;
        frame.admin = new ProxyAdmin();
        frame.rollup = AdminAwareProxy(payable(address(
            new TransparentUpgradeableProxy(address(rollupTemplate), address(frame.admin), "")
        )));

        (
            frame.delayedBridge,
            frame.sequencerInbox,
            frame.inbox,
            frame.rollupEventBridge,
            frame.outbox
        ) = bridgeCreator.createBridge(address(frame.admin), address(frame.rollup));

        frame.admin.transferOwnership(config.owner);
        frame.rollup.initialize(
            config,
            RollupLib.ContractDependencies({
                delayedBridge: frame.delayedBridge,
                sequencerInbox: frame.sequencerInbox,
                outbox: frame.outbox,
                rollupEventBridge: frame.rollupEventBridge,
                blockChallengeFactory: challengeFactory,
                rollupAdminLogic: rollupAdminLogic,
                rollupUserLogic: rollupUserLogic
            })
        );

        emit RollupCreated(address(frame.rollup), address(frame.inbox), address(frame.admin), address(frame.sequencerInbox), address(frame.delayedBridge));
        return address(frame.rollup);
    }
}
