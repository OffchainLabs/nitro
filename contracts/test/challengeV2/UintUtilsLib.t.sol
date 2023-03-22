// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/libraries/UintUtilsLib.sol";
import "./Utils.sol";

contract UintUtilsLibTest is Test {
    Random random = new Random();

    function testLsbZero() public {
        vm.expectRevert("Zero has no significant bits");
        UintUtilsLib.leastSignificantBit(0);
    }

    function testLsb() public {
        assertEq(UintUtilsLib.leastSignificantBit(1), 0); // 1
        assertEq(UintUtilsLib.leastSignificantBit(2), 1); // 10
        assertEq(UintUtilsLib.leastSignificantBit(3), 0); // 11
        assertEq(UintUtilsLib.leastSignificantBit(4), 2); // 100
        assertEq(UintUtilsLib.leastSignificantBit(5), 0); // 101
        assertEq(UintUtilsLib.leastSignificantBit(6), 1); // 110
        assertEq(UintUtilsLib.leastSignificantBit(7), 0); // 111
        assertEq(UintUtilsLib.leastSignificantBit(8), 3); // 1000
        assertEq(UintUtilsLib.leastSignificantBit(10), 1); // 1010
        assertEq(UintUtilsLib.leastSignificantBit(696320), 13); // 10101010000000000000
        assertEq(UintUtilsLib.leastSignificantBit(696321), 0); // 10101010000000000001
        assertEq(UintUtilsLib.leastSignificantBit(236945758459398306981350710526416285671374848), 14); // 1010101000000000000100000000010101010100000000000010101000000000000000001010100000000000010101001000101010000000000000000010100001000100000000000000
        assertEq(UintUtilsLib.leastSignificantBit(type(uint256).max), 0);
    }

    function testMoreLsb() public {
        uint256 randHash = uint256(random.hash());
        for (uint256 i = 0; i < 256; i++) {
            assertEq(UintUtilsLib.leastSignificantBit(1 << i), i);
            assertEq(UintUtilsLib.leastSignificantBit(type(uint256).max << i), i);
            assertEq(UintUtilsLib.leastSignificantBit((randHash | 1) << i), i);
        }
    }

    function testMsbZero() public {
        vm.expectRevert("Zero has no significant bits");
        UintUtilsLib.mostSignificantBit(0);
    }

    function testMsb() public {
        assertEq(UintUtilsLib.mostSignificantBit(1), 0); // 1
        assertEq(UintUtilsLib.mostSignificantBit(2), 1); // 10
        assertEq(UintUtilsLib.mostSignificantBit(3), 1); // 11
        assertEq(UintUtilsLib.mostSignificantBit(4), 2); // 100
        assertEq(UintUtilsLib.mostSignificantBit(5), 2); // 101
        assertEq(UintUtilsLib.mostSignificantBit(6), 2); // 110
        assertEq(UintUtilsLib.mostSignificantBit(7), 2); // 111
        assertEq(UintUtilsLib.mostSignificantBit(8), 3); // 1000
        assertEq(UintUtilsLib.mostSignificantBit(10), 3); // 1010
        assertEq(UintUtilsLib.mostSignificantBit(696320), 19); // 10101010000000000000
        assertEq(UintUtilsLib.mostSignificantBit(696321), 19); // 10101010000000000001
        assertEq(UintUtilsLib.mostSignificantBit(236945758459398306981350710526416285671374848), 147); // 1010101000000000000100000000010101010100000000000010101000000000000000001010100000000000010101001000101010000000000000000010100001000100000000000000
        assertEq(UintUtilsLib.mostSignificantBit(type(uint256).max), 255);
    }

    function testMoreMsb() public {
        uint256 randHash = uint256(random.hash());
        for (uint256 i = 0; i < 256; i++) {
            assertEq(UintUtilsLib.mostSignificantBit(1 << i), i);
            assertEq(UintUtilsLib.mostSignificantBit(type(uint256).max >> i), 255 - i);
            assertEq(UintUtilsLib.mostSignificantBit((randHash | (1 << 255)) >> i), 255 - i);
        }
    }
}
