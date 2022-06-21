// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../libraries/IGasRefunder.sol";
import {AlreadyInit, HadZeroInit, NotOrigin, DataTooLarge, NotRollup} from "../libraries/Error.sol";
import "./IDelayedMessageProvider.sol";

interface ISequencerInbox is IDelayedMessageProvider {
    struct MaxTimeVariation {
        uint256 delayBlocks;
        uint256 futureBlocks;
        uint256 delaySeconds;
        uint256 futureSeconds;
    }

    struct TimeBounds {
        uint64 minTimestamp;
        uint64 maxTimestamp;
        uint64 minBlockNumber;
        uint64 maxBlockNumber;
    }

    enum BatchDataLocation {
        TxInput,
        SeparateBatchEvent,
        NoData
    }

    event SequencerBatchDelivered(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        TimeBounds timeBounds,
        BatchDataLocation dataLocation
    );

    event OwnerFunctionCalled(uint256 indexed id);

    /// @dev a separate event that emits batch data when this isn't easily accessible in the tx.input
    event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data);

    /// @dev a valid keyset was added
    event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes);

    /// @dev a keyset was invalidated
    event InvalidateKeyset(bytes32 indexed keysetHash);

    /// @dev Thrown when someone attempts to read fewer messages than have already been read
    error DelayedBackwards();

    /// @dev Thrown when someone attempts to read more messages than exist
    error DelayedTooFar();

    /// @dev Force include can only read messages more blocks old than the delay period
    error ForceIncludeBlockTooSoon();

    /// @dev Force include can only read messages more seconds old than the delay period
    error ForceIncludeTimeTooSoon();

    /// @dev The message provided did not match the hash in the delayed inbox
    error IncorrectMessagePreimage();

    /// @dev This can only be called by the batch poster
    error NotBatchPoster();

    /// @dev The sequence number provided to this message was inconsistent with the number of batches already included
    error BadSequencerNumber(uint256 stored, uint256 received);

    /// @dev The batch data has the inbox authenticated bit set, but the batch data was not authenticated by the inbox
    error DataNotAuthenticated();

    /// @dev Tried to create an already valid Data Availability Service keyset
    error AlreadyValidDASKeyset(bytes32);

    /// @dev Tried to use or invalidate an already invalid Data Availability Service keyset
    error NoSuchKeyset(bytes32);

    function inboxAccs(uint256 index) external view returns (bytes32);

    function batchCount() external view returns (uint256);

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external;

    // Methods only callable by rollup owner

    /**
     * @notice Set max time variation from actual time for sequencer inbox
     * @param timeVariation the maximum time variation parameters
     */
    function setMaxTimeVariation(MaxTimeVariation memory timeVariation) external;

    /**
     * @notice Updates whether an address is authorized to be a batch poster at the sequencer inbox
     * @param addr the address
     * @param isBatchPoster if the specified address should be authorized as a batch poster
     */
    function setIsBatchPoster(address addr, bool isBatchPoster) external;

    function setValidKeyset(bytes calldata keysetBytes) external;

    function invalidateKeysetHash(bytes32 ksHash) external;
}
