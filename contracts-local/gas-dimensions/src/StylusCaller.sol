// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

contract StylusCaller {

    // the keccak wasm program does not follow the standard ABI. 
    // you just send it the preimage directly.
    function callKeccak(address programAddress, bytes memory input) public {
        bytes memory keccakArgs = abi.encodePacked(abi.encodePacked(uint8(0x01)), input);
        programAddress.call(keccakArgs);
    }
}