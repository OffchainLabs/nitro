//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct PcArray {
	uint64[] inner;
}

library PcArrays {
	function get(PcArray memory arr, uint256 index) internal pure returns (uint64) {
		return arr.inner[index];
	}

	function set(PcArray memory arr, uint256 index, uint64 val) internal pure {
		arr.inner[index] = val;
	}

	function length(PcArray memory arr) internal pure returns (uint256) {
		return arr.inner.length;
	}

	function push(PcArray memory arr, uint64 val) internal pure {
		uint64[] memory newInner = new uint64[](arr.inner.length + 1);
		for (uint256 i = 0; i < arr.inner.length; i++) {
			newInner[i] = arr.inner[i];
		}
		newInner[arr.inner.length] = val;
		arr.inner = newInner;
	}

	function pop(PcArray memory arr) internal pure returns (uint64 popped) {
		popped = arr.inner[arr.inner.length - 1];
		uint64[] memory newInner = new uint64[](arr.inner.length - 1);
		for (uint256 i = 0; i < newInner.length; i++) {
			newInner[i] = arr.inner[i];
		}
		arr.inner = newInner;
	}
}
