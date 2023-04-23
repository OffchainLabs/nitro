// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract ProgramTest {
    event Hash(bytes32 result);

    function callKeccak(address program, bytes calldata data) external {
        // in keccak.rs
        //     the input is the # of hashings followed by a preimage
        //     the output is the iterated hash of the preimage
        (bool success, bytes memory result) = address(program).call(data);
        require(success, "call failed");
        bytes32 hash = bytes32(result);
        emit Hash(hash);
        require(hash == keccak256(data[1:]));
    }

    function staticcallProgram(address program, bytes calldata data)
        external
        view
        returns (bytes memory)
    {
        (bool success, bytes memory result) = address(program).staticcall(data);
        require(success, "call failed");
        return result;
    }

    function checkRevertData(
        address program,
        bytes calldata data,
        bytes calldata expected
    ) external payable returns (bytes memory) {
        (bool success, bytes memory result) = address(program).call{value: msg.value}(data);
        require(!success, "unexpected success");
        require(result.length == expected.length, "wrong revert data length");
        for (uint256 i = 0; i < result.length; i++) {
            require(result[i] == expected[i], "revert data mismatch");
        }
        return result;
    }

    function fillBlock() external payable {
        bytes memory prefix = "\x19Ethereum Signed Message:\n32";
        bytes
            memory message = hex"1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8";
        bytes32 messageHash = keccak256(abi.encodePacked(prefix, message));
        address recovered = 0xdD4c825203f97984e7867F11eeCc813A036089D1;
        uint8 v = 28;
        bytes32 r = 0xb7cf302145348387b9e69fde82d8e634a0f8761e78da3bfa059efced97cbed0d;
        bytes32 s = 0x2a66b69167cafe0ccfc726aec6ee393fea3cf0e4f3f9c394705e0f56d9bfe1c9;
        while (true) {
            require(ecrecover(messageHash, v, r, s) == recovered);
        }
    }
}
