// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {
    AlreadyInit,
    HadZeroInit,
    BadPostUpgradeInit,
    NotOrigin,
    DataTooLarge,
    DelayedBackwards,
    DelayedTooFar,
    ForceIncludeBlockTooSoon,
    ForceIncludeTimeTooSoon,
    IncorrectMessagePreimage,
    NotBatchPoster,
    BadSequencerNumber,
    AlreadyValidDASKeyset,
    NoSuchKeyset,
    NotForked,
    NotBatchPosterManager,
    RollupNotChanged,
    DataBlobsNotSupported,
    InitParamZero,
    MissingDataHashes,
    NotOwner,
    InvalidHeaderFlag,
    NativeTokenMismatch,
    BadMaxTimeVariation,
    Deprecated,
    NotDelayBufferable,
    InvalidDelayedAccPreimage,
    DelayProofRequired,
    BadBufferConfig,
    ExtraGasNotUint64,
    KeysetTooLarge
} from "../libraries/Error.sol";
import "./IBridge.sol";
import "./IInboxBase.sol";
import "./ISequencerInbox.sol";
import "../rollup/IRollupLogic.sol";
import "./Messages.sol";
import "../precompiles/ArbGasInfo.sol";
import "../precompiles/ArbSys.sol";
import "../libraries/IReader4844.sol";

import {L1MessageType_batchPostingReport} from "../libraries/MessageTypes.sol";
import "../libraries/DelegateCallAware.sol";
import {IGasRefunder} from "../libraries/IGasRefunder.sol";
import {GasRefundEnabled} from "../libraries/GasRefundEnabled.sol";
import "../libraries/ArbitrumChecker.sol";
import {IERC20Bridge} from "./IERC20Bridge.sol";
import "./DelayBuffer.sol";

/**
 * @title  Accepts batches from the sequencer and adds them to the rollup inbox.
 * @notice Contains the inbox accumulator which is the ordering of all data and transactions to be processed by the rollup.
 *         As part of submitting a batch the sequencer is also expected to include items enqueued
 *         in the delayed inbox (Bridge.sol). If items in the delayed inbox are not included by a
 *         sequencer within a time limit they can be force included into the rollup inbox by anyone.
 */
