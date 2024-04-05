// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/Value.sol";
import "../state/Machine.sol";
import "../state/Module.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProverMath is IOneStepProver {
    using ValueLib for Value;
    using ValueStackLib for ValueStack;

    function executeEqz(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        Value memory v = mach.valueStack.pop();
        if (inst.opcode == Instructions.I32_EQZ) {
            require(v.valueType == ValueType.I32, "NOT_I32");
        } else if (inst.opcode == Instructions.I64_EQZ) {
            require(v.valueType == ValueType.I64, "NOT_I64");
        } else {
            revert("BAD_EQZ");
        }

        uint32 output;
        if (v.contents == 0) {
            output = 1;
        } else {
            output = 0;
        }

        mach.valueStack.push(ValueLib.newI32(output));
    }

    function signExtend(uint32 a) internal pure returns (uint64) {
        if (a & (1 << 31) != 0) {
            return uint64(a) | uint64(0xffffffff00000000);
        }
        return uint64(a);
    }

    function i64RelOp(
        uint64 a,
        uint64 b,
        uint16 relop
    ) internal pure returns (bool) {
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
            revert("BAD IRELOP");
        }
    }

    function executeI32RelOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint32 b = mach.valueStack.pop().assumeI32();
        uint32 a = mach.valueStack.pop().assumeI32();

        uint16 relop = inst.opcode - Instructions.I32_RELOP_BASE;
        uint64 a64;
        uint64 b64;

        if (
            relop == Instructions.IRELOP_LT_S ||
            relop == Instructions.IRELOP_GT_S ||
            relop == Instructions.IRELOP_LE_S ||
            relop == Instructions.IRELOP_GE_S
        ) {
            a64 = signExtend(a);
            b64 = signExtend(b);
        } else {
            a64 = uint64(a);
            b64 = uint64(b);
        }

        bool res = i64RelOp(a64, b64, relop);

        mach.valueStack.push(ValueLib.newBoolean(res));
    }

    function executeI64RelOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint64 b = mach.valueStack.pop().assumeI64();
        uint64 a = mach.valueStack.pop().assumeI64();

        uint16 relop = inst.opcode - Instructions.I64_RELOP_BASE;

        bool res = i64RelOp(a, b, relop);

        mach.valueStack.push(ValueLib.newBoolean(res));
    }

    function genericIUnOp(
        uint64 a,
        uint16 unop,
        uint16 bits
    ) internal pure returns (uint32) {
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

    function executeI32UnOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint32 a = mach.valueStack.pop().assumeI32();

        uint16 unop = inst.opcode - Instructions.I32_UNOP_BASE;

        uint32 res = genericIUnOp(a, unop, 32);

        mach.valueStack.push(ValueLib.newI32(res));
    }

    function executeI64UnOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint64 a = mach.valueStack.pop().assumeI64();

        uint16 unop = inst.opcode - Instructions.I64_UNOP_BASE;

        uint64 res = uint64(genericIUnOp(a, unop, 64));

        mach.valueStack.push(ValueLib.newI64(res));
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

    function genericBinOp(
        uint64 a,
        uint64 b,
        uint16 opcodeOffset
    ) internal pure returns (uint64, bool) {
        unchecked {
            if (opcodeOffset == 0) {
                // add
                return (a + b, false);
            } else if (opcodeOffset == 1) {
                // sub
                return (a - b, false);
            } else if (opcodeOffset == 2) {
                // mul
                return (a * b, false);
            } else if (opcodeOffset == 4) {
                // div_u
                if (b == 0) {
                    return (0, true);
                }
                return (a / b, false);
            } else if (opcodeOffset == 6) {
                // rem_u
                if (b == 0) {
                    return (0, true);
                }
                return (a % b, false);
            } else if (opcodeOffset == 7) {
                // and
                return (a & b, false);
            } else if (opcodeOffset == 8) {
                // or
                return (a | b, false);
            } else if (opcodeOffset == 9) {
                // xor
                return (a ^ b, false);
            } else {
                revert("INVALID_GENERIC_BIN_OP");
            }
        }
    }

    function executeI32BinOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint32 b = mach.valueStack.pop().assumeI32();
        uint32 a = mach.valueStack.pop().assumeI32();
        uint32 res;

        uint16 opcodeOffset = inst.opcode - Instructions.I32_ADD;

        unchecked {
            if (opcodeOffset == 3) {
                // div_s
                if (b == 0 || (int32(a) == -2147483648 && int32(b) == -1)) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
                res = uint32(int32(a) / int32(b));
            } else if (opcodeOffset == 5) {
                // rem_s
                if (b == 0) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
                res = uint32(int32(a) % int32(b));
            } else if (opcodeOffset == 10) {
                // shl
                res = a << (b % 32);
            } else if (opcodeOffset == 12) {
                // shr_u
                res = a >> (b % 32);
            } else if (opcodeOffset == 11) {
                // shr_s
                res = uint32(int32(a) >> (b % 32));
            } else if (opcodeOffset == 13) {
                // rotl
                res = rotl32(a, b);
            } else if (opcodeOffset == 14) {
                // rotr
                res = rotr32(a, b);
            } else {
                (uint64 computed, bool err) = genericBinOp(a, b, opcodeOffset);
                if (err) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
                res = uint32(computed);
            }
        }

        mach.valueStack.push(ValueLib.newI32(res));
    }

    function executeI64BinOp(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint64 b = mach.valueStack.pop().assumeI64();
        uint64 a = mach.valueStack.pop().assumeI64();
        uint64 res;

        uint16 opcodeOffset = inst.opcode - Instructions.I64_ADD;

        unchecked {
            if (opcodeOffset == 3) {
                // div_s
                if (b == 0 || (int64(a) == -9223372036854775808 && int64(b) == -1)) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
                res = uint64(int64(a) / int64(b));
            } else if (opcodeOffset == 5) {
                // rem_s
                if (b == 0) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
                res = uint64(int64(a) % int64(b));
            } else if (opcodeOffset == 10) {
                // shl
                res = a << (b % 64);
            } else if (opcodeOffset == 12) {
                // shr_u
                res = a >> (b % 64);
            } else if (opcodeOffset == 11) {
                // shr_s
                res = uint64(int64(a) >> (b % 64));
            } else if (opcodeOffset == 13) {
                // rotl
                res = rotl64(a, b);
            } else if (opcodeOffset == 14) {
                // rotr
                res = rotr64(a, b);
            } else {
                bool err;
                (res, err) = genericBinOp(a, b, opcodeOffset);
                if (err) {
                    mach.status = MachineStatus.ERRORED;
                    return;
                }
            }
        }

        mach.valueStack.push(ValueLib.newI64(res));
    }

    function executeI32WrapI64(
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        uint64 a = mach.valueStack.pop().assumeI64();

        uint32 a32 = uint32(a);

        mach.valueStack.push(ValueLib.newI32(a32));
    }

    function executeI64ExtendI32(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        uint32 a = mach.valueStack.pop().assumeI32();

        uint64 a64;

        if (inst.opcode == Instructions.I64_EXTEND_I32_S) {
            a64 = signExtend(a);
        } else {
            a64 = uint64(a);
        }

        mach.valueStack.push(ValueLib.newI64(a64));
    }

    function executeExtendSameType(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        ValueType ty;
        uint8 sourceBits;
        if (inst.opcode == Instructions.I32_EXTEND_8S) {
            ty = ValueType.I32;
            sourceBits = 8;
        } else if (inst.opcode == Instructions.I32_EXTEND_16S) {
            ty = ValueType.I32;
            sourceBits = 16;
        } else if (inst.opcode == Instructions.I64_EXTEND_8S) {
            ty = ValueType.I64;
            sourceBits = 8;
        } else if (inst.opcode == Instructions.I64_EXTEND_16S) {
            ty = ValueType.I64;
            sourceBits = 16;
        } else if (inst.opcode == Instructions.I64_EXTEND_32S) {
            ty = ValueType.I64;
            sourceBits = 32;
        } else {
            revert("INVALID_EXTEND_SAME_TYPE");
        }
        uint256 resultMask;
        if (ty == ValueType.I32) {
            resultMask = (1 << 32) - 1;
        } else {
            resultMask = (1 << 64) - 1;
        }
        Value memory val = mach.valueStack.pop();
        require(val.valueType == ty, "BAD_EXTEND_SAME_TYPE_TYPE");
        uint256 sourceMask = (1 << sourceBits) - 1;
        val.contents &= sourceMask;
        if (val.contents & (1 << (sourceBits - 1)) != 0) {
            // Extend sign flag
            val.contents |= resultMask & ~sourceMask;
        }
        mach.valueStack.push(val);
    }

    function executeReinterpret(
        Machine memory mach,
        Module memory,
        Instruction calldata inst,
        bytes calldata
    ) internal pure {
        ValueType destTy;
        ValueType sourceTy;
        if (inst.opcode == Instructions.I32_REINTERPRET_F32) {
            destTy = ValueType.I32;
            sourceTy = ValueType.F32;
        } else if (inst.opcode == Instructions.I64_REINTERPRET_F64) {
            destTy = ValueType.I64;
            sourceTy = ValueType.F64;
        } else if (inst.opcode == Instructions.F32_REINTERPRET_I32) {
            destTy = ValueType.F32;
            sourceTy = ValueType.I32;
        } else if (inst.opcode == Instructions.F64_REINTERPRET_I64) {
            destTy = ValueType.F64;
            sourceTy = ValueType.I64;
        } else {
            revert("INVALID_REINTERPRET");
        }
        Value memory val = mach.valueStack.pop();
        require(val.valueType == sourceTy, "INVALID_REINTERPRET_TYPE");
        val.valueType = destTy;
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
        if (opcode == Instructions.I32_EQZ || opcode == Instructions.I64_EQZ) {
            impl = executeEqz;
        } else if (
            opcode >= Instructions.I32_RELOP_BASE &&
            opcode <= Instructions.I32_RELOP_BASE + Instructions.IRELOP_LAST
        ) {
            impl = executeI32RelOp;
        } else if (
            opcode >= Instructions.I32_UNOP_BASE &&
            opcode <= Instructions.I32_UNOP_BASE + Instructions.IUNOP_LAST
        ) {
            impl = executeI32UnOp;
        } else if (opcode >= Instructions.I32_ADD && opcode <= Instructions.I32_ROTR) {
            impl = executeI32BinOp;
        } else if (
            opcode >= Instructions.I64_RELOP_BASE &&
            opcode <= Instructions.I64_RELOP_BASE + Instructions.IRELOP_LAST
        ) {
            impl = executeI64RelOp;
        } else if (
            opcode >= Instructions.I64_UNOP_BASE &&
            opcode <= Instructions.I64_UNOP_BASE + Instructions.IUNOP_LAST
        ) {
            impl = executeI64UnOp;
        } else if (opcode >= Instructions.I64_ADD && opcode <= Instructions.I64_ROTR) {
            impl = executeI64BinOp;
        } else if (opcode == Instructions.I32_WRAP_I64) {
            impl = executeI32WrapI64;
        } else if (
            opcode == Instructions.I64_EXTEND_I32_S || opcode == Instructions.I64_EXTEND_I32_U
        ) {
            impl = executeI64ExtendI32;
        } else if (opcode >= Instructions.I32_EXTEND_8S && opcode <= Instructions.I64_EXTEND_32S) {
            impl = executeExtendSameType;
        } else if (
            opcode >= Instructions.I32_REINTERPRET_F32 && opcode <= Instructions.F64_REINTERPRET_I64
        ) {
            impl = executeReinterpret;
        } else {
            revert("INVALID_OPCODE");
        }

        impl(mach, mod, inst, proof);
    }
}
