// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/Value.sol";
import "../state/Machine.sol";
import "../state/Deserialize.sol";
import "../state/ModuleMemory.sol";
import "./IOneStepProver.sol";
import "../bridge/Messages.sol";
import "../bridge/IBridge.sol";

contract OneStepProverHostIo is IOneStepProver {
    using GlobalStateLib for GlobalState;
    using MerkleProofLib for MerkleProof;
    using ModuleMemoryLib for ModuleMemory;
    using ValueLib for Value;
    using ValueStackLib for ValueStack;

    uint256 private constant LEAF_SIZE = 32;
    uint256 private constant INBOX_NUM = 2;
    uint64 private constant INBOX_HEADER_LEN = 40;
    uint64 private constant DELAYED_HEADER_LEN = 112 + 1;

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
        uint256 ptr = mach.valueStack.pop().assumeI32();
        uint32 idx = mach.valueStack.pop().assumeI32();

        if (idx >= GlobalStateLib.BYTES32_VALS_NUM) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 startLeafContents;
        MerkleProof memory merkleProof;
        (startLeafContents, proofOffset, merkleProof) = mod.moduleMemory.proveLeaf(
            leafIdx,
            proof,
            proofOffset
        );

        if (inst.opcode == Instructions.GET_GLOBAL_STATE_BYTES32) {
            mod.moduleMemory.merkleRoot = merkleProof.computeRootFromMemory(
                leafIdx,
                state.bytes32Vals[idx]
            );
        } else if (inst.opcode == Instructions.SET_GLOBAL_STATE_BYTES32) {
            state.bytes32Vals[idx] = startLeafContents;
        } else {
            revert("BAD_GLOBAL_STATE_OPCODE");
        }
    }

    function executeGetU64(Machine memory mach, GlobalState memory state) internal pure {
        uint32 idx = mach.valueStack.pop().assumeI32();

        if (idx >= GlobalStateLib.U64_VALS_NUM) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        mach.valueStack.push(ValueLib.newI64(state.u64Vals[idx]));
    }

    function executeSetU64(Machine memory mach, GlobalState memory state) internal pure {
        uint64 val = mach.valueStack.pop().assumeI64();
        uint32 idx = mach.valueStack.pop().assumeI32();

        if (idx >= GlobalStateLib.U64_VALS_NUM) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
        state.u64Vals[idx] = val;
    }

    function executeReadPreImage(
        ExecutionContext calldata,
        Machine memory mach,
        Module memory mod,
        Instruction calldata,
        bytes calldata proof
    ) internal pure {
        uint256 preimageOffset = mach.valueStack.pop().assumeI32();
        uint256 ptr = mach.valueStack.pop().assumeI32();
        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 leafContents;
        MerkleProof memory merkleProof;
        (leafContents, proofOffset, merkleProof) = mod.moduleMemory.proveLeaf(
            leafIdx,
            proof,
            proofOffset
        );

        bytes memory extracted;
        uint8 proofType = uint8(proof[proofOffset]);
        proofOffset++;
        if (proofType == 0) {
            bytes calldata preimage = proof[proofOffset:];
            require(keccak256(preimage) == leafContents, "BAD_PREIMAGE");

            uint256 preimageEnd = preimageOffset + 32;
            if (preimageEnd > preimage.length) {
                preimageEnd = preimage.length;
            }
            extracted = preimage[preimageOffset:preimageEnd];
        } else {
            // TODO: support proving via an authenticated contract
            revert("UNKNOWN_PREIMAGE_PROOF");
        }

        for (uint256 i = 0; i < extracted.length; i++) {
            leafContents = setLeafByte(leafContents, i, uint8(extracted[i]));
        }

        mod.moduleMemory.merkleRoot = merkleProof.computeRootFromMemory(leafIdx, leafContents);

        mach.valueStack.push(ValueLib.newI32(uint32(extracted.length)));
    }

    function validateSequencerInbox(
        ExecutionContext calldata execCtx,
        uint64 msgIndex,
        bytes calldata message
    ) internal view returns (bool) {
        require(message.length >= INBOX_HEADER_LEN, "BAD_SEQINBOX_PROOF");

        uint64 afterDelayedMsg;
        (afterDelayedMsg, ) = Deserialize.u64(message, 32);
        bytes32 messageHash = keccak256(message);
        bytes32 beforeAcc;
        bytes32 delayedAcc;

        if (msgIndex > 0) {
            beforeAcc = execCtx.bridge.sequencerInboxAccs(msgIndex - 1);
        }
        if (afterDelayedMsg > 0) {
            delayedAcc = execCtx.bridge.delayedInboxAccs(afterDelayedMsg - 1);
        }
        bytes32 acc = keccak256(abi.encodePacked(beforeAcc, messageHash, delayedAcc));
        require(acc == execCtx.bridge.sequencerInboxAccs(msgIndex), "BAD_SEQINBOX_MESSAGE");
        return true;
    }

    function validateDelayedInbox(
        ExecutionContext calldata execCtx,
        uint64 msgIndex,
        bytes calldata message
    ) internal view returns (bool) {
        require(message.length >= DELAYED_HEADER_LEN, "BAD_DELAYED_PROOF");

        bytes32 beforeAcc;

        if (msgIndex > 0) {
            beforeAcc = execCtx.bridge.delayedInboxAccs(msgIndex - 1);
        }

        bytes32 messageDataHash = keccak256(message[DELAYED_HEADER_LEN:]);
        bytes1 kind = message[0];
        uint256 sender;
        (sender, ) = Deserialize.u256(message, 1);

        bytes32 messageHash = keccak256(
            abi.encodePacked(kind, uint160(sender), message[33:DELAYED_HEADER_LEN], messageDataHash)
        );
        bytes32 acc = Messages.accumulateInboxMessage(beforeAcc, messageHash);

        require(acc == execCtx.bridge.delayedInboxAccs(msgIndex), "BAD_DELAYED_MESSAGE");
        return true;
    }

    function executeReadInboxMessage(
        ExecutionContext calldata execCtx,
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal view {
        uint256 messageOffset = mach.valueStack.pop().assumeI32();
        uint256 ptr = mach.valueStack.pop().assumeI32();
        uint256 msgIndex = mach.valueStack.pop().assumeI64();
        if (
            inst.argumentData == Instructions.INBOX_INDEX_SEQUENCER &&
            msgIndex >= execCtx.maxInboxMessagesRead
        ) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        if (ptr + 32 > mod.moduleMemory.size || ptr % LEAF_SIZE != 0) {
            mach.status = MachineStatus.ERRORED;
            return;
        }

        uint256 leafIdx = ptr / LEAF_SIZE;
        uint256 proofOffset = 0;
        bytes32 leafContents;
        MerkleProof memory merkleProof;
        (leafContents, proofOffset, merkleProof) = mod.moduleMemory.proveLeaf(
            leafIdx,
            proof,
            proofOffset
        );

        {
            // TODO: support proving via an authenticated contract
            require(proof[proofOffset] == 0, "UNKNOWN_INBOX_PROOF");
            proofOffset++;

            function(ExecutionContext calldata, uint64, bytes calldata)
                internal
                view
                returns (bool) inboxValidate;

            bool success;
            if (inst.argumentData == Instructions.INBOX_INDEX_SEQUENCER) {
                inboxValidate = validateSequencerInbox;
            } else if (inst.argumentData == Instructions.INBOX_INDEX_DELAYED) {
                inboxValidate = validateDelayedInbox;
            } else {
                mach.status = MachineStatus.ERRORED;
                return;
            }
            success = inboxValidate(execCtx, uint64(msgIndex), proof[proofOffset:]);
            if (!success) {
                mach.status = MachineStatus.ERRORED;
                return;
            }
        }

        require(proof.length >= proofOffset, "BAD_MESSAGE_PROOF");
        uint256 messageLength = proof.length - proofOffset;

        uint32 i = 0;
        for (; i < 32 && messageOffset + i < messageLength; i++) {
            leafContents = setLeafByte(
                leafContents,
                i,
                uint8(proof[proofOffset + messageOffset + i])
            );
        }

        mod.moduleMemory.merkleRoot = merkleProof.computeRootFromMemory(leafIdx, leafContents);
        mach.valueStack.push(ValueLib.newI32(i));
    }

    function executeHaltAndSetFinished(
        ExecutionContext calldata,
        Machine memory mach,
        Module memory,
        Instruction calldata,
        bytes calldata
    ) internal pure {
        mach.status = MachineStatus.FINISHED;
    }

    function executeGlobalStateAccess(
        ExecutionContext calldata,
        Machine memory mach,
        Module memory mod,
        Instruction calldata inst,
        bytes calldata proof
    ) internal pure {
        uint16 opcode = inst.opcode;

        GlobalState memory state;
        uint256 proofOffset = 0;
        (state, proofOffset) = Deserialize.globalState(proof, proofOffset);
        require(state.hash() == mach.globalStateHash, "BAD_GLOBAL_STATE");

        if (
            opcode == Instructions.GET_GLOBAL_STATE_BYTES32 ||
            opcode == Instructions.SET_GLOBAL_STATE_BYTES32
        ) {
            executeGetOrSetBytes32(mach, mod, state, inst, proof[proofOffset:]);
        } else if (opcode == Instructions.GET_GLOBAL_STATE_U64) {
            executeGetU64(mach, state);
        } else if (opcode == Instructions.SET_GLOBAL_STATE_U64) {
            executeSetU64(mach, state);
        } else {
            revert("INVALID_GLOBALSTATE_OPCODE");
        }

        mach.globalStateHash = state.hash();
    }

    function executeOneStep(
        ExecutionContext calldata execCtx,
        Machine calldata startMach,
        Module calldata startMod,
        Instruction calldata inst,
        bytes calldata proof
    ) external view override returns (Machine memory mach, Module memory mod) {
        mach = startMach;
        mod = startMod;

        uint16 opcode = inst.opcode;

        function(
            ExecutionContext calldata,
            Machine memory,
            Module memory,
            Instruction calldata,
            bytes calldata
        ) internal view impl;

        if (
            opcode >= Instructions.GET_GLOBAL_STATE_BYTES32 &&
            opcode <= Instructions.SET_GLOBAL_STATE_U64
        ) {
            impl = executeGlobalStateAccess;
        } else if (opcode == Instructions.READ_PRE_IMAGE) {
            impl = executeReadPreImage;
        } else if (opcode == Instructions.READ_INBOX_MESSAGE) {
            impl = executeReadInboxMessage;
        } else if (opcode == Instructions.HALT_AND_SET_FINISHED) {
            impl = executeHaltAndSetFinished;
        } else {
            revert("INVALID_MEMORY_OPCODE");
        }

        impl(execCtx, mach, mod, inst, proof);
    }
}
