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

    function executeGetOrSetLastBlockHash(
        Machine memory mach,
        Module memory mod,
        GlobalState memory state,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint256 ptr = ValueStacks.pop(mach.valueStack).contents;
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

        if (inst.opcode == Instructions.GET_LAST_BLOCK_HASH) {
            mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
                merkleProof,
                leafIdx,
                state.lastBlockHash
            );
        } else if (inst.opcode == Instructions.SET_LAST_BLOCK_HASH) {
            state.lastBlockHash = startLeafContents;
        } else {
            revert("BAD_BLOCK_HASH_OPCODE");
        }
    }

    function executeAdvanceInboxPosition(
        Machine memory mach,
        Module memory,
        GlobalState memory state,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        if (state.inboxPosition == ~uint64(0)) {
            mach.halted = true;
        } else {
            state.inboxPosition += 1;
        }
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
        GlobalState memory state,
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

    function executeGetPositionWithinMessage(
        Machine memory mach,
        Module memory,
        GlobalState memory state,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        ValueStacks.push(
            mach.valueStack,
            Values.newI64(state.positionWithinMessage)
        );
    }

    function executeSetPositionWithinMessage(
        Machine memory mach,
        Module memory,
        GlobalState memory state,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        state.positionWithinMessage = Values.assumeI64(
            ValueStacks.pop(mach.valueStack)
        );
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

    function executeGetInboxPosition(
        Machine memory mach,
        Module memory,
        GlobalState memory state,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        ValueStacks.push(mach.valueStack, Values.newI64(state.inboxPosition));
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

        function(
            Machine memory,
            Module memory,
            GlobalState memory,
            Instruction calldata,
            bytes calldata
        ) internal pure impl;
        if (
            opcode == Instructions.GET_LAST_BLOCK_HASH ||
            opcode == Instructions.SET_LAST_BLOCK_HASH
        ) {
            impl = executeGetOrSetLastBlockHash;
        } else if (opcode == Instructions.ADVANCE_INBOX_POSITION) {
            impl = executeAdvanceInboxPosition;
        } else if (opcode == Instructions.READ_PRE_IMAGE) {
            // Doesn't use global state
            executeReadPreImage(mach, mod, inst, proof);
            return (mach, mod);
        } else if (opcode == Instructions.READ_INBOX_MESSAGE) {
            impl = executeReadInboxMessage;
        } else if (opcode == Instructions.GET_POSITION_WITHIN_MESSAGE) {
            impl = executeGetPositionWithinMessage;
        } else if (opcode == Instructions.SET_POSITION_WITHIN_MESSAGE) {
            impl = executeSetPositionWithinMessage;
        } else if (opcode == Instructions.READ_DELAYED_INBOX_MESSAGE) {
            // Doesn't use global state
            executeReadDelayedInboxMessage(mach, mod, inst, proof);
            return (mach, mod);
        } else if (opcode == Instructions.GET_INBOX_POSITION) {
            impl = executeGetInboxPosition;
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }

        GlobalState memory state;
        uint256 proofOffset = 0;
        (state, proofOffset) = Deserialize.globalState(proof, proofOffset);
        require(
            GlobalStates.hash(state) == mach.globalStateHash,
            "BAD_GLOBAL_STATE"
        );

        impl(mach, mod, state, inst, proof[proofOffset:]);

        mach.globalStateHash = GlobalStates.hash(state);
    }
}
