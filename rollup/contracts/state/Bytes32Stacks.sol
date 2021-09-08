//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Bytes32Arrays.sol";

struct Bytes32Stack {
	Bytes32Array proved;
	bytes32 remaining_hash;
}

library Bytes32Stacks {
	function hash(Bytes32Stack memory stack) internal pure returns (bytes32 h) {
		h = stack.remaining_hash;
		uint256 len = Bytes32Arrays.length(stack.proved);
		for (uint256 i = 0; i < len; i++) {
			h = keccak256(abi.encodePacked("Bytes32 stack:", Bytes32Arrays.get(stack.proved, i), h));
		}
	}

	function pop(Bytes32Stack memory stack) internal pure returns (bytes32) {
		return Bytes32Arrays.pop(stack.proved);
	}

	function push(Bytes32Stack memory stack, bytes32 val) internal pure {
		return Bytes32Arrays.push(stack.proved, val);
	}

	function isEmpty(Bytes32Stack memory stack) internal pure returns (bool) {
		return Bytes32Arrays.length(stack.proved) == 0 && stack.remaining_hash == bytes32(0);
	}

	function hasProvenDepthLessThan(Bytes32Stack memory stack, uint256 bound) internal pure returns (bool) {
		return Bytes32Arrays.length(stack.proved) < bound && stack.remaining_hash == bytes32(0);
	}
}
