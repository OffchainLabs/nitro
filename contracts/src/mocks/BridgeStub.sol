// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./InboxStub.sol";
import {BadSequencerMessageNumber} from "../libraries/Error.sol";

import "../bridge/IBridge.sol";

contract BridgeStub is IBridge {
    struct InOutInfo {
        uint256 index;
        bool allowed;
    }

    mapping(address => InOutInfo) private allowedDelayedInboxesMap;
    //mapping(address => InOutInfo) private allowedOutboxesMap;

    IOwnable rollupItem;
    address[] public allowedDelayedInboxList;
    address[] public allowedOutboxList;

    address public override activeOutbox;

    // Accumulator for delayed inbox; tail represents hash of the current state; each element represents the inclusion of a new message.
    bytes32[] public override delayedInboxAccs;

    bytes32[] public override sequencerInboxAccs;

    address public sequencerInbox;
    uint256 public override sequencerReportedSubMessageCount;

    function setSequencerInbox(address _sequencerInbox) external override {
        sequencerInbox = _sequencerInbox;
        emit SequencerInboxUpdated(_sequencerInbox);
    }

    function allowedDelayedInboxes(address inbox) external view override returns (bool) {
        return allowedDelayedInboxesMap[inbox].allowed;
    }

    function allowedOutboxes(address) external pure override returns (bool) {
        return true;
    }

    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    ) external payable override returns (uint256) {
        require(allowedDelayedInboxesMap[msg.sender].allowed, "NOT_FROM_INBOX");
        return
            addMessageToDelayedAccumulator(
                kind,
                sender,
                block.number,
                block.timestamp, // solhint-disable-line not-rely-on-time
                block.basefee,
                messageDataHash
            );
    }

    function enqueueSequencerMessage(
        bytes32 dataHash,
        uint256 afterDelayedMessagesRead,
        uint256 prevMessageCount,
        uint256 newMessageCount
    )
        external
        returns (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        )
    {
        if (
            sequencerReportedSubMessageCount != prevMessageCount &&
            prevMessageCount != 0 &&
            sequencerReportedSubMessageCount != 0
        ) {
            revert BadSequencerMessageNumber(sequencerReportedSubMessageCount, prevMessageCount);
        }
        sequencerReportedSubMessageCount = newMessageCount;
        seqMessageIndex = sequencerInboxAccs.length;
        if (sequencerInboxAccs.length > 0) {
            beforeAcc = sequencerInboxAccs[sequencerInboxAccs.length - 1];
        }
        if (afterDelayedMessagesRead > 0) {
            delayedAcc = delayedInboxAccs[afterDelayedMessagesRead - 1];
        }
        acc = keccak256(abi.encodePacked(beforeAcc, dataHash, delayedAcc));
        sequencerInboxAccs.push(acc);
    }

    function submitBatchSpendingReport(address batchPoster, bytes32 dataHash)
        external
        returns (uint256)
    {
        // TODO: implement stub
    }

    function addMessageToDelayedAccumulator(
        uint8,
        address,
        uint256,
        uint256,
        uint256,
        bytes32 messageDataHash
    ) internal returns (uint256) {
        uint256 count = delayedInboxAccs.length;
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
            prevAcc = delayedInboxAccs[count - 1];
        }
        delayedInboxAccs.push(Messages.accumulateInboxMessage(prevAcc, messageHash));
        return count;
    }

    function executeCall(
        address,
        uint256,
        bytes calldata
    ) external pure override returns (bool, bytes memory) {
        revert("NOT_IMPLEMENTED_EXECUTE_CALL");
    }

    function setDelayedInbox(address inbox, bool enabled) external override {
        InOutInfo storage info = allowedDelayedInboxesMap[inbox];
        bool alreadyEnabled = info.allowed;
        emit InboxToggle(inbox, enabled);
        if (alreadyEnabled == enabled) {
            return;
        }
        if (enabled) {
            allowedDelayedInboxesMap[inbox] = InOutInfo(allowedDelayedInboxList.length, true);
            allowedDelayedInboxList.push(inbox);
        } else {
            allowedDelayedInboxList[info.index] = allowedDelayedInboxList[
                allowedDelayedInboxList.length - 1
            ];
            allowedDelayedInboxesMap[allowedDelayedInboxList[info.index]].index = info.index;
            allowedDelayedInboxList.pop();
            delete allowedDelayedInboxesMap[inbox];
        }
    }

    function setOutbox(
        address, /* outbox */
        bool /* enabled*/
    ) external pure override {
    }

    function delayedMessageCount() external view override returns (uint256) {
        return delayedInboxAccs.length;
    }

    function sequencerMessageCount() external view override returns (uint256) {
        return sequencerInboxAccs.length;
    }

    function rollup() external view override returns (IOwnable) {
        return rollupItem;
    }

    function acceptFundsFromOldBridge() external payable {}

    function initialize(IOwnable rollup_) external {
        rollupItem = rollup_;
    }
}
