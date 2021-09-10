//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Deserialize.sol";
import "../state/Machines.sol";
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

        uint16 opcode = Instructions.peek(mach.instructions).opcode;
        if (
            (opcode >= Instructions.I32_LOAD &&
                opcode <= Instructions.I64_LOAD32_U) ||
            (opcode >= Instructions.I32_STORE &&
                opcode <= Instructions.I64_STORE32)
        ) {
            mach = proverMem.executeOneStep(mach, proof[offset:]);
        } else {
            mach = prover0.executeOneStep(mach, proof[offset:]);
        }

        return Machines.hash(mach);
    }
}
