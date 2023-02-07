// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/Value.sol";
import "../state/Machine.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProverMemory is IOneStepProver {
    using MerkleProofLib for MerkleProof;
    using ModuleMemoryLib for ModuleMemory;
    using ValueLib for Value;
    using ValueStackLib for ValueStack;

    uint256 private constant LEAF_SIZE = 32;
    uint64 private constant PAGE_SIZE = 65536;

    function pullLeafByte(bytes32 leaf, uint256 idx) internal pure returns (uint8) {
        require(idx < LEAF_SIZE, "BAD_PULL_LEAF_BYTE_IDX");
        // Take into account that we are casting the leaf to a big-endian integer
        uint256 leafShift = (LEAF_SIZE - 1 - idx) * 8;
        return uint8(uint256(leaf) >> leafShift);
    }

    function setLeafByte(
        bytes32 oldLeaf,
        uint256 idx,
        uint8 val
    ) internal pure returns (bytes32) {
        require(idx < LEAF_SIZE, "BAD_SET_LEAF_BYTE_IDX");
        // Take into account that we are casting the leaf to a big-endian integer
        uint256 leafShift = (LEAF_SIZE - 1 - idx) * 8;
        uint256 newLeaf = uint256(oldLeaf);
        newLeaf &= ~(0xFF << leafShift);
        newLeaf |= uint256(val) << leafShift;
        return bytes32(newLeaf);
    }

    function executeMemoryLoad(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        ValueType ty;
        uint256 readBytes;
        bool signed;
        if (inst.opcode == Instructions.I32_LOAD) {
            ty = ValueType.I32;
            readBytes = 4;
            signed = false;
        } else if (inst.opcode == Instructions.I64_LOAD) {
            ty = ValueType.I64;
            readBytes = 8;
            signed = false;
        } else if (inst.opcode == Instructions.F32_LOAD) {
            ty = ValueType.F32;
            readBytes = 4;
            signed = false;
        } else if (inst.opcode == Instructions.F64_LOAD) {
            ty = ValueType.F64;
            readBytes = 8;
            signed = false;
        } else if (inst.opcode == Instructions.I32_LOAD8_S) {
            ty = ValueType.I32;
            readBytes = 1;
            signed = true;
        } else if (inst.opcode == Instructions.I32_LOAD8_U) {
            ty = ValueType.I32;
            readBytes = 1;
            signed = false;
        } else if (inst.opcode == Instructions.I32_LOAD16_S) {
            ty = ValueType.I32;
            readBytes = 2;
            signed = true;
        } else if (inst.opcode == Instructions.I32_LOAD16_U) {
            ty = ValueType.I32;
            readBytes = 2;
            signed = false;
        } else if (inst.opcode == Instructions.I64_LOAD8_S) {
            ty = ValueType.I64;
            readBytes = 1;
            signed = true;
        } else if (inst.opcode == Instructions.I64_LOAD8_U) {
            ty = ValueType.I64;
            readBytes = 1;
            signed = false;
        } else if (inst.opcode == Instructions.I64_LOAD16_S) {
            ty = ValueType.I64;
            readBytes = 2;
            signed = true;
        } else if (inst.opcode == Instructions.I64_LOAD16_U) {
            ty = ValueType.I64;
            readBytes = 2;
            signed = false;
        } else if (inst.opcode == Instructions.I64_LOAD32_S) {
            ty = ValueType.I64;
            readBytes = 4;
            signed = true;
        } else if (inst.opcode == Instructions.I64_LOAD32_U) {
            ty = ValueType.I64;
            readBytes = 4;
            signed = false;
        } else {
            revert("INVALID_MEMORY_LOAD_OPCODE");
        }

        // Neither of these can overflow as they're computed with much less than 256 bit integers.
        uint256 startIdx = inst.argumentData + mach.valueStack.pop().assumeI32();
        if (startIdx + readBytes > mod.moduleMemory.size) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        uint256 proofOffset = 0;
        uint256 lastProvedLeafIdx = ~uint256(0);
        bytes32 lastProvedLeafContents;
        uint64 readValue;
        for (uint256 i = 0; i < readBytes; i++) {
            uint256 idx = startIdx + i;
            uint256 leafIdx = idx / LEAF_SIZE;
            if (leafIdx != lastProvedLeafIdx) {
                // This hits the stack size if we phrase it as mod.moduleMemory.proveLeaf(...)
                (lastProvedLeafContents, proofOffset, ) = ModuleMemoryLib.proveLeaf(
                    mod.moduleMemory,
                    leafIdx,
                    proof,
                    proofOffset
                );
                lastProvedLeafIdx = leafIdx;
            }
            uint256 indexWithinLeaf = idx % LEAF_SIZE;
            readValue |=
                uint64(pullLeafByte(lastProvedLeafContents, indexWithinLeaf)) <<
                uint64(i * 8);
        }

        if (signed) {
            // Go down to the original uint size, change to signed, go up to correct size, convert back to unsigned
            if (readBytes == 1 && ty == ValueType.I32) {
                readValue = uint32(int32(int8(uint8(readValue))));
            } else if (readBytes == 1 && ty == ValueType.I64) {
                readValue = uint64(int64(int8(uint8(readValue))));
            } else if (readBytes == 2 && ty == ValueType.I32) {
                readValue = uint32(int32(int16(uint16(readValue))));
            } else if (readBytes == 2 && ty == ValueType.I64) {
                readValue = uint64(int64(int16(uint16(readValue))));
            } else if (readBytes == 4 && ty == ValueType.I64) {
                readValue = uint64(int64(int32(uint32(readValue))));
            } else {
                revert("BAD_READ_BYTES_SIGNED");
            }
        }

        mach.valueStack.push(Value({valueType: ty, contents: readValue}));
    }

    function executeMemoryStore(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint64 writeBytes;
        uint64 toWrite;
        {
            ValueType ty;
            if (inst.opcode == Instructions.I32_STORE) {
                ty = ValueType.I32;
                writeBytes = 4;
            } else if (inst.opcode == Instructions.I64_STORE) {
                ty = ValueType.I64;
                writeBytes = 8;
            } else if (inst.opcode == Instructions.F32_STORE) {
                ty = ValueType.F32;
                writeBytes = 4;
            } else if (inst.opcode == Instructions.F64_STORE) {
                ty = ValueType.F64;
                writeBytes = 8;
            } else if (inst.opcode == Instructions.I32_STORE8) {
                ty = ValueType.I32;
                writeBytes = 1;
            } else if (inst.opcode == Instructions.I32_STORE16) {
                ty = ValueType.I32;
                writeBytes = 2;
            } else if (inst.opcode == Instructions.I64_STORE8) {
                ty = ValueType.I64;
                writeBytes = 1;
            } else if (inst.opcode == Instructions.I64_STORE16) {
                ty = ValueType.I64;
                writeBytes = 2;
            } else if (inst.opcode == Instructions.I64_STORE32) {
                ty = ValueType.I64;
                writeBytes = 4;
            } else {
                revert("INVALID_MEMORY_STORE_OPCODE");
            }

            Value memory writingVal = mach.valueStack.pop();
            require(writingVal.valueType == ty, "BAD_STORE_TYPE");
            toWrite = uint64(writingVal.contents);
            if (writeBytes < 8) {
                toWrite &= (uint64(1) << (writeBytes * 8)) - 1;
            }
        }

        // Neither of these can overflow as they're computed with much less than 256 bit integers.
        uint256 startIdx = inst.argumentData + mach.valueStack.pop().assumeI32();
        if (startIdx + writeBytes > mod.moduleMemory.size) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        uint256 proofOffset = 0;
        uint256 lastProvedLeafIdx = ~uint256(0);
        MerkleProof memory lastProvedMerkle;
        bytes32 lastProvedLeafContents;
        for (uint256 i = 0; i < writeBytes; i++) {
            uint256 idx = startIdx + i;
            uint256 leafIdx = idx / LEAF_SIZE;
            if (leafIdx != lastProvedLeafIdx) {
                if (lastProvedLeafIdx != ~uint256(0)) {
                    // Apply the last leaf update
                    mod.moduleMemory.merkleRoot = lastProvedMerkle.computeRootFromMemory(
                        lastProvedLeafIdx,
                        lastProvedLeafContents
                    );
                }
                // This hits the stack size if we phrase it as mod.moduleMemory.proveLeaf(...)
                (lastProvedLeafContents, proofOffset, lastProvedMerkle) = ModuleMemoryLib.proveLeaf(
                    mod.moduleMemory,
                    leafIdx,
                    proof,
                    proofOffset
                );
                lastProvedLeafIdx = leafIdx;
            }
            uint256 indexWithinLeaf = idx % LEAF_SIZE;
            lastProvedLeafContents = setLeafByte(
                lastProvedLeafContents,
                indexWithinLeaf,
                uint8(toWrite)
            );
            toWrite >>= 8;
        }
        mod.moduleMemory.merkleRoot = lastProvedMerkle.computeRootFromMemory(
            lastProvedLeafIdx,
            lastProvedLeafContents
        );
    }

    function executeMemorySize(
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        uint32 pages = uint32(mod.moduleMemory.size / PAGE_SIZE);
        mach.valueStack.push(ValueLib.newI32(pages));
    }

    function executeMemoryGrow(
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        uint32 oldPages = uint32(mod.moduleMemory.size / PAGE_SIZE);
        uint32 growingPages = mach.valueStack.pop().assumeI32();
        // Safe as the input integers are too small to overflow a uint256
        uint256 newSize = uint256(oldPages) + uint256(growingPages);
        if (newSize <= mod.moduleMemory.maxSize) {
            mod.moduleMemory.size = uint64(newSize * PAGE_SIZE);
            mach.valueStack.push(ValueLib.newI32(oldPages));
        } else {
            mach.valueStack.push(ValueLib.newI32(~uint32(0)));
        }
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
        if (opcode >= Instructions.I32_LOAD && opcode <= Instructions.I64_LOAD32_U) {
            impl = executeMemoryLoad;
        } else if (opcode >= Instructions.I32_STORE && opcode <= Instructions.I64_STORE32) {
            impl = executeMemoryStore;
        } else if (opcode == Instructions.MEMORY_SIZE) {
            impl = executeMemorySize;
        } else if (opcode == Instructions.MEMORY_GROW) {
            impl = executeMemoryGrow;
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }

        impl(mach, mod, inst, proof);
    }
}
