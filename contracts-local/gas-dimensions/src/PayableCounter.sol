// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract PayableCounter {
    uint256 public number;

    function setNumber(uint256 newNumber) public {
        number = newNumber;
    }

    fallback() external payable {}

    receive() external payable {}
}
