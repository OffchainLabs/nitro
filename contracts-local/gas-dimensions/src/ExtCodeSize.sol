// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract ExtCodeSizeCallee {
    uint256 public number;

    function setNumber(uint256 _number) public {
        number = _number;
    }
}

contract ExtCodeSize {
    ExtCodeSizeCallee callee;
    bool q;

    constructor() {
        callee = new ExtCodeSizeCallee();
    }

    function getExtCodeSizeCold() public returns (uint256) {
        uint256 ret = address(callee).code.length;
        q = true;
        return ret;
    }

    function getExtCodeSizeWarm() public returns (uint256) {
        (bool success,) =
            address(callee).call{value: 0}(abi.encodeWithSelector(ExtCodeSizeCallee.setNumber.selector, 0x1337));
        if (success) {
            q = true;
        }
        return address(callee).code.length;
    }
}
