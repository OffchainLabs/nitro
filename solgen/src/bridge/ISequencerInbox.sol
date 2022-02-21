//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../libraries/IGasRefunder.sol";
import "../libraries/Error.sol";

interface ISequencerInbox {
    struct MaxTimeVariation {
        uint256 delayBlocks;
        uint256 futureBlocks;
        uint256 delaySeconds;
        uint256 futureSeconds;
    }

    /// @dev Thrown when someone attempts to read fewer messages than have already been read
    error DelayedBackwards();

    /// @dev Thrown when someone attempts to read more messages than exist
    error DelayedTooFar();

    /// @dev Thrown if the length of the header plus the length of the batch overflows
    error DataLengthOverflow();

    /// @dev Force include can only read messages more blocks old than the delay period
    error ForceIncludeBlockTooSoon();

    /// @dev Force include can only read messages more seconds old than the delay period
    error ForceIncludeTimeTooSoon();

    /// @dev The message provided did not match the hash in the delayed inbox
    error IncorrectMessagePreimage();

    /// @dev This can only be called by the batch poster
    error NotBatchPoster();

    /// @dev The sequence number provided to this message was inconsistent with the number of batches already included
    error BadSequencerNumber();

    function inboxAccs(uint256 index) external view returns (bytes32);

    function batchCount() external view returns (uint256);

    function setMaxTimeVariation(MaxTimeVariation memory timeVariation) external;

    function setIsBatchPoster(address addr, bool isBatchPoster_) external;

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external;
}
