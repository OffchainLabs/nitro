//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "../state/Deserialize.sol";
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

	function executeAdd(Machine memory mach, Instruction memory inst, bytes calldata) internal pure {
		Value memory a = ValueStacks.pop(mach.valueStack);
		Value memory b = ValueStacks.pop(mach.valueStack);
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

		ValueStacks.push(mach.valueStack, Value({
			valueType: ty,
			contents: contents
		}));
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

	function executeLocalGet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		Value memory proposedVal;
		uint256 offset = 0;
		MerkleProof memory merkle;
		(proposedVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRoot(merkle, inst.argumentData, proposedVal);
		require(recomputedRoot == frame.localsMerkleRoot, "WRONG_MERKLE_ROOT");
		ValueStacks.push(mach.valueStack, proposedVal);
	}

	function executeLocalSet(Machine memory mach, Instruction memory inst, bytes calldata proof) internal pure {
		Value memory newVal = ValueStacks.pop(mach.valueStack);
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		Value memory oldVal;
		uint256 offset = 0;
		MerkleProof memory merkle;
		(oldVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRoot(merkle, inst.argumentData, oldVal);
		require(recomputedRoot == frame.localsMerkleRoot, "WRONG_MERKLE_ROOT");
		frame.localsMerkleRoot = MerkleProofs.computeRoot(merkle, inst.argumentData, newVal);
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

	function handleTrap(Machine memory mach) internal pure {
		mach.halted = true;
	}

	function executeOneStep(Machine calldata startMach, bytes calldata proof) override view external returns (Machine memory mach) {
		mach = startMach;

		Instruction memory inst = Instructions.pop(mach.instructions);
		uint8 opcode = inst.opcode;

		function(Machine memory, Instruction memory, bytes calldata) internal view impl;
		if (opcode == Instructions.BLOCK) {
			impl = executeBlock;
		} else if (opcode == Instructions.BRANCH) {
			impl = executeBranch;
		} else if (opcode == Instructions.BRANCH_IF) {
			impl = executeBranchIf;
		} else if (opcode == Instructions.END_BLOCK) {
			impl = executeEndBlock;
		} else if (opcode == Instructions.END_BLOCK_IF) {
			impl = executeEndBlockIf;
		} else if (opcode == Instructions.LOCAL_GET) {
			impl = executeLocalGet;
		} else if (opcode == Instructions.LOCAL_SET) {
			impl = executeLocalSet;
		} else if (opcode == Instructions.INIT_FRAME) {
			impl = executeInitFrame;
		} else if (opcode == Instructions.DROP) {
			impl = executeDrop;
		} else if (opcode == Instructions.I32_EQZ) {
			impl = executeEqz;
		} else if (opcode >= Instructions.I32_CONST && opcode <= Instructions.F64_CONST) {
			impl = executeConstPush;
		} else if (opcode == Instructions.I32_ADD || opcode == Instructions.I64_ADD) {
			impl = executeAdd;
		} else {
			revert("TODO: instruction not implemented");
		}

		impl(mach, inst, proof);
	}
}
