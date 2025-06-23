// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract Caller {
    uint256 public n;

    function _doCallMemUnchanged(address target, uint256 transferValue) internal {
        n = 1;
        (bool success,) = target.call{value: transferValue}(abi.encodeWithSelector(Callee.setNumber.selector, 0x1337));
        if (success) n = 2;
    }

    function _doCallWithMemExpansion(address target, uint256 transferValue) internal {
        n = 1;
        bytes memory args = abi.encodeWithSelector(Callee.setNumber.selector, 0x1337);
        uint256 argsSize = args.length;
        bool success;
        assembly {
            let returnPtr := add(msize(), 0x20)
            success := call(gas(), target, transferValue, add(args, 0x20), argsSize, returnPtr, 32)
        }
        if (success) n = 2;
    }

    function warmNoTransferMemUnchanged(address target) public {
        uint256 currentValue = target.balance;
        _doCallMemUnchanged(target, 0);
        if (currentValue == target.balance) n = 3;
    }

    function warmPayableMemUnchanged(address target) public {
        n = 2;
        uint256 currentValue = target.balance;
        _doCallMemUnchanged(target, 0xABCD);
        if (currentValue == target.balance) n = 3;
    }

    function warmNoTransferMemExpansion(address target) public {
        uint256 currentValue = target.balance;
        _doCallWithMemExpansion(target, 0);
        if (currentValue == target.balance) n = 3;
    }

    function warmPayableMemExpansion(address target) public {
        uint256 currentValue = target.balance;
        _doCallWithMemExpansion(target, 0xABCD);
        if (currentValue == target.balance) n = 3;
    }

    function coldNoTransferMemUnchanged(address target) public {
        _doCallMemUnchanged(target, 0);
    }

    function coldPayableMemUnchanged(address target) public {
        _doCallMemUnchanged(target, 0xABCD);
    }

    function coldNoTransferMemExpansion(address target) public {
        _doCallWithMemExpansion(target, 0);
    }

    function coldPayableMemExpansion(address target) public {
        _doCallWithMemExpansion(target, 0xABCD);
    }

    receive() external payable {}
}

contract Callee {
    uint256 public number;

    function setNumber(uint256 _number) public payable {
        number = _number;
    }

    receive() external payable {}
}
