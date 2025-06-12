// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

/*
 * This contract is the solidity equivalent of the stylus create test contract.
 */
contract CreateTest {
    // solhint-disable no-complex-fallback
    // solhint-disable reason-string
    // solhint-disable avoid-low-level-calls
    // solhint-disable-next-line prettier/prettier
    fallback(
        bytes calldata input
    ) external returns (bytes memory) {
        uint8 kind = uint8(input[0]);
        input = input[1:];

        bytes32 endowment = bytes32(input[:32]);
        input = input[32:];

        address addr;

        if (kind == 2) {
            bytes32 salt = bytes32(input[:32]);
            input = input[32:];
            bytes memory code = input;
            assembly {
                addr := create2(endowment, add(code, 32), mload(code), salt)
            }
        } else {
            bytes memory code = input;
            assembly {
                addr := create(endowment, add(code, 32), mload(code))
            }
        }
        if (addr == address(0)) {
            revert("failed to create");
        }
        return addr.code;
    }
}
