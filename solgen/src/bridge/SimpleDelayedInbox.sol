// SPDX-License-Identifier: UNLICENSED
// Copyright 2021, Offchain Labs, Inc.

pragma solidity ^0.7.5;

import "./Messages.sol";

contract DelayedInbox {
    bytes32[] public delayedAccs;

    event MessageDelivered(
        uint256 indexed messageIndex,
        bytes32 indexed beforeInboxAcc,
        address inbox,
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    );

    function inboxAccs(uint256 index) external view returns (bytes32) {
        require (index<delayedAccs.length, "DELAYED_OUT_OF_BOUNDS");
        return delayedAccs[index];
    }

    function messageCount() external view returns (uint256) {
        return delayedAccs.length;
    }

    function addMessage(
        uint8 kind,
        address sender,
        bytes calldata data
    ) external {

        bytes32 messageDataHash = keccak256(data);
        bytes32 messageHash = Messages.messageHash(
            kind,
            sender,
            block.number,
            block.timestamp,
            delayedAccs.length, //TODO: inboxseqNum?
            tx.gasprice,
            messageDataHash
        );

        bytes32 prevDelayedAcc = 0;
        if (delayedAccs.length > 0) {
            prevDelayedAcc = delayedAccs[delayedAccs.length - 1];
        }
        delayedAccs.push(Messages.addMessageToInbox(prevDelayedAcc, messageHash));
        emit MessageDelivered(delayedAccs.length, prevDelayedAcc, msg.sender, kind, sender, messageDataHash);
    }

}