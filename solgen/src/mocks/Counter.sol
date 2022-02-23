//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

contract Simple {
    uint64 public counter;

    function increment() external {
        counter++;
    }
}
