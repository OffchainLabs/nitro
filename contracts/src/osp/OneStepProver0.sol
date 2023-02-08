// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/Value.sol";
import "../state/Machine.sol";
import "../state/Module.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProver0 is IOneStepProver {
    using MachineLib for Machine;
    using MerkleProofLib for MerkleProof;
    using StackFrameLib for StackFrameWindow;
    using ValueLib for Value;
    using ValueStackLib for ValueStack;

    function executeUnreachable(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        mach.status = MachineStatus.ERRORED;
    }

    function executeNop(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        // :)
    }

    function executeConstPush(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
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
        } else {
            revert("CONST_PUSH_INVALID_OPCODE");
        }

        mach.valueStack.push(Value({valueType: ty, contents: uint64(inst.argumentData)}));
    }

    function executeDrop(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        mach.valueStack.pop();
    }

    function executeSelect(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        uint32 selector = mach.valueStack.pop().assumeI32();
        Value memory b = mach.valueStack.pop();
        Value memory a = mach.valueStack.pop();

        if (selector != 0) {
            mach.valueStack.push(a);
        } else {
            mach.valueStack.push(b);
        }
    }

    function executeReturn(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        StackFrame memory frame = mach.frameStack.pop();
        mach.setPc(frame.returnPc);
    }

    function createReturnValue(Machine memory mach) internal pure returns (Value memory) {
        return ValueLib.newPc(mach.functionPc, mach.functionIdx, mach.moduleIdx);
    }

    function executeCall(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller module info to the stack
        StackFrame memory frame = mach.frameStack.peek();
        mach.valueStack.push(ValueLib.newI32(frame.callerModule));
        mach.valueStack.push(ValueLib.newI32(frame.callerModuleInternals));

        // Jump to the target
        uint32 idx = uint32(inst.argumentData);
        require(idx == inst.argumentData, "BAD_CALL_DATA");
        mach.functionIdx = idx;
        mach.functionPc = 0;
    }

    function executeCrossModuleCall(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller module info to the stack
        mach.valueStack.push(ValueLib.newI32(mach.moduleIdx));
        mach.valueStack.push(ValueLib.newI32(mod.internalsOffset));

        // Jump to the target
        uint32 func = uint32(inst.argumentData);
        uint32 module = uint32(inst.argumentData >> 32);
        require(inst.argumentData >> 64 == 0, "BAD_CROSS_MODULE_CALL_DATA");
        mach.moduleIdx = module;
        mach.functionIdx = func;
        mach.functionPc = 0;
    }

    function executeCrossModuleForward(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller's caller module info to the stack
        StackFrame memory frame = mach.frameStack.peek();
        mach.valueStack.push(ValueLib.newI32(frame.callerModule));
        mach.valueStack.push(ValueLib.newI32(frame.callerModuleInternals));

        // Jump to the target
        uint32 func = uint32(inst.argumentData);
        uint32 module = uint32(inst.argumentData >> 32);
        require(inst.argumentData >> 64 == 0, "BAD_CROSS_MODULE_CALL_DATA");
        mach.moduleIdx = module;
        mach.functionIdx = func;
        mach.functionPc = 0;
    }

    function executeCrossModuleDynamicCall(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Get the target from the stack
        uint32 func = mach.valueStack.pop().assumeI32();
        uint32 module = mach.valueStack.pop().assumeI32();

        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller module info to the stack
        mach.valueStack.push(ValueLib.newI32(mach.moduleIdx));
        mach.valueStack.push(ValueLib.newI32(mod.internalsOffset));

        // Jump to the target
        mach.moduleIdx = module;
        mach.functionIdx = func;
        mach.functionPc = 0;
    }

    function executeCallerModuleInternalCall(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller module info to the stack
        mach.valueStack.push(ValueLib.newI32(mach.moduleIdx));
        mach.valueStack.push(ValueLib.newI32(mod.internalsOffset));

        StackFrame memory frame = mach.frameStack.peek();
        if (frame.callerModuleInternals == 0) {
            // The caller module has no internals
            mach.status = MachineStatus.ERRORED;
            return;
        }

        // Jump to the target
        uint32 offset = uint32(inst.argumentData);
        require(offset == inst.argumentData, "BAD_CALLER_INTERNAL_CALL_DATA");
        mach.moduleIdx = frame.callerModule;
        mach.functionIdx = frame.callerModuleInternals + offset;
        mach.functionPc = 0;
    }

    function executeCallIndirect(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint32 funcIdx;
        {
            uint32 elementIdx = mach.valueStack.pop().assumeI32();

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
                bytes32 recomputed = keccak256(
                    abi.encodePacked("Call indirect:", tableIdx, wantedFuncTypeHash)
                );
                require(recomputed == bytes32(inst.argumentData), "BAD_CALL_INDIRECT_DATA");
                recomputed = tableMerkleProof.computeRootFromTable(
                    tableIdx,
                    tableType,
                    tableSize,
                    elemsRoot
                );
                require(recomputed == mod.tablesMerkleRoot, "BAD_TABLES_ROOT");

                // Check if the table access is out of bounds
                if (elementIdx >= tableSize) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
            }

            bytes32 elemFuncTypeHash;
            Value memory functionPointer;
            MerkleProof memory elementMerkleProof;
            (elemFuncTypeHash, offset) = Deserialize.b32(proof, offset);
            (functionPointer, offset) = Deserialize.value(proof, offset);
            (elementMerkleProof, offset) = Deserialize.merkleProof(proof, offset);
            bytes32 recomputedElemRoot = elementMerkleProof.computeRootFromElement(
                elementIdx,
                elemFuncTypeHash,
                functionPointer
            );
            require(recomputedElemRoot == elemsRoot, "BAD_ELEMENTS_ROOT");

            if (elemFuncTypeHash != wantedFuncTypeHash) {
                mach.status = MachineStatus.ERRORED;
                return;
            }

            if (functionPointer.valueType == ValueType.REF_NULL) {
                mach.status = MachineStatus.ERRORED;
                return;
            } else if (functionPointer.valueType == ValueType.FUNC_REF) {
                funcIdx = uint32(functionPointer.contents);
                require(funcIdx == functionPointer.contents, "BAD_FUNC_REF_CONTENTS");
            } else {
                revert("BAD_ELEM_TYPE");
            }
        }

        // Push the return pc to the stack
        mach.valueStack.push(createReturnValue(mach));

        // Push caller module info to the stack
        StackFrame memory frame = mach.frameStack.peek();
        mach.valueStack.push(ValueLib.newI32(frame.callerModule));
        mach.valueStack.push(ValueLib.newI32(frame.callerModuleInternals));

        // Jump to the target
        mach.functionIdx = funcIdx;
        mach.functionPc = 0;
    }

    function executeArbitraryJump(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        // Jump to target
        uint32 pc = uint32(inst.argumentData);
        require(pc == inst.argumentData, "BAD_CALL_DATA");
        mach.functionPc = pc;
    }

    function executeArbitraryJumpIf(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint32 cond = mach.valueStack.pop().assumeI32();
        if (cond != 0) {
            // Jump to target
            uint32 pc = uint32(inst.argumentData);
            require(pc == inst.argumentData, "BAD_CALL_DATA");
            mach.functionPc = pc;
        }
    }

    function merkleProveGetValue(
        bytes32 merkleRoot,
        uint256 index,
        bytes calldata proof
    ) internal pure returns (Value memory) {
        uint256 offset = 0;
        Value memory proposedVal;
        MerkleProof memory merkle;
        (proposedVal, offset) = Deserialize.value(proof, offset);
        (merkle, offset) = Deserialize.merkleProof(proof, offset);
        bytes32 recomputedRoot = merkle.computeRootFromValue(index, proposedVal);
        require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
        return proposedVal;
    }

    function merkleProveSetValue(
        bytes32 merkleRoot,
        uint256 index,
        Value memory newVal,
        bytes calldata proof
    ) internal pure returns (bytes32) {
        Value memory oldVal;
        uint256 offset = 0;
        MerkleProof memory merkle;
        (oldVal, offset) = Deserialize.value(proof, offset);
        (merkle, offset) = Deserialize.merkleProof(proof, offset);
        bytes32 recomputedRoot = merkle.computeRootFromValue(index, oldVal);
        require(recomputedRoot == merkleRoot, "WRONG_MERKLE_ROOT");
        return merkle.computeRootFromValue(index, newVal);
    }

    function executeLocalGet(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        StackFrame memory frame = mach.frameStack.peek();
        Value memory val = merkleProveGetValue(frame.localsMerkleRoot, inst.argumentData, proof);
        mach.valueStack.push(val);
    }

    function executeLocalSet(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        Value memory newVal = mach.valueStack.pop();
        StackFrame memory frame = mach.frameStack.peek();
        frame.localsMerkleRoot = merkleProveSetValue(
            frame.localsMerkleRoot,
            inst.argumentData,
            newVal,
            proof
        );
    }

    function executeGlobalGet(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        Value memory val = merkleProveGetValue(mod.globalsMerkleRoot, inst.argumentData, proof);
        mach.valueStack.push(val);
    }

    function executeGlobalSet(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        Value memory newVal = mach.valueStack.pop();
        mod.globalsMerkleRoot = merkleProveSetValue(
            mod.globalsMerkleRoot,
            inst.argumentData,
            newVal,
            proof
        );
    }

    function executeInitFrame(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        Value memory callerModuleInternals = mach.valueStack.pop();
        Value memory callerModule = mach.valueStack.pop();
        Value memory returnPc = mach.valueStack.pop();
        StackFrame memory newFrame = StackFrame({
            returnPc: returnPc,
            localsMerkleRoot: bytes32(inst.argumentData),
            callerModule: callerModule.assumeI32(),
            callerModuleInternals: callerModuleInternals.assumeI32()
        });
        mach.frameStack.push(newFrame);
    }

    function executeMoveInternal(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        Value memory val;
        if (inst.opcode == Instructions.MOVE_FROM_STACK_TO_INTERNAL) {
            val = mach.valueStack.pop();
            mach.internalStack.push(val);
        } else if (inst.opcode == Instructions.MOVE_FROM_INTERNAL_TO_STACK) {
            val = mach.internalStack.pop();
            mach.valueStack.push(val);
        } else {
            revert("MOVE_INTERNAL_INVALID_OPCODE");
        }
    }

    function executeDup(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        Value memory val = mach.valueStack.peek();
        mach.valueStack.push(val);
    }

    function executeOneStep(
        ExecutionContext calldata,
        Machine calldata startMach,
        Module calldata startMod,
        Instruction calldata inst,
        bytes calldata proof
    ) external pure override returns (Machine memory mach, Module memory mod) {
        mach = startMach;
        mod = startMod;

        uint16 opcode = inst.opcode;

        function(Machine memory, Module memory, Instruction calldata, bytes calldata)
            internal
            pure impl;
        if (opcode == Instructions.UNREACHABLE) {
            impl = executeUnreachable;
        } else if (opcode == Instructions.NOP) {
            impl = executeNop;
        } else if (opcode == Instructions.RETURN) {
            impl = executeReturn;
        } else if (opcode == Instructions.CALL) {
            impl = executeCall;
        } else if (opcode == Instructions.CROSS_MODULE_CALL) {
            impl = executeCrossModuleCall;
        } else if (opcode == Instructions.CROSS_MODULE_FORWARD) {
            impl = executeCrossModuleForward;
        } else if (opcode == Instructions.CROSS_MODULE_DYNAMIC_CALL) {
            impl = executeCrossModuleDynamicCall;
        } else if (opcode == Instructions.CALLER_MODULE_INTERNAL_CALL) {
            impl = executeCallerModuleInternalCall;
        } else if (opcode == Instructions.CALL_INDIRECT) {
            impl = executeCallIndirect;
        } else if (opcode == Instructions.ARBITRARY_JUMP) {
            impl = executeArbitraryJump;
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
        } else if (opcode >= Instructions.I32_CONST && opcode <= Instructions.F64_CONST) {
            impl = executeConstPush;
        } else if (
            opcode == Instructions.MOVE_FROM_STACK_TO_INTERNAL ||
            opcode == Instructions.MOVE_FROM_INTERNAL_TO_STACK
        ) {
            impl = executeMoveInternal;
        } else if (opcode == Instructions.DUP) {
            impl = executeDup;
        } else {
            revert("INVALID_OPCODE");
        }

        impl(mach, mod, inst, proof);
    }
}
