//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProver0 is IOneStepProver {
	function executeUnreachable(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		mach.halted = true;
	}

	function executeConstPush(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		uint16 opcode = inst.opcode;
		ValueType ty;
		if (opcode == Instructions.I32_CONST) {
			ty = ValueType.I32;
		} else if (opcode == Instructions.I64_CONST) {
			ty = ValueType.I64;
		} else if (opcode == Instructions.F32_CONST) {
			ty = ValueType.F32;
		} else if (opcode == Instructions.F64_CONST) {
			ty = ValueType.F64;
		} else if (opcode == Instructions.PUSH_STACK_BOUNDARY) {
			ty = ValueType.STACK_BOUNDARY;
		} else {
			revert("CONST_PUSH_INVALID_OPCODE");
		}

		ValueStacks.push(mach.valueStack, Value({
			valueType: ty,
			contents: uint64(inst.argumentData)
		}));
	}

	function executeEqz(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory v = ValueStacks.pop(mach.valueStack);

		if (v.contents == 0) {
			v.contents = 1;
		} else {
			v.contents = 0;
		}

		ValueStacks.push(mach.valueStack, v);
	}

	function executeDrop(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		ValueStacks.pop(mach.valueStack);
	}

	function executeI32Arith(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		uint32 b = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		uint32 a = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		uint32 res;

		uint16 opcodeOffset = inst.opcode - Instructions.I32_ADD;

		if (opcodeOffset == 0) {
			res = a + b;
		} else if (opcodeOffset == 1) {
			res = a - b;
		} else if (opcodeOffset == 2) {
			res = a * b;
		} else {
			revert("TODO: more operations");
		}

		ValueStacks.push(mach.valueStack, Values.newI32(res));
	}

	function executeI64Arith(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		uint64 b = Values.assumeI64(ValueStacks.pop(mach.valueStack));
		uint64 a = Values.assumeI64(ValueStacks.pop(mach.valueStack));
		uint64 res;

		uint16 opcodeOffset = inst.opcode - Instructions.I32_ADD;

		if (opcodeOffset == 0) {
			res = a + b;
		} else if (opcodeOffset == 1) {
			res = a - b;
		} else if (opcodeOffset == 2) {
			res = a * b;
		} else {
			revert("TODO: more operations");
		}

		ValueStacks.push(mach.valueStack, Values.newI64(res));
	}

	function executeBlock(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		bytes32 target = bytes32(inst.argumentData);
		if (target == 0) {
			Instruction[] memory proved = new Instruction[](1);
			proved[0] = inst;
			InstructionWindow memory selfWindow = InstructionWindow({
				proved: proved,
				remainingHash: Instructions.hash(mach.instructions)
			});
			target = Instructions.hash(selfWindow);
		}

		Bytes32Stacks.push(mach.blockStack, target);
	}

	function executeBranch(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		// Jump to target
		mach.instructions = InstructionWindow({
			proved: new Instruction[](0),
			remainingHash: Bytes32Stacks.pop(mach.blockStack)
		});
	}

	function executeBranchIf(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory cond = ValueStacks.pop(mach.valueStack);
		if (cond.contents != 0) {
			// Jump to target
			mach.instructions = InstructionWindow({
				proved: new Instruction[](0),
				remainingHash: Bytes32Stacks.pop(mach.blockStack)
			});
		}
	}

	function executeReturn(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		StackFrame memory frame = StackFrames.pop(mach.frameStack);
		if (frame.returnPc.valueType == ValueType.REF_NULL) {
			mach.halted = true;
			return;
		} else if (frame.returnPc.valueType != ValueType.REF) {
			revert("INVALID_RETURN_PC_TYPE");
		}
		mach.instructions = InstructionWindow({
			proved: new Instruction[](0),
			remainingHash: bytes32(frame.returnPc.contents)
		});
	}

	function executeCall(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		uint256 offset = 0;
		InstructionWindow memory proposedInstructions;
		MerkleProof memory merkle;
		(proposedInstructions, offset) = Deserialize.instructionWindow(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRoot(merkle, inst.argumentData, proposedInstructions);
		require(recomputedRoot == mach.functionsMerkleRoot, "WRONG_FUNC_MERKLE_ROOT");

		Value memory retPc = Value({
			valueType: ValueType.REF,
			contents: uint256(Instructions.hash(mach.instructions))
		});
		ValueStacks.push(mach.valueStack, retPc);
		mach.instructions = proposedInstructions;
	}

	function executeArbitraryJumpIf(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory cond = ValueStacks.pop(mach.valueStack);
		if (cond.contents != 0) {
			// Jump to target
			mach.instructions = InstructionWindow({
				proved: new Instruction[](0),
				remainingHash: bytes32(inst.argumentData)
			});
		}
	}

	function merkleProveGetValue(bytes32 merkleRoot, uint256 index, bytes calldata proof) internal pure returns (Value memory) {
		uint256 offset = 0;
		Value memory proposedVal;
		MerkleProof memory merkle;
		(proposedVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRoot(merkle, index, proposedVal);
		require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
		return proposedVal;
	}

	function merkleProveSetValue(bytes32 merkleRoot, uint256 index, Value memory newVal, bytes calldata proof) internal pure returns (bytes32) {
		Value memory oldVal;
		uint256 offset = 0;
		MerkleProof memory merkle;
		(oldVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRoot(merkle, index, oldVal);
		require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
		return MerkleProofs.computeRoot(merkle, index, newVal);
	}

	function executeLocalGet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		Value memory val = merkleProveGetValue(frame.localsMerkleRoot, inst.argumentData, proof);
		ValueStacks.push(mach.valueStack, val);
	}

	function executeLocalSet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		Value memory newVal = ValueStacks.pop(mach.valueStack);
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		frame.localsMerkleRoot = merkleProveSetValue(frame.localsMerkleRoot, inst.argumentData, newVal, proof);
	}

	function executeGlobalGet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		Value memory val = merkleProveGetValue(mach.globalsMerkleRoot, inst.argumentData, proof);
		ValueStacks.push(mach.valueStack, val);
	}

	function executeGlobalSet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		Value memory newVal = ValueStacks.pop(mach.valueStack);
		mach.globalsMerkleRoot = merkleProveSetValue(mach.globalsMerkleRoot, inst.argumentData, newVal, proof);
	}

	function executeEndBlock(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Bytes32Stacks.pop(mach.blockStack);
	}

	function executeEndBlockIf(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory cond = ValueStacks.peek(mach.valueStack);
		if (cond.contents != 0) {
			Bytes32Stacks.pop(mach.blockStack);
		}
	}

	function executeInitFrame(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory returnPc = ValueStacks.pop(mach.valueStack);
		StackFrame memory newFrame = StackFrame({
			returnPc: returnPc,
			localsMerkleRoot: bytes32(inst.argumentData)
		});
		StackFrames.push(mach.frameStack, newFrame);
	}

	function executeMoveInternal(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory val;
		if (inst.opcode == Instructions.MOVE_FROM_STACK_TO_INTERNAL) {
			val = ValueStacks.pop(mach.valueStack);
			ValueStacks.push(mach.internalStack, val);
		} else if (inst.opcode == Instructions.MOVE_FROM_INTERNAL_TO_STACK) {
			val = ValueStacks.pop(mach.internalStack);
			ValueStacks.push(mach.valueStack, val);
		} else {
			revert("MOVE_INTERNAL_INVALID_OPCODE");
		}
	}

	function executeIsStackBoundary(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory val = ValueStacks.pop(mach.valueStack);
		uint256 newContents = 0;
		if (val.valueType == ValueType.STACK_BOUNDARY) {
			newContents = 1;
		}
		ValueStacks.push(mach.valueStack, Value({
			valueType: ValueType.I32,
			contents: newContents
		}));
	}

	function executeDup(Machine memory mach, Instruction memory, bytes calldata) internal pure {
		Value memory val = ValueStacks.peek(mach.valueStack);
		ValueStacks.push(mach.valueStack, val);
	}

	function handleTrap(Machine memory mach) internal pure {
		mach.halted = true;
	}

	function executeOneStep(Machine calldata startMach, bytes calldata proof) override view external returns (Machine memory mach) {
		mach = startMach;

		Instruction memory inst = Instructions.pop(mach.instructions);
		uint16 opcode = inst.opcode;

		function(Machine memory, Instruction memory, bytes calldata) internal view impl;
		if (opcode == Instructions.UNREACHABLE) {
			impl = executeUnreachable;
		} else if (opcode == Instructions.BLOCK) {
			impl = executeBlock;
		} else if (opcode == Instructions.BRANCH) {
			impl = executeBranch;
		} else if (opcode == Instructions.BRANCH_IF) {
			impl = executeBranchIf;
		} else if (opcode == Instructions.RETURN) {
			impl = executeReturn;
		} else if (opcode == Instructions.CALL) {
			impl = executeCall;
		} else if (opcode == Instructions.END_BLOCK) {
			impl = executeEndBlock;
		} else if (opcode == Instructions.END_BLOCK_IF) {
			impl = executeEndBlockIf;
		} else if (opcode == Instructions.ARBITRARY_JUMP_IF) {
			impl = executeArbitraryJumpIf;
		} else if (opcode == Instructions.LOCAL_GET) {
			impl = executeLocalGet;
		} else if (opcode == Instructions.LOCAL_SET) {
			impl = executeLocalSet;
		} else if (opcode == Instructions.GLOBAL_GET) {
			impl = executeGlobalGet;
		} else if (opcode == Instructions.GLOBAL_SET) {
			impl = executeGlobalSet;
		} else if (opcode == Instructions.INIT_FRAME) {
			impl = executeInitFrame;
		} else if (opcode == Instructions.DROP) {
			impl = executeDrop;
		} else if (opcode == Instructions.I32_EQZ) {
			impl = executeEqz;
		} else if (opcode >= Instructions.I32_CONST && opcode <= Instructions.F64_CONST || opcode == Instructions.PUSH_STACK_BOUNDARY) {
			impl = executeConstPush;
		} else if (opcode >= Instructions.I32_ADD && opcode <= Instructions.I32_MUL) {
			impl = executeI32Arith;
		} else if (opcode >= Instructions.I64_ADD && opcode <= Instructions.I64_MUL) {
			impl = executeI64Arith;
		} else if (opcode == Instructions.MOVE_FROM_STACK_TO_INTERNAL || opcode == Instructions.MOVE_FROM_INTERNAL_TO_STACK) {
			impl = executeMoveInternal;
		} else if (opcode == Instructions.IS_STACK_BOUNDARY) {
			impl = executeIsStackBoundary;
		} else if (opcode == Instructions.DUP) {
			impl = executeDup;
		} else {
			revert("INVALID_OPCODE");
		}

		impl(mach, inst, proof);
	}
}
