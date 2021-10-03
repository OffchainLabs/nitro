//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Deserialize.sol";
import "../state/Machines.sol";
import "../state/MerkleProofs.sol";
import "./IOneStepProver.sol";

contract OneStepProofEntry {
    IOneStepProver prover0;
    IOneStepProver proverMem;
    IOneStepProver proverMath;
    IOneStepProver proverHostIo;

    constructor(
        IOneStepProver prover0_,
        IOneStepProver proverMem_,
        IOneStepProver proverMath_,
        IOneStepProver proverHostIo_
    ) {
        prover0 = prover0_;
        proverMem = proverMem_;
        proverMath = proverMath_;
        proverHostIo = proverHostIo_;
    }

    function proveOneStep(bytes32 beforeHash, bytes calldata proof)
        external
        view
        returns (bytes32 afterHash)
    {
        Machine memory mach;
        uint256 offset = 0;
        (mach, offset) = Deserialize.machine(proof, offset);
        require(Machines.hash(mach) == beforeHash, "MACHINE_BEFORE_HASH");

        Module memory mod;
        MerkleProof memory modProof;
        (mod, offset) = Deserialize.module(proof, offset);
        (modProof, offset) = Deserialize.merkleProof(proof, offset);
        require(
            MerkleProofs.computeRootFromModule(modProof, mach.moduleIdx, mod) ==
                mach.modulesRoot,
            "MODULES_ROOT"
        );

        Instruction memory inst;
        {
            MerkleProof memory instProof;
            MerkleProof memory funcProof;
            (inst, offset) = Deserialize.instruction(proof, offset);
            (instProof, offset) = Deserialize.merkleProof(proof, offset);
            (funcProof, offset) = Deserialize.merkleProof(proof, offset);
            bytes32 codeHash = MerkleProofs.computeRootFromInstruction(
                instProof,
                mach.functionPc,
                inst
            );
            bytes32 recomputedRoot = MerkleProofs.computeRootFromFunction(
                funcProof,
                mach.functionIdx,
                codeHash
            );
            require(
                recomputedRoot == mod.functionsMerkleRoot,
                "BAD_FUNCTIONS_ROOT"
            );
        }

        uint256 oldModIdx = mach.moduleIdx;
        mach.functionPc += 1;
        uint16 opcode = inst.opcode;
        if (
            (opcode >= Instructions.I32_LOAD &&
                opcode <= Instructions.I64_LOAD32_U) ||
            (opcode >= Instructions.I32_STORE &&
                opcode <= Instructions.I64_STORE32) ||
            opcode == Instructions.MEMORY_SIZE ||
            opcode == Instructions.MEMORY_GROW
        ) {
            (mach, mod) = proverMem.executeOneStep(
                mach,
                mod,
                inst,
                proof[offset:]
            );
        } else if (
            (opcode == Instructions.I32_EQZ ||
                opcode == Instructions.I64_EQZ) ||
            (opcode >= Instructions.I32_RELOP_BASE &&
                opcode <=
                Instructions.I32_RELOP_BASE + Instructions.IRELOP_LAST) ||
            (opcode >= Instructions.I32_UNOP_BASE &&
                opcode <=
                Instructions.I32_UNOP_BASE + Instructions.IUNOP_LAST) ||
            (opcode >= Instructions.I32_ADD &&
                opcode <= Instructions.I32_ROTR) ||
            (opcode >= Instructions.I64_RELOP_BASE &&
                opcode <=
                Instructions.I64_RELOP_BASE + Instructions.IRELOP_LAST) ||
            (opcode >= Instructions.I64_UNOP_BASE &&
                opcode <=
                Instructions.I64_UNOP_BASE + Instructions.IUNOP_LAST) ||
            (opcode >= Instructions.I64_ADD &&
                opcode <= Instructions.I64_ROTR) ||
            (opcode == Instructions.I32_WRAP_I64) ||
            (opcode == Instructions.I64_EXTEND_I32_S ||
                opcode == Instructions.I64_EXTEND_I32_U) ||
            (opcode >= Instructions.I32_EXTEND_8S &&
                opcode <= Instructions.I64_EXTEND_32S) ||
            (opcode >= Instructions.I32_REINTERPRET_F32 &&
                opcode <= Instructions.F64_REINTERPRET_I64)
        ) {
            (mach, mod) = proverMath.executeOneStep(
                mach,
                mod,
                inst,
                proof[offset:]
            );
        } else if (
            (opcode == Instructions.GET_LAST_BLOCK_HASH ||
                opcode == Instructions.SET_LAST_BLOCK_HASH) ||
            opcode == Instructions.ADVANCE_INBOX_POSITION ||
            opcode == Instructions.READ_PRE_IMAGE ||
            opcode == Instructions.READ_INBOX_MESSAGE ||
            opcode == Instructions.GET_POSITION_WITHIN_MESSAGE ||
            opcode == Instructions.SET_POSITION_WITHIN_MESSAGE
        ) {
            (mach, mod) = proverHostIo.executeOneStep(
                mach,
                mod,
                inst,
                proof[offset:]
            );
        } else {
            (mach, mod) = prover0.executeOneStep(
                mach,
                mod,
                inst,
                proof[offset:]
            );
        }

        mach.modulesRoot = MerkleProofs.computeRootFromModule(
            modProof,
            oldModIdx,
            mod
        );

        return Machines.hash(mach);
    }
}
