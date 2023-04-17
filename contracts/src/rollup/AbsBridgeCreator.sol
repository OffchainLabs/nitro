// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/IBridge.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/IInbox.sol";
import "../bridge/Outbox.sol";
import "../rollup/IBridgeCreator.sol";
import "./IRollupEventInbox.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

abstract contract AbsBridgeCreator is Ownable, IBridgeCreator {
    IBridge public bridgeTemplate;
    SequencerInbox public sequencerInboxTemplate;
    IInbox public inboxTemplate;
    IRollupEventInbox public rollupEventInboxTemplate;
    Outbox public outboxTemplate;

    event TemplatesUpdated();

    constructor() Ownable() {
        sequencerInboxTemplate = new SequencerInbox();
        outboxTemplate = new Outbox();
    }

    function updateTemplates(
        address _bridgeTemplate,
        address _sequencerInboxTemplate,
        address _inboxTemplate,
        address _rollupEventInboxTemplate,
        address _outboxTemplate
    ) external onlyOwner {
        bridgeTemplate = IBridge(_bridgeTemplate);
        sequencerInboxTemplate = SequencerInbox(_sequencerInboxTemplate);
        inboxTemplate = IInbox(_inboxTemplate);
        rollupEventInboxTemplate = IRollupEventInbox(_rollupEventInboxTemplate);
        outboxTemplate = Outbox(_outboxTemplate);

        emit TemplatesUpdated();
    }

    struct CreateBridgeFrame {
        ProxyAdmin admin;
        IBridge bridge;
        SequencerInbox sequencerInbox;
        IInbox inbox;
        IRollupEventInbox rollupEventInbox;
        Outbox outbox;
    }

    function _createBridge(
        address adminProxy,
        address rollup,
        address nativeToken,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation
    )
        internal
        returns (
            IBridge,
            SequencerInbox,
            IInbox,
            IRollupEventInbox,
            Outbox
        )
    {
        CreateBridgeFrame memory frame;
        {
            frame.bridge = IBridge(
                address(new TransparentUpgradeableProxy(address(bridgeTemplate), adminProxy, ""))
            );
            frame.sequencerInbox = SequencerInbox(
                address(
                    new TransparentUpgradeableProxy(address(sequencerInboxTemplate), adminProxy, "")
                )
            );
            frame.inbox = IInbox(
                address(new TransparentUpgradeableProxy(address(inboxTemplate), adminProxy, ""))
            );
            frame.rollupEventInbox = IRollupEventInbox(
                address(
                    new TransparentUpgradeableProxy(
                        address(rollupEventInboxTemplate),
                        adminProxy,
                        ""
                    )
                )
            );
            frame.outbox = Outbox(
                address(new TransparentUpgradeableProxy(address(outboxTemplate), adminProxy, ""))
            );
        }

        _initializeBridge(frame.bridge, IOwnable(rollup), nativeToken);
        frame.sequencerInbox.initialize(IBridge(frame.bridge), maxTimeVariation);
        frame.inbox.initialize(IBridge(frame.bridge), ISequencerInbox(frame.sequencerInbox));
        frame.rollupEventInbox.initialize(IBridge(frame.bridge));
        frame.outbox.initialize(IBridge(frame.bridge));

        return (
            frame.bridge,
            frame.sequencerInbox,
            frame.inbox,
            frame.rollupEventInbox,
            frame.outbox
        );
    }

    function _initializeBridge(
        IBridge bridge,
        IOwnable rollup,
        address
    ) internal virtual;
}
