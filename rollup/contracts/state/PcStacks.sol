//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./PcArrays.sol";

struct PcStack {
	PcArray proved;
	bytes32 remainingHash;
}

library PcStacks {
	function hash(PcStack memory stack) internal pure returns (bytes32 h) {
		h = stack.remainingHash;
		uint256 len = PcArrays.length(stack.proved);
		for (uint256 i = 0; i < len; i++) {
			h = keccak256(abi.encodePacked("Program counter stack:", PcArrays.get(stack.proved, i), h));
		}
	}

	function pop(PcStack memory stack) internal pure returns (uint64) {
		return PcArrays.pop(stack.proved);
	}

	function push(PcStack memory stack, uint64 val) internal pure {
		return PcArrays.push(stack.proved, val);
	}

	function isEmpty(PcStack memory stack) internal pure returns (bool) {
		return PcArrays.length(stack.proved) == 0 && stack.remainingHash == bytes32(0);
	}

	function hasProvenDepthLessThan(PcStack memory stack, uint256 bound) internal pure returns (bool) {
		return PcArrays.length(stack.proved) < bound && stack.remainingHash == bytes32(0);
	}
}
