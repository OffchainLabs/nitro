// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

struct PcArray {
    uint32[] inner;
}

library PcArrayLib {
    function get(PcArray memory arr, uint256 index) internal pure returns (uint32) {
        return arr.inner[index];
    }

    function set(
        PcArray memory arr,
        uint256 index,
        uint32 val
    ) internal pure {
        arr.inner[index] = val;
    }

    function length(PcArray memory arr) internal pure returns (uint256) {
        return arr.inner.length;
    }

    function push(PcArray memory arr, uint32 val) internal pure {
        uint32[] memory newInner = new uint32[](arr.inner.length + 1);
        for (uint256 i = 0; i < arr.inner.length; i++) {
            newInner[i] = arr.inner[i];
        }
        newInner[arr.inner.length] = val;
        arr.inner = newInner;
    }

    function pop(PcArray memory arr) internal pure returns (uint32 popped) {
        popped = arr.inner[arr.inner.length - 1];
        uint32[] memory newInner = new uint32[](arr.inner.length - 1);
        for (uint256 i = 0; i < newInner.length; i++) {
            newInner[i] = arr.inner[i];
        }
        arr.inner = newInner;
    }
}
