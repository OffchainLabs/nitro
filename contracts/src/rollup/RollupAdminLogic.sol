// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupAdmin.sol";
import "./IRollupLogic.sol";
import "./RollupCore.sol";
import "../bridge/IOutbox.sol";
import "../bridge/ISequencerInbox.sol";
import "../libraries/DoubleLogicUUPSUpgradeable.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

import {NO_CHAL_INDEX} from "../libraries/Constants.sol";

contract RollupAdminLogic is RollupCore, IRollupAdmin, DoubleLogicUUPSUpgradeable {
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
        connectedContracts.bridge.setDelayedInbox(address(connectedContracts.rollupEventInbox), true);

        connectedContracts.rollupEventInbox.rollupInitialized(config.chainId);
        connectedContracts.sequencerInbox.addSequencerL2Batch(0, "", 1, IGasRefunder(address(0)), 0, 1);

        validatorUtils = connectedContracts.validatorUtils;
        validatorWalletCreator = connectedContracts.validatorWalletCreator;
        challengeManager = connectedContracts.challengeManager;

        confirmPeriodBlocks = config.confirmPeriodBlocks;
        chainId = config.chainId;
        baseStake = config.baseStake;
        wasmModuleRoot = config.wasmModuleRoot;
        // A little over 15 minutes
        minimumAssertionPeriod = 75;

        // the owner can't access the rollup user facet where escrow is redeemable
        require(config.loserStakeEscrow != _getAdmin(), "INVALID_ESCROW_ADMIN");
        // this next check shouldn't be an issue if the owner controls an AdminProxy
        // that accesses the admin facet, but still seems like a good extra precaution
        require(config.loserStakeEscrow != config.owner, "INVALID_ESCROW_OWNER");
        loserStakeEscrow = config.loserStakeEscrow;

        stakeToken = config.stakeToken;

        GlobalState memory emptyGlobalState;
        ExecutionState memory emptyExecutionState = ExecutionState(emptyGlobalState, MachineStatus.FINISHED);
        bytes32 parentAssertionHash = bytes32(0);
        bytes32 inboxAcc = bytes32(0);
        bytes32 genesisHash = RollupLib.assertionHash({
            parentAssertionHash: parentAssertionHash,
            afterState: emptyExecutionState,
            inboxAcc: inboxAcc
        });

        uint64 inboxMaxCount = 1; // force the first assertion to read a message
        AssertionNode memory initialAssertion = AssertionNodeLib.createAssertion(
            inboxMaxCount,
            0, // prev assertion
            uint64(block.number), // deadline block (not challengeable)
            true,
            RollupLib.configHash({
                wasmModuleRoot: wasmModuleRoot,
                requiredStake: baseStake,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: confirmPeriodBlocks
            })
        );
        initializeCore(initialAssertion, genesisHash);

        AssertionInputs memory assertionInputs;
        assertionInputs.afterState = emptyExecutionState;
        emit AssertionCreated(
            genesisHash,
            parentAssertionHash,
            assertionInputs,
            inboxAcc,
            inboxMaxCount,
            wasmModuleRoot,
            baseStake,
            address(challengeManager),
            confirmPeriodBlocks
        );

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
        emit OwnerFunctionCalled(0);
    }

    /**
     * @notice Disable an old outbox from interacting with the bridge
     * @param _outbox Outbox contract to remove
     */
    function removeOldOutbox(address _outbox) external override {
        require(_outbox != address(outbox), "CUR_OUTBOX");
        bridge.setOutbox(_outbox, false);
        emit OwnerFunctionCalled(1);
    }

    /**
     * @notice Enable or disable an inbox contract
     * @param _inbox Inbox contract to add or remove
     * @param _enabled New status of inbox
     */
    function setDelayedInbox(address _inbox, bool _enabled) external override {
        bridge.setDelayedInbox(address(_inbox), _enabled);
        emit OwnerFunctionCalled(2);
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
        emit OwnerFunctionCalled(3);
    }

    /**
     * @notice Resume interaction with the rollup contract
     */
    function resume() external override {
        _unpause();
        emit OwnerFunctionCalled(4);
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
            isValidator[_validator[i]] = _val[i];
        }
        emit OwnerFunctionCalled(6);
    }

    /**
     * @notice Set a new owner address for the rollup
     * @dev it is expected that only the rollup admin can use this facet to set a new owner
     * @param newOwner address of new rollup owner
     */
    function setOwner(address newOwner) external override {
        _changeAdmin(newOwner);
        emit OwnerFunctionCalled(7);
    }

    /**
     * @notice Set minimum assertion period for the rollup
     * @param newPeriod new minimum period for assertions
     */
    function setMinimumAssertionPeriod(uint256 newPeriod) external override {
        minimumAssertionPeriod = newPeriod;
        emit OwnerFunctionCalled(8);
    }

    /**
     * @notice Set number of blocks until a assertion is considered confirmed
     * @param newConfirmPeriod new number of blocks
     */
    function setConfirmPeriodBlocks(uint64 newConfirmPeriod) external override {
        require(newConfirmPeriod > 0, "INVALID_CONFIRM_PERIOD");
        confirmPeriodBlocks = newConfirmPeriod;
        emit OwnerFunctionCalled(9);
    }

    /**
     * @notice Set base stake required for an assertion
     * @param newBaseStake minimum amount of stake required
     */
    function setBaseStake(uint256 newBaseStake) external override {
        baseStake = newBaseStake;
        emit OwnerFunctionCalled(12);
    }

    /**
     * @notice Set the token used for stake, where address(0) == eth
     * @dev Before changing the base stake token, you might need to change the
     * implementation of the Rollup User facet!
     * @param newStakeToken address of token used for staking
     */
    function setStakeToken(address newStakeToken) external override whenPaused {
        /*
         * To change the stake token without breaking consistency one would need to:
         * Pause the system, have all stakers remove their funds,
         * update the user logic to handle ERC20s, change the stake token, then resume.
         *
         * Note: To avoid loss of funds stakers must remove their funds and claim all the
         * available withdrawable funds before the system is paused.
         */
        // TODO: HN: should we drop this function?
        bool expectERC20Support = newStakeToken != address(0);
        // this assumes the rollup isn't its own admin. if needed, instead use a ProxyAdmin by OZ!
        bool actualERC20Support = IRollupUser(address(this)).isERC20Enabled();
        require(actualERC20Support == expectERC20Support, "NO_USER_LOGIC_SUPPORT");
        require(stakerCount() == 0, "NO_ACTIVE_STAKERS");
        require(totalWithdrawableFunds == 0, "NO_PENDING_WITHDRAW");
        stakeToken = newStakeToken;
        emit OwnerFunctionCalled(13);
    }

    function forceRefundStaker(address[] calldata staker) external override whenPaused {
        require(staker.length > 0, "EMPTY_ARRAY");
        for (uint256 i = 0; i < staker.length; i++) {
            requireInactiveStaker(staker[i]);
            reduceStakeTo(staker[i], 0);
        }
        emit OwnerFunctionCalled(22);
    }

    function forceCreateAssertion(
        bytes32 prevAssertionId,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash
    ) external override whenPaused {
        // CHRIS: TODO: update this comment
        // We've removed this in favour of the following flow:
        // 1. Pause the rollup
        // 2. Update the wasm module root
        // 3. Create an alternative prev
        // 4. Refund the stake to the party that created the correct but ERRORED assertion
        // 5. Unpause the rollup
        // require(prevAssertionId == latestConfirmed(), "ONLY_LATEST_CONFIRMED");

        // Normally, a new assertion is created using its prev's confirmPeriodBlocks
        // in the case of a force create, we use the rollup's current confirmPeriodBlocks
        createNewAssertion(assertion, prevAssertionId, confirmPeriodBlocks, expectedAssertionHash);

        emit OwnerFunctionCalled(23);
    }

    function forceConfirmAssertion(
        bytes32 assertionId,
        bytes32 parentAssertionHash,
        ExecutionState calldata confirmState,
        bytes32 inboxAcc
    ) external override whenPaused {
        // this skips deadline, staker and zombie validation
        confirmAssertionInternal(assertionId, parentAssertionHash, confirmState, inboxAcc);
        emit OwnerFunctionCalled(24);
    }

    function setLoserStakeEscrow(address newLoserStakerEscrow) external override {
        // escrow holder can't be proxy admin, since escrow is only redeemable through
        // the primary user logic contract
        require(newLoserStakerEscrow != _getAdmin(), "INVALID_ESCROW");
        loserStakeEscrow = newLoserStakerEscrow;
        emit OwnerFunctionCalled(25);
    }

    /**
     * @notice Set the proving WASM module root
     * @param newWasmModuleRoot new module root
     */
    function setWasmModuleRoot(bytes32 newWasmModuleRoot) external override {
        wasmModuleRoot = newWasmModuleRoot;
        emit OwnerFunctionCalled(26);
    }

    /**
     * @notice set a new sequencer inbox contract
     * @param _sequencerInbox new address of sequencer inbox
     */
    function setSequencerInbox(address _sequencerInbox) external override {
        bridge.setSequencerInbox(_sequencerInbox);
        emit OwnerFunctionCalled(27);
    }

    /**
     * @notice sets the rollup's inbox reference. Does not update the bridge's view.
     * @param newInbox new address of inbox
     */
    function setInbox(IInbox newInbox) external {
        // TODO: HN: this is not synced with the bridge
        inbox = newInbox;
        emit OwnerFunctionCalled(28);
    }

    /**
     * @notice set the validatorWhitelistDisabled flag
     * @param _validatorWhitelistDisabled new value of validatorWhitelistDisabled, i.e. true = disabled
     */
    function setValidatorWhitelistDisabled(bool _validatorWhitelistDisabled) external {
        validatorWhitelistDisabled = _validatorWhitelistDisabled;
        emit OwnerFunctionCalled(30);
    }
}
