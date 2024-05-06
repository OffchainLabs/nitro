// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/Bridge.sol";
import "../bridge/SequencerInbox.sol";
import "../bridge/Inbox.sol";
import "../bridge/Outbox.sol";
import "./RollupEventInbox.sol";
import "../bridge/ERC20Bridge.sol";
import "../bridge/ERC20Inbox.sol";
import "../rollup/ERC20RollupEventInbox.sol";
import "../bridge/ERC20Outbox.sol";

import "../bridge/IBridge.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

contract BridgeCreator is Ownable {
    BridgeTemplates public ethBasedTemplates;
    BridgeTemplates public erc20BasedTemplates;

    event TemplatesUpdated();
    event ERC20TemplatesUpdated();

    struct BridgeTemplates {
        IBridge bridge;
        ISequencerInbox sequencerInbox;
        ISequencerInbox delayBufferableSequencerInbox;
        IInboxBase inbox;
        IRollupEventInbox rollupEventInbox;
        IOutbox outbox;
    }

    struct BridgeContracts {
        IBridge bridge;
        IInboxBase inbox;
        ISequencerInbox sequencerInbox;
        IRollupEventInbox rollupEventInbox;
        IOutbox outbox;
    }

    constructor(
        BridgeTemplates memory _ethBasedTemplates,
        BridgeTemplates memory _erc20BasedTemplates
    ) Ownable() {
        ethBasedTemplates = _ethBasedTemplates;
        erc20BasedTemplates = _erc20BasedTemplates;
    }

    function updateTemplates(BridgeTemplates calldata _newTemplates) external onlyOwner {
        ethBasedTemplates = _newTemplates;
        emit TemplatesUpdated();
    }

    function updateERC20Templates(BridgeTemplates calldata _newTemplates) external onlyOwner {
        erc20BasedTemplates = _newTemplates;
        emit ERC20TemplatesUpdated();
    }

    function _createBridge(
        address adminProxy,
        BridgeTemplates memory templates,
        bool isDelayBufferable
    ) internal returns (BridgeContracts memory) {
        BridgeContracts memory frame;
        frame.bridge = IBridge(
            address(new TransparentUpgradeableProxy(address(templates.bridge), adminProxy, ""))
        );
        frame.sequencerInbox = ISequencerInbox(
            address(
                new TransparentUpgradeableProxy(
                    address(
                        isDelayBufferable
                            ? templates.delayBufferableSequencerInbox
                            : templates.sequencerInbox
                    ),
                    adminProxy,
                    ""
                )
            )
        );
        frame.inbox = IInboxBase(
            address(new TransparentUpgradeableProxy(address(templates.inbox), adminProxy, ""))
        );
        frame.rollupEventInbox = IRollupEventInbox(
            address(
                new TransparentUpgradeableProxy(address(templates.rollupEventInbox), adminProxy, "")
            )
        );
        frame.outbox = IOutbox(
            address(new TransparentUpgradeableProxy(address(templates.outbox), adminProxy, ""))
        );
        return frame;
    }

    function createBridge(
        address adminProxy,
        address rollup,
        address nativeToken,
        ISequencerInbox.MaxTimeVariation calldata maxTimeVariation,
        BufferConfig calldata bufferConfig
    ) external returns (BridgeContracts memory) {
        // create delay bufferable sequencer inbox if threshold is non-zero
        bool isDelayBufferable = bufferConfig.threshold != 0;

        // create ETH-based bridge if address zero is provided for native token, otherwise create ERC20-based bridge
        BridgeContracts memory frame = _createBridge(
            adminProxy,
            nativeToken == address(0) ? ethBasedTemplates : erc20BasedTemplates,
            isDelayBufferable
        );

        // init contracts
        if (nativeToken == address(0)) {
            IEthBridge(address(frame.bridge)).initialize(IOwnable(rollup));
        } else {
            IERC20Bridge(address(frame.bridge)).initialize(IOwnable(rollup), nativeToken);
        }
        frame.sequencerInbox.initialize(IBridge(frame.bridge), maxTimeVariation, bufferConfig);
        frame.inbox.initialize(frame.bridge, frame.sequencerInbox);
        frame.rollupEventInbox.initialize(frame.bridge);
        frame.outbox.initialize(frame.bridge);

        return frame;
    }
}
