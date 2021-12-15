//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ValueStacks.sol";
import "./PcStacks.sol";
import "./Instructions.sol";
import "./StackFrames.sol";

enum MachineStatus {
	RUNNING,
	FINISHED,
	ERRORED,
	TOO_FAR
}

struct Machine {
	MachineStatus status;
	ValueStack valueStack;
	ValueStack internalStack;
	PcStack blockStack;
	StackFrameWindow frameStack;
	bytes32 globalStateHash;
	uint32 moduleIdx;
	uint32 functionIdx;
	uint32 functionPc;
	bytes32 modulesRoot;
}

library Machines {
	function hash(Machine memory mach) internal pure returns (bytes32) {
		// Warning: the non-running hashes are replicated in BlockChallenge
		if (mach.status == MachineStatus.RUNNING) {
			return keccak256(abi.encodePacked(
				"Machine running:",
				ValueStacks.hash(mach.valueStack),
				ValueStacks.hash(mach.internalStack),
				PcStacks.hash(mach.blockStack),
				StackFrames.hash(mach.frameStack),
				mach.globalStateHash,
				mach.moduleIdx,
				mach.functionIdx,
				mach.functionPc,
				mach.modulesRoot
			));
		} else if (mach.status == MachineStatus.FINISHED) {
			return keccak256(abi.encodePacked(
				"Machine finished:",
				mach.globalStateHash
			));
		} else if (mach.status == MachineStatus.ERRORED) {
			return keccak256(abi.encodePacked("Machine errored:"));
		} else if (mach.status == MachineStatus.TOO_FAR) {
			return keccak256(abi.encodePacked("Machine too far:"));
		} else {
			revert("BAD_MACH_STATUS");
		}
	}
}
