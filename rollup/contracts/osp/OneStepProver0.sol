//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "./IOneStepProver.sol";

contract OneStepProver0 is IOneStepProver {
	function executeConstPush(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		uint8 opcode = inst.opcode;
		ValueType ty;
		if (opcode == Instructions.I32_CONST) {
			ty = ValueType.I32;
		} else if (opcode == Instructions.I64_CONST) {
			ty = ValueType.I64;
		} else if (opcode == Instructions.F32_CONST) {
			ty = ValueType.F32;
		} else if (opcode == Instructions.F64_CONST) {
			ty = ValueType.F64;
		} else {
			revert("CONST_PUSH_INVALID_OPCODE");
		}

		ValueStacks.push(mach.value_stack, Value({
			value_type: ty,
			contents: uint64(inst.argument_data)
		}));
	}

	function executeEqz(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory v = ValueStacks.pop(mach.value_stack);

		if (v.contents == 0) {
			v.contents = 1;
		} else {
			v.contents = 0;
		}

		ValueStacks.push(mach.value_stack, v);
	}

	function executeDrop(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		ValueStacks.pop(mach.value_stack);
	}

	function executeAdd(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory a = ValueStacks.pop(mach.value_stack);
		Value memory b = ValueStacks.pop(mach.value_stack);
		uint64 contents = a.contents + b.contents;

		uint8 opcode = inst.opcode;
		ValueType ty;
		if (opcode == Instructions.I32_ADD) {
			ty = ValueType.I32;
			contents &= (1 << 32) - 1;
		} else if (opcode == Instructions.I64_ADD) {
			ty = ValueType.I64;
		} else {
			revert("TODO: floating point math");
		}

		ValueStacks.push(mach.value_stack, Value({
			value_type: ty,
			contents: contents
		}));
	}

	function executeBlock(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		bytes32 target = bytes32(inst.argument_data);
		if (target == 0) {
			Instruction[] memory proved = new Instruction[](1);
			proved[0] = inst;
			InstructionWindow memory selfWindow = InstructionWindow({
				proved: proved,
				remaining_hash: Instructions.hash(mach.instructions)
			});
			target = Instructions.hash(selfWindow);
		}

		Bytes32Stacks.push(mach.block_stack, target);
	}

	function executeBranch(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		// Jump to target
		mach.instructions = InstructionWindow({
			proved: new Instruction[](0),
			remaining_hash: Bytes32Stacks.pop(mach.block_stack)
		});
	}

	function executeBranchIf(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory cond = ValueStacks.pop(mach.value_stack);
		if (cond.contents != 0) {
			// Jump to target
			mach.instructions = InstructionWindow({
				proved: new Instruction[](0),
				remaining_hash: Bytes32Stacks.pop(mach.block_stack)
			});
		}
	}

	function executeEndBlock(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Bytes32Stacks.pop(mach.block_stack);
	}

	function executeEndBlockIf(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory cond = ValueStacks.peek(mach.value_stack);
		if (cond.contents != 0) {
			Bytes32Stacks.pop(mach.block_stack);
		}
	}

	function executeInitFrame(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory return_pc = ValueStacks.pop(mach.value_stack);
		StackFrame memory new_frame = StackFrame({
			return_pc: return_pc,
			locals_merkle_root: bytes32(inst.argument_data)
		});
		StackFrames.push(mach.frame_stack, new_frame);
	}

	function handleTrap(Machine memory mach) internal pure {
		mach.halted = true;
	}

	function executeOneStep(Machine calldata startMach, bytes calldata proof) override view external returns (Machine memory mach) {
		mach = startMach;

		Instruction memory inst = Instructions.pop(mach.instructions);
		uint8 opcode = inst.opcode;

		uint256 pops;
		function(Machine memory, Instruction memory, bytes calldata) internal view impl;
		if (opcode == Instructions.BLOCK) {
			impl = executeBlock;
		} else if (opcode == Instructions.BRANCH) {
			impl = executeBranch;
		} else if (opcode == Instructions.BRANCH_IF) {
			pops = 1;
			impl = executeBranchIf;
		} else if (opcode == Instructions.END_BLOCK) {
			impl = executeEndBlock;
		} else if (opcode == Instructions.END_BLOCK_IF) {
			pops = 1;
			impl = executeEndBlockIf;
		} else if (opcode == Instructions.INIT_FRAME) {
			pops = 1;
			impl = executeInitFrame;
		} else if (opcode == Instructions.DROP) {
			pops = 1;
			impl = executeDrop;
		} else if (opcode == Instructions.I32_EQZ) {
			pops = 1;
			impl = executeEqz;
		} else if (opcode >= Instructions.I32_CONST && opcode <= Instructions.F64_CONST) {
			impl = executeConstPush;
		} else if (opcode == Instructions.I32_ADD || opcode == Instructions.I64_ADD) {
			pops = 2;
			impl = executeAdd;
		} else {
			revert("TODO: instruction not implemented");
		}

		if (ValueStacks.hasProvenDepthLessThan(mach.value_stack, pops)) {
			// Shouldn't be possible due to wasm strict typing and validation
			handleTrap(mach);
			return mach;
		}

		impl(mach, inst, proof);
	}
}
