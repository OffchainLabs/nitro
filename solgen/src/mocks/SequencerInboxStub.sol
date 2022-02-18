//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../bridge/IBridge.sol";
import "../bridge/Messages.sol";
import "../bridge/ISequencerInbox.sol";
import "../libraries/IGasRefunder.sol";

contract SequencerInboxStub is ISequencerInbox {
    bytes32[] public override inboxAccs;
    uint256 public totalDelayedMessagesRead;

    IBridge public delayedBridge;

    mapping(address => bool) public isBatchPoster;

    constructor(IBridge _delayedBridge, address _sequencer) {
        delayedBridge = _delayedBridge;
        isBatchPoster[_sequencer] = true;

        bytes memory header = abi.encodePacked(
            uint64(0),
            uint64(0),
            uint64(0),
            uint64(0),
            uint64(0)
        );
        bytes32 headerHash = keccak256(header);
        bytes32 acc = keccak256(abi.encodePacked(bytes32(0), headerHash, bytes32(0)));
        inboxAccs.push(acc);
        bytes memory data;
        TimeBounds memory timeBounds;
        emit SequencerBatchDelivered(0, bytes32(0), acc, bytes32(0), 0, timeBounds, data);
    }

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder
    ) external {
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "ORIGIN_ONLY");
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        uint256 calldataSize;
        assembly {
            calldataSize := calldatasize()
        }

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(data, afterDelayedMessagesRead);

        TimeBounds memory emptyTimeBounds;
        emptyTimeBounds.maxTimestamp = ~uint64(0);
        emptyTimeBounds.maxBlockNumber = ~uint64(0);
        emit SequencerBatchDeliveredFromOrigin(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            emptyTimeBounds
        );
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder
    ) external override {
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(data, afterDelayedMessagesRead);
        TimeBounds memory emptyTimeBounds;
        emptyTimeBounds.maxTimestamp = ~uint64(0);
        emptyTimeBounds.maxBlockNumber = ~uint64(0);
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            emptyTimeBounds,
            data
        );
    }

    function addSequencerL2BatchImpl(
        bytes calldata data,
        uint256 afterDelayedMessagesRead
    )
        internal
        returns (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        )
    {
        require(
            afterDelayedMessagesRead >= totalDelayedMessagesRead,
            "DELAYED_BACKWARDS"
        );
        require(
            delayedBridge.messageCount() >= afterDelayedMessagesRead,
            "DELAYED_TOO_FAR"
        );

        uint256 fullDataLen = 40 + data.length;
        require(fullDataLen >= 40, "DATA_LEN_OVERFLOW");
        bytes memory fullData = new bytes(fullDataLen);

        bytes memory header = abi.encodePacked(
            uint64(0),
            ~uint64(0),
            uint64(0),
            ~uint64(0),
            uint64(afterDelayedMessagesRead)
        );
        require(header.length == 40, "BAD_HEADER_LEN");
        for (uint256 i = 0; i < 40; i++) {
            fullData[i] = header[i];
        }
        // copy data into fullData at offset 40 (the extra 32 offset is because solidity puts the array len first)
        assembly {
            calldatacopy(add(fullData, 72), data.offset, data.length)
        }

        if (inboxAccs.length > 0) {
            beforeAcc = inboxAccs[inboxAccs.length - 1];
        }
        if (afterDelayedMessagesRead > 0) {
            delayedAcc = delayedBridge.inboxAccs(afterDelayedMessagesRead - 1);
        }
        bytes32 fullDataHash = keccak256(fullData);
        acc = keccak256(abi.encodePacked(beforeAcc, fullDataHash, delayedAcc));
        inboxAccs.push(acc);
        totalDelayedMessagesRead = afterDelayedMessagesRead;
    }

    function batchCount() external view override returns (uint256) {
        return inboxAccs.length;
    }

    function setMaxTimeVariation(
        ISequencerInbox.MaxTimeVariation memory timeVariation
    ) external override {}

    function setIsBatchPoster(address addr, bool isBatchPoster_)
        external
        override
    {
        isBatchPoster[addr] = isBatchPoster_;
    }
}
