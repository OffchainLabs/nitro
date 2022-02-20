//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "./IBridge.sol";
import "./ISequencerInbox.sol";
import "./Messages.sol";

import { GasRefundEnabled, IGasRefunder } from "../libraries/IGasRefunder.sol";
import "../libraries/DelegateCallAware.sol";
import { MAX_DATA_SIZE } from "../libraries/Constants.sol";

/**
 * @title Accepts batches from the sequencer and adds them to the rollup inbox.
 * @notice Contains the inbox accumulator which is the ordering of all data and transactions to be processed by the rollup.
 * As part of submitting a batch the sequencer is also expected to include items enqueued
 * in the delayed inbox (Bridge.sol). If items in the delayed inbox are not included by a
 * sequencer within a time limit they can be force included into the rollup inbox by anyone.
 */
contract SequencerInbox is DelegateCallAware, GasRefundEnabled, ISequencerInbox {
    bytes32[] public override inboxAccs;
    uint256 public totalDelayedMessagesRead;

    IBridge public delayedBridge;
    
    /// @dev The size of the batch header
    uint256 constant public HEADER_LENGTH = 40;

    address public rollup;
    mapping(address => bool) public isBatchPoster;
    ISequencerInbox.MaxTimeVariation public maxTimeVariation;

    struct TimeBounds {
        uint64 minTimestamp;
        uint64 maxTimestamp;
        uint64 minBlockNumber;
        uint64 maxBlockNumber;
    }

    enum BatchDataLocation {
        TxInput,
        SeparateBatchEvent,
        NoData
    }

    event SequencerBatchDelivered(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        TimeBounds timeBounds,
        BatchDataLocation dataLocation
    );

    /// @dev a separate event that emits batch data when this isn't easily accessible in the tx.input
    event SequencerBatchData(bytes data);

    function initialize(
        IBridge _delayedBridge, 
        address rollup_, 
        ISequencerInbox.MaxTimeVariation calldata maxTimeVariation_
    ) external onlyDelegated {
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

    /// @notice Force messages from the delayed inbox to be included in the chain
    /// Callable by any address, but message can only be force-included after maxTimeVariation.delayBlocks and maxTimeVariation.delaySeconds
    /// has elapsed. As part of normal behaviour the sequencer will include these messages
    /// so it's only necessary to call this if the sequencer is down, or not including
    /// any delayed messages.
    /// @param _totalDelayedMessagesRead The total number of messages to read up to
    /// @param kind The kind of the last message to be included
    /// @param l1BlockAndTimestamp The l1 block and the l1 timestamp of the last message to be included
    /// @param gasPriceL1 The l1 gas price of the last message to be included
    /// @param sender The sender of the last message to be included
    /// @param messageDataHash The messageDataHash of the last message to be included
    function forceInclusion(
        uint256 _totalDelayedMessagesRead,
        uint8 kind,
        uint256[2] calldata l1BlockAndTimestamp,
        uint256 gasPriceL1,
        address sender,
        bytes32 messageDataHash
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
                _totalDelayedMessagesRead - 1,
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

        (
            bytes32 dataHash,
            TimeBounds memory timeBounds
        ) = formEmptyDataHash(_totalDelayedMessagesRead);
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(dataHash, _totalDelayedMessagesRead);
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

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external refundsGasWithCalldata(gasRefunder, payable(msg.sender)) {
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "ORIGIN_ONLY");
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (
            bytes32 dataHash,
            TimeBounds memory timeBounds
        ) = formDataHash(data, afterDelayedMessagesRead);
        (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(dataHash, afterDelayedMessagesRead);
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.TxInput
        );
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external override refundsGasNoCalldata(gasRefunder, payable(msg.sender)) {
        require(
            isBatchPoster[msg.sender] || msg.sender == rollup,
            "NOT_BATCH_POSTER"
        );

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");

        TimeBounds memory timeBounds;
        bytes32 beforeAcc;
        bytes32 delayedAcc;
        bytes32 afterAcc;
        {
            bytes32 dataHash;
            (dataHash, timeBounds) = formDataHash(data, afterDelayedMessagesRead);
            (
                beforeAcc,
                delayedAcc,
                afterAcc
            ) = addSequencerL2BatchImpl(dataHash, afterDelayedMessagesRead);
        }
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.SeparateBatchEvent
        );
        emit SequencerBatchData(data);
    }

    function packHeader(uint256 afterDelayedMessagesRead) internal view returns (bytes memory, TimeBounds memory) {
        TimeBounds memory timeBounds = getTimeBounds();
        bytes memory header = abi.encodePacked(
            timeBounds.minTimestamp,
            timeBounds.maxTimestamp,
            timeBounds.minBlockNumber,
            timeBounds.maxBlockNumber,
            uint64(afterDelayedMessagesRead)
        );
        require(header.length == HEADER_LENGTH, "BAD_HEADER_LEN");
        return (header, timeBounds);
    }

    function formDataHash(bytes calldata data, uint256 afterDelayedMessagesRead) internal view returns (bytes32, TimeBounds memory) {
        uint256 fullDataLen = HEADER_LENGTH + data.length;
        require(fullDataLen >= HEADER_LENGTH, "DATA_LEN_OVERFLOW");
        require(fullDataLen <= MAX_DATA_SIZE, "DATA_TOO_LARGE");
        bytes memory fullData = new bytes(fullDataLen);
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);

        for (uint256 i = 0; i < HEADER_LENGTH; i++) {
            fullData[i] = header[i];
        }
        // copy data into fullData at offset of HEADER_LENGTH (the extra 32 offset is because solidity puts the array len first)
        assembly {
            calldatacopy(add(fullData, add(HEADER_LENGTH, 32)), data.offset, data.length)
        }
        return (keccak256(fullData), timeBounds);
    }


    function formEmptyDataHash(uint256 afterDelayedMessagesRead) internal view returns (bytes32, TimeBounds memory) {
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);
        return (keccak256(header), timeBounds);
    }

    function addSequencerL2BatchImpl(
        bytes32 dataHash,
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

        if (inboxAccs.length > 0) {
            beforeAcc = inboxAccs[inboxAccs.length - 1];
        }
        if (afterDelayedMessagesRead > 0) {
            delayedAcc = delayedBridge.inboxAccs(afterDelayedMessagesRead - 1);
        }

        acc = keccak256(abi.encodePacked(beforeAcc, dataHash, delayedAcc));
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
