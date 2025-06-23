// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract Counter {
    uint256 public number;

    function setNumber(uint256 newNumber) public {
        number = newNumber;
    }

    function increment() public {
        uint256 newNumber = number; // make extra sure to cause a cold SLOAD
        newNumber = newNumber + 1;
        uint256 oldNumber = number; // make sure to cause a warm SLOAD
        number = newNumber + oldNumber;
    }

    function noSpecials() public {
        assembly {
            let x := add(0x1337, 0x6969)
            return(x, 0x20)
        }
    }
}
