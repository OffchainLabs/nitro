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

    function staticcallEvmData(
        address program,
        uint64 gas,
        bytes calldata data
    ) external view returns (bytes memory) {
        (bool success, bytes memory result) = address(program).staticcall{gas: gas}(data);
        require(success, "call failed");

        bytes32 selectedBlockNumber;
        bytes32 foundBlockhash;
        assembly {
            selectedBlockNumber := mload(add(add(result, 0), 32))
            foundBlockhash := mload(add(add(result, 32), 32))
        }
        require(foundBlockhash == blockhash(uint256(selectedBlockNumber)), "unexpected blockhash");

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
        bytes32 bridgeToNova = 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
        address cryptoIsCute = 0x361594F5429D23ECE0A88E4fBE529E1c49D524d8;
        uint8 v = 27;
        bytes32 r = 0xc6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f;
        bytes32 s = 0x5fdbcefe2675e96219cdae57a7894280bf80fd40d44ce146a35e169ea6a78fd3;
        while (true) {
            require(ecrecover(bridgeToNova, v, r, s) == cryptoIsCute, "WRONG_ARBINAUT");
        }
    }
}
