// Copyright 2022-2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../precompiles/ArbSys.sol";

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

    function staticcallProgram(
        address program,
        bytes calldata data
    ) external view returns (bytes memory) {
        (bool success, bytes memory result) = address(program).staticcall(data);
        require(success, "call failed");
        return result;
    }

    function assert256(
        bytes memory data,
        string memory text,
        uint256 expected
    ) internal pure returns (bytes memory) {
        uint256 value = abi.decode(data, (uint256));
        require(value == expected, text);

        bytes memory rest = new bytes(data.length - 32);
        for (uint256 i = 32; i < data.length; i++) {
            rest[i - 32] = data[i];
        }
        return rest;
    }

    function staticcallEvmData(
        address program,
        address fundedAccount,
        uint64 gas,
        bytes calldata data
    ) external view returns (bytes memory) {
        (bool success, bytes memory result) = address(program).staticcall{gas: gas}(data);
        require(success, "call failed");

        address arbPrecompile = address(0x69);
        address ethPrecompile = address(0x01);

        result = assert256(result, "block number ", block.number - 1);
        result = assert256(result, "chain id     ", block.chainid);
        result = assert256(result, "base fee     ", block.basefee);
        result = assert256(result, "gas price    ", tx.gasprice);
        result = assert256(result, "gas limit    ", block.gaslimit);
        result = assert256(result, "value        ", 0);
        result = assert256(result, "timestamp    ", block.timestamp);
        result = assert256(result, "balance      ", fundedAccount.balance);
        result = assert256(result, "rust address ", uint256(uint160(program)));
        result = assert256(result, "sender       ", uint256(uint160(address(this))));
        result = assert256(result, "origin       ", uint256(uint160(tx.origin)));
        result = assert256(result, "coinbase     ", uint256(uint160(address(block.coinbase))));
        result = assert256(result, "rust codehash", uint256(program.codehash));
        result = assert256(result, "arb codehash ", uint256(arbPrecompile.codehash));
        result = assert256(result, "eth codehash ", uint256(ethPrecompile.codehash));

        bytes memory code = new bytes(program.code.length);
        for (uint256 i = 0; i < program.code.length; i++) {
            code[i] = result[i];
        }
        require(keccak256(code) == keccak256(program.code), "code");
        bytes memory rest = new bytes(result.length - program.code.length);
        for (uint256 i = program.code.length; i < result.length; i++) {
            rest[i - program.code.length] = result[i];
        }

        result = rest;
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

    function mathTest(
        address program
    ) external {
        uint256 value = 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
        value = mulmod(
            value,
            0xc6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f,
            0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f
        );
        value = addmod(
            value,
            0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f,
            0xc6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f
        );
        unchecked {
            value /= 0xeddecf107b5740ce;
            value = value ** 0xfffffffefffffc2f;
            value = value % 0xc6178c2de1078cd3;
        }

        (bool success, bytes memory result) = address(program).call("");
        require(success, "call failed");
        require(keccak256(result) == keccak256(abi.encodePacked(value)));
    }
}
