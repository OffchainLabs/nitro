// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/ERC20Bridge.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/ERC20Inbox.sol";
import "../bridge/Outbox.sol";
import "./RollupEventInbox.sol";

import "../bridge/IBridge.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

contract ERC20BridgeCreator is Ownable {
    ERC20Bridge public bridgeTemplate;
    SequencerInbox public sequencerInboxTemplate;
    ERC20Inbox public inboxTemplate;
    RollupEventInbox public rollupEventInboxTemplate;
    Outbox public outboxTemplate;

    event TemplatesUpdated();

    constructor() Ownable() {
        bridgeTemplate = new ERC20Bridge();
        sequencerInboxTemplate = new SequencerInbox();
        inboxTemplate = new ERC20Inbox();
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
        bridgeTemplate = ERC20Bridge(_bridgeTemplate);
        sequencerInboxTemplate = SequencerInbox(_sequencerInboxTemplate);
        inboxTemplate = ERC20Inbox(_inboxTemplate);
        rollupEventInboxTemplate = RollupEventInbox(_rollupEventInboxTemplate);
        outboxTemplate = Outbox(_outboxTemplate);

        emit TemplatesUpdated();
    }

    struct CreateBridgeFrame {
        ProxyAdmin admin;
        ERC20Bridge bridge;
        SequencerInbox sequencerInbox;
        ERC20Inbox inbox;
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
            ERC20Bridge,
            SequencerInbox,
            ERC20Inbox,
            RollupEventInbox,
            Outbox
        )
    {
        CreateBridgeFrame memory frame;
        {
            frame.bridge = ERC20Bridge(
                address(new TransparentUpgradeableProxy(address(bridgeTemplate), adminProxy, ""))
            );
            frame.sequencerInbox = SequencerInbox(
                address(
                    new TransparentUpgradeableProxy(address(sequencerInboxTemplate), adminProxy, "")
                )
            );
            frame.inbox = ERC20Inbox(
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
