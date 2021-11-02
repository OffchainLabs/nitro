//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.7.5;

import "./IBridge.sol";
import "./Messages.sol";
import "../utils/IGasRefunder.sol";

contract SequencerInbox {
	bytes32[] public inboxAccs;
	uint256 public batchCount;
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
        uint256[4] timeBounds,
        bytes data
    );

    event SequencerBatchDeliveredFromOrigin(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        uint256[4] timeBounds
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

    function getTimeBounds() internal view returns (uint256[4] memory) {
        uint256[4] memory bounds;
        if (block.timestamp > maxDelaySeconds) {
            bounds[0] = block.timestamp - maxDelaySeconds;
        } else {
            bounds[0] = 0;
        }
        bounds[1] = block.timestamp + maxFutureSeconds;
        if (block.number > maxDelayBlocks) {
            bounds[2] = block.number - maxDelayBlocks;
        } else {
            bounds[2] = 0;
        }
        bounds[3] = block.number + maxFutureBlocks;
        return bounds;
    }

    function forceInclusion(
        uint256 _totalDelayedMessagesRead,
        uint8 kind,
        uint256[2] calldata l1BlockAndTimestamp,
        uint256 inboxSeqNum,
        uint256 gasPriceL1,
        address sender,
        bytes32 messageDataHash
    ) external {
        require(_totalDelayedMessagesRead > totalDelayedMessagesRead, "DELAYED_BACKWARDS");
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
            require(l1BlockAndTimestamp[0] + maxDelayBlocks < block.number, "MAX_DELAY_BLOCKS");
            require(l1BlockAndTimestamp[1] + maxDelaySeconds < block.timestamp, "MAX_DELAY_TIME");

            // Verify that message hash represents the last message sequence of delayed message to be included
            bytes32 prevDelayedAcc = 0;
            if (_totalDelayedMessagesRead > 1) {
                prevDelayedAcc = delayedBridge.inboxAccs(_totalDelayedMessagesRead - 2);
            }
            require(
                delayedBridge.inboxAccs(_totalDelayedMessagesRead - 1) ==
                    Messages.addMessageToInbox(prevDelayedAcc, messageHash),
                "DELAYED_ACCUMULATOR"
            );
        }

        bytes calldata emptyData;
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc, uint256[4] memory timeBounds) = addSequencerL2BatchImpl(
            emptyData,
            _totalDelayedMessagesRead
        );
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
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "ORIGIN_ONLY");
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        uint256 startGasLeft = gasleft();
        uint256 calldataSize;
        assembly {
            calldataSize := calldatasize()
        }

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc, uint256[4] memory timeBounds) = addSequencerL2BatchImpl(
            data,
            afterDelayedMessagesRead
        );
        emit SequencerBatchDeliveredFromOrigin(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            totalDelayedMessagesRead,
            timeBounds
        );

        if (gasRefunder != IGasRefunder(0)) {
            gasRefunder.onGasSpent(msg.sender, startGasLeft - gasleft(), calldataSize);
        }
    }

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external {
        require(isBatchPoster[msg.sender], "NOT_BATCH_POSTER");

        uint256 startGasLeft = gasleft();

        require(inboxAccs.length == sequenceNumber, "BAD_SEQ_NUM");
        (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 afterAcc, uint256[4] memory timeBounds) = addSequencerL2BatchImpl(
            data,
            afterDelayedMessagesRead
        );
        emit SequencerBatchDelivered(
            inboxAccs.length - 1,
            beforeAcc,
            afterAcc,
            delayedAcc,
            afterDelayedMessagesRead,
            timeBounds,
            data
        );

        if (gasRefunder != IGasRefunder(0)) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            gasRefunder.onGasSpent(msg.sender, startGasLeft - gasleft(), calldataSize);
        }
    }

    function addSequencerL2BatchImpl(
        bytes calldata data,
        uint256 afterDelayedMessagesRead
    ) internal returns (bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc, uint256[4] memory timeBounds) {
        require(afterDelayedMessagesRead >= totalDelayedMessagesRead, "DELAYED_BACKWARDS");
        require(delayedBridge.messageCount() >= afterDelayedMessagesRead, "DELAYED_TOO_FAR");

        uint256 fullDataLen = 40 + data.length;
        require(fullDataLen >= 40, "DATA_LEN_OVERFLOW");
        bytes memory fullData = new bytes(fullDataLen);
        bytes memory header = abi.encodePacked(
            uint64(block.timestamp - maxDelaySeconds),
            uint64(block.timestamp + maxFutureSeconds),
            uint64(block.number - maxDelayBlocks),
            uint64(block.number + maxFutureBlocks),
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
        timeBounds = getTimeBounds();
        bytes32 fullDataHash = keccak256(fullData);
        acc = keccak256(abi.encodePacked(beforeAcc, fullDataHash, delayedAcc, timeBounds));
        inboxAccs.push(acc);
        totalDelayedMessagesRead = afterDelayedMessagesRead;
    }
}
