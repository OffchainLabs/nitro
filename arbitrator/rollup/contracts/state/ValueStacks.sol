//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";
import "./ValueArrays.sol";

struct ValueStack {
	ValueArray proved;
	bytes32 remainingHash;
}

library ValueStacks {
	function hash(ValueStack memory stack) internal pure returns (bytes32 h) {
		h = stack.remainingHash;
		uint256 len = ValueArrays.length(stack.proved);
		for (uint256 i = 0; i < len; i++) {
			h = keccak256(abi.encodePacked("Value stack:", Values.hash(ValueArrays.get(stack.proved, i)), h));
		}
	}

	function peek(ValueStack memory stack) internal pure returns (Value memory) {
		uint256 len = ValueArrays.length(stack.proved);
		return ValueArrays.get(stack.proved, len - 1);
	}

	function pop(ValueStack memory stack) internal pure returns (Value memory) {
		return ValueArrays.pop(stack.proved);
	}

	function push(ValueStack memory stack, Value memory val) internal pure {
		return ValueArrays.push(stack.proved, val);
	}

	function isEmpty(ValueStack memory stack) internal pure returns (bool) {
		return ValueArrays.length(stack.proved) == 0 && stack.remainingHash == bytes32(0);
	}

	function hasProvenDepthLessThan(ValueStack memory stack, uint256 bound) internal pure returns (bool) {
		return ValueArrays.length(stack.proved) < bound && stack.remainingHash == bytes32(0);
	}
}
