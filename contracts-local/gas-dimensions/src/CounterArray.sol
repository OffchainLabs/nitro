// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract CounterArray {
    uint256[] public counters;

    // create the array
    constructor() {
        counters = new uint256[](20);
    }

    // alllow caller to set data in the array
    function setCounters(uint256[] memory newCounters) public {
        require(newCounters.length == 20, "Array must be 20 elements long");
        for (uint256 i = 0; i < 20; i++) {
            counters[i] = newCounters[i];
        }
    }

    function refund1() public {
        assembly {
            let slot := counters.slot
            let slotKey := keccak256(slot, 0x20)
            let data := sload(slotKey)
            sstore(slotKey, 0)
        }
    }

    function getSlotKey(uint256 index) public pure returns (bytes32) {
        // Declare variable in Solidity scope
        bytes32 elementSlot;
        assembly {
            let slot := counters.slot
            let slotKey := keccak256(slot, 0x20)
            // Assign to the Solidity variable using :=
            elementSlot := add(slotKey, index)
        }
        return elementSlot;
    }

    function refundFromCalldata(bytes32 slotKey) public {
        assembly {
            sstore(slotKey, 0)
        }
    }

    // manually SSTORE to zero out the storage
    // unrolled loop assuming 20 elements
    function refunder() public {
        assembly {
            // Get the slot of the counters array
            let slot := counters.slot
            //
            let slotKey := keccak256(slot, 0x20)
            // Zero out the storage slot
            sstore(slotKey, 0)
            let elementSlot := add(slotKey, 1)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 2)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 3)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 4)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 5)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 6)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 7)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 8)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 9)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 10)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 11)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 12)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 13)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 14)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 15)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 16)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 17)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 18)
            // Zero out the storage slot
            sstore(elementSlot, 0)
            elementSlot := add(slotKey, 19)
            // Zero out the storage slot
            sstore(elementSlot, 0)
        }
    }
}
