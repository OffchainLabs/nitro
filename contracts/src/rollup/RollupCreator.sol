// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./BridgeCreator.sol";

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

import "./RollupProxy.sol";
import "./IRollupAdmin.sol";

contract RollupCreator is Ownable {
    event RollupCreated(
        address indexed rollupAddress, address inboxAddress, address adminProxy, address sequencerInbox, address bridge
    );
    event TemplatesUpdated();

    BridgeCreator public bridgeCreator;
    IOneStepProofEntry public osp;
    IEdgeChallengeManager public challengeManagerTemplate;
    IRollupAdmin public rollupAdminLogic;
    IRollupUser public rollupUserLogic;

    address public validatorWalletCreator;

    constructor() Ownable() {}

    function setTemplates(
        BridgeCreator _bridgeCreator,
        IOneStepProofEntry _osp,
        IEdgeChallengeManager _challengeManagerLogic,
        IRollupAdmin _rollupAdminLogic,
        IRollupUser _rollupUserLogic,
        address _validatorWalletCreator
    ) external onlyOwner {
        bridgeCreator = _bridgeCreator;
        osp = _osp;
        challengeManagerTemplate = _challengeManagerLogic;
        rollupAdminLogic = _rollupAdminLogic;
        rollupUserLogic = _rollupUserLogic;
        validatorWalletCreator = _validatorWalletCreator;
        emit TemplatesUpdated();
    }

    // internal function to workaround stack limit
    function createChallengeManager(address rollupAddr, address proxyAdminAddr, Config memory config)
        internal
        returns (IEdgeChallengeManager)
    {
        IEdgeChallengeManager challengeManager = IEdgeChallengeManager(
            address(
                new TransparentUpgradeableProxy(
                    address(challengeManagerTemplate),
                    proxyAdminAddr,
                    ""
                )
            )
        );

        challengeManager.initialize({
            _assertionChain: IAssertionChain(rollupAddr),
            _challengePeriodBlocks: config.confirmPeriodBlocks,
            _oneStepProofEntry: osp,
            layerZeroBlockEdgeHeight: config.layerZeroBlockEdgeHeight,
            layerZeroBigStepEdgeHeight: config.layerZeroBigStepEdgeHeight,
            layerZeroSmallStepEdgeHeight: config.layerZeroSmallStepEdgeHeight,
            _stakeToken: IERC20(config.stakeToken),
            _stakeAmount: config.miniStakeValue,
            _excessStakeReceiver: config.owner,
            _numBigStepLevel: config.numBigStepLevel
        });

        return challengeManager;
    }

    struct DeployedContracts {
        RollupProxy rollup;
        IInbox inbox;
        ISequencerInbox sequencerInbox;
        IBridge bridge;
        IRollupEventInbox rollupEventInbox;
        ProxyAdmin proxyAdmin;
        IEdgeChallengeManager challengeManager;
        IOutbox outbox;
    }

    /**
     * @notice Create a new rollup
     * @dev After this setup:
     * @dev - Rollup should be the owner of bridge
     * @dev - RollupOwner should be the owner of Rollup's ProxyAdmin
     * @dev - RollupOwner should be the owner of Rollup
     * @dev - Bridge should have a single inbox and outbox
     * @dev - Validators and batch poster should be set if provided
     * @param config       The configuration for the rollup
     * @param _batchPoster The address of the batch poster, not used when set to zero address
     * @param _validators  The list of validator addresses, not used when set to empty list
     * @return The address of the newly created rollup
     */
    function createRollup(
        Config memory config,
        address _batchPoster,
        address[] memory _validators,
        bool disableValidatorWhitelist,
        uint256 maxDataSize
    ) public returns (address) {
        // Make sure the immutable maxDataSize is as expected
        require(maxDataSize == bridgeCreator.sequencerInboxTemplate().maxDataSize(), "SI_MAX_DATA_SIZE_MISMATCH");
        require(maxDataSize == bridgeCreator.inboxTemplate().maxDataSize(), "I_MAX_DATA_SIZE_MISMATCH");
        DeployedContracts memory deployed;

        deployed.proxyAdmin = new ProxyAdmin();
        deployed.proxyAdmin.transferOwnership(config.owner);

        // Create the rollup proxy to figure out the address and initialize it later
        deployed.rollup =
        new RollupProxy{salt: keccak256(abi.encode(config, _batchPoster, _validators, disableValidatorWhitelist, maxDataSize))}();

        (deployed.bridge, deployed.sequencerInbox, deployed.inbox, deployed.rollupEventInbox, deployed.outbox) =
        bridgeCreator.createBridge(
            address(deployed.proxyAdmin), address(deployed.rollup), config.sequencerInboxMaxTimeVariation
        );

        deployed.challengeManager =
            createChallengeManager(address(deployed.rollup), address(deployed.proxyAdmin), config);

        // initialize the rollup with this contract as owner to set batch poster and validators
        // it will transfer the ownership back to the actual owner later
        address actualOwner = config.owner;
        config.owner = address(this);

        deployed.rollup.initializeProxy(
            config,
            ContractDependencies({
                bridge: deployed.bridge,
                sequencerInbox: deployed.sequencerInbox,
                inbox: deployed.inbox,
                outbox: deployed.outbox,
                rollupEventInbox: deployed.rollupEventInbox,
                challengeManager: deployed.challengeManager,
                rollupAdminLogic: address(rollupAdminLogic),
                rollupUserLogic: rollupUserLogic,
                validatorWalletCreator: validatorWalletCreator
            })
        );

        // setting batch poster, if the address provided is not zero address
        if (_batchPoster != address(0)) {
            deployed.sequencerInbox.setIsBatchPoster(_batchPoster, true);
        }
        // Call setValidator on the newly created rollup contract just if validator set is not empty
        if (_validators.length != 0) {
            bool[] memory _vals = new bool[](_validators.length);
            for (uint256 i = 0; i < _validators.length; i++) {
                _vals[i] = true;
            }
            IRollupAdmin(address(deployed.rollup)).setValidator(_validators, _vals);
        }
        if (disableValidatorWhitelist == true) {
            IRollupAdmin(address(deployed.rollup)).setValidatorWhitelistDisabled(disableValidatorWhitelist);
        }
        IRollupAdmin(address(deployed.rollup)).setOwner(actualOwner);

        emit RollupCreated(
            address(deployed.rollup),
            address(deployed.inbox),
            address(deployed.proxyAdmin),
            address(deployed.sequencerInbox),
            address(deployed.bridge)
        );
        return address(deployed.rollup);
    }
}
