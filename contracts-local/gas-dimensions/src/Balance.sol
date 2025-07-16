// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract Balance {
    uint256 public number;

    function callBalanceCold() public {
        uint256 thisBalance = address(this).balance;
        uint256 beefbalance = address(0xdeadbeef).balance;
        number = thisBalance + beefbalance;
    }

    function callBalanceWarm() public {
        address target = address(0xdeadbeef);
        (bool success,) = target.call{value: 0}("");
        if (success) {
            number = 2;
        }
        uint256 beefbalance2 = target.balance;
        number = number + beefbalance2;
    }
}
