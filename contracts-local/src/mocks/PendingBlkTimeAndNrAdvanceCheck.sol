// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../precompiles/ArbSys.sol";

contract PendingBlkTimeAndNrAdvanceCheck {
    uint256 immutable deployedAt;
    uint256 immutable deployedAtBlock;
    ArbSys constant ARB_SYS = ArbSys(address(100));

    constructor() {
        deployedAt = block.timestamp;
        deployedAtBlock = ARB_SYS.arbBlockNumber();
    }

    function isAdvancing() external {
        require(block.timestamp > deployedAt, "Time didn't advance");
        require(ARB_SYS.arbBlockNumber() > deployedAtBlock, "Block didn't advance");
    }

    function checkArbBlockHashReturnsLatest(
        bytes32 expected
    ) external {
        bytes32 gotBlockHash = ARB_SYS.arbBlockHash(ARB_SYS.arbBlockNumber() - 1);
        require(gotBlockHash != bytes32(0), "ZERO_BLOCK_HASH");
        require(gotBlockHash == expected, "WRONG_BLOCK_HASH");
    }
}
