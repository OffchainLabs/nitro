// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupAdmin.sol";
import "./IRollupLogic.sol";
import "./RollupCore.sol";
import "../bridge/IOutbox.sol";
import "../bridge/ISequencerInbox.sol";
import "../libraries/DoubleLogicUUPSUpgradeable.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract RollupAdminLogic is RollupCore, IRollupAdmin, DoubleLogicUUPSUpgradeable {
    using AssertionStateLib for AssertionState;
    using EnumerableSetUpgradeable for EnumerableSetUpgradeable.AddressSet;

    function initialize(Config calldata config, ContractDependencies calldata connectedContracts)
        external
        override
        onlyProxy
        initializer
    {
        rollupDeploymentBlock = block.number;
        bridge = connectedContracts.bridge;
        connectedContracts.bridge.setDelayedInbox(address(connectedContracts.inbox), true);
        connectedContracts.bridge.setSequencerInbox(address(connectedContracts.sequencerInbox));

        inbox = connectedContracts.inbox;
        outbox = connectedContracts.outbox;
        connectedContracts.bridge.setOutbox(address(connectedContracts.outbox), true);
        rollupEventInbox = connectedContracts.rollupEventInbox;

        // dont need to connect and initialize the event inbox if it's already been initialized
        if (!bridge.allowedDelayedInboxes(address(connectedContracts.rollupEventInbox))) {
            connectedContracts.bridge.setDelayedInbox(address(connectedContracts.rollupEventInbox), true);
            connectedContracts.rollupEventInbox.rollupInitialized(config.chainId, config.chainConfig);
        }

        if (connectedContracts.sequencerInbox.totalDelayedMessagesRead() == 0) {
            connectedContracts.sequencerInbox.addSequencerL2Batch(0, "", 1, IGasRefunder(address(0)), 0, 1);
        }

        validatorWalletCreator = connectedContracts.validatorWalletCreator;
        challengeManager = connectedContracts.challengeManager;

        confirmPeriodBlocks = config.confirmPeriodBlocks;
        chainId = config.chainId;
        baseStake = config.baseStake;
        wasmModuleRoot = config.wasmModuleRoot;
        // A little over 15 minutes
        minimumAssertionPeriod = 75;
        // ValidatorAfkBlocks is defaulted to 28 days assuming a 12 seconds block time. 
        // Since it can take 14 days under normal circumstances to confirm an assertion, this means 
        // the validators will have been inactive for a further 14 days before the whitelist is removed.
        validatorAfkBlocks = 201600;
        challengeGracePeriodBlocks = config.challengeGracePeriodBlocks;

        // loser stake is now sent directly to loserStakeEscrow, it must not
        // be address(0) because some token do not allow transfers to address(0)
        require(config.loserStakeEscrow != address(0), "INVALID_ESCROW_0");
        loserStakeEscrow = config.loserStakeEscrow;

        stakeToken = config.stakeToken;
        anyTrustFastConfirmer = config.anyTrustFastConfirmer;

        bytes32 parentAssertionHash = bytes32(0);
        bytes32 inboxAcc = bytes32(0);
        bytes32 genesisHash = RollupLib.assertionHash({
            parentAssertionHash: parentAssertionHash,
            afterStateHash: config.genesisAssertionState.hash(),
            inboxAcc: inboxAcc
        });

        uint256 currentInboxCount = bridge.sequencerMessageCount();
        // ensure to move the inbox forward by at least one message
        if (currentInboxCount == config.genesisInboxCount) {
            currentInboxCount += 1;
        }
        AssertionNode memory initialAssertion = AssertionNodeLib.createAssertion(
            true,
            RollupLib.configHash({
                wasmModuleRoot: wasmModuleRoot,
                requiredStake: baseStake,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: confirmPeriodBlocks,
                nextInboxPosition: uint64(currentInboxCount)
            })
        );
        initializeCore(initialAssertion, genesisHash);

        AssertionInputs memory assertionInputs;
        assertionInputs.afterState = config.genesisAssertionState;
        emit AssertionCreated(
            genesisHash,
            parentAssertionHash,
            assertionInputs,
            inboxAcc,
            currentInboxCount,
            wasmModuleRoot,
            baseStake,
            address(challengeManager),
            confirmPeriodBlocks
        );
        if (_hostChainIsArbitrum) {
            _assertionCreatedAtArbSysBlock[genesisHash] = ArbSys(address(100)).arbBlockNumber();
        }

        emit RollupInitialized(config.wasmModuleRoot, config.chainId);
    }

    /**
     * Functions are only to reach this logic contract if the caller is the owner
     * so there is no need for a redundant onlyOwner check
     */

    /**
     * @notice Add a contract authorized to put messages into this rollup's inbox
     * @param _outbox Outbox contract to add
     */
    function setOutbox(IOutbox _outbox) external override {
        outbox = _outbox;
        bridge.setOutbox(address(_outbox), true);
        emit OutboxSet(address(_outbox));
        // previously: emit OwnerFunctionCalled(0);
    }

    /**
     * @notice Disable an old outbox from interacting with the bridge
     * @param _outbox Outbox contract to remove
     */
    function removeOldOutbox(address _outbox) external override {
        require(_outbox != address(outbox), "CUR_OUTBOX");
        bridge.setOutbox(_outbox, false);
        emit OldOutboxRemoved(address(_outbox));
        // previously: emit OwnerFunctionCalled(1);
    }

    /**
     * @notice Enable or disable an inbox contract
     * @param _inbox Inbox contract to add or remove
     * @param _enabled New status of inbox
     */
    function setDelayedInbox(address _inbox, bool _enabled) external override {
        bridge.setDelayedInbox(address(_inbox), _enabled);
        emit DelayedInboxSet(address(_inbox), _enabled);
        // previously: emit OwnerFunctionCalled(2);
    }

    /**
     * @notice Pause interaction with the rollup contract.
     * The time spent paused is not incremented in the rollup's timing for assertion validation.
     * @dev this function may be frontrun by a validator (ie to create a assertion before the system is paused).
     * The pause should be called atomically with required checks to be sure the system is paused in a consistent state.
     * The RollupAdmin may execute a check against the Rollup's latest assertion num or the OldChallengeManager, then execute this function atomically with it.
     */
    function pause() external override {
        _pause();
        // previously: emit OwnerFunctionCalled(3);
    }

    /**
     * @notice Resume interaction with the rollup contract
     */
    function resume() external override {
        _unpause();
        // previously: emit OwnerFunctionCalled(4);
    }

    /// @notice allows the admin to upgrade the primary logic contract (ie rollup admin logic, aka this)
    /// @dev this function doesn't revert as this primary logic contract is only
    /// reachable by the proxy's admin
    function _authorizeUpgrade(address newImplementation) internal override {}

    /// @notice allows the admin to upgrade the secondary logic contract (ie rollup user logic)
    /// @dev this function doesn't revert as this primary logic contract is only
    /// reachable by the proxy's admin
    function _authorizeSecondaryUpgrade(address newImplementation) internal override {}

    /**
     * @notice Set the addresses of the validator whitelist
     * @dev It is expected that both arrays are same length, and validator at
     * position i corresponds to the value at position i
     * @param _validator addresses to set in the whitelist
     * @param _val value to set in the whitelist for corresponding address
     */
    function setValidator(address[] calldata _validator, bool[] calldata _val) external override {
        require(_validator.length > 0, "EMPTY_ARRAY");
        require(_validator.length == _val.length, "WRONG_LENGTH");

        for (uint256 i = 0; i < _validator.length; i++) {
            if (_val[i]) validators.add(_validator[i]);
            else validators.remove(_validator[i]);
        }

        emit ValidatorsSet(_validator, _val);
        // previously: emit OwnerFunctionCalled(6);
    }

    /**
     * @notice Set a new owner address for the rollup
     * @dev it is expected that only the rollup admin can use this facet to set a new owner
     * @param newOwner address of new rollup owner
     */
    function setOwner(address newOwner) external override {
        _changeAdmin(newOwner);
        // previously: emit OwnerFunctionCalled(7);
    }

    /**
     * @notice Set minimum assertion period for the rollup
     * @param newPeriod new minimum period for assertions
     */
    function setMinimumAssertionPeriod(uint256 newPeriod) external override {
        minimumAssertionPeriod = newPeriod;
        emit MinimumAssertionPeriodSet(newPeriod);
        // previously: emit OwnerFunctionCalled(8);
    }

    /**
     * @notice Set validator afk blocks for the rollup
     * @param  newAfkBlocks new number of blocks before a validator is considered afk (0 to disable)
     * @dev    ValidatorAfkBlocks is the number of blocks since the last confirmed 
     *         assertion (or its first child) before the validator whitelist is removed.
     *         It's important that this time is greater than the max amount of time it can take to
     *         to confirm an assertion via the normal method. Therefore we need it to be greater
     *         than max(2* confirmPeriod, 2 * challengePeriod) with some additional margin.
     */
    function setValidatorAfkBlocks(uint64 newAfkBlocks) external override {
        validatorAfkBlocks = newAfkBlocks;
        emit ValidatorAfkBlocksSet(newAfkBlocks);
    }

    /**
     * @notice Set number of blocks until a assertion is considered confirmed
     * @param newConfirmPeriod new number of blocks
     */
    function setConfirmPeriodBlocks(uint64 newConfirmPeriod) external override {
        require(newConfirmPeriod > 0, "INVALID_CONFIRM_PERIOD");
        confirmPeriodBlocks = newConfirmPeriod;
        emit ConfirmPeriodBlocksSet(newConfirmPeriod);
        // previously: emit OwnerFunctionCalled(9);
    }

    /**
     * @notice Set base stake required for an assertion
     * @param newBaseStake minimum amount of stake required
     */
    function setBaseStake(uint256 newBaseStake) external override {
        // we do not currently allow base stake to be reduced since as doing so might allow a malicious party
        // to withdraw some (up to the difference between baseStake and newBaseStake) honest funds from this contract
        // The sequence of events is as follows:
        // 1. The malicious party creates a sibling assertion, stake size is currently S
        // 2. The base stake is then reduced to S'
        // 3. The malicious party uses a different address to create a child of the malicious assertion, using stake size S'
        // 4. This allows the malicious party to withdraw the stake S, since assertions with children set the staker to "inactive"
        require(newBaseStake > baseStake, "BASE_STAKE_MUST_BE_INCREASED");
        baseStake = newBaseStake;
        emit BaseStakeSet(newBaseStake);
        // previously: emit OwnerFunctionCalled(12);
    }

    function forceRefundStaker(address[] calldata staker) external override whenPaused {
        require(staker.length > 0, "EMPTY_ARRAY");
        for (uint256 i = 0; i < staker.length; i++) {
            requireInactiveStaker(staker[i]);
            reduceStakeTo(staker[i], 0);
        }
        emit StakersForceRefunded(staker);
        // previously: emit OwnerFunctionCalled(22);
    }

    function forceCreateAssertion(
        bytes32 prevAssertionHash,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash
    ) external override whenPaused {
        // To update the wasm module root in the case of a bug:
        // 0. pause the contract
        // 1. update the wasm module root in the contract
        // 2. update the config hash of the assertion after which you wish to use the new wasm module root (functionality not written yet)
        // 3. force refund the stake of the current leaf assertion(s)
        // 4. create a new assertion using the assertion with the updated config has as a prev
        // 5. force confirm it - this is necessary to set latestConfirmed on the correct line
        // 6. unpause the contract

        // Normally, a new assertion is created using its prev's confirmPeriodBlocks
        // in the case of a force create, we use the rollup's current confirmPeriodBlocks
        createNewAssertion(assertion, prevAssertionHash, expectedAssertionHash);

        emit AssertionForceCreated(expectedAssertionHash);
        // previously: emit OwnerFunctionCalled(23);
    }

    function forceConfirmAssertion(
        bytes32 assertionHash,
        bytes32 parentAssertionHash,
        AssertionState calldata confirmState,
        bytes32 inboxAcc
    ) external override whenPaused {
        // this skip deadline, prev, challenge validations
        confirmAssertionInternal(assertionHash, parentAssertionHash, confirmState, inboxAcc);
        emit AssertionForceConfirmed(assertionHash);
        // previously: emit OwnerFunctionCalled(24);
    }

    function setLoserStakeEscrow(address newLoserStakerEscrow) external override {
        // loser stake is now sent directly to loserStakeEscrow, it must not
        // be address(0) because some token do not allow transfers to address(0)
        require(newLoserStakerEscrow != address(0), "INVALID_ESCROW_0");
        loserStakeEscrow = newLoserStakerEscrow;
        emit LoserStakeEscrowSet(newLoserStakerEscrow);
        // previously: emit OwnerFunctionCalled(25);
    }

    /**
     * @notice Set the proving WASM module root
     * @param newWasmModuleRoot new module root
     */
    function setWasmModuleRoot(bytes32 newWasmModuleRoot) external override {
        wasmModuleRoot = newWasmModuleRoot;
        emit WasmModuleRootSet(newWasmModuleRoot);
        // previously: emit OwnerFunctionCalled(26);
    }

    /**
     * @notice set a new sequencer inbox contract
     * @param _sequencerInbox new address of sequencer inbox
     */
    function setSequencerInbox(address _sequencerInbox) external override {
        bridge.setSequencerInbox(_sequencerInbox);
        emit SequencerInboxSet(_sequencerInbox);
        // previously: emit OwnerFunctionCalled(27);
    }

    /**
     * @notice sets the rollup's inbox reference. Does not update the bridge's view.
     * @param newInbox new address of inbox
     */
    function setInbox(IInboxBase newInbox) external {
        inbox = newInbox;
        emit InboxSet(address(newInbox));
        // previously: emit OwnerFunctionCalled(28);
    }

    /**
     * @notice set the validatorWhitelistDisabled flag
     * @param _validatorWhitelistDisabled new value of validatorWhitelistDisabled, i.e. true = disabled
     */
    function setValidatorWhitelistDisabled(bool _validatorWhitelistDisabled) external {
        validatorWhitelistDisabled = _validatorWhitelistDisabled;
        emit ValidatorWhitelistDisabledSet(_validatorWhitelistDisabled);
        // previously: emit OwnerFunctionCalled(30);
    }

    /**
     * @notice set the anyTrustFastConfirmer address
     * @param _anyTrustFastConfirmer new value of anyTrustFastConfirmer
     */
    function setAnyTrustFastConfirmer(address _anyTrustFastConfirmer) external {
        anyTrustFastConfirmer = _anyTrustFastConfirmer;
        emit AnyTrustFastConfirmerSet(_anyTrustFastConfirmer);
        // previously: emit OwnerFunctionCalled(31);
    }

    /**
     * @notice set a new challengeManager contract
     * @param _challengeManager new value of challengeManager
     */
    function setChallengeManager(address _challengeManager) external {
        challengeManager = IEdgeChallengeManager(_challengeManager);
        emit ChallengeManagerSet(_challengeManager);
        // previously: emit OwnerFunctionCalled(32);
    }
}
