// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract Createe {
    uint256 public x;

    constructor(uint256 _x) payable {
        x = _x;
    }

    receive() external payable {}
}

contract Creator {
    receive() external payable {}

    function _doCreateMemUnchanged(uint256 amount, uint256 arg) internal returns (address) {
        Createe createe = new Createe{value: amount}(arg);
        return address(createe);

    }

    // arg doesn't matter here because we don't actually call the constructor LOL
    function _doCreateMemExpansion(uint256 amount) internal {
        bytes memory code = type(Createe).creationCode;
        assembly {
            let m := msize()
            let offsetDiff := add(sub(m, code), 0x20)
            let unused :=create(amount, add(code, 0x20), offsetDiff)
            pop(unused)
        }
    }

    function createNoTransferMemUnchanged() public returns (address) {
        return _doCreateMemUnchanged(0x0, 0x42);
    }

    function createNoTransferMemExpansion() public {
        _doCreateMemExpansion(0x0);
    }

    function createPayableMemUnchanged() public payable returns (address) {
        return _doCreateMemUnchanged(0x69420, 0x42);
    }

    function createPayableMemExpansion() public payable {
        _doCreateMemExpansion(0x69420);
    }
}
