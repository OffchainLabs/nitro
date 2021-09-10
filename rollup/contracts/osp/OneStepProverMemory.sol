//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProverMemory is IOneStepProver {
    uint256 constant LEAF_SIZE = 32;

	function pullByte(bytes32 source, uint256 idx) internal pure returns (uint8) {
		require(idx < LEAF_SIZE, "BAD_PULL_BYTE_IDX");
		// Take into account that we are casting the source to a big-endian integer
		uint256 leafShift = (LEAF_SIZE - 1 - idx) * 8;
		return uint8(uint256(source) >> leafShift);
	}

    function executeMemoryLoad(
        Machine memory mach,
        Instruction memory inst,
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
        uint256 startIdx = inst.argumentData +
            ValueStacks.pop(mach.valueStack).contents;
        if (startIdx + readBytes > mach.machineMemory.size) {
            mach.halted = true;
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
                (lastProvedLeafContents, proofOffset) = MachineMemories
                    .proveLeaf(mach.machineMemory, leafIdx, proof, proofOffset);
                lastProvedLeafIdx = leafIdx;
            }
            uint256 indexWithinLeaf = idx % LEAF_SIZE;
			readValue |= uint64(pullByte(lastProvedLeafContents, indexWithinLeaf)) << uint64(i * 8);
        }

		if (signed) {
			// Go down to the original uint size, change to signed, go up to correct size, convert back to unsigned
			if (readBytes == 1) {
				readValue = uint64(int64(int8(uint8(readValue))));
			} else if (readBytes == 2) {
				readValue = uint64(int64(int16(uint16(readValue))));
			} else if (readBytes == 4) {
				readValue = uint64(int64(int32(uint32(readValue))));
			} else {
				revert("BAD_READ_BYTES_SIGNED");
			}
		}

		ValueStacks.push(mach.valueStack, Value({
			valueType: ty,
			contents: readValue
		}));
    }

    function executeOneStep(Machine calldata startMach, bytes calldata proof)
        external
        view
        override
        returns (Machine memory mach)
    {
        mach = startMach;

        Instruction memory inst = Instructions.pop(mach.instructions);
        uint16 opcode = inst.opcode;

        function(Machine memory, Instruction memory, bytes calldata)
            internal
            view impl;
        if (
            opcode >= Instructions.I32_LOAD &&
            opcode <= Instructions.I64_LOAD32_U
        ) {
            impl = executeMemoryLoad;
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }

        impl(mach, inst, proof);
    }
}
