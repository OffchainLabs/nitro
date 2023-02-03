// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./AbsRollupCreator.sol";
import "./BridgeCreator.sol";

contract RollupCreator is AbsRollupCreator, IEthRollupCreator {
    constructor() AbsRollupCreator() {}

    // After this setup:
    // Rollup should be the owner of bridge
    // RollupOwner should be the owner of Rollup's ProxyAdmin
    // RollupOwner should be the owner of Rollup
    // Bridge should have a single inbox and outbox
    function createRollup(Config memory config, address expectedRollupAddr)
        external
        override
        returns (address)
    {
        return _createRollup(config, expectedRollupAddr, address(0));
    }

    function _createBridge(
        address adminProxy,
        address rollup,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation,
        address
    )
        internal
        override
        returns (
            IBridge,
            SequencerInbox,
            IInbox,
            IRollupEventInbox,
            Outbox
        )
    {
        return
            BridgeCreator(address(bridgeCreator)).createBridge(
                adminProxy,
                rollup,
                maxTimeVariation
            );
    }
}
