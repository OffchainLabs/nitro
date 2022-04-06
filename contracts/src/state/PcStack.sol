// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./PcArray.sol";

struct PcStack {
    PcArray proved;
    bytes32 remainingHash;
}

library PcStackLib {
    using PcArrayLib for PcArray;

    function hash(PcStack memory stack) internal pure returns (bytes32 h) {
        h = stack.remainingHash;
        uint256 len = stack.proved.length();
        for (uint256 i = 0; i < len; i++) {
            h = keccak256(abi.encodePacked("Program counter stack:", stack.proved.get(i), h));
        }
    }

    function pop(PcStack memory stack) internal pure returns (uint32) {
        return stack.proved.pop();
    }

    function push(PcStack memory stack, uint32 val) internal pure {
        return stack.proved.push(val);
    }
}
