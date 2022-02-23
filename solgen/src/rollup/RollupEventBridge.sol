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

import "./IRollupLogic.sol";

import "../bridge/IBridge.sol";
import "../bridge/IMessageProvider.sol";
import "../libraries/DelegateCallAware.sol";
import {ROLLUP_PROTOCOL_EVENT_TYPE, INITIALIZATION_MSG_TYPE} from "../libraries/MessageTypes.sol";

/**
 * @title The inbox for rollup protocol events
 */
contract RollupEventBridge is IMessageProvider, DelegateCallAware {
    uint8 internal constant CREATE_NODE_EVENT = 0;
    uint8 internal constant CONFIRM_NODE_EVENT = 1;
    uint8 internal constant REJECT_NODE_EVENT = 2;
    uint8 internal constant STAKE_CREATED_EVENT = 3;

    IBridge bridge;
    address rollup;

    modifier onlyRollup() {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        _;
    }

    function initialize(address _bridge, address _rollup) external onlyDelegated {
        require(rollup == address(0), "ALREADY_INIT");
        bridge = IBridge(_bridge);
        rollup = _rollup;
    }

    function rollupInitialized(address owner, uint256 chainId) external onlyRollup {
        bytes memory initMsg = abi.encodePacked(owner, chainId);
        uint256 num = bridge.enqueueDelayedMessage(
            INITIALIZATION_MSG_TYPE,
            address(0),
            keccak256(initMsg)
        );
        emit InboxMessageDelivered(num, initMsg);
    }

    function nodeCreated(
        uint256 nodeNum,
        uint256 prev,
        uint256 deadline,
        address asserter
    ) external onlyRollup {
        deliverToBridge(
            abi.encodePacked(
                CREATE_NODE_EVENT,
                nodeNum,
                prev,
                block.number,
                deadline,
                uint256(uint160(bytes20(asserter)))
            )
        );
    }

    function nodeConfirmed(uint256 nodeNum) external onlyRollup {
        deliverToBridge(abi.encodePacked(CONFIRM_NODE_EVENT, nodeNum));
    }

    function nodeRejected(uint256 nodeNum) external onlyRollup {
        deliverToBridge(abi.encodePacked(REJECT_NODE_EVENT, nodeNum));
    }

    function stakeCreated(address staker, uint256 nodeNum) external onlyRollup {
        deliverToBridge(
            abi.encodePacked(
                STAKE_CREATED_EVENT,
                uint256(uint160(bytes20(staker))),
                nodeNum,
                block.number
            )
        );
    }

    function deliverToBridge(bytes memory message) private {
        emit InboxMessageDelivered(
            bridge.enqueueDelayedMessage(
                ROLLUP_PROTOCOL_EVENT_TYPE,
                msg.sender,
                keccak256(message)
            ),
            message
        );
    }
}
