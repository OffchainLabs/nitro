// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "../src/AssertionChain.sol";

contract AssertionChainTest is Test {
    AssertionChain public chain;

    function setUp() public {
        chain = new AssertionChain();
    }
}
