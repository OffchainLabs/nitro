// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/ValueArray.sol";

contract ValueArrayTester {
    using ValueArrayLib for ValueArray;

    function test() external pure {
        ValueArray memory arr = ValueArray(new Value[](2));
        require(arr.length() == 2, "START_LEN");
        arr.set(0, ValueLib.newI32(1));
        arr.set(1, ValueLib.newI32(2));
        arr.push(ValueLib.newI32(3));
        require(arr.length() == 3, "PUSH_LEN");
        for (uint256 i = 0; i < arr.length(); i++) {
            Value memory val = arr.get(i);
            require(val.valueType == ValueType.I32, "PUSH_VAL_TYPE");
            require(val.contents == i + 1, "PUSH_VAL_CONTENTS");
        }
        Value memory popped = arr.pop();
        require(popped.valueType == ValueType.I32, "POP_RET_TYPE");
        require(popped.contents == 3, "POP_RET_CONTENTS");
        require(arr.length() == 2, "POP_LEN");
        for (uint256 i = 0; i < arr.length(); i++) {
            Value memory val = arr.get(i);
            require(val.valueType == ValueType.I32, "POP_VAL_TYPE");
            require(val.contents == i + 1, "POP_VAL_CONTENTS");
        }
    }
}
