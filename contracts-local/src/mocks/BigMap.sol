// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract BigMap {
    mapping(uint256 => uint256) public data;
    uint256 size;

    function clearAndAddValues(uint256 clear, uint256 add) external {
        uint256 i = size;
        while (i < size + add) {
            data[i] = 8675309;
            i++;
        }
        size = i;
        for (uint256 j = 0; j < clear; j++) {
            data[j] = 0;
        }
    }
}
