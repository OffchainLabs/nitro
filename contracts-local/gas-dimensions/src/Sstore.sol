// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract Sstore {
    uint256 public zeroFromStart;
    uint256 public nonZeroFromStart;

    constructor() {
        nonZeroFromStart = 2;
    }

    function sstoreColdZeroToZero() public {
        zeroFromStart = 0;
    }

    function sstoreColdZeroToNonZero() public {
        zeroFromStart = 1;
    }

    function sstoreColdNonZeroValueToZero() public {
        nonZeroFromStart = 0;
    }

    function sstoreColdNonZeroToSameNonZeroValue() public {
        nonZeroFromStart = 2;
    }

    function sstoreColdNonZeroToDifferentNonZeroValue() public {
        nonZeroFromStart = 3;
    }

    function sstoreWarmZeroToZero() public returns (uint256) {
        uint256 x = zeroFromStart;
        zeroFromStart = 0;
        return x;
    }

    function sstoreWarmZeroToNonZeroValue() public returns (uint256) {
        uint256 x = zeroFromStart;
        zeroFromStart = 1;
        return x;
    }

    function sstoreWarmNonZeroValueToZero() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 0;
        return x;
    }

    function sstoreWarmNonZeroToSameNonZeroValue() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 2;
        return x;
    }

    function sstoreWarmNonZeroToDifferentNonZeroValue() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 3;
        return x;
    }

    function sstoreMultipleWarmNonZeroToNonZeroToNonZero() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 3;
        x = nonZeroFromStart;
        nonZeroFromStart = 4;
        return x;
    }

    function sstoreMultipleWarmNonZeroToNonZeroToSameNonZero() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 3;
        x = nonZeroFromStart;
        nonZeroFromStart = 2;
        return x;
    }

    function sstoreMultipleWarmNonZeroToZeroToNonZero() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 0;
        x = nonZeroFromStart;
        nonZeroFromStart = 4;
        return x;
    }

    function sstoreMultipleWarmNonZeroToZeroToSameNonZero() public returns (uint256) {
        uint256 x = nonZeroFromStart;
        nonZeroFromStart = 0;
        x = nonZeroFromStart;
        nonZeroFromStart = 2;
        return x;
    }

    function sstoreMultipleWarmZeroToNonZeroToNonZero() public returns (uint256) {
        uint256 x = zeroFromStart;
        zeroFromStart = 1;
        x = zeroFromStart;
        zeroFromStart = 2;
        return x;
    }

    function sstoreMultipleWarmZeroToNonZeroBackToZero() public returns (uint256) {
        uint256 x = zeroFromStart;
        zeroFromStart = 1;
        x = zeroFromStart;
        zeroFromStart = 0;
        return x;
    }
}
