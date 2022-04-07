// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IBridge.sol";
import "./ISequencerInbox.sol";
import "./Messages.sol";

import {GasRefundEnabled, IGasRefunder} from "../libraries/IGasRefunder.sol";
import "../libraries/DelegateCallAware.sol";
import {MAX_DATA_SIZE} from "../libraries/Constants.sol";

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
    uint256 public constant HEADER_LENGTH = 40;
    /// @dev If the first batch data byte after the header has this bit set,
    /// the sequencer inbox has authenticated the data. Currently not used.
    bytes1 public constant DATA_AUTHENTICATED_FLAG = 0x40;

    address public rollup;
    mapping(address => bool) public isBatchPoster;
    ISequencerInbox.MaxTimeVariation public maxTimeVariation;

    function initialize(
        IBridge delayedBridge_,
        address rollup_,
        ISequencerInbox.MaxTimeVariation calldata maxTimeVariation_
    ) external onlyDelegated {
        if (delayedBridge != IBridge(address(0))) revert AlreadyInit();
        if (delayedBridge_ == IBridge(address(0))) revert HadZeroInit();
        delayedBridge = delayedBridge_;
        rollup = rollup_;
        maxTimeVariation = maxTimeVariation_;
    }

    function getTimeBounds() internal view virtual returns (TimeBounds memory) {
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
    /// @param l1BlockAndTime The l1 block and the l1 timestamp of the last message to be included
    /// @param baseFeeL1 The l1 gas price of the last message to be included
    /// @param sender The sender of the last message to be included
    /// @param messageDataHash The messageDataHash of the last message to be included
    function forceInclusion(
        uint256 _totalDelayedMessagesRead,
        uint8 kind,
        uint64[2] calldata l1BlockAndTime,
        uint256 baseFeeL1,
        address sender,
        bytes32 messageDataHash
    ) external {
        if (_totalDelayedMessagesRead <= totalDelayedMessagesRead) revert DelayedBackwards();
        bytes32 messageHash = Messages.messageHash(
            kind,
            sender,
            l1BlockAndTime[0],
            l1BlockAndTime[1],
            _totalDelayedMessagesRead - 1,
            baseFeeL1,
            messageDataHash
        );
        // Can only force-include after the Sequencer-only window has expired.
        if (l1BlockAndTime[0] + maxTimeVariation.delayBlocks >= block.number)
            revert ForceIncludeBlockTooSoon();
        if (l1BlockAndTime[1] + maxTimeVariation.delaySeconds >= block.timestamp)
            revert ForceIncludeTimeTooSoon();

        // Verify that message hash represents the last message sequence of delayed message to be included
        bytes32 prevDelayedAcc = 0;
        if (_totalDelayedMessagesRead > 1) {
            prevDelayedAcc = delayedBridge.inboxAccs(_totalDelayedMessagesRead - 2);
        }
        if (
            delayedBridge.inboxAccs(_totalDelayedMessagesRead - 1) !=
            Messages.accumulateInboxMessage(prevDelayedAcc, messageHash)
        ) revert IncorrectMessagePreimage();

        (bytes32 dataHash, TimeBounds memory timeBounds) = formEmptyDataHash(
            _totalDelayedMessagesRead
        );
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            dataHash,
            _totalDelayedMessagesRead
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

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external refundsGasWithCalldata(gasRefunder, payable(msg.sender)) {
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        if (inboxAccs.length != sequenceNumber) revert BadSequencerNumber();
        (bytes32 dataHash, TimeBounds memory timeBounds) = formDataHash(
            data,
            afterDelayedMessagesRead
        );
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            dataHash,
            afterDelayedMessagesRead
        );
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
        if (!isBatchPoster[msg.sender] && msg.sender != rollup) revert NotBatchPoster();
        if (inboxAccs.length != sequenceNumber) revert BadSequencerNumber();

        (bytes32 dataHash, TimeBounds memory timeBounds) = formDataHash(
            data,
            afterDelayedMessagesRead
        );
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            dataHash,
            afterDelayedMessagesRead
        );
        emit SequencerBatchDelivered(
            sequenceNumber,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.SeparateBatchEvent
        );
        emit SequencerBatchData(sequenceNumber, data);
    }

    function packHeader(uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes memory, TimeBounds memory)
    {
        TimeBounds memory timeBounds = getTimeBounds();
        bytes memory header = abi.encodePacked(
            timeBounds.minTimestamp,
            timeBounds.maxTimestamp,
            timeBounds.minBlockNumber,
            timeBounds.maxBlockNumber,
            uint64(afterDelayedMessagesRead)
        );
        // This must always be true from the packed encoding
        assert(header.length == HEADER_LENGTH);
        return (header, timeBounds);
    }

    function formDataHash(bytes calldata data, uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes32, TimeBounds memory)
    {
        uint256 fullDataLen = HEADER_LENGTH + data.length;
        if (fullDataLen < HEADER_LENGTH) revert DataLengthOverflow();
        if (fullDataLen > MAX_DATA_SIZE) revert DataTooLarge(fullDataLen, MAX_DATA_SIZE);
        bytes memory fullData = new bytes(fullDataLen);
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);

        for (uint256 i = 0; i < HEADER_LENGTH; i++) {
            fullData[i] = header[i];
        }
        if (data.length > 0 && (data[0] & DATA_AUTHENTICATED_FLAG) == DATA_AUTHENTICATED_FLAG) {
            revert DataNotAuthenticated();
        }
        // copy data into fullData at offset of HEADER_LENGTH (the extra 32 offset is because solidity puts the array len first)
        assembly {
            calldatacopy(add(fullData, add(HEADER_LENGTH, 32)), data.offset, data.length)
        }
        return (keccak256(fullData), timeBounds);
    }

    function formEmptyDataHash(uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes32, TimeBounds memory)
    {
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);
        return (keccak256(header), timeBounds);
    }

    function addSequencerL2BatchImpl(bytes32 dataHash, uint256 afterDelayedMessagesRead)
        internal
        returns (
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        )
    {
        if (afterDelayedMessagesRead < totalDelayedMessagesRead) revert DelayedBackwards();
        if (afterDelayedMessagesRead > delayedBridge.messageCount()) revert DelayedTooFar();

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

    function setMaxTimeVariation(ISequencerInbox.MaxTimeVariation memory maxTimeVariation_)
        external
        override
    {
        if (msg.sender != rollup) revert NotRollup(msg.sender, rollup);
        maxTimeVariation = maxTimeVariation_;
    }

    function setIsBatchPoster(address addr, bool isBatchPoster_) external override {
        if (msg.sender != rollup) revert NotRollup(msg.sender, rollup);
        isBatchPoster[addr] = isBatchPoster_;
    }
}
