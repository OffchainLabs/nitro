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
    using GlobalStateLib for GlobalState;

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

    // Copied from OldChallengeLib.sol
    function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot)
        internal
        pure
        returns (bytes32)
    {
        // Start the value stack with the function call ABI for the entrypoint
        Value[] memory startingValues = new Value[](3);
        startingValues[0] = ValueLib.newRefNull();
        startingValues[1] = ValueLib.newI32(0);
        startingValues[2] = ValueLib.newI32(0);
        ValueArray memory valuesArray = ValueArray({inner: startingValues});
        ValueStack memory values = ValueStack({proved: valuesArray, remainingHash: 0});
        ValueStack memory internalStack;
        StackFrameWindow memory frameStack;

        Machine memory mach = Machine({
            status: MachineStatus.RUNNING,
            valueStack: values,
            internalStack: internalStack,
            frameStack: frameStack,
            globalStateHash: globalStateHash,
            moduleIdx: 0,
            functionIdx: 0,
            functionPc: 0,
            modulesRoot: wasmModuleRoot
        });
        return mach.hash();
    }

    function getMachineHash(ExecutionState calldata execState) external pure override returns (bytes32) {
        if (execState.machineStatus == MachineStatus.FINISHED) {
            return keccak256(abi.encodePacked("Machine finished:", execState.globalState.hash()));
        } else if (execState.machineStatus == MachineStatus.ERRORED) {
            return keccak256(abi.encodePacked("Machine errored:", execState.globalState.hash()));
        } else {
            revert("BAD_MACHINE_STATUS");
        }
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
                GlobalState memory globalState;
                (globalState, offset) = Deserialize.globalState(proof, offset);
                require(globalState.hash() == mach.globalStateHash, "BAD_GLOBAL_STATE");
                if (mach.status == MachineStatus.FINISHED && machineStep == 0 && globalState.getInboxPosition() < execCtx.maxInboxMessagesRead) {
                    // Kickstart the machine
                    return getStartMachineHash(mach.globalStateHash, execCtx.initialWasmModuleRoot);
                }
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
