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
}
