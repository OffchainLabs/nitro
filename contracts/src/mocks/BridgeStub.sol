// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./InboxStub.sol";

import "../bridge/IBridge.sol";

contract BridgeStub is IBridge {
    struct InOutInfo {
        uint256 index;
        bool allowed;
    }

    mapping(address => InOutInfo) private allowedInboxesMap;
    //mapping(address => InOutInfo) private allowedOutboxesMap;

    address[] public allowedInboxList;
    address[] public allowedOutboxList;

    address public override activeOutbox;

    // Accumulator for delayed inbox; tail represents hash of the current state; each element represents the inclusion of a new message.
    bytes32[] public override inboxAccs;

    function allowedInboxes(address inbox) external view override returns (bool) {
        return allowedInboxesMap[inbox].allowed;
    }

    function allowedOutboxes(address) external pure override returns (bool) {
        revert("NOT_IMPLEMENTED");
    }

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
                block.basefee,
                messageDataHash
            );
    }

    function addMessageToAccumulator(
        uint8,
        address,
        uint256,
        uint256,
        uint256,
        bytes32 messageDataHash
    ) internal returns (uint256) {
        uint256 count = inboxAccs.length;
        bytes32 messageHash = Messages.messageHash(
            0,
            address(uint160(0)),
            0,
            0,
            0,
            0,
            messageDataHash
        );
        bytes32 prevAcc = 0;
        if (count > 0) {
            prevAcc = inboxAccs[count - 1];
        }
        inboxAccs.push(Messages.accumulateInboxMessage(prevAcc, messageHash));
        return count;
    }

    function executeCall(
        address,
        uint256,
        bytes calldata
    ) external pure override returns (bool, bytes memory) {
        revert("NOT_IMPLEMENTED");
    }

    function setInbox(address inbox, bool enabled) external override {
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

    function setOutbox(
        address, /* outbox */
        bool /* enabled*/
    ) external pure override {
        revert("NOT_IMPLEMENTED");
    }

    function messageCount() external view override returns (uint256) {
        return inboxAccs.length;
    }
}
