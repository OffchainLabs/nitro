//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ValueStacks.sol";
import "./Bytes32Stacks.sol";
import "./Instructions.sol";
import "./StackFrames.sol";

struct Machine {
	ValueStack value_stack;
	Bytes32Stack block_stack;
	InstructionWindow instructions;
	StackFrameWindow frame_stack;
	bool halted;
}

library Machines {
	function hash(Machine memory mach) internal pure returns (bytes32) {
		if (mach.halted) {
			return bytes32(0);
		}
		return keccak256(abi.encodePacked(
			"Machine:",
			ValueStacks.hash(mach.value_stack),
			Bytes32Stacks.hash(mach.block_stack),
			StackFrames.hash(mach.frame_stack),
			Instructions.hash(mach.instructions)
		));
	}
}
