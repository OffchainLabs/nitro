// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/Deserialize.sol";
import "../state/Machine.sol";
import "../state/MerkleProof.sol";
import "./IOneStepProver.sol";
import "./IOneStepProofEntry.sol";

contract OneStepProofEntry is IOneStepProofEntry {
    using MerkleProofLib for MerkleProof;
    using MachineLib for Machine;

    IOneStepProver public prover0;
    IOneStepProver public proverMem;
    IOneStepProver public proverMath;
    IOneStepProver public proverHostIo;

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

    function proveOneStep(
        ExecutionContext calldata execCtx,
        uint256 machineStep,
        bytes32 beforeHash,
        bytes calldata proof
    ) external view override returns (bytes32 afterHash) {
        Machine memory mach;
        Module memory mod;
        MerkleProof memory modProof;
        Instruction memory inst;

        {
            uint256 offset = 0;
            (mach, offset) = Deserialize.machine(proof, offset);
            require(mach.hash() == beforeHash, "MACHINE_BEFORE_HASH");
            if (mach.status != MachineStatus.RUNNING) {
                // Machine is halted.
                // WARNING: at this point, most machine fields are unconstrained.
                return mach.hash();
            }

            if (machineStep + 1 == OneStepProofEntryLib.MAX_STEPS) {
                mach.status = MachineStatus.ERRORED;
                return mach.hash();
            }

            (mod, offset) = Deserialize.module(proof, offset);
            (modProof, offset) = Deserialize.merkleProof(proof, offset);
            require(
                modProof.computeRootFromModule(mach.moduleIdx, mod) == mach.modulesRoot,
                "MODULES_ROOT"
            );

            {
                MerkleProof memory instProof;
                MerkleProof memory funcProof;
                (inst, offset) = Deserialize.instruction(proof, offset);
                (instProof, offset) = Deserialize.merkleProof(proof, offset);
                (funcProof, offset) = Deserialize.merkleProof(proof, offset);
                bytes32 codeHash = instProof.computeRootFromInstruction(mach.functionPc, inst);
                bytes32 recomputedRoot = funcProof.computeRootFromFunction(
                    mach.functionIdx,
                    codeHash
                );
                require(recomputedRoot == mod.functionsMerkleRoot, "BAD_FUNCTIONS_ROOT");
            }
            proof = proof[offset:];
        }

        uint256 oldModIdx = mach.moduleIdx;
        mach.functionPc += 1;
        uint16 opcode = inst.opcode;
        IOneStepProver prover;
        if (
            (opcode >= Instructions.I32_LOAD && opcode <= Instructions.I64_LOAD32_U) ||
            (opcode >= Instructions.I32_STORE && opcode <= Instructions.I64_STORE32) ||
            opcode == Instructions.MEMORY_SIZE ||
            opcode == Instructions.MEMORY_GROW
        ) {
            prover = proverMem;
        } else if (
            (opcode == Instructions.I32_EQZ || opcode == Instructions.I64_EQZ) ||
            (opcode >= Instructions.I32_RELOP_BASE &&
                opcode <= Instructions.I32_RELOP_BASE + Instructions.IRELOP_LAST) ||
            (opcode >= Instructions.I32_UNOP_BASE &&
                opcode <= Instructions.I32_UNOP_BASE + Instructions.IUNOP_LAST) ||
            (opcode >= Instructions.I32_ADD && opcode <= Instructions.I32_ROTR) ||
            (opcode >= Instructions.I64_RELOP_BASE &&
                opcode <= Instructions.I64_RELOP_BASE + Instructions.IRELOP_LAST) ||
            (opcode >= Instructions.I64_UNOP_BASE &&
                opcode <= Instructions.I64_UNOP_BASE + Instructions.IUNOP_LAST) ||
            (opcode >= Instructions.I64_ADD && opcode <= Instructions.I64_ROTR) ||
            (opcode == Instructions.I32_WRAP_I64) ||
            (opcode == Instructions.I64_EXTEND_I32_S || opcode == Instructions.I64_EXTEND_I32_U) ||
            (opcode >= Instructions.I32_EXTEND_8S && opcode <= Instructions.I64_EXTEND_32S) ||
            (opcode >= Instructions.I32_REINTERPRET_F32 &&
                opcode <= Instructions.F64_REINTERPRET_I64)
        ) {
            prover = proverMath;
        } else if (
            (opcode >= Instructions.GET_GLOBAL_STATE_BYTES32 &&
                opcode <= Instructions.SET_GLOBAL_STATE_U64) ||
            (opcode >= Instructions.READ_PRE_IMAGE && opcode <= Instructions.HALT_AND_SET_FINISHED)
        ) {
            prover = proverHostIo;
        } else {
            prover = prover0;
        }

        (mach, mod) = prover.executeOneStep(execCtx, mach, mod, inst, proof);

        mach.modulesRoot = modProof.computeRootFromModule(oldModIdx, mod);

        return mach.hash();
    }
}
