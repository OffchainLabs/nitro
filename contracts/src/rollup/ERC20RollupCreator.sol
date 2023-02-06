// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./AbsRollupCreator.sol";
import "./ERC20BridgeCreator.sol";

contract ERC20RollupCreator is AbsRollupCreator, IERC20RollupCreator {
    constructor() AbsRollupCreator() {}

    // After this setup:
    // Rollup should be the owner of bridge
    // RollupOwner should be the owner of Rollup's ProxyAdmin
    // RollupOwner should be the owner of Rollup
    // Bridge should have a single inbox and outbox
    function createRollup(
        Config memory config,
        address expectedRollupAddr,
        address nativeToken
    ) external override returns (address) {
        return _createRollup(config, expectedRollupAddr, nativeToken);
    }

    function _createBridge(
        address adminProxy,
        address rollup,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation,
        address nativeToken
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
            ERC20BridgeCreator(address(bridgeCreator)).createBridge(
                adminProxy,
                rollup,
                nativeToken,
                maxTimeVariation
            );
    }
}
