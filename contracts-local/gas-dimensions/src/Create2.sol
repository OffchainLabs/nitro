// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Createe} from "./Create.sol";

contract CreatorTwo {
    receive() external payable {}

    function _doCreateTwoMemUnchanged(uint256 amount, uint256 arg, bytes32 salt) internal returns (address) {
        Createe createe = new Createe{value: amount, salt: salt}(arg);
        return address(createe);

    }

    // arg doesn't matter here because we don't actually call the constructor LOL
    function _doCreateTwoMemExpansion(uint256 amount, bytes32 salt) internal {
        bytes memory code = type(Createe).creationCode;
        assembly {
            let m := msize()
            let offsetDiff := add(sub(m, code), 0x20)
            let unused :=create2(amount, add(code, 0x20), offsetDiff, salt)
            pop(unused)
        }
    }

    function createTwoNoTransferMemUnchanged(bytes32 _salt) public returns (address) {
        return _doCreateTwoMemUnchanged(0x0, 0x42, _salt);
    }

    function createTwoNoTransferMemExpansion(bytes32 _salt) public {
        _doCreateTwoMemExpansion(0x0, _salt);
    }

    function createTwoPayableMemUnchanged(bytes32 _salt) public payable returns (address) {
        return _doCreateTwoMemUnchanged(0x69420, 0x42, _salt);
    }

    function createTwoPayableMemExpansion(bytes32 _salt) public payable {
        _doCreateTwoMemExpansion(0x42069, _salt);
    }
}

