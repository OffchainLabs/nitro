//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Values.sol";
import "../state/Machines.sol";
import "../state/Deserialize.sol";
import "./IOneStepProver.sol";

contract OneStepProverHostIo is IOneStepProver {
    uint256 constant LEAF_SIZE = 32;

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

    function executeGetOrSetBytes32(
        Machine memory mach,
        Module memory mod,
        GlobalState memory state,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint256 ptr = ValueStacks.pop(mach.valueStack).contents;
        uint32 idx = Values.assumeI32(ValueStacks.pop(mach.valueStack));

        if (idx >= GlobalStates.BYTES32_VALS_NUM) {
            mach.halted = true;
            return;
        }
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.halted = true;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 startLeafContents;
        MerkleProof memory merkleProof;
        (startLeafContents, proofOffset, merkleProof) = ModuleMemories
            .proveLeaf(mod.moduleMemory, leafIdx, proof, proofOffset);

        if (inst.opcode == Instructions.GET_GLOBAL_STATE_BYTES32) {
            mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
                merkleProof,
                leafIdx,
                state.bytes32_vals[idx]
            );
        } else if (inst.opcode == Instructions.SET_GLOBAL_STATE_BYTES32) {
            state.bytes32_vals[idx] = startLeafContents;
        } else {
            revert("BAD_GLOBAL_STATE_OPCODE");
        }
    }

    function executeGetU64(
        Machine memory mach,
        GlobalState memory state
    ) internal pure {
        uint32 idx = Values.assumeI32(ValueStacks.pop(mach.valueStack));

        if (idx >= GlobalStates.U64_VALS_NUM) {
            mach.halted = true;
            return;
        }

        ValueStacks.push(
            mach.valueStack,
            Values.newI64(state.u64_vals[idx])
        );
    }

    function executeSetU64(
        Machine memory mach,
        GlobalState memory state
    ) internal pure {
        uint64 val = Values.assumeI64(ValueStacks.pop(mach.valueStack));
        uint32 idx = Values.assumeI32(ValueStacks.pop(mach.valueStack));

        if (idx >= GlobalStates.U64_VALS_NUM) {
            mach.halted = true;
            return;
        }
        state.u64_vals[idx] = val;
    }

    function executeReadPreImage(
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata proof
    ) internal pure {
        uint256 preimageOffset = ValueStacks.pop(mach.valueStack).contents;
        uint256 ptr = ValueStacks.pop(mach.valueStack).contents;
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.halted = true;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 leafContents;
        MerkleProof memory merkleProof;
        (leafContents, proofOffset, merkleProof) = ModuleMemories.proveLeaf(
            mod.moduleMemory,
            leafIdx,
            proof,
            proofOffset
        );

        bytes memory preimage = proof[proofOffset:];
        require(keccak256(preimage) == leafContents, "BAD_PREIMAGE");

        uint32 i = 0;
        for (; i < 32 && preimageOffset + i < preimage.length; i++) {
            leafContents = setLeafByte(
                leafContents,
                i,
                uint8(preimage[preimageOffset + i])
            );
        }

        mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
            merkleProof,
            leafIdx,
            leafContents
        );

        ValueStacks.push(mach.valueStack, Values.newI32(i));
    }

    function executeReadInboxMessage(
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata proof
    ) internal pure {
        uint256 messageOffset = ValueStacks.pop(mach.valueStack).contents;
        uint256 ptr = ValueStacks.pop(mach.valueStack).contents;
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.halted = true;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 leafContents;
        MerkleProof memory merkleProof;
        (leafContents, proofOffset, merkleProof) = ModuleMemories.proveLeaf(
            mod.moduleMemory,
            leafIdx,
            proof,
            proofOffset
        );

        revert("TODO: proper inbox API");
        bytes memory message; // TODO
        uint32 i = 0;
        for (; i < 32 && messageOffset + i < message.length; i++) {
            leafContents = setLeafByte(
                leafContents,
                i,
                uint8(message[messageOffset + i])
            );
        }

        mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
            merkleProof,
            leafIdx,
            leafContents
        );
        ValueStacks.push(mach.valueStack, Values.newI32(i));
    }

    function executeReadDelayedInboxMessage(
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata proof
    ) internal pure {
        uint256 messageOffset = ValueStacks.pop(mach.valueStack).contents;
        uint256 ptr = ValueStacks.pop(mach.valueStack).contents;
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.halted = true;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 leafContents;
        MerkleProof memory merkleProof;
        (leafContents, proofOffset, merkleProof) = ModuleMemories.proveLeaf(
            mod.moduleMemory,
            leafIdx,
            proof,
            proofOffset
        );

        revert("TODO: proper inbox API");
        bytes memory message; // TODO
        uint32 i = 0;
        for (; i < 32 && messageOffset + i < message.length; i++) {
            leafContents = setLeafByte(
                leafContents,
                i,
                uint8(message[messageOffset + i])
            );
        }

        mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
            merkleProof,
            leafIdx,
            leafContents
        );
        ValueStacks.push(mach.valueStack, Values.newI32(i));
    }

    function executeGlobalStateAccess(
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint16 opcode = inst.opcode;

        GlobalState memory state;
        uint256 proofOffset = 0;
        (state, proofOffset) = Deserialize.globalState(proof, proofOffset);
        require(
            GlobalStates.hash(state) == mach.globalStateHash,
            "BAD_GLOBAL_STATE"
        );

        if (opcode == Instructions.GET_GLOBAL_STATE_BYTES32 ||
            opcode == Instructions.SET_GLOBAL_STATE_BYTES32) {
            executeGetOrSetBytes32(mach, mod, state, inst, proof[proofOffset:]);
        } else if (opcode == Instructions.GET_GLOBAL_STATE_U64) {
            executeGetU64(mach, state);
        } else if (opcode == Instructions.SET_GLOBAL_STATE_U64) {
            executeSetU64(mach, state);
        } else {
            revert("INVALID_GLOBALSTATE_OPCODE");
        }

        mach.globalStateHash = GlobalStates.hash(state);

    }

    function executeOneStep(
        Machine calldata startMach,
        Module calldata startMod,
        Instruction calldata inst,
        bytes calldata proof
    ) external pure override returns (Machine memory mach, Module memory mod) {
        mach = startMach;
        mod = startMod;

        uint16 opcode = inst.opcode;

        function(Machine memory, Module memory, Instruction calldata, bytes calldata) internal pure impl;

        if (opcode >= Instructions.GET_GLOBAL_STATE_BYTES32 &&
            opcode <= Instructions.SET_GLOBAL_STATE_U64)
        {
            impl = executeGlobalStateAccess;
        } else if (opcode == Instructions.READ_PRE_IMAGE) {
            impl = executeReadPreImage;
        } else if (opcode == Instructions.READ_INBOX_MESSAGE) {
            impl = executeReadInboxMessage;
        } else if (opcode == Instructions.READ_DELAYED_INBOX_MESSAGE) {
            impl = executeReadDelayedInboxMessage;
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }


        impl(mach, mod, inst, proof);

    }
}
