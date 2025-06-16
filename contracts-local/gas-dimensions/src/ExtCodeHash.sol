// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract ExtCodeHashCallee {
    uint256 public number;

    function setNumber(uint256 _number) public {
        number = _number;
    }
}

contract ExtCodeHash {
    ExtCodeHashCallee callee;
    bool q;

    constructor() {
        callee = new ExtCodeHashCallee();
    }

    function getExtCodeHashCold() public returns (bytes32) {
        bytes32 ret = address(callee).codehash;
        q = true;
        return ret;
    }

    function getExtCodeHashWarm() public returns (bytes32) {
        (bool success,) =
            address(callee).call{value: 0}(abi.encodeWithSelector(ExtCodeHashCallee.setNumber.selector, 0x1337));
        if (success) {
            q = true;
        }
        return address(callee).codehash;
    }
}
