// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract CallCoder {
    uint256 public n;

    function _doCallCodeMemUnchanged(address target, uint256 transferValue) internal {
        n = 1;
        bytes memory args = abi.encodeWithSelector(CallCodee.setNumber.selector, 0x1337);
        uint256 argsSize = args.length;
        uint256 returnValue;
        bool success;
        assembly {
            success := callcode(gas(), target, transferValue, add(args, 0x20), argsSize, returnValue, 32)
        }
        if (success) n = 2;
    }

    function _doCallCodeWithMemExpansion(address target, uint256 transferValue) internal {
        n = 1;
        bytes memory args = abi.encodeWithSelector(CallCodee.setNumber.selector, 0x1337);
        uint256 argsSize = args.length;
        bool success;
        assembly {
            let returnPtr := add(msize(), 0x20)
            success := callcode(gas(), target, transferValue, add(args, 0x20), argsSize, returnPtr, 32)
        }
        if (success) n = 2;
    }

    function warmNoTransferMemUnchanged(address target) public {
        uint256 currentValue = target.balance;
        _doCallCodeMemUnchanged(target, 0);
        if (currentValue == target.balance) n = 3;
    }

    function warmPayableMemUnchanged(address target) public {
        n = 2;
        uint256 currentValue = target.balance;
        _doCallCodeMemUnchanged(target, 0xABCD);
        if (currentValue == target.balance) n = 3;
    }

    function warmNoTransferMemExpansion(address target) public {
        uint256 currentValue = target.balance;
        _doCallCodeWithMemExpansion(target, 0);
        if (currentValue == target.balance) n = 3;
    }

    function warmPayableMemExpansion(address target) public {
        uint256 currentValue = target.balance;
        _doCallCodeWithMemExpansion(target, 0xABCD);
        if (currentValue == target.balance) n = 3;
    }

    function coldNoTransferMemUnchanged(address target) public {
        _doCallCodeMemUnchanged(target, 0);
    }

    function coldPayableMemUnchanged(address target) public {
        _doCallCodeMemUnchanged(target, 0xABCD);
    }

    function coldNoTransferMemExpansion(address target) public {
        _doCallCodeWithMemExpansion(target, 0);
    }

    function coldPayableMemExpansion(address target) public {
        _doCallCodeWithMemExpansion(target, 0xABCD);
    }

    receive() external payable {}
}

contract CallCodee {
    uint256 public n;

    function setNumber(uint256 _n) public payable {
        n = _n;
    }

    receive() external payable {}
}
