//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

enum ValueType {
	I32,
	I64,
	F32,
	F64,
	REF_NULL,
	REF,
	REF_EXTERN
}

struct Value {
	ValueType value_type;
	uint64 contents;
}

library Values {
	function hash(Value memory val) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Value:", val.value_type, val.contents));
	}

	function maxValueType() internal pure returns (ValueType) {
		return ValueType.REF_EXTERN;
	}

	function isNumeric(ValueType val) internal pure returns (bool) {
		return val == ValueType.I32 || val == ValueType.I64 || val == ValueType.F32 || val == ValueType.F64;
	}

	function isNumeric(Value memory val) internal pure returns (bool) {
		return isNumeric(val.value_type);
	}

	function newInt32(int32 x) internal pure returns (Value memory) {
		return Value({
			value_type: ValueType.I32,
			contents: uint64(uint32(x))
		});
	}
}
