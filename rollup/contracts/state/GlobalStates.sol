//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct GlobalState {
	bytes32[1] bytes32_vals;
	uint64[2] u64_vals;
}


library GlobalStates {
	uint16 constant BYTES32_VALS_NUM = 1;
	uint16 constant U64_VALS_NUM = 2;
	function hash(GlobalState memory state) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked(
			"Global state:",
			state.bytes32_vals[0],
			state.u64_vals[0],
			state.u64_vals[1]
		));
	}

}
