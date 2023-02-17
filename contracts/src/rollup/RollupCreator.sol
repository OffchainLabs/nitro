// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./BridgeCreator.sol";

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

import "./RollupProxy.sol";

contract RollupCreator is Ownable {
    event RollupCreated(
        address indexed rollupAddress,
        address inboxAddress,
        address adminProxy,
        address sequencerInbox,
        address bridge
    );
    event TemplatesUpdated();

    BridgeCreator public bridgeCreator;
    IOneStepProofEntry public osp;
    IChallengeManager public challengeManagerTemplate;
    IRollupAdmin public rollupAdminLogic;
    IRollupUser public rollupUserLogic;

    address public validatorUtils;
    address public validatorWalletCreator;

    constructor() Ownable() {}

    function setTemplates(
        BridgeCreator _bridgeCreator,
        IOneStepProofEntry _osp,
        IChallengeManager _challengeManagerLogic,
        IRollupAdmin _rollupAdminLogic,
        IRollupUser _rollupUserLogic,
        address _validatorUtils,
        address _validatorWalletCreator
    ) external onlyOwner {
        bridgeCreator = _bridgeCreator;
        osp = _osp;
        challengeManagerTemplate = _challengeManagerLogic;
        rollupAdminLogic = _rollupAdminLogic;
        rollupUserLogic = _rollupUserLogic;
        validatorUtils = _validatorUtils;
        validatorWalletCreator = _validatorWalletCreator;
        emit TemplatesUpdated();
    }

    struct CreateRollupFrame {
        ProxyAdmin admin;
        IBridge bridge;
        ISequencerInbox sequencerInbox;
        IInbox inbox;
        IRollupEventInbox rollupEventInbox;
        IOutbox outbox;
        RollupProxy rollup;
    }

    // After this setup:
    // Rollup should be the owner of bridge
    // RollupOwner should be the owner of Rollup's ProxyAdmin
    // RollupOwner should be the owner of Rollup
    // Bridge should have a single inbox and outbox
    function createRollup(Config memory config, address expectedRollupAddr)
        external
        returns (address)
    {
        CreateRollupFrame memory frame;
        frame.admin = new ProxyAdmin();

        (
            frame.bridge,
            frame.sequencerInbox,
            frame.inbox,
            frame.rollupEventInbox,
            frame.outbox
        ) = bridgeCreator.createBridge(
            address(frame.admin),
            expectedRollupAddr,
            config.sequencerInboxMaxTimeVariation
        );

        frame.admin.transferOwnership(config.owner);

        IChallengeManager challengeManager = IChallengeManager(
            address(
                new TransparentUpgradeableProxy(
                    address(challengeManagerTemplate),
                    address(frame.admin),
                    ""
                )
            )
        );
        
        challengeManager.initialize({
            _assertionChain: IAssertionChain(expectedRollupAddr),
            _miniStakeValue: config.miniStakeValue,
            _challengePeriod: config.challengePeriodSeconds,
            _oneStepProofEntry: osp
        });
        // oldChallengeManager.initialize(
        //     IOldChallengeResultReceiver(expectedRollupAddr),
        //     frame.sequencerInbox,
        //     frame.bridge,
        //     osp
        // );

        frame.rollup = new RollupProxy(
            config,
            ContractDependencies({
                bridge: frame.bridge,
                sequencerInbox: frame.sequencerInbox,
                inbox: frame.inbox,
                outbox: frame.outbox,
                rollupEventInbox: frame.rollupEventInbox,
                challengeManager: challengeManager,
                rollupAdminLogic: rollupAdminLogic,
                rollupUserLogic: rollupUserLogic,
                validatorUtils: validatorUtils,
                validatorWalletCreator: validatorWalletCreator
            })
        );
        require(address(frame.rollup) == expectedRollupAddr, "WRONG_ROLLUP_ADDR");

        emit RollupCreated(
            address(frame.rollup),
            address(frame.inbox),
            address(frame.admin),
            address(frame.sequencerInbox),
            address(frame.bridge)
        );
        return address(frame.rollup);
    }
}
