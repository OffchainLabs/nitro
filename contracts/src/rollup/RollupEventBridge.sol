// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupEventBridge.sol";
import "../bridge/IBridge.sol";
import "../bridge/IMessageProvider.sol";
import "../libraries/DelegateCallAware.sol";
import {INITIALIZATION_MSG_TYPE} from "../libraries/MessageTypes.sol";

/**
 * @title The inbox for rollup protocol events
 */
contract RollupEventBridge is IRollupEventBridge, IMessageProvider, DelegateCallAware {
    uint8 internal constant CREATE_NODE_EVENT = 0;
    uint8 internal constant CONFIRM_NODE_EVENT = 1;
    uint8 internal constant REJECT_NODE_EVENT = 2;
    uint8 internal constant STAKE_CREATED_EVENT = 3;

    IBridge public bridge;
    address public rollup;

    modifier onlyRollup() {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        _;
    }

    function initialize(address _bridge, address _rollup) external onlyDelegated {
        require(rollup == address(0), "ALREADY_INIT");
        bridge = IBridge(_bridge);
        rollup = _rollup;
    }

    function rollupInitialized(uint256 chainId) external onlyRollup {
        bytes memory initMsg = abi.encodePacked(chainId);
        uint256 num = bridge.enqueueDelayedMessage(
            INITIALIZATION_MSG_TYPE,
            address(0),
            keccak256(initMsg)
        );
        emit InboxMessageDelivered(num, initMsg);
    }
}
