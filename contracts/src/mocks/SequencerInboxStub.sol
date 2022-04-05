// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/SequencerInbox.sol";

contract SequencerInboxStub is SequencerInbox {
    constructor(
        IBridge delayedBridge_,
        address sequencer_,
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation_
    ) {
        delayedBridge = delayedBridge_;
        rollup = msg.sender;
        maxTimeVariation = maxTimeVariation_;
        isBatchPoster[sequencer_] = true;
    }

    function addInitMessage() external {
        (bytes32 dataHash, TimeBounds memory timeBounds) = formEmptyDataHash(0);
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            dataHash,
            0
        );
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
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
