// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/libraries/ArrayUtilsLib.sol";
import "./Utils.sol";

contract ArrayUtilsLibTest is Test {
    Random random = new Random();

    function areEqual(bytes32[] memory a, bytes32[] memory b) internal {
        assertEq(a.length, b.length, "Len unequal");

        for (uint i = 0; i < a.length; i++) {
            assertEq(a[i], b[i]);
        }
    }

    function testAppendSingle() public {
        bytes32 r = random.hash();
        bytes32[] memory e = new bytes32[](1);
        e[0] = r;

        bytes32[] memory n = new bytes32[](0);
        bytes32[] memory n2 = ArrayUtilsLib.append(n, r);
        areEqual(n2, e);
    }

    function testAppend() public {
        bytes32[] memory o = random.hashes(3);
        bytes32[] memory expected = new bytes32[](4);
        expected[0] = o[0];
        expected[1] = o[1];
        expected[2] = o[2];
        bytes32 n = random.hash();
        expected[3] = n;
        bytes32[] memory actual = ArrayUtilsLib.append(o, n);

        areEqual(actual, expected);
        assertEq(actual.length, 4, "len 4");

        bytes32[] memory expected2 = new bytes32[](5);
        expected2[0] = o[0];
        expected2[1] = o[1];
        expected2[2] = o[2];
        expected2[3] = n;
        bytes32 n2 = random.hash();
        expected2[4] = n2;
        bytes32[] memory actual2 = ArrayUtilsLib.append(actual, n2);

        areEqual(actual2, expected2);
        assertEq(actual2.length, 5, "len 5");
    }

    function testSliceAll() public {
        bytes32[] memory o = random.hashes(5);
        
        bytes32[] memory s = ArrayUtilsLib.slice(o, 0, o.length);
        areEqual(o, s);
    }

    function testSliceStart() public {
        bytes32[] memory o = random.hashes(5);
        bytes32[] memory s = ArrayUtilsLib.slice(o, 0, 3);

        bytes32[] memory e = new bytes32[](3);
        e[0] = o[0];
        e[1] = o[1];
        e[2] = o[2];
        areEqual(e, s);
    }

    function testSliceEnd() public {
        bytes32[] memory o = random.hashes(5);
        bytes32[] memory s = ArrayUtilsLib.slice(o, 3, 5);

        bytes32[] memory e = new bytes32[](2);
        e[0] = o[3];
        e[1] = o[4];
        areEqual(e, s);
    }

    function testSliceMiddle() public {
        bytes32[] memory o = random.hashes(5);
        bytes32[] memory s = ArrayUtilsLib.slice(o, 2, 4);

        bytes32[] memory e = new bytes32[](2);
        e[0] = o[2];
        e[1] = o[3];
        areEqual(e, s);
    }

    function testSliceOutOfBoundStart() public {
        bytes32[] memory o = random.hashes(5);
        vm.expectRevert("End not less or equal than length");
        ArrayUtilsLib.slice(o, 5, 6);
    }

    function testSliceOutOfBoundEnd() public {
        bytes32[] memory o = random.hashes(5);
        vm.expectRevert("End not less or equal than length");
        ArrayUtilsLib.slice(o, 3, 6);
    }

    function testSliceStartGtEnd() public {
        bytes32[] memory o = random.hashes(5);
        vm.expectRevert("Start not less than end");
        ArrayUtilsLib.slice(o, 3, 3);
    }

    function testConcat() public {
        bytes32[] memory o = random.hashes(2);
        bytes32[] memory o2 = random.hashes(3);
        bytes32[] memory expected = new bytes32[](5);
        expected[0] = o[0];
        expected[1] = o[1];
        expected[2] = o2[0];
        expected[3] = o2[1];
        expected[4] = o2[2];

        bytes32[] memory r = ArrayUtilsLib.concat(o, o2);
        areEqual(r, expected);
    }

    function testConcatEmpty() public {
        bytes32[] memory o = random.hashes(2);
        bytes32[] memory o2 = new bytes32[](0);

        bytes32[] memory r = ArrayUtilsLib.concat(o, o2);
        areEqual(r, o);

        bytes32[] memory r2 = ArrayUtilsLib.concat(o2, o);
        areEqual(r2, o);

        bytes32[] memory r3 = ArrayUtilsLib.concat(o2, o2);
        areEqual(r3, o2);
    }
}
