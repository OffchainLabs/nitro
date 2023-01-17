// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/NativeTokenBridge.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/NativeTokenInbox.sol";
import "../bridge/Outbox.sol";
import "./RollupEventInbox.sol";

import "../bridge/IBridge.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

contract ERC20BridgeCreator is Ownable {
    NativeTokenBridge public bridgeTemplate;
    SequencerInbox public sequencerInboxTemplate;
    NativeTokenInbox public inboxTemplate;
    RollupEventInbox public rollupEventInboxTemplate;
    Outbox public outboxTemplate;

    event TemplatesUpdated();

    constructor() Ownable() {
        bridgeTemplate = new NativeTokenBridge();
        sequencerInboxTemplate = new SequencerInbox();
        inboxTemplate = new NativeTokenInbox();
        rollupEventInboxTemplate = new RollupEventInbox();
        outboxTemplate = new Outbox();
    }

    function updateTemplates(
        address _bridgeTemplate,
        address _sequencerInboxTemplate,
        address _inboxTemplate,
        address _rollupEventInboxTemplate,
        address _outboxTemplate
    ) external onlyOwner {
        bridgeTemplate = NativeTokenBridge(_bridgeTemplate);
        sequencerInboxTemplate = SequencerInbox(_sequencerInboxTemplate);
        inboxTemplate = NativeTokenInbox(_inboxTemplate);
        rollupEventInboxTemplate = RollupEventInbox(_rollupEventInboxTemplate);
        outboxTemplate = Outbox(_outboxTemplate);

        emit TemplatesUpdated();
    }

    struct CreateBridgeFrame {
        ProxyAdmin admin;
        NativeTokenBridge bridge;
        SequencerInbox sequencerInbox;
        NativeTokenInbox inbox;
        RollupEventInbox rollupEventInbox;
        Outbox outbox;
    }

    function createBridge(
        address adminProxy,
        address rollup,
        address nativeToken,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation
    )
        external
        returns (
            NativeTokenBridge,
            SequencerInbox,
            NativeTokenInbox,
            RollupEventInbox,
            Outbox
        )
    {
        CreateBridgeFrame memory frame;
        {
            frame.bridge = NativeTokenBridge(
                address(new TransparentUpgradeableProxy(address(bridgeTemplate), adminProxy, ""))
            );
            frame.sequencerInbox = SequencerInbox(
                address(
                    new TransparentUpgradeableProxy(address(sequencerInboxTemplate), adminProxy, "")
                )
            );
            frame.inbox = NativeTokenInbox(
                address(new TransparentUpgradeableProxy(address(inboxTemplate), adminProxy, ""))
            );
            frame.rollupEventInbox = RollupEventInbox(
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

        frame.bridge.initialize(IOwnable(rollup), nativeToken);
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
}
