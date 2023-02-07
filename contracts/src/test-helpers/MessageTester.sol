// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/Messages.sol";

contract MessageTester {
    function messageHash(
        uint8 messageType,
        address sender,
        uint64 blockNumber,
        uint64 timestamp,
        uint256 inboxSeqNum,
        uint256 gasPriceL1,
        bytes32 messageDataHash
    ) public pure returns (bytes32) {
        return
            Messages.messageHash(
                messageType,
                sender,
                blockNumber,
                timestamp,
                inboxSeqNum,
                gasPriceL1,
                messageDataHash
            );
    }

    function accumulateInboxMessage(bytes32 inbox, bytes32 message) public pure returns (bytes32) {
        return Messages.accumulateInboxMessage(inbox, message);
    }
}
