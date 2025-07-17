// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract Sload {
    uint256 public a;

    constructor() {
        a = 3;
    }

    function warmSload() public returns (uint256) {
        a = 4;
        uint256 b = a;
        return b;
    }

    function coldSload() public returns (uint256) {
        uint256 b = a;
        a = 5;
        return b;
    }
}
