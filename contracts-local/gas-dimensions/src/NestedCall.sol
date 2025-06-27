// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract NestedCall {
    uint256 public mainValue;
    uint256 public nestedValue;
    uint256 public callCount;

    // Entrypoint function
    function entrypoint(address target) public {
        mainValue = 0x0FFC13A1171AB5;
        callCount++;
        this.intermediateCall(target);
    }

    // Intermediate function that does a DELEGATECALL inside it
    function intermediateCall(address target) public {
        mainValue = 0x0A4B1CAFE;
        (bool success,) = target.delegatecall(
            abi.encodeWithSelector(NestedTarget.performNestedAction.selector)
        );
        require(success, "Delegatecall failed");
    }
}

contract NestedTarget {
    function performNestedAction() public {
        // Need to access the storage slot directly 
        // since we're in delegate called contract context
        assembly {
            sstore(0x1, 0xA4B1) // slot 1 for nestedValue
        }
    }
}
