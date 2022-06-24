// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {NotContract, NotRollupOrOwner} from "../libraries/Error.sol";
import "./IOwnable.sol";

/// @dev Thrown when an un-authorized address tries to access an only-inbox function
/// @param sender The un-authorized sender
error NotDelayedInbox(address sender);

/// @dev Thrown when an un-authorized address tries to access an only-sequencer-inbox function
/// @param sender The un-authorized sender
error NotSequencerInbox(address sender);

/// @dev Thrown when an un-authorized address tries to access an only-outbox function
/// @param sender The un-authorized sender
error NotOutbox(address sender);

/// @dev the provided outbox address isn't valid
/// @param outbox address of outbox being set
error InvalidOutboxSet(address outbox);

interface IBridge {
    event MessageDelivered(
        uint256 indexed messageIndex,
        bytes32 indexed beforeInboxAcc,
        address inbox,
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 baseFeeL1,
        uint64 timestamp
    );

    event BridgeCallTriggered(
        address indexed outbox,
        address indexed to,
        uint256 value,
        bytes data
    );

    event InboxToggle(address indexed inbox, bool enabled);

    event OutboxToggle(address indexed outbox, bool enabled);

    event SequencerInboxUpdated(address newSequencerInbox);

    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    ) external payable returns (uint256);

    function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead)
        external
        returns (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        );

    function submitBatchSpendingReport(address batchPoster, bytes32 dataHash)
        external
        returns (uint256 msgNum);

    function executeCall(
        address to,
        uint256 value,
        bytes calldata data
    ) external returns (bool success, bytes memory returnData);

    // These are only callable by the admin
    function setDelayedInbox(address inbox, bool enabled) external;

    function setOutbox(address inbox, bool enabled) external;

    function setSequencerInbox(address _sequencerInbox) external;

    // View functions

    function sequencerInbox() external view returns (address);

    function activeOutbox() external view returns (address);

    function allowedDelayedInboxes(address inbox) external view returns (bool);

    function allowedOutboxes(address outbox) external view returns (bool);

    function delayedInboxAccs(uint256 index) external view returns (bytes32);

    function sequencerInboxAccs(uint256 index) external view returns (bytes32);

    function delayedMessageCount() external view returns (uint256);

    function sequencerMessageCount() external view returns (uint256);

    function rollup() external view returns (IOwnable);
}
