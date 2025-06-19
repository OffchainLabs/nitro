// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract StaticCaller {
    uint256 public n;

    function _doStaticCall(address target) internal returns (bool success) {
        bytes memory returnData = new bytes(32);
        bytes memory args = abi.encodeWithSignature("getNumber()");
        uint256 argsSize = args.length;
        assembly {
            success :=
                staticcall(
                    gas(), // Forward all available gas
                    target, // Target contract
                    add(args, 32), // Input data (skip length)
                    argsSize, // Input size
                    add(returnData, 32), // Output location (skip length)
                    32 // Output size (32 bytes for uint256)
                )
        }
        if (success) {
            n = uint256(bytes32(returnData));
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallEmptyWarm(address emptyAddress) public {
        uint256 bal = emptyAddress.balance;
        bool success = _doStaticCall(emptyAddress);
        if (success && bal == 0) {
            n = 0x42;
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallNonEmptyWarm(address nonEmptyAddress) public {
        uint256 bal = nonEmptyAddress.balance;
        bool success = _doStaticCall(nonEmptyAddress);
        if (success && bal == 0) {
            n = 0x42;
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallEmptyCold(address emptyAddress) public {
        _doStaticCall(emptyAddress);
    }

    function testStaticCallNonEmptyCold(address nonEmptyAddress) public {
        _doStaticCall(nonEmptyAddress);
    }

    // for valid non empty static calls, force the memory expansion in the
    // return data
    function _doNonEmptyStaticCallMemExpanded(address target) internal returns (bool success) {
        bytes memory args = abi.encodeWithSignature("getNumber()");
        uint256 argsSize = args.length;
        assembly {
            // do NOT allocate memory for the return data
            // instead send a return pointer that is way past the end of memory
            let returnPtr := add(msize(), 0x20)
            let gasLeft := gas()
            // Make the static call
            success := staticcall(gasLeft, target, add(args, 0x20), argsSize, returnPtr, 32)
        }
        if (success) {
            n = 1;
        } else {
            n = 0x1337;
        }
    }

    // for invalid empty static calls, since the contract does not have code to call
    // there will be no return data so we have to force the memory expansion on the args
    function _doEmptyStaticCallMemExpanded(address target) internal returns (bool success) {
        bytes memory returnData;
        assembly {
            // do NOT allocate memory for the return data
            // instead send a return pointer that is way past the end of memory
            let ptr := add(msize(), 0x20)
            let gasLeft := gas()
            // Make the static call
            success := staticcall(gasLeft, target, ptr, 4, ptr, 32)
            // Update the free memory pointer
            // We need space for success flag (32 bytes) + return data (32 bytes)
            mstore(0x40, add(ptr, 0x20))
            if success { returnData := mload(ptr) }
        }
        if (success) {
            n = uint256(bytes32(returnData));
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallEmptyWarmMemExpansion(address emptyAddress) public {
        uint256 bal = emptyAddress.balance;
        bool success = _doEmptyStaticCallMemExpanded(emptyAddress);
        if (success && bal == 0) {
            n = 0x42;
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallNonEmptyWarmMemExpansion(address nonEmptyAddress) public {
        uint256 bal = nonEmptyAddress.balance;
        bool success = _doNonEmptyStaticCallMemExpanded(nonEmptyAddress);
        if (success && bal == 0) {
            n = 0x42;
        } else {
            n = 0x1337;
        }
    }

    function testStaticCallEmptyColdMemExpansion(address emptyAddress) public {
        _doEmptyStaticCallMemExpanded(emptyAddress);
    }

    function testStaticCallNonEmptyColdMemExpansion(address nonEmptyAddress) public {
        _doNonEmptyStaticCallMemExpanded(nonEmptyAddress);
    }
}

contract StaticCallee {
    uint256 private number;

    constructor() {
        number = 0x414243;
    }

    function getNumber() public view returns (uint256) {
        return number;
    }
}
