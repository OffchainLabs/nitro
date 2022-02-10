//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct GlobalState {
	bytes32[2] bytes32_vals;
	uint64[2] u64_vals;
}


library GlobalStateLib {
	uint16 constant BYTES32_VALS_NUM = 2;
	uint16 constant U64_VALS_NUM = 2;
	function hash(GlobalState memory state) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked(
			"Global state:",
			state.bytes32_vals[0],
			state.bytes32_vals[1],
			state.u64_vals[0],
			state.u64_vals[1]
		));
	}

	function getBlockHash(GlobalState memory state) internal pure returns (bytes32) {
		return state.bytes32_vals[0];
	}

	function getSendRoot(GlobalState memory state) internal pure returns (bytes32) {
		return state.bytes32_vals[1];
	}

	function getInboxPosition(GlobalState memory state) internal pure returns (uint64) {
		return state.u64_vals[0];
	}

	function getPositionInMessage(GlobalState memory state) internal pure returns (uint64) {
		return state.u64_vals[1];
	}
}
