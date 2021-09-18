//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "../state/Modules.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProver0 is IOneStepProver {
	function executeUnreachable(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		mach.halted = true;
	}

	function executeNop(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		// :)
	}

	function executeConstPush(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
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

	function executeEqz(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		Value memory v = ValueStacks.pop(mach.valueStack);

		if (v.contents == 0) {
			v.contents = 1;
		} else {
			v.contents = 0;
		}

		ValueStacks.push(mach.valueStack, v);
	}

	function executeDrop(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		ValueStacks.pop(mach.valueStack);
	}

	function executeSelect(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		uint32 selector = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		Value memory b = ValueStacks.pop(mach.valueStack);
		Value memory a = ValueStacks.pop(mach.valueStack);

		if (selector != 0) {
			ValueStacks.push(mach.valueStack, a);
		} else {
			ValueStacks.push(mach.valueStack, b);
		}
	}

	function signExtend(uint32 a) internal pure returns (uint64) {
		if (a & (1<<31) != 0) {
			return uint64(a) | uint64(0xffffffff00000000);
		}
		return uint64(a);
	}

	function I64RelOp(uint64 a, uint64 b, uint16 relop) internal pure returns (bool) {
		if (relop == Instructions.IRELOP_EQ) {
			return (a == b);
		} else if (relop == Instructions.IRELOP_NE) {
			return (a != b);
		} else if (relop == Instructions.IRELOP_LT_S) {
			return (int64(a) < int64(b));
		} else if (relop == Instructions.IRELOP_LT_U) {
			return (a < b);
		} else if (relop == Instructions.IRELOP_GT_S) {
			return (int64(a) > int64(b));
		} else if (relop == Instructions.IRELOP_GT_U) {
			return (a > b);
		} else if (relop == Instructions.IRELOP_LE_S) {
			return (int64(a) <= int64(b));
		} else if (relop == Instructions.IRELOP_LE_U) {
			return (a <= b);
		} else if (relop == Instructions.IRELOP_GE_S) {
			return (int64(a) >= int64(b));
		} else if (relop == Instructions.IRELOP_GE_U) {
			return (a >= b);
		} else {
			revert ("BAD IRELOP");
		}
	}

	function executeI32RelOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint32 b = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		uint32 a = Values.assumeI32(ValueStacks.pop(mach.valueStack));

		uint16 relop = inst.opcode - Instructions.I32_RELOP_BASE;
		uint64 a64;
		uint64 b64;

		if (relop == Instructions.IRELOP_LT_S || relop == Instructions.IRELOP_GT_S ||
			relop == Instructions.IRELOP_LE_S || relop == Instructions.IRELOP_GE_S) {
			a64 = signExtend(a);
			b64 = signExtend(b);
		} else {
			a64 = uint64(a);
			b64 = uint64(b);
		}

		bool res = I64RelOp(a64, b64, relop);

		ValueStacks.push(mach.valueStack, Values.newBoolean(res));
	}

	function executeI64RelOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint64 b = Values.assumeI64(ValueStacks.pop(mach.valueStack));
		uint64 a = Values.assumeI64(ValueStacks.pop(mach.valueStack));

		uint16 relop = inst.opcode - Instructions.I64_RELOP_BASE;

		bool res = I64RelOp(a, b, relop);

		ValueStacks.push(mach.valueStack, Values.newBoolean(res));
	}


	function genericIUnOp(uint64 a, uint16 unop, uint16 bits) internal pure returns (uint32) {
		require(bits == 32 || bits == 64, "WRONG USE OF genericUnOp");
		if (unop == Instructions.IUNOP_CLZ) {
			/* curbits is one-based to keep with unsigned mathematics */
			uint32 curbit = bits;
			while (curbit > 0 && (a & (1 << (curbit - 1)) == 0)) {
				curbit -= 1;
			}
			return (bits - curbit);
		} else if (unop == Instructions.IUNOP_CTZ) {
			uint32 curbit = 0;
			while (curbit < bits && ((a & (1 << curbit)) == 0)) {
				curbit += 1;
			}
			return curbit;
		} else if (unop == Instructions.IUNOP_POPCNT) {
			uint32 curbit = 0;
			uint32 res = 0;
			while (curbit < bits) {
				if ((a & (1 << curbit)) != 0) {
					res += 1;
				}
				curbit++;
			}
			return res;
		}
		revert("BAD IUnOp");
	}

	function executeI32UnOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint32 a = Values.assumeI32(ValueStacks.pop(mach.valueStack));

		uint16 unop = inst.opcode - Instructions.I32_UNOP_BASE;

		uint32 res = genericIUnOp(a, unop, 32);

		ValueStacks.push(mach.valueStack, Values.newI32(res));
	}

	function executeI64UnOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint64 a = Values.assumeI64(ValueStacks.pop(mach.valueStack));

		uint16 unop = inst.opcode - Instructions.I64_UNOP_BASE;

		uint64 res = uint64(genericIUnOp(a, unop, 64));

		ValueStacks.push(mach.valueStack, Values.newI64(res));
	}

	function rotl32(uint32 a, uint32 b) internal pure returns (uint32) {
		b %= 32;
		return (a << b) | (a >> (32 - b));
	}

	function rotl64(uint64 a, uint64 b) internal pure returns (uint64) {
		b %= 64;
		return (a << b) | (a >> (64 - b));
	}

	function rotr32(uint32 a, uint32 b) internal pure returns (uint32) {
		b %= 32;
		return (a >> b) | (a << (32 - b));
	}

	function rotr64(uint64 a, uint64 b) internal pure returns (uint64) {
		b %= 64;
		return (a >> b) | (a << (64 - b));
	}

	function genericBinOp(uint64 a, uint64 b, uint16 opcodeOffset) internal pure returns (uint64) {
		unchecked {
			if (opcodeOffset == 0) {
				// add
				return a + b;
			} else if (opcodeOffset == 1) {
				// sub
				return a - b;
			} else if (opcodeOffset == 2) {
				// mul
				return a * b;
			} else if (opcodeOffset == 4) {
				// div_u
				if (b == 0) {
					return 0;
				}
				return a / b;
			} else if (opcodeOffset == 6) {
				// rem_u
				if (b == 0) {
					return 0;
				}
				return a % b;
			} else if (opcodeOffset == 7) {
				// and
				return a & b;
			} else if (opcodeOffset == 8) {
				// or
				return a | b;
			} else if (opcodeOffset == 9) {
				// xor
				return a ^ b;
			} else {
				revert("INVALID_GENERIC_BIN_OP");
			}
		}
	}

	function executeI32BinOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint32 b = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		uint32 a = Values.assumeI32(ValueStacks.pop(mach.valueStack));
		uint32 res;

		uint16 opcodeOffset = inst.opcode - Instructions.I32_ADD;

		unchecked {
			if (opcodeOffset == 3) {
				// div_s
				if (b == 0) {
					res = 0;
				} else {
					res = uint32(int32(a) / int32(b));
				}
			} else if (opcodeOffset == 5) {
				// rem_s
				if (b == 0) {
					res = 0;
				} else {
					res = uint32(int32(a) % int32(b));
				}
			} else if (opcodeOffset == 10) {
				// shl
				res = a << (b % 32);
			} else if (opcodeOffset == 12) {
				// shr_u
				res = a >> (b % 32);
			} else if (opcodeOffset == 11) {
				// shr_s
				res = uint32(int32(a) >> b);
			} else if (opcodeOffset == 13) {
				// rotl
				res = rotl32(a, b);
			} else if (opcodeOffset == 14) {
				// rotr
				res = rotr32(a, b);
			} else {
				res = uint32(genericBinOp(a, b, opcodeOffset));
			}
		}

		ValueStacks.push(mach.valueStack, Values.newI32(res));
	}

	function executeI64BinOp(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint64 b = Values.assumeI64(ValueStacks.pop(mach.valueStack));
		uint64 a = Values.assumeI64(ValueStacks.pop(mach.valueStack));
		uint64 res;

		uint16 opcodeOffset = inst.opcode - Instructions.I64_ADD;

		unchecked {
			if (opcodeOffset == 3) {
				// div_s
				if (b == 0) {
					res = 0;
				} else {
					res = uint64(int64(a) / int64(b));
				}
			} else if (opcodeOffset == 5) {
				// rem_s
				if (b == 0) {
					res = 0;
				} else {
					res = uint64(int64(a) % int64(b));
				}
			} else if (opcodeOffset == 10) {
				// shl
				res = a << (b % 64);
			} else if (opcodeOffset == 12) {
				// shr_u
				res = a >> (b % 64);
			} else if (opcodeOffset == 11) {
				// shr_s
				res = uint64(int64(a) >> b);
			} else if (opcodeOffset == 13) {
				// rotl
				res = rotl64(a, b);
			} else if (opcodeOffset == 14) {
				// rotr
				res = rotr64(a, b);
			} else {
				res = genericBinOp(a, b, opcodeOffset);
			}
		}

		ValueStacks.push(mach.valueStack, Values.newI64(res));
	}

	function executeI32WrapI64(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		uint64 a = Values.assumeI64(ValueStacks.pop(mach.valueStack));

		uint32 a32 = uint32(a);

		ValueStacks.push(mach.valueStack, Values.newI32(a32));
	}

	function executeI64ExtendI32(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint32 a = Values.assumeI32(ValueStacks.pop(mach.valueStack));

		uint64 a64;

		if (inst.opcode == Instructions.I64_EXTEND_I32_S) {
			a64 = signExtend(a);
		} else {
			a64 = uint64(a);
		}

		ValueStacks.push(mach.valueStack, Values.newI64(a64));
	}

	function executeBlock(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		uint32 targetPc = uint32(inst.argumentData);
		require(targetPc == inst.argumentData, "BAD_BLOCK_PC");
		PcStacks.push(mach.blockStack, targetPc);
	}

	function executeBranch(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		mach.functionPc = PcStacks.pop(mach.blockStack);
	}

	function executeBranchIf(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		Value memory cond = ValueStacks.pop(mach.valueStack);
		if (cond.contents != 0) {
			// Jump to target
			mach.functionPc = PcStacks.pop(mach.blockStack);
		}
	}

	function executeReturn(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		StackFrame memory frame = StackFrames.pop(mach.frameStack);
		if (frame.returnPc.valueType == ValueType.REF_NULL) {
			mach.halted = true;
			return;
		} else if (frame.returnPc.valueType != ValueType.INTERNAL_REF) {
			revert("INVALID_RETURN_PC_TYPE");
		}
		uint256 data = frame.returnPc.contents;
		uint32 pc = uint32(data);
		uint32 func = uint32(data >> 32);
		uint32 mod = uint32(data >> 64);
		require(data >> 96 == 0, "INVALID_RETURN_PC_DATA");
		mach.functionPc = pc;
		mach.functionIdx = func;
		mach.moduleIdx = mod;
	}

	function createReturnValue(Machine memory mach) internal pure returns (Value memory) {
		uint256 returnData = 0;
		returnData |= mach.functionPc;
		returnData |= uint256(mach.functionIdx) << 32;
		returnData |= uint256(mach.moduleIdx) << 64;
		return Value({
			valueType: ValueType.INTERNAL_REF,
			contents: returnData
		});
	}

	function executeCall(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		// Push the return pc to the stack
		ValueStacks.push(mach.valueStack, createReturnValue(mach));

		// Jump to the target
		uint32 idx = uint32(inst.argumentData);
		require(idx == inst.argumentData, "BAD_CALL_DATA");
		mach.functionIdx = idx;
		mach.functionPc = 0;
	}

	function executeCrossModuleCall(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		// Push the return pc to the stack
		ValueStacks.push(mach.valueStack, createReturnValue(mach));

		// Jump to the target
		uint32 func = uint32(inst.argumentData);
		uint32 module = uint32(inst.argumentData >> 32);
		require(inst.argumentData >> 64 == 0, "BAD_CROSS_MODULE_CALL_DATA");
		mach.moduleIdx = module;
		mach.functionIdx = func;
		mach.functionPc = 0;
	}

	function executeCallIndirect(Machine memory mach, Module memory mod, Instruction calldata inst, bytes calldata proof) internal pure {
		uint32 funcIdx;
		{
			uint32 elementIdx = Values.assumeI32(ValueStacks.pop(mach.valueStack));

			// Prove metadata about the instruction and tables
			bytes32 elemsRoot;
			bytes32 wantedFuncTypeHash;
			uint256 offset = 0;
			{
				uint64 tableIdx;
				uint8 tableType;
				uint64 tableSize;
				MerkleProof memory tableMerkleProof;
				(tableIdx, offset) = Deserialize.u64(proof, offset);
				(wantedFuncTypeHash, offset) = Deserialize.b32(proof, offset);
				(tableType, offset) = Deserialize.u8(proof, offset);
				(tableSize, offset) = Deserialize.u64(proof, offset);
				(elemsRoot, offset) = Deserialize.b32(proof, offset);
				(tableMerkleProof, offset) = Deserialize.merkleProof(proof, offset);

				// Validate the information by recomputing known hashes
				bytes32 recomputed = keccak256(abi.encodePacked("Call indirect:", tableIdx, wantedFuncTypeHash));
				require(recomputed == bytes32(inst.argumentData), "BAD_CALL_INDIRECT_DATA");
				recomputed = MerkleProofs.computeRootFromTable(tableMerkleProof, tableIdx, tableType, tableSize, elemsRoot);
				require(recomputed == mod.tablesMerkleRoot, "BAD_TABLES_ROOT");

				// Check if the table access is out of bounds
				if (elementIdx >= tableSize) {
					mach.halted = true;
					return;
				}
			}

			bytes32 elemFuncTypeHash;
			Value memory functionPointer;
			MerkleProof memory elementMerkleProof;
			(elemFuncTypeHash, offset) = Deserialize.b32(proof, offset);
			(functionPointer, offset) = Deserialize.value(proof, offset);
			(elementMerkleProof, offset) = Deserialize.merkleProof(proof, offset);
			bytes32 recomputedElemRoot = MerkleProofs.computeRootFromElement(elementMerkleProof, elementIdx, elemFuncTypeHash, functionPointer);
			require(recomputedElemRoot == elemsRoot, "BAD_ELEMENTS_ROOT");

			if (elemFuncTypeHash != wantedFuncTypeHash) {
				mach.halted = true;
				return;
			}

			if (functionPointer.valueType == ValueType.REF_NULL) {
				mach.halted = true;
				return;
			} else if (functionPointer.valueType == ValueType.FUNC_REF) {
				funcIdx = uint32(functionPointer.contents);
				require(funcIdx == functionPointer.contents, "BAD_FUNC_REF_CONTENTS");
			} else {
				revert("BAD_ELEM_TYPE");
			}
		}

		// Push the return pc to the stack
		ValueStacks.push(mach.valueStack, createReturnValue(mach));

		// Jump to the target
		mach.functionIdx = funcIdx;
		mach.functionPc = 0;
	}

	function executeArbitraryJumpIf(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		Value memory cond = ValueStacks.pop(mach.valueStack);
		if (cond.contents != 0) {
			// Jump to target
			uint32 pc = uint32(inst.argumentData);
			require(pc == inst.argumentData, "BAD_CALL_DATA");
			mach.functionPc = pc;
		}
	}

	function merkleProveGetValue(bytes32 merkleRoot, uint256 index, bytes calldata proof) internal pure returns (Value memory) {
		uint256 offset = 0;
		Value memory proposedVal;
		MerkleProof memory merkle;
		(proposedVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRootFromValue(merkle, index, proposedVal);
		require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
		return proposedVal;
	}

	function merkleProveSetValue(bytes32 merkleRoot, uint256 index, Value memory newVal, bytes calldata proof) internal pure returns (bytes32) {
		Value memory oldVal;
		uint256 offset = 0;
		MerkleProof memory merkle;
		(oldVal, offset) = Deserialize.value(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRootFromValue(merkle, index, oldVal);
		require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
		return MerkleProofs.computeRootFromValue(merkle, index, newVal);
	}

	function executeLocalGet(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata proof) internal pure {
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		Value memory val = merkleProveGetValue(frame.localsMerkleRoot, inst.argumentData, proof);
		ValueStacks.push(mach.valueStack, val);
	}

	function executeLocalSet(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata proof) internal pure {
		Value memory newVal = ValueStacks.pop(mach.valueStack);
		StackFrame memory frame = StackFrames.peek(mach.frameStack);
		frame.localsMerkleRoot = merkleProveSetValue(frame.localsMerkleRoot, inst.argumentData, newVal, proof);
	}

	function executeGlobalGet(Machine memory mach, Module memory mod, Instruction calldata inst, bytes calldata proof) internal pure {
		Value memory val = merkleProveGetValue(mod.globalsMerkleRoot, inst.argumentData, proof);
		ValueStacks.push(mach.valueStack, val);
	}

	function executeGlobalSet(Machine memory mach, Module memory mod, Instruction calldata inst, bytes calldata proof) internal pure {
		Value memory newVal = ValueStacks.pop(mach.valueStack);
		mod.globalsMerkleRoot = merkleProveSetValue(mod.globalsMerkleRoot, inst.argumentData, newVal, proof);
	}

	function executeEndBlock(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		PcStacks.pop(mach.blockStack);
	}

	function executeEndBlockIf(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		Value memory cond = ValueStacks.peek(mach.valueStack);
		if (cond.contents != 0) {
			PcStacks.pop(mach.blockStack);
		}
	}

	function executeInitFrame(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
		Value memory returnPc = ValueStacks.pop(mach.valueStack);
		StackFrame memory newFrame = StackFrame({
			returnPc: returnPc,
			localsMerkleRoot: bytes32(inst.argumentData)
		});
		StackFrames.push(mach.frameStack, newFrame);
	}

	function executeMoveInternal(Machine memory mach, Module memory, Instruction calldata inst, bytes calldata) internal pure {
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

	function executeIsStackBoundary(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
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

	function executeDup(Machine memory mach, Module memory, Instruction calldata, bytes calldata) internal pure {
		Value memory val = ValueStacks.peek(mach.valueStack);
		ValueStacks.push(mach.valueStack, val);
	}

	function handleTrap(Machine memory mach) internal pure {
		mach.halted = true;
	}

	function executeOneStep(Machine calldata startMach, Module calldata startMod, Instruction calldata inst, bytes calldata proof) override view external returns (Machine memory mach, Module memory mod) {
		mach = startMach;
		mod = startMod;

		uint16 opcode = inst.opcode;

		function(Machine memory, Module memory, Instruction calldata, bytes calldata) internal view impl;
		if (opcode == Instructions.UNREACHABLE) {
			impl = executeUnreachable;
		} else if (opcode == Instructions.NOP) {
			impl = executeNop;
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
		} else if (opcode == Instructions.CROSS_MODULE_CALL) {
			impl = executeCrossModuleCall;
		} else if (opcode == Instructions.CALL_INDIRECT) {
			impl = executeCallIndirect;
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
		} else if (opcode == Instructions.SELECT) {
			impl = executeSelect;
		} else if (opcode == Instructions.I32_EQZ) {
			impl = executeEqz;
		} else if (opcode >= Instructions.I32_CONST && opcode <= Instructions.F64_CONST || opcode == Instructions.PUSH_STACK_BOUNDARY) {
			impl = executeConstPush;
		} else if (opcode >= Instructions.I32_RELOP_BASE && opcode <= Instructions.I32_RELOP_BASE + Instructions.IRELOP_LAST) {
			impl = executeI32RelOp;
		} else if (opcode >= Instructions.I32_UNOP_BASE && opcode <= Instructions.I32_UNOP_BASE + Instructions.IUNOP_LAST) {
			impl = executeI32UnOp;
		} else if (opcode >= Instructions.I32_ADD && opcode <= Instructions.I32_ROTR) {
			impl = executeI32BinOp;
		} else if (opcode >= Instructions.I64_RELOP_BASE && opcode <= Instructions.I64_RELOP_BASE + Instructions.IRELOP_LAST) {
			impl = executeI64RelOp;
		} else if (opcode >= Instructions.I64_UNOP_BASE && opcode <= Instructions.I64_UNOP_BASE + Instructions.IUNOP_LAST) {
			impl = executeI64UnOp;
		} else if (opcode >= Instructions.I64_ADD && opcode <= Instructions.I64_ROTR) {
			impl = executeI64BinOp;
		} else if (opcode == Instructions.I32_WRAP_I64) {
			impl = executeI32WrapI64;
		} else if (opcode == Instructions.I64_EXTEND_I32_S || opcode == Instructions.I64_EXTEND_I32_U) {
			impl = executeI64ExtendI32;
		} else if (opcode == Instructions.MOVE_FROM_STACK_TO_INTERNAL || opcode == Instructions.MOVE_FROM_INTERNAL_TO_STACK) {
			impl = executeMoveInternal;
		} else if (opcode == Instructions.IS_STACK_BOUNDARY) {
			impl = executeIsStackBoundary;
		} else if (opcode == Instructions.DUP) {
			impl = executeDup;
		} else {
			revert("INVALID_OPCODE");
		}

		impl(mach, mod, inst, proof);
	}
}
