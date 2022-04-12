// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract Simple {
    uint64 public counter;

    event CounterEvent(uint64 count);
    event NullEvent();

    function increment() external {
        counter++;
    }

    function incrementEmit() external {
        counter++;
        emit CounterEvent(counter);
    }

    function emitNullEvent() external {
        emit NullEvent();
    }

    function checkBlockHashes() external view returns (uint256) {
        require(blockhash(block.number - 1) != blockhash(block.number - 2), "SAME_BLOCK_HASH");
        return block.number;
    }
}
