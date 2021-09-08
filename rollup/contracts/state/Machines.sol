//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ValueStacks.sol";
import "./Bytes32Stacks.sol";
import "./Instructions.sol";
import "./StackFrames.sol";

struct Machine {
	ValueStack valueStack;
	Bytes32Stack blockStack;
	StackFrameWindow frameStack;
	InstructionWindow instructions;
	bytes32 globalsMerkleRoot;
	bytes32 functionsMerkleRoot;
	bool halted;
}

library Machines {
	function hash(Machine memory mach) internal pure returns (bytes32) {
		if (mach.halted) {
			return bytes32(0);
		}
		return keccak256(abi.encodePacked(
			"Machine:",
			ValueStacks.hash(mach.valueStack),
			Bytes32Stacks.hash(mach.blockStack),
			StackFrames.hash(mach.frameStack),
			Instructions.hash(mach.instructions),
			mach.globalsMerkleRoot,
			mach.functionsMerkleRoot
		));
	}
}
