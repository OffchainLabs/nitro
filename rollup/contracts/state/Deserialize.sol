//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";
import "./ValueStacks.sol";
import "./Bytes32Stacks.sol";
import "./Machines.sol";
import "./Instructions.sol";
import "./StackFrames.sol";
import "./MerkleProofs.sol";

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

	function bytes32Window(bytes calldata proof, uint256 startOffset) internal pure returns (Bytes32Stack memory stack, uint256 offset) {
		offset = startOffset;
		bytes32 remainingHash;
		(remainingHash, offset) = b32(proof, offset);
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

	function instructionWindow(bytes calldata proof, uint256 startOffset) internal pure returns (InstructionWindow memory window, uint256 offset) {
		offset = startOffset;
		bytes32 remainingHash;
		Instruction[] memory proved = new Instruction[](1);
		(remainingHash, offset) = b32(proof, offset);
		(proved[0], offset) = instruction(proof, offset);
		window = InstructionWindow({
			proved: proved,
			remainingHash: remainingHash
		});
	}

	function stackFrame(bytes calldata proof, uint256 startOffset) internal pure returns (StackFrame memory window, uint256 offset) {
		offset = startOffset;
		Value memory returnPc;
		bytes32 localsMerkleRoot;
		(returnPc, offset) = value(proof, offset);
		(localsMerkleRoot, offset) = b32(proof, offset);
		window = StackFrame({
			returnPc: returnPc,
			localsMerkleRoot: localsMerkleRoot
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

	function machine(bytes calldata proof, uint256 startOffset) internal pure returns (Machine memory mach, uint256 offset) {
		offset = startOffset;
		ValueStack memory values;
		ValueStack memory internalStack;
		Bytes32Stack memory blocks;
		InstructionWindow memory instructions;
		StackFrameWindow memory frameStack;
		bytes32 globalsMerkleRoot;
		bytes32 functionsMerkleRoot;
		(values, offset) = valueStack(proof, offset);
		(internalStack, offset) = valueStack(proof, offset);
		(blocks, offset) = bytes32Window(proof, offset);
		(frameStack, offset) = stackFrameWindow(proof, offset);
		(instructions, offset) = instructionWindow(proof, offset);
		(globalsMerkleRoot, offset) = b32(proof, offset);
		(functionsMerkleRoot, offset) = b32(proof, offset);
		mach = Machine({
			valueStack: values,
			internalStack: internalStack,
			blockStack: blocks,
			frameStack: frameStack,
			instructions: instructions,
			globalsMerkleRoot: globalsMerkleRoot,
			functionsMerkleRoot: functionsMerkleRoot,
			halted: false
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
