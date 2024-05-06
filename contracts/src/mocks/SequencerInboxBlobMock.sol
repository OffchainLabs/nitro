// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/SequencerInbox.sol";

contract SequencerInboxBlobMock is SequencerInbox {
    constructor(
        uint256 maxDataSize_,
        IReader4844 reader_,
        bool isUsingFeeToken_,
        bool isDelayBufferable_
    ) SequencerInbox(maxDataSize_, reader_, isUsingFeeToken_, isDelayBufferable_) {}

    /// @dev    Form a hash of the data being provided in 4844 data blobs
    /// @param  afterDelayedMessagesRead The delayed messages count read up to
    /// @return The data hash
    /// @return The timebounds within which the message should be processed
    /// @return The normalized amount of gas used for blob posting
    function formBlobDataHash(uint256 afterDelayedMessagesRead)
        internal
        view
        override
        returns (
            bytes32,
            IBridge.TimeBounds memory,
            uint256
        )
    {
        bytes32[3] memory dataHashes = [bytes32(0), bytes32(0), bytes32(0)];
        if (dataHashes.length == 0) revert MissingDataHashes();

        (bytes memory header, IBridge.TimeBounds memory timeBounds) = packHeader(
            afterDelayedMessagesRead
        );
        uint256 BLOB_BASE_FEE = 1 gwei;
        uint256 blobCost = BLOB_BASE_FEE * GAS_PER_BLOB * dataHashes.length;
        return (
            keccak256(bytes.concat(header, DATA_BLOB_HEADER_FLAG, abi.encodePacked(dataHashes))),
            timeBounds,
            block.basefee > 0 ? blobCost / block.basefee : 0
        );
    }
}
