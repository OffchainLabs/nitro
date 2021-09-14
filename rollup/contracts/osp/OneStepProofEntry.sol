//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Deserialize.sol";
import "../state/Machines.sol";
import "../state/MerkleProofs.sol";
import "./IOneStepProver.sol";

contract OneStepProofEntry {
    IOneStepProver prover0;
    IOneStepProver proverMem;

    constructor(IOneStepProver prover0_, IOneStepProver proverMem_) {
        prover0 = prover0_;
        proverMem = proverMem_;
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

        Instruction memory inst;
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
            recomputedRoot == mach.functionsMerkleRoot,
            "BAD_FUNCTIONS_ROOT"
        );

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
            mach = proverMem.executeOneStep(mach, inst, proof[offset:]);
        } else {
            mach = prover0.executeOneStep(mach, inst, proof[offset:]);
        }

        return Machines.hash(mach);
    }
}
