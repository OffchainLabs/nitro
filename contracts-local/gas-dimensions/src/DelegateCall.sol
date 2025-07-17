// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract DelegateCaller {
    uint256 public value;

    function testDelegateCallEmptyWarm(address emptyAddress) public {
        // Warm up the address in the access list by accessing its balance
        uint256 balance = emptyAddress.balance;
        // Perform delegatecall to empty address
        (bool success,) = emptyAddress.delegatecall("");
        if (success && balance > 0) {
            value = 4;
        }
    }

    function testDelegateCallNonEmptyWarm(address nonEmptyAddress) public {
        // Warm up the address in the access list by accessing its balance
        uint256 balance = nonEmptyAddress.balance;
        // Perform delegatecall to non-empty address
        (bool success,) = nonEmptyAddress.delegatecall(abi.encodeWithSignature("setValue(uint256)", 42));
        if (success && balance > 0) {
            value = 4;
        }
    }

    function testDelegateCallEmptyCold(address emptyAddress) public {
        // Perform delegatecall to empty address
        (bool success,) = emptyAddress.delegatecall("");
        if (success) {
            value = 4;
        }
    }

    function testDelegateCallNonEmptyCold(address nonEmptyAddress) public {
        // Perform delegatecall to non-empty address
        (bool success,) = nonEmptyAddress.delegatecall(abi.encodeWithSignature("setValue(uint256)", 42));
        if (success) {
            value = 4;
        }
    }

    function testDelegateCallEmptyWarmMemExpansion(address emptyAddress) public {
        // Warm up the address in the access list by accessing its balance
        uint256 balance = emptyAddress.balance;
        bool success;
        assembly {
            let ptr := msize()
            let gasLeft := gas()
            success := delegatecall(gasLeft, emptyAddress, ptr, 0x40, ptr, 0x40)
        }
        if (success && balance == 0) {
            value = 3;
        } else {
            value = 4;
        }
    }

    function testDelegateCallNonEmptyWarmMemExpansion(address nonEmptyAddress) public {
        // Warm up the address in the access list by accessing its balance
        uint256 balance = nonEmptyAddress.balance;
        bytes memory args = abi.encodeWithSignature("setValue(uint256)", 43);
        uint256 argsSize = args.length;
        bool success;
        assembly {
            let ptr := msize()
            let gasLeft := gas()
            success := delegatecall(gasLeft, nonEmptyAddress, add(args, 0x20), argsSize, ptr, 0x40)
        }
        if (success && balance == 0) {
            value = 3;
        } else {
            value = 4;
        }
    }

    function testDelegateCallEmptyColdMemExpansion(address emptyAddress) public {
        bool success;
        assembly {
            let ptr := msize()
            let gasLeft := gas()
            success := delegatecall(gasLeft, emptyAddress, ptr, 0x40, ptr, 0x40)
        }
        if (success) {
            value = 3;
        } else {
            value = 4;
        }
    }

    function testDelegateCallNonEmptyColdMemExpansion(address nonEmptyAddress) public {
        bytes memory args = abi.encodeWithSignature("setValue(uint256)", 42);
        uint256 argsSize = args.length;
        bool success;
        assembly {
            let ptr := msize()
            let gasLeft := gas()
            success := delegatecall(gasLeft, nonEmptyAddress, add(args, 0x20), argsSize, ptr, 0x40)
        }
        if (success) {
            value = 4;
        }
    }
}

contract DelegateCallee {
    // Storage layout must match the calling contract for delegatecall to work as expected
    uint256 public value;

    /**
     * @dev Set the value in storage
     * When called via delegatecall, this will modify the storage of the calling contract
     */
    function setValue(uint256 _value) public returns (bool) {
        value = _value;
        return true;
    }
}
