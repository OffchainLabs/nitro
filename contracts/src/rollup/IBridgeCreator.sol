// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./RollupLib.sol";
import "./IRollupCore.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/Outbox.sol";
import "../rollup/RollupEventInbox.sol";

interface IBridgeCreator {
    function updateTemplates(
        address _bridgeTemplate,
        address _sequencerInboxTemplate,
        address _inboxTemplate,
        address _rollupEventInboxTemplate,
        address _outboxTemplate
    ) external;

    function bridgeTemplate() external view returns (IBridge);

    function sequencerInboxTemplate() external view returns (SequencerInbox);

    function inboxTemplate() external view returns (IInbox);

    function rollupEventInboxTemplate() external view returns (IRollupEventInbox);

    function outboxTemplate() external view returns (Outbox);
}

interface IEthBridgeCreator is IBridgeCreator {
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
            IRollupEventInbox,
            Outbox
        );
}

interface IERC20BridgeCreator is IBridgeCreator {
    function createBridge(
        address adminProxy,
        address rollup,
        address nativeToken,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation
    )
        external
        returns (
            IBridge,
            SequencerInbox,
            IInbox,
            IRollupEventInbox,
            Outbox
        );
}
