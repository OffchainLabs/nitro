//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../utils/IGasRefunder.sol";

interface ISequencerInbox {
    function inboxAccs(uint256 index) external view returns (bytes32);

    function batchCount() external view returns (uint256);

    function setMaxTimeVariation(
        uint256 maxDelayBlocks,
        uint256 maxFutureBlocks,
        uint256 maxDelaySeconds,
        uint256 maxFutureSeconds
    ) external;

    function setIsBatchPoster(address addr, bool isBatchPoster_) external;

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external;
}
