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
        Machine memory,
        Module memory,
        GlobalState memory state,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        state.inboxPosition += 1;
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
        (leafContents, proofOffset, merkleProof) = ModuleMemories
            .proveLeaf(mod.moduleMemory, leafIdx, proof, proofOffset);

        bytes memory preimage = proof[proofOffset:];
        require(keccak256(preimage) == leafContents, "BAD_PREIMAGE");

        uint32 i = 0;
        for (; i < 32 && preimageOffset + i < preimage.length; i++) {
            leafContents = setLeafByte(leafContents, i, uint8(preimage[preimageOffset + i]));
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
        (leafContents, proofOffset, merkleProof) = ModuleMemories
            .proveLeaf(mod.moduleMemory, leafIdx, proof, proofOffset);

        // TODO
        require(state.inboxPosition == 0, "TODO: proper inbox");
        bytes memory message = hex"f8859431b98d14007bdee637298086988a0bbd311845238080f86c808504a817c800825208942ed530faddb7349c1efdbf4410db2de835a004e4880de0b6b3a7640000802ba06217a3ed3379e98821117e66536aa59dc9f402eb1c998111e4e087bc5ec9b09ea0092d3bccf7d31fd025ea79560583064a511a02a2001e31d91927e4b80c9ccaa7";
        uint32 i = 0;
        for (; i < 32 && messageOffset + i < message.length; i++) {
            leafContents = setLeafByte(leafContents, i, uint8(message[messageOffset + i]));
        }

        mod.moduleMemory.merkleRoot = MerkleProofs.computeRootFromMemory(
            merkleProof,
            leafIdx,
            leafContents
        );
        ValueStacks.push(mach.valueStack, Values.newI32(i));
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
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }

        GlobalState memory state;
        uint256 proofOffset = 0;
        (state, proofOffset) = Deserialize.globalState(proof, proofOffset);
        require(GlobalStates.hash(state) == mach.globalStateHash, "BAD_GLOBAL_STATE");

        impl(mach, mod, state, inst, proof[proofOffset:]);

        mach.globalStateHash = GlobalStates.hash(state);
    }
}
