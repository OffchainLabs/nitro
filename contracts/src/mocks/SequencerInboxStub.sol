// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/SequencerInbox.sol";
import {INITIALIZATION_MSG_TYPE} from "../libraries/MessageTypes.sol";

contract SequencerInboxStub is SequencerInbox {
    constructor(
        IBridge bridge_,
        address sequencer_,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation_
    ) {
        bridge = bridge_;
        rollup = IOwnable(msg.sender);
        maxTimeVariation = maxTimeVariation_;
        isBatchPoster[sequencer_] = true;
    }

    function addInitMessage(uint256 chainId) external {
        bytes memory initMsg = abi.encodePacked(chainId);
        uint256 num = bridge.enqueueDelayedMessage(
            INITIALIZATION_MSG_TYPE,
            address(0),
            keccak256(initMsg)
        );
        require(num == 0, "ALREADY_DELAYED_INIT");
        emit InboxMessageDelivered(num, initMsg);
        (bytes32 dataHash, TimeBounds memory timeBounds) = formEmptyDataHash(1);
        (
            uint256 sequencerMessageCount,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(dataHash, 1, 0, 0, 1);
        require(sequencerMessageCount == 0, "ALREADY_SEQ_INIT");
        emit SequencerBatchDelivered(
            sequencerMessageCount,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.NoData
        );
    }

    function getTimeBounds() internal view override returns (TimeBounds memory bounds) {
        this; // silence warning about function not being view
        return bounds;
    }
}
