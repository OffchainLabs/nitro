// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../precompiles/ArbWasm.sol";

contract ProgramTest {
    event Hash(bytes32 result);

    function callKeccak(address program, bytes calldata data) external {
        // in keccak.rs
        //     the input is the # of hashings followed by a preimage
        //     the output is the iterated hash of the preimage

        bytes memory result = ArbWasm(address(0x71)).callProgram(program, data);
        bytes32 hash = bytes32(result);
        emit Hash(hash);
        require(hash == keccak256(data[1:]));
    }
}
