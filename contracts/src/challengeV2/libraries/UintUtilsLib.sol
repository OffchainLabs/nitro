// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

/// @title  Uint utils library
/// @notice Some additional bit inspection tools
library UintUtilsLib {
    /// @notice The least significant bit in the bit representation of a uint
    /// @dev    Zero indexed from the least sig bit. Eg 1010 => 1, 1100 => 2, 1001 => 0
    ///         Finds lsb in linear (uint size) time
    /// @param x Cannot be zero, since zero that has no signficant bits
    function leastSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x > 0, "Zero has no significant bits");

        // isolate the least sig bit
        uint256 isolated = ((x - 1) & x) ^ x;
        
        // since we removed all higher bits, least sig == most sig
        return mostSignificantBit(isolated);
    }

    /// @notice The most significant bit in the bit representation of a uint
    /// @dev    Zero indexed from the least sig bit. Eg 1010 => 3, 110 => 2, 1 => 0
    ///         Taken from https://solidity-by-example.org/bitwise/
    ///         Finds msb in log (uint size) time
    /// @param x Cannot be zero, since zero has no sigificant bits
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