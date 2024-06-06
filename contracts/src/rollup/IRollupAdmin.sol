// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupCore.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/IOutbox.sol";
import "../bridge/IOwnable.sol";
import "./Config.sol";

interface IRollupAdmin {
    /// @dev Outbox address was set
    event OutboxSet(address outbox);

    /// @dev Old outbox was removed
    event OldOutboxRemoved(address outbox);

    /// @dev Inbox was enabled or disabled
    event DelayedInboxSet(address inbox, bool enabled);

    /// @dev A list of validators was set
    event ValidatorsSet(address[] validators, bool[] enabled);

    /// @dev A new minimum assertion period was set
    event MinimumAssertionPeriodSet(uint256 newPeriod);

    /// @dev A new validator afk blocks was set
    event ValidatorAfkBlocksSet(uint256 newPeriod);

    /// @dev New confirm period blocks was set
    event ConfirmPeriodBlocksSet(uint64 newConfirmPeriod);

    /// @dev Base stake was set
    event BaseStakeSet(uint256 newBaseStake);

    /// @dev Stakers were force refunded
    event StakersForceRefunded(address[] staker);

    /// @dev An assertion was force created
    event AssertionForceCreated(bytes32 indexed assertionHash);

    /// @dev An assertion was force confirmed
    event AssertionForceConfirmed(bytes32 indexed assertionHash);

    /// @dev New loser stake escrow set
    event LoserStakeEscrowSet(address newLoserStakerEscrow);

    /// @dev New wasm module root was set
    event WasmModuleRootSet(bytes32 newWasmModuleRoot);

    /// @dev New sequencer inbox was set
    event SequencerInboxSet(address newSequencerInbox);

    /// @dev New inbox set
    event InboxSet(address inbox);

    /// @dev Validator whitelist was disabled or enabled
    event ValidatorWhitelistDisabledSet(bool _validatorWhitelistDisabled);

    /// @dev AnyTrust fast confirmer was set
    event AnyTrustFastConfirmerSet(address anyTrustFastConfirmer);

    /// @dev Challenge manager was set
    event ChallengeManagerSet(address challengeManager);


    function initialize(Config calldata config, ContractDependencies calldata connectedContracts) external;

    /**
     * @notice Add a contract authorized to put messages into this rollup's inbox
     * @param _outbox Outbox contract to add
     */
    function setOutbox(IOutbox _outbox) external;

    /**
     * @notice Disable an old outbox from interacting with the bridge
     * @param _outbox Outbox contract to remove
     */
    function removeOldOutbox(address _outbox) external;

    /**
     * @notice Enable or disable an inbox contract
     * @param _inbox Inbox contract to add or remove
     * @param _enabled New status of inbox
     */
    function setDelayedInbox(address _inbox, bool _enabled) external;

    /**
     * @notice Pause interaction with the rollup contract
     */
    function pause() external;

    /**
     * @notice Resume interaction with the rollup contract
     */
    function resume() external;

    /**
     * @notice Set the addresses of the validator whitelist
     * @dev It is expected that both arrays are same length, and validator at
     * position i corresponds to the value at position i
     * @param _validator addresses to set in the whitelist
     * @param _val value to set in the whitelist for corresponding address
     */
    function setValidator(address[] memory _validator, bool[] memory _val) external;

    /**
     * @notice Set a new owner address for the rollup proxy
     * @param newOwner address of new rollup owner
     */
    function setOwner(address newOwner) external;

    /**
     * @notice Set minimum assertion period for the rollup
     * @param newPeriod new minimum period for assertions
     */
    function setMinimumAssertionPeriod(uint256 newPeriod) external;

    /**
     * @notice Set validator afk blocks for the rollup
     * @param  newAfkBlocks new number of blocks before a validator is considered afk (0 to disable)
     * @dev    ValidatorAfkBlocks is the number of blocks since the last confirmed 
     *         assertion (or its first child) before the validator whitelist is removed.
     *         It's important that this time is greater than the max amount of time it can take to
     *         to confirm an assertion via the normal method. Therefore we need it to be greater
     *         than max(2* confirmPeriod, 2 * challengePeriod) with some additional margin.
     */
    function setValidatorAfkBlocks(uint64 newAfkBlocks) external;

    /**
     * @notice Set number of blocks until a assertion is considered confirmed
     * @param newConfirmPeriod new number of blocks until a assertion is confirmed
     */
    function setConfirmPeriodBlocks(uint64 newConfirmPeriod) external;

    /**
     * @notice Set base stake required for an assertion
     * @param newBaseStake maximum avmgas to be used per block
     */
    function setBaseStake(uint256 newBaseStake) external;

    function forceRefundStaker(address[] memory stacker) external;

    function forceCreateAssertion(
        bytes32 prevAssertionHash,
        AssertionInputs calldata assertion,
        bytes32 expectedAssertionHash
    ) external;

    function forceConfirmAssertion(
        bytes32 assertionHash,
        bytes32 parentAssertionHash,
        AssertionState calldata confirmState,
        bytes32 inboxAcc
    ) external;

    function setLoserStakeEscrow(address newLoserStakerEscrow) external;

    /**
     * @notice Set the proving WASM module root
     * @param newWasmModuleRoot new module root
     */
    function setWasmModuleRoot(bytes32 newWasmModuleRoot) external;

    /**
     * @notice set a new sequencer inbox contract
     * @param _sequencerInbox new address of sequencer inbox
     */
    function setSequencerInbox(address _sequencerInbox) external;

    /**
     * @notice set the validatorWhitelistDisabled flag
     * @param _validatorWhitelistDisabled new value of validatorWhitelistDisabled, i.e. true = disabled
     */
    function setValidatorWhitelistDisabled(bool _validatorWhitelistDisabled) external;

    /**
     * @notice set a new challengeManager contract
     * @param _challengeManager new value of challengeManager
     */
    function setChallengeManager(address _challengeManager) external;
}
