// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/SequencerInbox.sol";

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

    function addInitMessage() external {
        (bytes32 dataHash, TimeBounds memory timeBounds) = formEmptyDataHash(0);
        (
            uint256 sequencerMessageCount,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(dataHash, 0, 0);
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
