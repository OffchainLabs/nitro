// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {
    AlreadyInit,
    HadZeroInit,
    NotOrigin,
    DataTooLarge,
    NotRollup,
    DelayedBackwards,
    DelayedTooFar,
    ForceIncludeBlockTooSoon,
    ForceIncludeTimeTooSoon,
    IncorrectMessagePreimage,
    NotBatchPoster,
    BadSequencerNumber,
    DataNotAuthenticated,
    AlreadyValidDASKeyset,
    NoSuchKeyset,
    NotForked,
    Paused
} from "../libraries/Error.sol";
import "./IBridge.sol";
import "./IInbox.sol";
import "./ISequencerInbox.sol";
import "../rollup/IRollupLogic.sol";
import "./Messages.sol";

import {L1MessageType_batchPostingReport} from "../libraries/MessageTypes.sol";
import {GasRefundEnabled, IGasRefunder} from "../libraries/IGasRefunder.sol";
import "../libraries/DelegateCallAware.sol";
import {MAX_DATA_SIZE} from "../libraries/Constants.sol";

interface IDataHashesReader {
    function getDataHashes() external view returns (bytes memory);
}

/**
 * @title Accepts batches from the sequencer and adds them to the rollup inbox.
 * @notice Contains the inbox accumulator which is the ordering of all data and transactions to be processed by the rollup.
 * As part of submitting a batch the sequencer is also expected to include items enqueued
 * in the delayed inbox (Bridge.sol). If items in the delayed inbox are not included by a
 * sequencer within a time limit they can be force included into the rollup inbox by anyone.
 */
