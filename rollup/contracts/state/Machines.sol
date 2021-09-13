//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ValueStacks.sol";
import "./PcStacks.sol";
import "./Instructions.sol";
import "./StackFrames.sol";
import "./MachineMemories.sol";

struct Machine {
	ValueStack valueStack;
	ValueStack internalStack;
	PcStack blockStack;
	StackFrameWindow frameStack;
	uint64 functionIdx;
	uint64 functionPc;
	bytes32 globalsMerkleRoot;
	MachineMemory machineMemory;
	bytes32 tablesMerkleRoot;
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
			ValueStacks.hash(mach.internalStack),
			PcStacks.hash(mach.blockStack),
			StackFrames.hash(mach.frameStack),
			mach.functionIdx,
			mach.functionPc,
			mach.globalsMerkleRoot,
			MachineMemories.hash(mach.machineMemory),
			mach.tablesMerkleRoot,
			mach.functionsMerkleRoot
		));
	}
}
