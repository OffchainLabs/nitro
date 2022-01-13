//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../bridge/IBridge.sol";
import "../bridge/Messages.sol";
import "../bridge/ISequencerInbox.sol";

contract SequencerInboxStub is ISequencerInbox {
	bytes32[] public override inboxAccs;
    uint256 public totalDelayedMessagesRead;

    IBridge public delayedBridge;

    mapping(address => bool) public isBatchPoster;
    uint256 public maxDelayBlocks;
    uint256 public maxFutureBlocks;
    uint256 public maxDelaySeconds;
    uint256 public maxFutureSeconds;

    event SequencerBatchDelivered(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        uint64[4] timeBounds,
        bytes data
    );

    event SequencerBatchDeliveredFromOrigin(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        uint64[4] timeBounds
    );

    constructor(
        IBridge _delayedBridge,
        address _sequencer
    ) {
        delayedBridge = _delayedBridge;
        isBatchPoster[_sequencer] = true;

		maxDelaySeconds = 60*60*24;
		maxFutureSeconds = 60*60;

		maxDelayBlocks = maxDelaySeconds * 15;
		maxFutureBlocks = 12;
    }

    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        uint256 gasRefunder
    ) external {
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "ORIGIN_ONLY");
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        uint256 calldataSize;
        assembly {
            calldataSize := calldatasize()
        }

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            data,
            afterDelayedMessagesRead
        );

        uint64[4] memory emptyTimeBounds;
        emit SequencerBatchDeliveredFromOrigin(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            emptyTimeBounds
        );

        if (gasRefunder != 0) {
            revert("NOT_SUPPORTED");
        }
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        uint256 gasRefunder
    ) external {
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc) = addSequencerL2BatchImpl(
            data,
            afterDelayedMessagesRead
        );
        uint64[4] memory emptyTimeBounds;
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            emptyTimeBounds,
            data
        );

        if (gasRefunder != 0) {
            revert("NOT_SUPPORTED");
        }
    }

    function addSequencerL2BatchImpl(
        bytes calldata data,
        uint256 afterDelayedMessagesRead
    ) internal returns (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc) {
        require(afterDelayedMessagesRead >= totalDelayedMessagesRead, "DELAYED_BACKWARDS");
        require(delayedBridge.messageCount() >= afterDelayedMessagesRead, "DELAYED_TOO_FAR");

        uint256 fullDataLen = 40 + data.length;
        require(fullDataLen >= 40, "DATA_LEN_OVERFLOW");
        bytes memory fullData = new bytes(fullDataLen);

        bytes memory header = abi.encodePacked(
            uint64(0),
            uint64(0),
            uint64(0),
            uint64(0),
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

    function batchCount() override external view returns (uint256) {
        return inboxAccs.length;
    }

    function setMaxTimeVariation(
        uint256 maxDelayBlocks_,
        uint256 maxFutureBlocks_,
        uint256 maxDelaySeconds_,
        uint256 maxFutureSeconds_
    ) external override {
        maxDelayBlocks = maxDelayBlocks_;
        maxFutureBlocks = maxFutureBlocks_;
        maxDelaySeconds = maxDelaySeconds_;
        maxFutureSeconds = maxFutureSeconds_;
    }

    function setIsBatchPoster(address addr, bool isBatchPoster_) external override {
        isBatchPoster[addr] = isBatchPoster_;
    }
}
