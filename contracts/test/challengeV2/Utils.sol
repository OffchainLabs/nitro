// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

contract Random {
    bytes32 private seed = 0xf19f64ef5b8c788ff3f087b4f75bc6596a6aaa3c9048bbbbe990fa0870261385;

    function hash() public returns (bytes32) {
        seed = keccak256(abi.encodePacked(seed));
        return seed;
    }

    function addr() public returns (address) {
        seed = keccak256(abi.encodePacked(seed));
        return address(bytes20(seed));
    }
}
   