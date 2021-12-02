//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";
import "./ValueStacks.sol";
import "./PcStacks.sol";
import "./Machines.sol";
import "./Instructions.sol";
import "./StackFrames.sol";
import "./MerkleProofs.sol";
import "./ModuleMemories.sol";
import "./Modules.sol";
import "./GlobalStates.sol";

library Deserialize {
	function u8(bytes calldata proof, uint256 startOffset) internal pure returns (uint8 ret, uint256 offset) {
		offset = startOffset;
		ret = uint8(proof[offset]);
		offset++;
	}

	function u16(bytes calldata proof, uint256 startOffset) internal pure returns (uint16 ret, uint256 offset) {
		offset = startOffset;
		for (uint256 i = 0; i < 16/8; i++) {
			ret <<= 8;
			ret |= uint8(proof[offset]);
			offset++;
		}
	}

	function u32(bytes calldata proof, uint256 startOffset) internal pure returns (uint32 ret, uint256 offset) {
		offset = startOffset;
		for (uint256 i = 0; i < 32/8; i++) {
			ret <<= 8;
			ret |= uint8(proof[offset]);
			offset++;
		}
	}

	function u64(bytes calldata proof, uint256 startOffset) internal pure returns (uint64 ret, uint256 offset) {
		offset = startOffset;
		for (uint256 i = 0; i < 64/8; i++) {
			ret <<= 8;
			ret |= uint8(proof[offset]);
			offset++;
		}
	}

	function u256(bytes calldata proof, uint256 startOffset) internal pure returns (uint256 ret, uint256 offset) {
		offset = startOffset;
		for (uint256 i = 0; i < 256/8; i++) {
			ret <<= 8;
			ret |= uint8(proof[offset]);
			offset++;
		}
	}

	function b32(bytes calldata proof, uint256 startOffset) internal pure returns (bytes32 ret, uint256 offset) {
		offset = startOffset;
		uint256 retInt;
		(retInt, offset) = u256(proof, offset);
		ret = bytes32(retInt);
	}

	function value(bytes calldata proof, uint256 startOffset) internal pure returns (Value memory val, uint256 offset)  {
		offset = startOffset;
		uint8 typeInt = uint8(proof[offset]);
		offset++;
		require(typeInt <= uint8(Values.maxValueType()), "BAD_VALUE_TYPE");
		uint256 contents;
		(contents, offset) = u256(proof, offset);
		val = Value({
			valueType: ValueType(typeInt),
			contents: contents
		});
	}

	function valueStack(bytes calldata proof, uint256 startOffset) internal pure returns (ValueStack memory stack, uint256 offset) {
		offset = startOffset;
		bytes32 remainingHash;
		(remainingHash, offset) = b32(proof, offset);
		uint256 provedLength;
		(provedLength, offset) = u256(proof, offset);
		Value[] memory proved = new Value[](provedLength);
		for (uint256 i = 0; i < proved.length; i++) {
			(proved[i], offset) = value(proof, offset);
		}
		stack = ValueStack({
			proved: ValueArray(proved),
			remainingHash: remainingHash
		});
	}

	function pcStack(bytes calldata proof, uint256 startOffset) internal pure returns (PcStack memory stack, uint256 offset) {
		offset = startOffset;
		bytes32 remainingHash;
		(remainingHash, offset) = b32(proof, offset);
		uint256 provedLength;
		(provedLength, offset) = u256(proof, offset);
		uint32[] memory proved = new uint32[](provedLength);
		for (uint256 i = 0; i < proved.length; i++) {
			(proved[i], offset) = u32(proof, offset);
		}
		stack = PcStack({
			proved: PcArray(proved),
			remainingHash: remainingHash
		});
	}

	function instruction(bytes calldata proof, uint256 startOffset) internal pure returns (Instruction memory inst, uint256 offset) {
		offset = startOffset;
		uint16 opcode;
		uint256 data;
		(opcode, offset) = u16(proof, offset);
		(data, offset) = u256(proof, offset);
		inst = Instruction({
			opcode: opcode,
			argumentData: data
		});
	}

	function stackFrame(bytes calldata proof, uint256 startOffset) internal pure returns (StackFrame memory window, uint256 offset) {
		offset = startOffset;
		Value memory returnPc;
		bytes32 localsMerkleRoot;
		uint32 callerModule;
		uint32 callerModuleInternals;
		(returnPc, offset) = value(proof, offset);
		(localsMerkleRoot, offset) = b32(proof, offset);
		(callerModule, offset) = u32(proof, offset);
		(callerModuleInternals, offset) = u32(proof, offset);
		window = StackFrame({
			returnPc: returnPc,
			localsMerkleRoot: localsMerkleRoot,
			callerModule: callerModule,
			callerModuleInternals: callerModuleInternals
		});
	}

	function stackFrameWindow(bytes calldata proof, uint256 startOffset) internal pure returns (StackFrameWindow memory window, uint256 offset) {
		offset = startOffset;
		bytes32 remainingHash;
		(remainingHash, offset) = b32(proof, offset);
		StackFrame[] memory proved;
		if (proof[offset] != 0) {
			offset++;
			proved = new StackFrame[](1);
			(proved[0], offset) = stackFrame(proof, offset);
		} else {
			offset++;
			proved = new StackFrame[](0);
		}
		window = StackFrameWindow({
			proved: proved,
			remainingHash: remainingHash
		});
	}

	function moduleMemory(bytes calldata proof, uint256 startOffset) internal pure returns (ModuleMemory memory mem, uint256 offset) {
		offset = startOffset;
		uint64 size;
		bytes32 root;
		(size, offset) = u64(proof, offset);
		(root, offset) = b32(proof, offset);
		mem = ModuleMemory({
			size: size,
			merkleRoot: root
		});
	}

	function module(bytes calldata proof, uint256 startOffset) internal pure returns (Module memory mod, uint256 offset) {
		offset = startOffset;
		bytes32 globalsMerkleRoot;
		ModuleMemory memory mem;
		bytes32 tablesMerkleRoot;
		bytes32 functionsMerkleRoot;
		uint32 internalsOffset;
		(globalsMerkleRoot, offset) = b32(proof, offset);
		(mem, offset) = moduleMemory(proof, offset);
		(tablesMerkleRoot, offset) = b32(proof, offset);
		(functionsMerkleRoot, offset) = b32(proof, offset);
		(internalsOffset, offset) = u32(proof, offset);
		mod = Module({
			globalsMerkleRoot: globalsMerkleRoot,
			moduleMemory: mem,
			tablesMerkleRoot: tablesMerkleRoot,
			functionsMerkleRoot: functionsMerkleRoot,
			internalsOffset: internalsOffset
		});
	}

	function globalState(bytes calldata proof, uint256 startOffset) internal pure returns (GlobalState memory state, uint256 offset) {
		offset = startOffset;

		// using constant ints for array size requires newer solidity
		bytes32[1] memory bytes32_vals;
		uint64[2] memory u64_vals;

		for (uint8 i = 0; i< GlobalStates.BYTES32_VALS_NUM; i++) {
			(bytes32_vals[i], offset) = b32(proof, offset);
		}
		for (uint8 i = 0; i< GlobalStates.U64_VALS_NUM; i++) {
			(u64_vals[i], offset) = u64(proof, offset);
		}
		state = GlobalState({
			bytes32_vals: bytes32_vals,
			u64_vals: u64_vals
		});
	}

	function machine(bytes calldata proof, uint256 startOffset) internal pure returns (Machine memory mach, uint256 offset) {
		offset = startOffset;
		MachineStatus status;
		{
			uint8 status_u8;
			(status_u8, offset) = u8(proof, offset);
			if (status_u8 == 0) {
				status = MachineStatus.RUNNING;
			} else if (status_u8 == 1) {
				status = MachineStatus.FINISHED;
			} else if (status_u8 == 2) {
				status = MachineStatus.ERRORED;
			} else if (status_u8 == 3) {
				status = MachineStatus.TOO_FAR;
			} else {
				revert("UNKNOWN_MACH_STATUS");
			}
		}
		ValueStack memory values;
		ValueStack memory internalStack;
		PcStack memory blocks;
		bytes32 globalStateHash;
		uint32 moduleIdx;
		uint32 functionIdx;
		uint32 functionPc;
		StackFrameWindow memory frameStack;
		bytes32 modulesRoot;
		(values, offset) = valueStack(proof, offset);
		(internalStack, offset) = valueStack(proof, offset);
		(blocks, offset) = pcStack(proof, offset);
		(frameStack, offset) = stackFrameWindow(proof, offset);
		(globalStateHash, offset) = b32(proof, offset);
		(moduleIdx, offset) = u32(proof, offset);
		(functionIdx, offset) = u32(proof, offset);
		(functionPc, offset) = u32(proof, offset);
		(modulesRoot, offset) = b32(proof, offset);
		mach = Machine({
			status: status,
			valueStack: values,
			internalStack: internalStack,
			blockStack: blocks,
			frameStack: frameStack,
			globalStateHash: globalStateHash,
			moduleIdx: moduleIdx,
			functionIdx: functionIdx,
			functionPc: functionPc,
			modulesRoot: modulesRoot
		});
	}

	function merkleProof(bytes calldata proof, uint256 startOffset) internal pure returns (MerkleProof memory merkle, uint256 offset) {
		offset = startOffset;
		uint8 length;
		(length, offset) = u8(proof, offset);
		bytes32[] memory counterparts = new bytes32[](length);
		for (uint8 i = 0; i < length; i++) {
			(counterparts[i], offset) = b32(proof, offset);
		}
		merkle = MerkleProof(counterparts);
	}
}
