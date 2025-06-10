// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

contract PayableCounter {
    uint256 public number;

    function setNumber(uint256 newNumber) public {
        number = newNumber;
    }

    fallback() external payable {}

    receive() external payable {}
}
