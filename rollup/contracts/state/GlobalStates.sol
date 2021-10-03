//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct GlobalState {
	bytes32 lastBlockHash;
	uint256 inboxPosition;
	uint64 positionWithinMessage;
}

library GlobalStates {
	function hash(GlobalState memory state) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked(
			"Global state:",
			state.lastBlockHash,
			state.inboxPosition,
			state.positionWithinMessage
		));
	}
}
