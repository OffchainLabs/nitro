//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

import "../state/ValueArrays.sol";

contract ValueArrayTester {
    function test() external pure {
        ValueArray memory arr = ValueArray(new Value[](2));
        require(ValueArrays.length(arr) == 2, "START_LEN");
        ValueArrays.set(arr, 0, Values.newInt32(1));
        ValueArrays.set(arr, 1, Values.newInt32(2));
        ValueArrays.push(arr, Values.newInt32(3));
        require(ValueArrays.length(arr) == 3, "PUSH_LEN");
        for (uint256 i = 0; i < ValueArrays.length(arr); i++) {
            Value memory val = ValueArrays.get(arr, i);
            require(val.valueType == ValueType.I32, "PUSH_VAL_TYPE");
            require(val.contents == i + 1, "PUSH_VAL_CONTENTS");
        }
        Value memory popped = ValueArrays.pop(arr);
        require(popped.valueType == ValueType.I32, "POP_RET_TYPE");
        require(popped.contents == 3, "POP_RET_CONTENTS");
        require(ValueArrays.length(arr) == 2, "POP_LEN");
        for (uint256 i = 0; i < ValueArrays.length(arr); i++) {
            Value memory val = ValueArrays.get(arr, i);
            require(val.valueType == ValueType.I32, "POP_VAL_TYPE");
            require(val.contents == i + 1, "POP_VAL_CONTENTS");
        }
    }
}
