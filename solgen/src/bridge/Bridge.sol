//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "./Inbox.sol";
import "./Outbox.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts/utils/Address.sol";

import "./IBridge.sol";

/**
 * @title Staging ground for incoming and outgoing messages
 * @notice Holds the inbox accumulator for delayed messages, and is the ETH escrow
 * for value sent with these messages.
 * Since the escrow is held here, this contract also contains a list of allowed
 * outboxes that can make calls from here and withdraw this escrow.
 */
contract Bridge is OwnableUpgradeable, IBridge {
    using Address for address;
    struct InOutInfo {
        uint256 index;
        bool allowed;
    }

    mapping(address => InOutInfo) private allowedInboxesMap;
    mapping(address => InOutInfo) private allowedOutboxesMap;

    address[] public allowedInboxList;
    address[] public allowedOutboxList;

    address public override activeOutbox;

    /// @dev Accumulator for delayed inbox messages; tail represents hash of the current state; each element represents the inclusion of a new message.
    bytes32[] public override inboxAccs;

    function initialize() external initializer {
        __Ownable_init();
    }

    function allowedInboxes(address inbox) external view override returns (bool) {
        return allowedInboxesMap[inbox].allowed;
    }

    function allowedOutboxes(address outbox) external view override returns (bool) {
        return allowedOutboxesMap[outbox].allowed;
    }

    /**
     * @dev Enqueue a message in the delayed inbox accumulator.
     * These messages are later sequenced in the SequencerInbox, either by the sequencer as
     * part of a normal batch, or by force inclusion.
     */
    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    ) external payable override returns (uint256) {
        require(allowedInboxesMap[msg.sender].allowed, "NOT_FROM_INBOX");
        return
            addMessageToAccumulator(
                kind,
                sender,
                block.number,
                block.timestamp, // solhint-disable-line not-rely-on-time
                tx.gasprice,
                messageDataHash
            );
    }

    function addMessageToAccumulator(
        uint8 kind,
        address sender,
        uint256 blockNumber,
        uint256 blockTimestamp,
        uint256 gasPrice,
        bytes32 messageDataHash
    ) internal returns (uint256) {
        uint256 count = inboxAccs.length;
        bytes32 messageHash = Messages.messageHash(
            kind,
            sender,
            blockNumber,
            blockTimestamp,
            count,
            gasPrice,
            messageDataHash
        );
        bytes32 prevAcc = 0;
        if (count > 0) {
            prevAcc = inboxAccs[count - 1];
        }
        inboxAccs.push(Messages.accumulateInboxMessage(prevAcc, messageHash));
        emit MessageDelivered(count, prevAcc, msg.sender, kind, sender, messageDataHash, gasPrice, blockTimestamp);
        return count;
    }

    function executeCall(
        address destAddr,
        uint256 amount,
        bytes calldata data
    ) external override returns (bool success, bytes memory returnData) {
        require(allowedOutboxesMap[msg.sender].allowed, "NOT_FROM_OUTBOX");
        if (data.length > 0) require(destAddr.isContract(), "NO_CODE_AT_DEST");
        address currentOutbox = activeOutbox;
        activeOutbox = msg.sender;
        // We set and reset active outbox around external call so activeOutbox remains valid during call
        (success, returnData) = destAddr.call{ value: amount }(data);
        activeOutbox = currentOutbox;
        emit BridgeCallTriggered(msg.sender, destAddr, amount, data);
    }

    function setInbox(address inbox, bool enabled) external override onlyOwner {
        InOutInfo storage info = allowedInboxesMap[inbox];
        bool alreadyEnabled = info.allowed;
        emit InboxToggle(inbox, enabled);
        if ((alreadyEnabled && enabled) || (!alreadyEnabled && !enabled)) {
            return;
        }
        if (enabled) {
            allowedInboxesMap[inbox] = InOutInfo(allowedInboxList.length, true);
            allowedInboxList.push(inbox);
        } else {
            allowedInboxList[info.index] = allowedInboxList[allowedInboxList.length - 1];
            allowedInboxesMap[allowedInboxList[info.index]].index = info.index;
            allowedInboxList.pop();
            delete allowedInboxesMap[inbox];
        }
    }

    function setOutbox(address outbox, bool enabled) external override onlyOwner {
        InOutInfo storage info = allowedOutboxesMap[outbox];
        bool alreadyEnabled = info.allowed;
        emit OutboxToggle(outbox, enabled);
        if ((alreadyEnabled && enabled) || (!alreadyEnabled && !enabled)) {
            return;
        }
        if (enabled) {
            allowedOutboxesMap[outbox] = InOutInfo(allowedOutboxList.length, true);
            allowedOutboxList.push(outbox);
        } else {
            allowedOutboxList[info.index] = allowedOutboxList[allowedOutboxList.length - 1];
            allowedOutboxesMap[allowedOutboxList[info.index]].index = info.index;
            allowedOutboxList.pop();
            delete allowedOutboxesMap[outbox];
        }
    }

    function messageCount() external view override returns (uint256) {
        return inboxAccs.length;
    }
}
