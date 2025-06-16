// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract OutOfGas {
    uint256 public x;
    bool public gasErrorOccurred;

    function outOfGas() public {
        while (true) {
            // This loop will run forever, consuming all gas
            x++;
        }
    }

    function callOutOfGas() public {
        gasErrorOccurred = false;
        try this.outOfGas{gas: 100000}() {
            // This block will never execute because outOfGas always fails
            gasErrorOccurred = false;
        } catch {
            // This block will execute when outOfGas runs out of gas
            gasErrorOccurred = true;
        }
    }
}