contract SequencerInbox is DelegateCallAware, GasRefundEnabled, ISequencerInbox {
    uint256 public totalDelayedMessagesRead;

    IBridge public bridge;

    /// @inheritdoc ISequencerInbox
    uint256 public constant HEADER_LENGTH = 40;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant DATA_AUTHENTICATED_FLAG = 0x40;

    IOwnable public rollup;
    mapping(address => bool) public isBatchPoster;
    ISequencerInbox.MaxTimeVariation public maxTimeVariation;

    mapping(bytes32 => DasKeySetInfo) public dasKeySetInfo;

    modifier onlyRollupOwner() {
        if (msg.sender != rollup.owner()) revert NotOwner(msg.sender, address(rollup));
        _;
    }

    uint256 internal immutable deployTimeChainId = block.chainid;

    function _chainIdChanged() internal view returns (bool) {
        return deployTimeChainId != block.chainid;
    }

    function initialize(
        IBridge bridge_,
        ISequencerInbox.MaxTimeVariation calldata maxTimeVariation_
    ) external onlyDelegated {
        if (bridge != IBridge(address(0))) revert AlreadyInit();
        if (bridge_ == IBridge(address(0))) revert HadZeroInit();
        bridge = bridge_;
        rollup = bridge_.rollup();
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

    /// @inheritdoc ISequencerInbox
    function removeDelayAfterFork() external {
        if (!_chainIdChanged()) revert NotForked();
        maxTimeVariation = ISequencerInbox.MaxTimeVariation({
            delayBlocks: 1,
            futureBlocks: 1,
            delaySeconds: 1,
            futureSeconds: 1
        });
    }

    /// @inheritdoc ISequencerInbox
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
            prevDelayedAcc = bridge.delayedInboxAccs(_totalDelayedMessagesRead - 2);
        }
        if (
            bridge.delayedInboxAccs(_totalDelayedMessagesRead - 1) !=
            Messages.accumulateInboxMessage(prevDelayedAcc, messageHash)
        ) revert IncorrectMessagePreimage();

        (bytes32 dataHash, TimeBounds memory timeBounds) = formEmptyDataHash(
            _totalDelayedMessagesRead
        );
        uint256 __totalDelayedMessagesRead = _totalDelayedMessagesRead;
        uint256 prevSeqMsgCount = bridge.sequencerReportedSubMessageCount();
        uint256 newSeqMsgCount = prevSeqMsgCount +
            _totalDelayedMessagesRead -
            totalDelayedMessagesRead;
        (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(
                dataHash,
                __totalDelayedMessagesRead,
                0,
                prevSeqMsgCount,
                newSeqMsgCount
            );
        emit SequencerBatchDelivered(
            seqMessageIndex,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.NoData
        );
    }

    /// @dev Deprecated in favor of the variant specifying message counts for consistency
    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external refundsGas(gasRefunder) {
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();

        (bytes32 dataHash, TimeBounds memory timeBounds) = formDataHash(
            data,
            afterDelayedMessagesRead
        );
        (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(dataHash, afterDelayedMessagesRead, data.length, 0, 0);
        if (seqMessageIndex != sequenceNumber)
            revert BadSequencerNumber(seqMessageIndex, sequenceNumber);
        emit SequencerBatchDelivered(
            sequenceNumber,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            BatchDataLocation.TxInput
        );
    }

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external refundsGas(gasRefunder) {
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        (bytes32 dataHash, TimeBounds memory timeBounds) = formDataHash(
            data,
            afterDelayedMessagesRead
        );
        // Reformat the stack to prevent "Stack too deep"
        uint256 sequenceNumber_ = sequenceNumber;
        TimeBounds memory timeBounds_ = timeBounds;
        bytes32 dataHash_ = dataHash;
        uint256 dataLength = data.length;
        uint256 afterDelayedMessagesRead_ = afterDelayedMessagesRead;
        uint256 prevMessageCount_ = prevMessageCount;
        uint256 newMessageCount_ = newMessageCount;
        (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(
                dataHash_,
                afterDelayedMessagesRead_,
                dataLength,
                prevMessageCount_,
                newMessageCount_
            );
        if (seqMessageIndex != sequenceNumber_ && sequenceNumber_ != ~uint256(0))
            revert BadSequencerNumber(seqMessageIndex, sequenceNumber_);
        emit SequencerBatchDelivered(
            seqMessageIndex,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds_,
            BatchDataLocation.TxInput
        );
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external override refundsGas(gasRefunder) {
        if (!isBatchPoster[msg.sender] && msg.sender != address(rollup)) revert NotBatchPoster();
        (bytes32 dataHash, TimeBounds memory timeBounds) = formDataHash(
            data,
            afterDelayedMessagesRead
        );
        uint256 seqMessageIndex;
        {
            // Reformat the stack to prevent "Stack too deep"
            uint256 sequenceNumber_ = sequenceNumber;
            TimeBounds memory timeBounds_ = timeBounds;
            bytes32 dataHash_ = dataHash;
            uint256 afterDelayedMessagesRead_ = afterDelayedMessagesRead;
            uint256 prevMessageCount_ = prevMessageCount;
            uint256 newMessageCount_ = newMessageCount;
            // we set the calldata length posted to 0 here since the caller isn't the origin
            // of the tx, so they might have not paid tx input cost for the calldata
            bytes32 beforeAcc;
            bytes32 delayedAcc;
            bytes32 afterAcc;
            (seqMessageIndex, beforeAcc, delayedAcc, afterAcc) = addSequencerL2BatchImpl(
                dataHash_,
                afterDelayedMessagesRead_,
                0,
                prevMessageCount_,
                newMessageCount_
            );
            if (seqMessageIndex != sequenceNumber_ && sequenceNumber_ != ~uint256(0))
                revert BadSequencerNumber(seqMessageIndex, sequenceNumber_);
            emit SequencerBatchDelivered(
                seqMessageIndex,
                beforeAcc,
                afterAcc,
                delayedAcc,
                totalDelayedMessagesRead,
                timeBounds_,
                BatchDataLocation.SeparateBatchEvent
            );
        }
        emit SequencerBatchData(seqMessageIndex, data);
    }

    modifier validateBatchData(bytes calldata data) {
        uint256 fullDataLen = HEADER_LENGTH + data.length;
        if (fullDataLen > MAX_DATA_SIZE) revert DataTooLarge(fullDataLen, MAX_DATA_SIZE);
        if (data.length > 0 && (data[0] & DATA_AUTHENTICATED_FLAG) == DATA_AUTHENTICATED_FLAG) {
            revert DataNotAuthenticated();
        }
        // the first byte is used to identify the type of batch data
        // das batches expect to have the type byte set, followed by the keyset (so they should have at least 33 bytes)
        if (data.length >= 33 && data[0] & 0x80 != 0) {
            // we skip the first byte, then read the next 32 bytes for the keyset
            bytes32 dasKeysetHash = bytes32(data[1:33]);
            if (!dasKeySetInfo[dasKeysetHash].isValidKeyset) revert NoSuchKeyset(dasKeysetHash);
        }
        _;
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
        validateBatchData(data)
        returns (bytes32, TimeBounds memory)
    {
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);
        bytes32 dataHash = keccak256(bytes.concat(header, data));
        return (dataHash, timeBounds);
    }

    function formEmptyDataHash(uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes32, TimeBounds memory)
    {
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);
        return (keccak256(header), timeBounds);
    }

    function addSequencerL2BatchImpl(
        bytes32 dataHash,
        uint256 afterDelayedMessagesRead,
        uint256 calldataLengthPosted,
        uint256 prevMessageCount,
        uint256 newMessageCount
    )
        internal
        returns (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        )
    {
        if (afterDelayedMessagesRead < totalDelayedMessagesRead) revert DelayedBackwards();
        if (afterDelayedMessagesRead > bridge.delayedMessageCount()) revert DelayedTooFar();

        (seqMessageIndex, beforeAcc, delayedAcc, acc) = bridge.enqueueSequencerMessage(
            dataHash,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount
        );

        totalDelayedMessagesRead = afterDelayedMessagesRead;

        if (calldataLengthPosted > 0) {
            // this msg isn't included in the current sequencer batch, but instead added to
            // the delayed messages queue that is yet to be included
            address batchPoster = msg.sender;
            bytes memory spendingReportMsg = abi.encodePacked(
                block.timestamp,
                batchPoster,
                dataHash,
                seqMessageIndex,
                block.basefee
            );
            uint256 msgNum = bridge.submitBatchSpendingReport(
                batchPoster,
                keccak256(spendingReportMsg)
            );
            // this is the same event used by Inbox.sol after including a message to the delayed message accumulator
            emit InboxMessageDelivered(msgNum, spendingReportMsg);
        }
    }

    function addSequencerL2BatchWithBlobs(
        uint256 sequenceNumber,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external override refundsGas(gasRefunder) {
        if (!isBatchPoster[msg.sender] && msg.sender != address(rollup)) revert NotBatchPoster();
        if (!(prevMessageCount == 0 && newMessageCount == 1)) revert Paused();
        (bytes memory header, TimeBounds memory timeBounds) = packHeader(afterDelayedMessagesRead);
        bytes memory data = bytes.concat(
            DATA_AUTHENTICATED_FLAG,
            IDataHashesReader(0x30bdaE426d3CBD42e9d41D23958Fac6AD8310f81).getDataHashes()
        );
        bytes32 dataHash = keccak256(bytes.concat(header, data));
        uint256 seqMessageIndex;
        {
            // Reformat the stack to prevent "Stack too deep"
            uint256 sequenceNumber_ = sequenceNumber;
            TimeBounds memory timeBounds_ = timeBounds;
            bytes32 dataHash_ = dataHash;
            uint256 afterDelayedMessagesRead_ = afterDelayedMessagesRead;
            uint256 prevMessageCount_ = prevMessageCount;
            uint256 newMessageCount_ = newMessageCount;
            // we set the calldata length posted to 0 here since the caller isn't the origin
            // of the tx, so they might have not paid tx input cost for the calldata
            bytes32 beforeAcc;
            bytes32 delayedAcc;
            bytes32 afterAcc;
            (seqMessageIndex, beforeAcc, delayedAcc, afterAcc) = addSequencerL2BatchImpl(
                dataHash_,
                afterDelayedMessagesRead_,
                0,
                prevMessageCount_,
                newMessageCount_
            );
            if (seqMessageIndex != sequenceNumber_ && sequenceNumber_ != ~uint256(0))
                revert BadSequencerNumber(seqMessageIndex, sequenceNumber_);
            emit SequencerBatchDelivered(
                seqMessageIndex,
                beforeAcc,
                afterAcc,
                delayedAcc,
                totalDelayedMessagesRead,
                timeBounds_,
                BatchDataLocation.SeparateBatchEvent
            );
        }
        emit SequencerBatchData(seqMessageIndex, data);
    }

    function inboxAccs(uint256 index) external view returns (bytes32) {
        return bridge.sequencerInboxAccs(index);
    }

    function batchCount() external view returns (uint256) {
        return bridge.sequencerMessageCount();
    }

    /// @inheritdoc ISequencerInbox
    function setMaxTimeVariation(ISequencerInbox.MaxTimeVariation memory maxTimeVariation_)
        external
        onlyRollupOwner
    {
        maxTimeVariation = maxTimeVariation_;
        emit OwnerFunctionCalled(0);
    }

    /// @inheritdoc ISequencerInbox
    function setIsBatchPoster(address addr, bool isBatchPoster_) external onlyRollupOwner {
        isBatchPoster[addr] = isBatchPoster_;
        emit OwnerFunctionCalled(1);
    }

    /// @inheritdoc ISequencerInbox
    function setValidKeyset(bytes calldata keysetBytes) external onlyRollupOwner {
        uint256 ksWord = uint256(keccak256(bytes.concat(hex"fe", keccak256(keysetBytes))));
        bytes32 ksHash = bytes32(ksWord ^ (1 << 255));
        require(keysetBytes.length < 64 * 1024, "keyset is too large");

        if (dasKeySetInfo[ksHash].isValidKeyset) revert AlreadyValidDASKeyset(ksHash);
        dasKeySetInfo[ksHash] = DasKeySetInfo({
            isValidKeyset: true,
            creationBlock: uint64(block.number)
        });
        emit SetValidKeyset(ksHash, keysetBytes);
        emit OwnerFunctionCalled(2);
    }

    /// @inheritdoc ISequencerInbox
    function invalidateKeysetHash(bytes32 ksHash) external onlyRollupOwner {
        if (!dasKeySetInfo[ksHash].isValidKeyset) revert NoSuchKeyset(ksHash);
        // we don't delete the block creation value since its used to fetch the SetValidKeyset
        // event efficiently. The event provides the hash preimage of the key.
        // this is still needed when syncing the chain after a keyset is invalidated.
        dasKeySetInfo[ksHash].isValidKeyset = false;
        emit InvalidateKeyset(ksHash);
        emit OwnerFunctionCalled(3);
    }

    function isValidKeysetHash(bytes32 ksHash) external view returns (bool) {
        return dasKeySetInfo[ksHash].isValidKeyset;
    }

    /// @inheritdoc ISequencerInbox
    function getKeysetCreationBlock(bytes32 ksHash) external view returns (uint256) {
        DasKeySetInfo memory ksInfo = dasKeySetInfo[ksHash];
        if (ksInfo.creationBlock == 0) revert NoSuchKeyset(ksHash);
        return uint256(ksInfo.creationBlock);
    }
}
