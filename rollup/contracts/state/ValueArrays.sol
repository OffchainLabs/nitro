//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";

struct ValueArray {
	Value[] inner;
}

library ValueArrays {
	function get(ValueArray memory arr, uint256 index) internal pure returns (Value memory) {
		return arr.inner[index];
	}

	function set(ValueArray memory arr, uint256 index, Value memory val) internal pure {
		arr.inner[index] = val;
	}

	function length(ValueArray memory arr) internal pure returns (uint256) {
		return arr.inner.length;
	}

	function push(ValueArray memory arr, Value memory val) internal pure {
		Value[] memory new_inner = new Value[](arr.inner.length + 1);
		for (uint256 i = 0; i < arr.inner.length; i++) {
			new_inner[i] = arr.inner[i];
		}
		new_inner[arr.inner.length] = val;
		arr.inner = new_inner;
	}

	function pop(ValueArray memory arr) internal pure returns (Value memory popped) {
		popped = arr.inner[arr.inner.length - 1];
		Value[] memory new_inner = new Value[](arr.inner.length - 1);
		for (uint256 i = 0; i < new_inner.length; i++) {
			new_inner[i] = arr.inner[i];
		}
		arr.inner = new_inner;
	}
}