contract SequencerInbox is DelegateCallAware, GasRefundEnabled, ISequencerInbox {
    uint256 public totalDelayedMessagesRead;

    IBridge public bridge;

    /// @inheritdoc ISequencerInbox
    uint256 public constant HEADER_LENGTH = 40;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant DATA_AUTHENTICATED_FLAG = 0x40;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant DATA_BLOB_HEADER_FLAG = DATA_AUTHENTICATED_FLAG | 0x10;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant DAS_MESSAGE_HEADER_FLAG = 0x80;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant TREE_DAS_MESSAGE_HEADER_FLAG = 0x08;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant BROTLI_MESSAGE_HEADER_FLAG = 0x00;

    /// @inheritdoc ISequencerInbox
    bytes1 public constant ZERO_HEAVY_MESSAGE_HEADER_FLAG = 0x20;

    // GAS_PER_BLOB from EIP-4844
    uint256 internal constant GAS_PER_BLOB = 1 << 17;

    IOwnable public rollup;

    mapping(address => bool) public isBatchPoster;

    // we previously stored the max time variation in a (uint,uint,uint,uint) struct here
    // solhint-disable-next-line var-name-mixedcase
    ISequencerInbox.MaxTimeVariation private __LEGACY_MAX_TIME_VARIATION;

    mapping(bytes32 => DasKeySetInfo) public dasKeySetInfo;

    modifier onlyRollupOwner() {
        if (msg.sender != rollup.owner()) revert NotOwner(msg.sender, rollup.owner());
        _;
    }

    modifier onlyRollupOwnerOrBatchPosterManager() {
        if (msg.sender != rollup.owner() && msg.sender != batchPosterManager) {
            revert NotBatchPosterManager(msg.sender);
        }
        _;
    }

    mapping(address => bool) public isSequencer;
    IReader4844 public immutable reader4844;

    // see ISequencerInbox.MaxTimeVariation
    uint64 internal delayBlocks;
    uint64 internal futureBlocks;
    uint64 internal delaySeconds;
    uint64 internal futureSeconds;

    /// @inheritdoc ISequencerInbox
    address public batchPosterManager;

    // On L1 this should be set to 117964: 90% of Geth's 128KB tx size limit, leaving ~13KB for proving
    uint256 public immutable maxDataSize;
    uint256 internal immutable deployTimeChainId = block.chainid;
    // If the chain this SequencerInbox is deployed on is an Arbitrum chain.
    bool internal immutable hostChainIsArbitrum = ArbitrumChecker.runningOnArbitrum();
    // True if the chain this SequencerInbox is deployed on uses custom fee token
    bool public immutable isUsingFeeToken;
    // True if the SequencerInbox is delay bufferable
    bool public immutable isDelayBufferable;

    using DelayBuffer for BufferData;
    BufferData public buffer;

    constructor(
        uint256 _maxDataSize,
        IReader4844 reader4844_,
        bool _isUsingFeeToken,
        bool _isDelayBufferable
    ) {
        maxDataSize = _maxDataSize;
        if (hostChainIsArbitrum) {
            if (reader4844_ != IReader4844(address(0))) revert DataBlobsNotSupported();
        } else {
            if (reader4844_ == IReader4844(address(0))) revert InitParamZero("Reader4844");
        }
        reader4844 = reader4844_;
        isUsingFeeToken = _isUsingFeeToken;
        isDelayBufferable = _isDelayBufferable;
    }

    function _chainIdChanged() internal view returns (bool) {
        return deployTimeChainId != block.chainid;
    }

    function postUpgradeInit(BufferConfig memory bufferConfig_)
        external
        onlyDelegated
        onlyProxyOwner
    {
        if (!isDelayBufferable) revert NotDelayBufferable();

        // Assuming we would not upgrade from a version that does not have the buffer initialized
        // If that is the case, postUpgradeInit do not need to be called
        if (buffer.bufferBlocks != 0) {
            revert AlreadyInit();
        }

        _setBufferConfig(bufferConfig_);
    }

    function initialize(
        IBridge bridge_,
        ISequencerInbox.MaxTimeVariation calldata maxTimeVariation_,
        BufferConfig memory bufferConfig_
    ) external onlyDelegated {
        if (bridge != IBridge(address(0))) revert AlreadyInit();
        if (bridge_ == IBridge(address(0))) revert HadZeroInit();

        // Make sure logic contract was created by proper value for 'isUsingFeeToken'.
        // Bridge in ETH based chains doesn't implement nativeToken(). In future it might implement it and return address(0)
        bool actualIsUsingFeeToken = false;
        try IERC20Bridge(address(bridge_)).nativeToken() returns (address feeToken) {
            if (feeToken != address(0)) {
                actualIsUsingFeeToken = true;
            }
        } catch {}
        if (isUsingFeeToken != actualIsUsingFeeToken) {
            revert NativeTokenMismatch();
        }

        bridge = bridge_;
        rollup = bridge_.rollup();

        _setMaxTimeVariation(maxTimeVariation_);

        if (isDelayBufferable) {
            _setBufferConfig(bufferConfig_);
        }
    }

    /// @notice Allows the rollup owner to sync the rollup address
    function updateRollupAddress() external {
        if (msg.sender != IOwnable(rollup).owner())
            revert NotOwner(msg.sender, IOwnable(rollup).owner());
        IOwnable newRollup = bridge.rollup();
        if (rollup == newRollup) revert RollupNotChanged();
        rollup = newRollup;
    }

    function getTimeBounds() internal view virtual returns (IBridge.TimeBounds memory) {
        IBridge.TimeBounds memory bounds;
        (
            uint64 delayBlocks_,
            uint64 futureBlocks_,
            uint64 delaySeconds_,
            uint64 futureSeconds_
        ) = maxTimeVariationInternal();
        if (block.timestamp > delaySeconds_) {
            bounds.minTimestamp = uint64(block.timestamp) - delaySeconds_;
        }
        bounds.maxTimestamp = uint64(block.timestamp) + futureSeconds_;
        if (block.number > delayBlocks_) {
            bounds.minBlockNumber = uint64(block.number) - delayBlocks_;
        }
        bounds.maxBlockNumber = uint64(block.number) + futureBlocks_;
        return bounds;
    }

    /// @inheritdoc ISequencerInbox
    function removeDelayAfterFork() external {
        if (!_chainIdChanged()) revert NotForked();
        delayBlocks = 1;
        futureBlocks = 1;
        delaySeconds = 1;
        futureSeconds = 1;
    }

    function maxTimeVariation()
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256
        )
    {
        (
            uint64 delayBlocks_,
            uint64 futureBlocks_,
            uint64 delaySeconds_,
            uint64 futureSeconds_
        ) = maxTimeVariationInternal();

        return (
            uint256(delayBlocks_),
            uint256(futureBlocks_),
            uint256(delaySeconds_),
            uint256(futureSeconds_)
        );
    }

    function maxTimeVariationInternal()
        internal
        view
        returns (
            uint64,
            uint64,
            uint64,
            uint64
        )
    {
        if (_chainIdChanged()) {
            return (1, 1, 1, 1);
        } else {
            return (delayBlocks, futureBlocks, delaySeconds, futureSeconds);
        }
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

        uint256 delayBlocks_ = delayBlocks;

        if (isDelayBufferable) {
            // proactively apply any pending delay buffer updates before the force included message l1BlockAndTime
            buffer.update(l1BlockAndTime[0]);
            delayBlocks_ = delayBufferableBlocks(buffer.bufferBlocks);
        }
        // Can only force-include after the Sequencer-only window has expired.
        if (l1BlockAndTime[0] + delayBlocks_ >= block.number) revert ForceIncludeBlockTooSoon();

        // Verify that message hash represents the last message sequence of delayed message to be included
        bytes32 prevDelayedAcc = 0;
        if (_totalDelayedMessagesRead > 1) {
            prevDelayedAcc = bridge.delayedInboxAccs(_totalDelayedMessagesRead - 2);
        }
        if (
            bridge.delayedInboxAccs(_totalDelayedMessagesRead - 1) !=
            Messages.accumulateInboxMessage(prevDelayedAcc, messageHash)
        ) revert IncorrectMessagePreimage();

        (bytes32 dataHash, IBridge.TimeBounds memory timeBounds) = formEmptyDataHash(
            _totalDelayedMessagesRead
        );
        uint256 __totalDelayedMessagesRead = _totalDelayedMessagesRead;
        uint256 prevSeqMsgCount = bridge.sequencerReportedSubMessageCount();
        uint256 newSeqMsgCount = prevSeqMsgCount; // force inclusion should not modify sequencer message count
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
            IBridge.BatchDataLocation.NoData
        );
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchFromOrigin(
        uint256,
        bytes calldata,
        uint256,
        IGasRefunder
    ) external pure {
        revert Deprecated();
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external refundsGas(gasRefunder, IReader4844(address(0))) {
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        if (isDelayProofRequired(afterDelayedMessagesRead)) revert DelayProofRequired();

        addSequencerL2BatchFromCalldataImpl(
            sequenceNumber,
            data,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount,
            true
        );
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchFromBlobs(
        uint256 sequenceNumber,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external refundsGas(gasRefunder, reader4844) {
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        if (isDelayProofRequired(afterDelayedMessagesRead)) revert DelayProofRequired();

        addSequencerL2BatchFromBlobsImpl(
            sequenceNumber,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount
        );
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchFromBlobsDelayProof(
        uint256 sequenceNumber,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount,
        DelayProof calldata delayProof
    ) external refundsGas(gasRefunder, reader4844) {
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        if (!isDelayBufferable) revert NotDelayBufferable();

        delayProofImpl(afterDelayedMessagesRead, delayProof);
        addSequencerL2BatchFromBlobsImpl(
            sequenceNumber,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount
        );
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchFromOriginDelayProof(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount,
        DelayProof calldata delayProof
    ) external refundsGas(gasRefunder, IReader4844(address(0))) {
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        if (!isBatchPoster[msg.sender]) revert NotBatchPoster();
        if (!isDelayBufferable) revert NotDelayBufferable();

        delayProofImpl(afterDelayedMessagesRead, delayProof);
        addSequencerL2BatchFromCalldataImpl(
            sequenceNumber,
            data,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount,
            true
        );
    }

    function addSequencerL2BatchFromBlobsImpl(
        uint256 sequenceNumber,
        uint256 afterDelayedMessagesRead,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) internal {
        (
            bytes32 dataHash,
            IBridge.TimeBounds memory timeBounds,
            uint256 blobGas
        ) = formBlobDataHash(afterDelayedMessagesRead);

        // we use addSequencerL2BatchImpl for submitting the message
        // normally this would also submit a batch spending report but that is skipped if we pass
        // an empty call data size, then we submit a separate batch spending report later
        (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(
                dataHash,
                afterDelayedMessagesRead,
                0,
                prevMessageCount,
                newMessageCount
            );

        // ~uint256(0) is type(uint256).max, but ever so slightly cheaper
        if (seqMessageIndex != sequenceNumber && sequenceNumber != ~uint256(0)) {
            revert BadSequencerNumber(seqMessageIndex, sequenceNumber);
        }

        emit SequencerBatchDelivered(
            sequenceNumber,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            IBridge.BatchDataLocation.Blob
        );

        // blobs are currently not supported on host arbitrum chains, when support is added it may
        // consume gas in a different way to L1, so explicitly block host arb chains so that if support for blobs
        // on arb is added it will need to explicitly turned on in the sequencer inbox
        if (hostChainIsArbitrum) revert DataBlobsNotSupported();

        // submit a batch spending report to refund the entity that produced the blob batch data
        // same as using calldata, we only submit spending report if the caller is the origin of the tx
        // such that one cannot "double-claim" batch posting refund in the same tx
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender == tx.origin && !isUsingFeeToken) {
            submitBatchSpendingReport(dataHash, seqMessageIndex, block.basefee, blobGas);
        }
    }

    function addSequencerL2BatchFromCalldataImpl(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        uint256 prevMessageCount,
        uint256 newMessageCount,
        bool isFromOrigin
    ) internal {
        (bytes32 dataHash, IBridge.TimeBounds memory timeBounds) = formCallDataHash(
            data,
            afterDelayedMessagesRead
        );
        (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 afterAcc
        ) = addSequencerL2BatchImpl(
                dataHash,
                afterDelayedMessagesRead,
                isFromOrigin ? data.length : 0,
                prevMessageCount,
                newMessageCount
            );

        // ~uint256(0) is type(uint256).max, but ever so slightly cheaper
        if (seqMessageIndex != sequenceNumber && sequenceNumber != ~uint256(0)) {
            revert BadSequencerNumber(seqMessageIndex, sequenceNumber);
        }

        emit SequencerBatchDelivered(
            seqMessageIndex,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds,
            isFromOrigin
                ? IBridge.BatchDataLocation.TxInput
                : IBridge.BatchDataLocation.SeparateBatchEvent
        );

        if (!isFromOrigin) {
            emit SequencerBatchData(seqMessageIndex, data);
        }
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount
    ) external override refundsGas(gasRefunder, IReader4844(address(0))) {
        if (!isBatchPoster[msg.sender] && msg.sender != address(rollup)) revert NotBatchPoster();
        if (isDelayProofRequired(afterDelayedMessagesRead)) revert DelayProofRequired();

        addSequencerL2BatchFromCalldataImpl(
            sequenceNumber,
            data,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount,
            false
        );
    }

    /// @inheritdoc ISequencerInbox
    function addSequencerL2BatchDelayProof(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder,
        uint256 prevMessageCount,
        uint256 newMessageCount,
        DelayProof calldata delayProof
    ) external refundsGas(gasRefunder, IReader4844(address(0))) {
        if (!isBatchPoster[msg.sender] && msg.sender != address(rollup)) revert NotBatchPoster();
        if (!isDelayBufferable) revert NotDelayBufferable();

        delayProofImpl(afterDelayedMessagesRead, delayProof);
        addSequencerL2BatchFromCalldataImpl(
            sequenceNumber,
            data,
            afterDelayedMessagesRead,
            prevMessageCount,
            newMessageCount,
            false
        );
    }

    function delayProofImpl(uint256 afterDelayedMessagesRead, DelayProof memory delayProof)
        internal
    {
        // buffer update depends on new delayed messages. if none are read, no buffer update is proccessed
        if (afterDelayedMessagesRead > totalDelayedMessagesRead) {
            if (buffer.isUpdatable()) {
                // delayedAcc of the 1st new delayed message
                bytes32 delayedAcc = bridge.delayedInboxAccs(totalDelayedMessagesRead);
                // validate delayProof against the delayed accumulator
                if (
                    !Messages.isValidDelayedAccPreimage(
                        delayedAcc,
                        delayProof.beforeDelayedAcc,
                        delayProof.delayedMessage
                    )
                ) {
                    revert InvalidDelayedAccPreimage();
                }
                buffer.update(delayProof.delayedMessage.blockNumber);
            }
        }
    }

    function isDelayProofRequired(uint256 afterDelayedMessagesRead) internal view returns (bool) {
        // if no new delayed messages are read, no buffer updates can be applied, so no proof required
        // if the buffer is synced, the buffer cannot be depleted, so no proof is required
        return
            isDelayBufferable &&
            afterDelayedMessagesRead > totalDelayedMessagesRead &&
            !buffer.isSynced();
    }

    function packHeader(uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes memory, IBridge.TimeBounds memory)
    {
        IBridge.TimeBounds memory timeBounds = getTimeBounds();
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

    /// @dev    Form a hash for a sequencer message with no batch data
    /// @param  afterDelayedMessagesRead The delayed messages count read up to
    /// @return The data hash
    /// @return The timebounds within which the message should be processed
    function formEmptyDataHash(uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes32, IBridge.TimeBounds memory)
    {
        (bytes memory header, IBridge.TimeBounds memory timeBounds) = packHeader(
            afterDelayedMessagesRead
        );
        return (keccak256(header), timeBounds);
    }

    /// @dev    Since the data is supplied from calldata, the batch poster can choose the data type
    ///         We need to ensure that this data cannot cause a collision with data supplied via another method (eg blobs)
    ///         therefore we restrict which flags can be provided as a header in this field
    ///         This also safe guards unused flags for future use, as we know they would have been disallowed up until this point
    /// @param  headerByte The first byte in the calldata
    function isValidCallDataFlag(bytes1 headerByte) internal pure returns (bool) {
        return
            headerByte == BROTLI_MESSAGE_HEADER_FLAG ||
            headerByte == DAS_MESSAGE_HEADER_FLAG ||
            (headerByte == (DAS_MESSAGE_HEADER_FLAG | TREE_DAS_MESSAGE_HEADER_FLAG)) ||
            headerByte == ZERO_HEAVY_MESSAGE_HEADER_FLAG;
    }

    /// @dev    Form a hash of the data taken from the calldata
    /// @param  data The calldata to be hashed
    /// @param  afterDelayedMessagesRead The delayed messages count read up to
    /// @return The data hash
    /// @return The timebounds within which the message should be processed
    function formCallDataHash(bytes calldata data, uint256 afterDelayedMessagesRead)
        internal
        view
        returns (bytes32, IBridge.TimeBounds memory)
    {
        uint256 fullDataLen = HEADER_LENGTH + data.length;
        if (fullDataLen > maxDataSize) revert DataTooLarge(fullDataLen, maxDataSize);

        (bytes memory header, IBridge.TimeBounds memory timeBounds) = packHeader(
            afterDelayedMessagesRead
        );

        // the batch poster is allowed to submit an empty batch, they can use this to progress the
        // delayed inbox without providing extra batch data
        if (data.length > 0) {
            // The first data byte cannot be the same as any that have been set via other methods (eg 4844 blob header) as this
            // would allow the supplier of the data to spoof an incorrect 4844 data batch
            if (!isValidCallDataFlag(data[0])) revert InvalidHeaderFlag(data[0]);

            // the first byte is used to identify the type of batch data
            // das batches expect to have the type byte set, followed by the keyset (so they should have at least 33 bytes)
            // if invalid data is supplied here the state transition function will process it as an empty block
            // however we can provide a nice additional check here for the batch poster
            if (data[0] & DAS_MESSAGE_HEADER_FLAG != 0 && data.length >= 33) {
                // we skip the first byte, then read the next 32 bytes for the keyset
                bytes32 dasKeysetHash = bytes32(data[1:33]);
                if (!dasKeySetInfo[dasKeysetHash].isValidKeyset) revert NoSuchKeyset(dasKeysetHash);
            }
        }
        return (keccak256(bytes.concat(header, data)), timeBounds);
    }

    /// @dev    Form a hash of the data being provided in 4844 data blobs
    /// @param  afterDelayedMessagesRead The delayed messages count read up to
    /// @return The data hash
    /// @return The timebounds within which the message should be processed
    /// @return The normalized amount of gas used for blob posting
    function formBlobDataHash(uint256 afterDelayedMessagesRead)
        internal
        view
        virtual
        returns (
            bytes32,
            IBridge.TimeBounds memory,
            uint256
        )
    {
        bytes32[] memory dataHashes = reader4844.getDataHashes();
        if (dataHashes.length == 0) revert MissingDataHashes();

        (bytes memory header, IBridge.TimeBounds memory timeBounds) = packHeader(
            afterDelayedMessagesRead
        );

        uint256 blobCost = reader4844.getBlobBaseFee() * GAS_PER_BLOB * dataHashes.length;
        return (
            keccak256(bytes.concat(header, DATA_BLOB_HEADER_FLAG, abi.encodePacked(dataHashes))),
            timeBounds,
            block.basefee > 0 ? blobCost / block.basefee : 0
        );
    }

    /// @dev   Submit a batch spending report message so that the batch poster can be reimbursed on the rollup
    ///        This function expect msg.sender is tx.origin, and will always record tx.origin as the spender
    /// @param dataHash The hash of the message the spending report is being submitted for
    /// @param seqMessageIndex The index of the message to submit the spending report for
    /// @param gasPrice The gas price that was paid for the data (standard gas or data gas)
    function submitBatchSpendingReport(
        bytes32 dataHash,
        uint256 seqMessageIndex,
        uint256 gasPrice,
        uint256 extraGas
    ) internal {
        // report the account who paid the gas (tx.origin) for the tx as batch poster
        // if msg.sender is used and is a contract, it might not be able to spend the refund on l2
        // solhint-disable-next-line avoid-tx-origin
        address batchPoster = tx.origin;

        // this msg isn't included in the current sequencer batch, but instead added to
        // the delayed messages queue that is yet to be included
        if (hostChainIsArbitrum) {
            // Include extra gas for the host chain's L1 gas charging
            uint256 l1Fees = ArbGasInfo(address(0x6c)).getCurrentTxL1GasFees();
            extraGas += l1Fees / block.basefee;
        }
        if (extraGas > type(uint64).max) revert ExtraGasNotUint64();
        bytes memory spendingReportMsg = abi.encodePacked(
            block.timestamp,
            batchPoster,
            dataHash,
            seqMessageIndex,
            gasPrice,
            uint64(extraGas)
        );

        uint256 msgNum = bridge.submitBatchSpendingReport(
            batchPoster,
            keccak256(spendingReportMsg)
        );
        // this is the same event used by Inbox.sol after including a message to the delayed message accumulator
        emit InboxMessageDelivered(msgNum, spendingReportMsg);
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

        if (calldataLengthPosted > 0 && !isUsingFeeToken) {
            // only report batch poster spendings if chain is using ETH as native currency
            submitBatchSpendingReport(dataHash, seqMessageIndex, block.basefee, 0);
        }
    }

    function inboxAccs(uint256 index) external view returns (bytes32) {
        return bridge.sequencerInboxAccs(index);
    }

    function batchCount() external view returns (uint256) {
        return bridge.sequencerMessageCount();
    }

    /// @inheritdoc ISequencerInbox
    function forceInclusionDeadline(uint64 blockNumber) external view returns (uint64) {
        uint64 _delayBlocks = delayBlocks;
        if (isDelayBufferable) {
            uint64 _buffer = buffer.calcPendingBuffer(blockNumber);
            _delayBlocks = delayBufferableBlocks(_buffer);
        }
        return blockNumber + _delayBlocks;
    }

    /// @notice Calculates the buffer dependent delay blocks
    function delayBufferableBlocks(uint64 _buffer) internal view returns (uint64) {
        return _buffer < delayBlocks ? _buffer : delayBlocks;
    }

    function _setBufferConfig(BufferConfig memory bufferConfig_) internal {
        if (!isDelayBufferable) revert NotDelayBufferable();
        if (!DelayBuffer.isValidBufferConfig(bufferConfig_)) revert BadBufferConfig();

        if (buffer.bufferBlocks == 0 || buffer.bufferBlocks > bufferConfig_.max) {
            buffer.bufferBlocks = bufferConfig_.max;
        }
        if (buffer.bufferBlocks < bufferConfig_.threshold) {
            buffer.bufferBlocks = bufferConfig_.threshold;
        }
        buffer.max = bufferConfig_.max;
        buffer.threshold = bufferConfig_.threshold;
        buffer.replenishRateInBasis = bufferConfig_.replenishRateInBasis;

        // if all delayed messages are read, the buffer is considered synced
        if (bridge.delayedMessageCount() == totalDelayedMessagesRead) {
            buffer.update(uint64(block.number));
        }
    }

    function _setMaxTimeVariation(ISequencerInbox.MaxTimeVariation memory maxTimeVariation_)
        internal
    {
        if (
            maxTimeVariation_.delayBlocks > type(uint64).max ||
            maxTimeVariation_.futureBlocks > type(uint64).max ||
            maxTimeVariation_.delaySeconds > type(uint64).max ||
            maxTimeVariation_.futureSeconds > type(uint64).max
        ) {
            revert BadMaxTimeVariation();
        }
        delayBlocks = uint64(maxTimeVariation_.delayBlocks);
        futureBlocks = uint64(maxTimeVariation_.futureBlocks);
        delaySeconds = uint64(maxTimeVariation_.delaySeconds);
        futureSeconds = uint64(maxTimeVariation_.futureSeconds);
    }

    /// @inheritdoc ISequencerInbox
    function setMaxTimeVariation(ISequencerInbox.MaxTimeVariation memory maxTimeVariation_)
        external
        onlyRollupOwner
    {
        _setMaxTimeVariation(maxTimeVariation_);
        emit MaxTimeVariationSet(maxTimeVariation_);
        emit OwnerFunctionCalled(0);
    }

    /// @inheritdoc ISequencerInbox
    function setIsBatchPoster(address addr, bool isBatchPoster_)
        external
        onlyRollupOwnerOrBatchPosterManager
    {
        isBatchPoster[addr] = isBatchPoster_;
        emit BatchPosterSet(addr, isBatchPoster_);
        emit OwnerFunctionCalled(1);
    }

    /// @inheritdoc ISequencerInbox
    function setValidKeyset(bytes calldata keysetBytes) external onlyRollupOwner {
        uint256 ksWord = uint256(keccak256(bytes.concat(hex"fe", keccak256(keysetBytes))));
        bytes32 ksHash = bytes32(ksWord ^ (1 << 255));
        if (keysetBytes.length >= 64 * 1024) revert KeysetTooLarge();

        if (dasKeySetInfo[ksHash].isValidKeyset) revert AlreadyValidDASKeyset(ksHash);
        uint256 creationBlock = block.number;
        if (hostChainIsArbitrum) {
            creationBlock = ArbSys(address(100)).arbBlockNumber();
        }
        dasKeySetInfo[ksHash] = DasKeySetInfo({
            isValidKeyset: true,
            creationBlock: uint64(creationBlock)
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

    /// @inheritdoc ISequencerInbox
    function setIsSequencer(address addr, bool isSequencer_)
        external
        onlyRollupOwnerOrBatchPosterManager
    {
        isSequencer[addr] = isSequencer_;
        emit SequencerSet(addr, isSequencer_);
        emit OwnerFunctionCalled(4);
    }

    /// @inheritdoc ISequencerInbox
    function setBatchPosterManager(address newBatchPosterManager) external onlyRollupOwner {
        batchPosterManager = newBatchPosterManager;
        emit BatchPosterManagerSet(newBatchPosterManager);
        emit OwnerFunctionCalled(5);
    }

    function setBufferConfig(BufferConfig memory bufferConfig_) external onlyRollupOwner {
        _setBufferConfig(bufferConfig_);
        emit BufferConfigSet(bufferConfig_);
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
