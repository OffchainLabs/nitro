// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./Value.sol";

struct ErrorGuard {
    bytes32 frameStack;
    bytes32 valueStack;
    Value onErrorPc;
}

struct GuardStack {
    ErrorGuard[] proved;
    bytes32 remainingHash;
}

library GuardStackLib {
    using ValueLib for Value;

    function newErrorGuard(
        bytes32 frameStack,
        bytes32 valueStack,
        Value memory onErrorPc
    ) internal pure returns (ErrorGuard memory) {
        return ErrorGuard({frameStack: frameStack, valueStack: valueStack, onErrorPc: onErrorPc});
    }

    function hash(ErrorGuard memory guard) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(
                    "Error guard:",
                    guard.frameStack,
                    guard.valueStack,
                    guard.onErrorPc.hash()
                )
            );
    }

    function hash(GuardStack memory guards) internal pure returns (bytes32 h) {
        h = guards.remainingHash;
        for (uint256 i = 0; i < guards.proved.length; i++) {
            h = keccak256(abi.encodePacked("Guard stack:", hash(guards.proved[i]), h));
        }
    }

    function empty(GuardStack memory guards) internal pure returns (bool) {
        return guards.proved.length == 0 && guards.remainingHash == 0;
    }

    function peek(GuardStack memory guards) internal pure returns (ErrorGuard memory) {
        require(guards.proved.length == 1, "BAD_GUARDS_LENGTH");
        return guards.proved[0];
    }

    function pop(GuardStack memory guards) internal pure returns (ErrorGuard memory frame) {
        require(guards.proved.length == 1, "BAD_GUARDS_LENGTH");
        frame = guards.proved[0];
        guards.proved = new ErrorGuard[](0);
    }

    function push(GuardStack memory guards, ErrorGuard memory guard) internal pure {
        ErrorGuard[] memory newProved = new ErrorGuard[](guards.proved.length + 1);
        for (uint256 i = 0; i < guards.proved.length; i++) {
            newProved[i] = guards.proved[i];
        }
        newProved[guards.proved.length] = guard;
        guards.proved = newProved;
    }
}
