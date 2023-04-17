// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./AbsRollupEventInbox.sol";
import "../bridge/IERC20Bridge.sol";

/**
 * @title The inbox for rollup protocol events
 */
contract ERC20RollupEventInbox is AbsRollupEventInbox {
    constructor() AbsRollupEventInbox() {}

    function _enqueueInitializationMsg(bytes memory initMsg) internal override returns (uint256) {
        uint256 tokenAmount = 0;
        return
            IERC20Bridge(address(bridge)).enqueueDelayedMessage(
                INITIALIZATION_MSG_TYPE,
                address(0),
                keccak256(initMsg),
                tokenAmount
            );
    }
}
