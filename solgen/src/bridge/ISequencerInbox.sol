//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../libraries/IGasRefunder.sol";

interface ISequencerInbox {
    struct MaxTimeVariation {
        uint256 delayBlocks;
        uint256 futureBlocks;
        uint256 delaySeconds;
        uint256 futureSeconds;
    }

    function inboxAccs(uint256 index) external view returns (bytes32);

    function batchCount() external view returns (uint256);

    function setMaxTimeVariation(MaxTimeVariation memory timeVariation) external;

    function setIsBatchPoster(address addr, bool isBatchPoster_) external;

    function addSequencerL2Batch(
        uint256 sequenceNumber,
        bytes calldata data,
        uint256 afterDelayedMessagesRead,
        IGasRefunder gasRefunder
    ) external;
}
