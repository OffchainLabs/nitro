// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

// CHRIS: TODO: docs and tests

library UintUtilsLib {
    // Find the index (from least sig) of the least significant bit
    function leastSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x > 0, "Zero has no significant bits");

        uint256 i = 0;
        while ((x <<= 1) != 0) {
            ++i;
        }
        return 256 - i - 1;
    }

    // take from https://solidity-by-example.org/bitwise/
    // Find the index (from least sig) of the most significant bit using binary search
    function mostSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x != 0, "Zero has no significant bits");

        // x >= 2 ** 128
        if (x >= 0x100000000000000000000000000000000) {
            x >>= 128;
            msb += 128;
        }
        // x >= 2 ** 64
        if (x >= 0x10000000000000000) {
            x >>= 64;
            msb += 64;
        }
        // x >= 2 ** 32
        if (x >= 0x100000000) {
            x >>= 32;
            msb += 32;
        }
        // x >= 2 ** 16
        if (x >= 0x10000) {
            x >>= 16;
            msb += 16;
        }
        // x >= 2 ** 8
        if (x >= 0x100) {
            x >>= 8;
            msb += 8;
        }
        // x >= 2 ** 4
        if (x >= 0x10) {
            x >>= 4;
            msb += 4;
        }
        // x >= 2 ** 2
        if (x >= 0x4) {
            x >>= 2;
            msb += 2;
        }
        // x >= 2 ** 1
        if (x >= 0x2) msb += 1;
    }
}