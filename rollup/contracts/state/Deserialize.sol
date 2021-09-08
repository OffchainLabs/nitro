//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";
import "./ValueStacks.sol";
import "./Bytes32Stacks.sol";
import "./Machines.sol";
import "./Instructions.sol";
import "./StackFrames.sol";

library Deserialize {
	function u8(bytes calldata proof, uint256 startOffset) internal pure returns (uint8 ret, uint256 offset) {
		offset = startOffset;
		ret = uint8(proof[offset]);
		offset++;
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
		uint8 type_int = uint8(proof[offset]);
		offset++;
		require(type_int <= uint8(Values.maxValueType()), "BAD_VALUE_TYPE");
		uint64 contents;
		(contents, offset) = u64(proof, offset);
		val = Value({
			value_type: ValueType(type_int),
			contents: contents
		});
	}

	function valueStack(bytes calldata proof, uint256 startOffset) internal pure returns (ValueStack memory stack, uint256 offset) {
		offset = startOffset;
		bytes32 remaining_hash;
		(remaining_hash, offset) = b32(proof, offset);
		uint256 proved_length;
		(proved_length, offset) = u256(proof, offset);
		Value[] memory proved = new Value[](proved_length);
		for (uint256 i = 0; i < proved.length; i++) {
			(proved[i], offset) = value(proof, offset);
		}
		stack = ValueStack({
			proved: ValueArray(proved),
			remaining_hash: remaining_hash
		});
	}

	function bytes32Window(bytes calldata proof, uint256 startOffset) internal pure returns (Bytes32Stack memory stack, uint256 offset) {
		offset = startOffset;
		bytes32 remaining_hash;
		(remaining_hash, offset) = b32(proof, offset);
		bytes32[] memory proved;
		if (proof[offset] != 0) {
			offset++;
			proved = new bytes32[](1);
			(proved[0], offset) = b32(proof, offset);
		} else {
			offset++;
			proved = new bytes32[](0);
		}
		stack = Bytes32Stack({
			proved: Bytes32Array(proved),
			remaining_hash: remaining_hash
		});
	}

	function instruction(bytes calldata proof, uint256 startOffset) internal pure returns (Instruction memory inst, uint256 offset) {
		offset = startOffset;
		uint8 opcode;
		uint256 data;
		(opcode, offset) = u8(proof, offset);
		(data, offset) = u256(proof, offset);
		inst = Instruction({
			opcode: opcode,
			argument_data: data
		});
	}

	function instructionWindow(bytes calldata proof, uint256 startOffset) internal pure returns (InstructionWindow memory window, uint256 offset) {
		offset = startOffset;
		bytes32 remaining_hash;
		(remaining_hash, offset) = b32(proof, offset);
		Instruction[] memory proved;
		if (proof[offset] != 0) {
			offset++;
			proved = new Instruction[](1);
			(proved[0], offset) = instruction(proof, offset);
		} else {
			offset++;
			proved = new Instruction[](0);
		}
		window = InstructionWindow({
			proved: proved,
			remaining_hash: remaining_hash
		});
	}

	function stackFrame(bytes calldata proof, uint256 startOffset) internal pure returns (StackFrame memory window, uint256 offset) {
		offset = startOffset;
		Value memory return_pc;
		bytes32 locals_merkle_root;
		(return_pc, offset) = value(proof, offset);
		(locals_merkle_root, offset) = b32(proof, offset);
		window = StackFrame({
			return_pc: return_pc,
			locals_merkle_root: locals_merkle_root
		});
	}

	function stackFrameWindow(bytes calldata proof, uint256 startOffset) internal pure returns (StackFrameWindow memory window, uint256 offset) {
		offset = startOffset;
		bytes32 remaining_hash;
		(remaining_hash, offset) = b32(proof, offset);
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
			remaining_hash: remaining_hash
		});
	}

	function machine(bytes calldata proof, uint256 startOffset) internal pure returns (Machine memory mach, uint256 offset) {
		offset = startOffset;
		ValueStack memory value_stack;
		Bytes32Stack memory block_stack;
		InstructionWindow memory instructions;
		StackFrameWindow memory frame_stack;
		(value_stack, offset) = valueStack(proof, offset);
		(block_stack, offset) = bytes32Window(proof, offset);
		(frame_stack, offset) = stackFrameWindow(proof, offset);
		(instructions, offset) = instructionWindow(proof, offset);
		mach = Machine({
			value_stack: value_stack,
			block_stack: block_stack,
			frame_stack: frame_stack,
			instructions: instructions,
			halted: false
		});
	}
}
