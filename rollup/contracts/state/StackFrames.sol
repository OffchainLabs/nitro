//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";

struct StackFrame {
	Value return_pc;
	bytes32 locals_merkle_root;
}

struct StackFrameWindow {
	StackFrame[] proved;
	bytes32 remaining_hash;
}

library StackFrames {
	function hash(StackFrame memory frame) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Stack frame:", Values.hash(frame.return_pc), frame.locals_merkle_root));
	}

	function hash(StackFrameWindow memory window) internal pure returns (bytes32 h) {
		h = window.remaining_hash;
		for (uint256 i = 0; i < window.proved.length; i++) {
			h = keccak256(abi.encodePacked("Stack frame stack:", hash(window.proved[i]), h));
		}
	}

	function peek(StackFrameWindow memory window) internal pure returns (StackFrame memory) {
		require(window.proved.length == 1, "BAD_WINDOW_LENGTH");
		return window.proved[0];
	}

	function pop(StackFrameWindow memory window) internal pure returns (StackFrame memory frame) {
		require(window.proved.length == 1, "BAD_WINDOW_LENGTH");
		frame = window.proved[0];
		window.proved = new StackFrame[](0);
	}

	function push(StackFrameWindow memory window, StackFrame memory frame) internal pure {
		StackFrame[] memory new_proved = new StackFrame[](window.proved.length + 1);
		for (uint256 i = 0; i < window.proved.length; i++) {
			new_proved[i] = window.proved[i];
		}
		new_proved[window.proved.length] = frame;
		window.proved = new_proved;
	}
}
