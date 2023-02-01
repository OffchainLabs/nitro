// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../rollup/AbsBridgeCreator.sol";
import "../bridge/Bridge.sol";
import "../bridge/IEthBridge.sol";
import "../bridge/Inbox.sol";

contract BridgeCreator is AbsBridgeCreator {
    constructor() AbsBridgeCreator() {
        bridgeTemplate = new Bridge();
        inboxTemplate = new Inbox();
    }

    function createBridge(
        address adminProxy,
        address rollup,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation
    )
        external
        returns (
            IBridge,
            SequencerInbox,
            IInbox,
            RollupEventInbox,
            Outbox
        )
    {
        return _createBridge(adminProxy, rollup, address(0), maxTimeVariation);
    }

    function _initializeBridge(
        IBridge bridge,
        IOwnable rollup,
        address
    ) internal override {
        IEthBridge(address(bridge)).initialize(IOwnable(rollup));
    }
}
