//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "./IBridge.sol";
import "./ISequencerInbox.sol";
import "./Messages.sol";
import "../libraries/IGasRefunder.sol";
import { MAX_DATA_SIZE } from "../libraries/Constants.sol";

/**
 * @title Accepts batches from the sequencer and adds them to the rollup inbox.
 * @notice Contains the inbox accumulator which is the ordering of all data and transactions to be processed by the rollup.
 * As part of submitting a batch the sequencer is also expected to include items enqueued
 * in the delayed inbox (Bridge.sol). If items in the delayed inbox are not included by a
 * sequencer within a time limit they can be force included into the rollup inbox by anyone.
 */
contract SequencerInbox is ISequencerInbox {
    bytes32[] public override inboxAccs;
    uint256 public totalDelayedMessagesRead;

    IBridge public delayedBridge;

    address public rollup;
    mapping(address => bool) public isBatchPoster;
    ISequencerInbox.MaxTimeVariation public maxTimeVariation;

    struct TimeBounds {
        uint64 minTimestamp;
        uint64 maxTimestamp;
        uint64 minBlockNumber;
        uint64 maxBlockNumber;
    }

    event SequencerBatchDelivered(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        TimeBounds timeBounds,
        bytes data
    );

    event SequencerBatchDeliveredFromOrigin(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        TimeBounds timeBounds
    );

    function initialize(
        IBridge _delayedBridge, 
        address rollup_, 
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation_
    ) external {
        require(delayedBridge == IBridge(address(0)), "ALREADY_INIT");
        require(_delayedBridge != IBridge(address(0)), "ZERO_BRIDGE");
        delayedBridge = _delayedBridge;
        rollup = rollup_;
        maxTimeVariation = maxTimeVariation_;
    }

    function getTimeBounds() internal view returns (TimeBounds memory) {
        TimeBounds memory bounds;
        if (block.timestamp > maxTimeVariation.delaySeconds) {
            bounds.minTimestamp = uint64(block.timestamp - maxTimeVariation.delaySeconds);
        }
        bounds.maxTimestamp = uint64(block.timestamp + maxTimeVariation.futureSeconds);
        if (block.number > maxTimeVariation.delayBlocks) {
            bounds.minBlockNumber = uint64(block.number - maxTimeVariation.delayBlocks);
        }
        bounds.maxBlockNumber = uint64(block.number + maxTimeVariation.futureBlocks);
        return bounds;
    }

    function forceInclusion(
        uint256 _totalDelayedMessagesRead,
        uint8 kind,
        uint256[2] calldata l1BlockAndTimestamp,
        uint256 inboxSeqNum,
        uint256 gasPriceL1,
        address sender,
        bytes32 messageDataHash,
        bytes calldata emptyData
    ) external {
        require(
            _totalDelayedMessagesRead > totalDelayedMessagesRead,
            "DELAYED_BACKWARDS"
        );
        {
            bytes32 messageHash = Messages.messageHash(
                kind,
                sender,
                l1BlockAndTimestamp[0],
                l1BlockAndTimestamp[1],
                inboxSeqNum,
                gasPriceL1,
                messageDataHash
            );
            // Can only force-include after the Sequencer-only window has expired.
            require(
                l1BlockAndTimestamp[0] + maxTimeVariation.delayBlocks <
                    block.number,
                "MAX_DELAY_BLOCKS"
            );
            require(
                l1BlockAndTimestamp[1] + maxTimeVariation.delaySeconds <
                    block.timestamp,
                "MAX_DELAY_TIME"
            );

            // Verify that message hash represents the last message sequence of delayed message to be included
            bytes32 prevDelayedAcc = 0;
            if (_totalDelayedMessagesRead > 1) {
                prevDelayedAcc = delayedBridge.inboxAccs(
                    _totalDelayedMessagesRead - 2
                );
            }
            require(
                delayedBridge.inboxAccs(_totalDelayedMessagesRead - 1) ==
                    Messages.accumulateInboxMessage(prevDelayedAcc, messageHash),
                "DELAYED_ACCUMULATOR"
            );
        }

        require(emptyData.length == 0, "NOT_EMPTY");
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc,
            TimeBounds memory timeBounds
        ) = addSequencerL2BatchImpl(emptyData, _totalDelayedMessagesRead);
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            emptyData
        );
    }

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external {
        uint256 startGasLeft = gasleft();
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "ORIGIN_ONLY");
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc,
            TimeBounds memory timeBounds
        ) = addSequencerL2BatchImpl(data, afterDelayedMessagesRead);
        emit SequencerBatchDeliveredFromOrigin(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds
        );

        if (address(gasRefunder) != address(0)) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            gasRefunder.onGasSpent(
                payable(msg.sender),
                startGasLeft - gasleft(),
                calldataSize
            );
        }
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external override {
        uint256 startGasLeft = gasleft();
        require(
            isBatchPoster[msg.sender] || msg.sender == rollup,
            "NOT_BATCH_POSTER"
        );

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc,
            TimeBounds memory timeBounds
        ) = addSequencerL2BatchImpl(data, afterDelayedMessagesRead);
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            timeBounds,
            data
        );

        if (address(gasRefunder) != address(0)) {
            gasRefunder.onGasSpent(
                payable(msg.sender),
                startGasLeft - gasleft(),
                0
            );
        }
    }

    function addSequencerL2BatchImpl(
        bytes calldata data,
        uint256 afterDelayedMessagesRead
    )
        internal
        returns (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc,
            TimeBounds memory timeBounds
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
        require(fullDataLen <= MAX_DATA_SIZE, "DATA_TOO_LARGE");
        bytes memory fullData = new bytes(fullDataLen);
        timeBounds = getTimeBounds();
        bytes memory header = abi.encodePacked(
            timeBounds.minTimestamp,
            timeBounds.maxTimestamp,
            timeBounds.minBlockNumber,
            timeBounds.maxBlockNumber,
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
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation_
    ) external override {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        maxTimeVariation = maxTimeVariation_;
    }

    function setIsBatchPoster(address addr, bool isBatchPoster_)
        external
        override
    {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        isBatchPoster[addr] = isBatchPoster_;
    }
}
