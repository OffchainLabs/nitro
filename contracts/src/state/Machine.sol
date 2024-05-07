// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./ValueStack.sol";
import "./Instructions.sol";
import "./MultiStack.sol";
import "./StackFrame.sol";

enum MachineStatus {
    RUNNING,
    FINISHED,
    ERRORED
}

struct Machine {
    MachineStatus status;
    ValueStack valueStack;
    MultiStack valueMultiStack;
    ValueStack internalStack;
    StackFrameWindow frameStack;
    MultiStack frameMultiStack;
    bytes32 globalStateHash;
    uint32 moduleIdx;
    uint32 functionIdx;
    uint32 functionPc;
    bytes32 recoveryPc;
    bytes32 modulesRoot;
}

library MachineLib {
    using StackFrameLib for StackFrameWindow;
    using ValueStackLib for ValueStack;
    using MultiStackLib for MultiStack;

    bytes32 internal constant NO_RECOVERY_PC = ~bytes32(0);

    function hash(Machine memory mach) internal pure returns (bytes32) {
        // Warning: the non-running hashes are replicated in Challenge
        if (mach.status == MachineStatus.RUNNING) {
            bytes32 valueMultiHash = mach.valueMultiStack.hash(
                mach.valueStack.hash(),
                mach.recoveryPc != NO_RECOVERY_PC
            );
            bytes32 frameMultiHash = mach.frameMultiStack.hash(
                mach.frameStack.hash(),
                mach.recoveryPc != NO_RECOVERY_PC
            );
            bytes memory preimage = abi.encodePacked(
                "Machine running:",
                valueMultiHash,
                mach.internalStack.hash(),
                frameMultiHash,
                mach.globalStateHash,
                mach.moduleIdx,
                mach.functionIdx,
                mach.functionPc,
                mach.recoveryPc,
                mach.modulesRoot
            );
            return keccak256(preimage);
        } else if (mach.status == MachineStatus.FINISHED) {
            return keccak256(abi.encodePacked("Machine finished:", mach.globalStateHash));
        } else if (mach.status == MachineStatus.ERRORED) {
            return keccak256(abi.encodePacked("Machine errored:", mach.globalStateHash));
        } else {
            revert("BAD_MACH_STATUS");
        }
    }

    function switchCoThreadStacks(Machine memory mach) internal pure {
        bytes32 newActiveValue = mach.valueMultiStack.inactiveStackHash;
        bytes32 newActiveFrame = mach.frameMultiStack.inactiveStackHash;
        if (
            newActiveFrame == MultiStackLib.NO_STACK_HASH ||
            newActiveValue == MultiStackLib.NO_STACK_HASH
        ) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
        mach.frameMultiStack.inactiveStackHash = mach.frameStack.hash();
        mach.valueMultiStack.inactiveStackHash = mach.valueStack.hash();
        mach.frameStack.overwrite(newActiveFrame);
        mach.valueStack.overwrite(newActiveValue);
    }

    function setPcFromData(Machine memory mach, uint256 data) internal pure returns (bool) {
        if (data >> 96 != 0) {
            return false;
        }

        mach.functionPc = uint32(data);
        mach.functionIdx = uint32(data >> 32);
        mach.moduleIdx = uint32(data >> 64);
        return true;
    }

    function setPcFromRecovery(Machine memory mach) internal pure returns (bool) {
        if (!setPcFromData(mach, uint256(mach.recoveryPc))) {
            return false;
        }
        mach.recoveryPc = NO_RECOVERY_PC;
        return true;
    }

    function setRecoveryFromPc(Machine memory mach, uint32 offset) internal pure returns (bool) {
        if (mach.recoveryPc != NO_RECOVERY_PC) {
            return false;
        }

        uint256 result;
        result = uint256(mach.moduleIdx) << 64;
        result = result | (uint256(mach.functionIdx) << 32);
        result = result | uint256(mach.functionPc + offset - 1);
        mach.recoveryPc = bytes32(result);
        return true;
    }

    function setPc(Machine memory mach, Value memory pc) internal pure {
        if (pc.valueType == ValueType.REF_NULL) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
        if (pc.valueType != ValueType.INTERNAL_REF) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
        if (!setPcFromData(mach, pc.contents)) {
            mach.status = MachineStatus.ERRORED;
            return;
        }
    }
}
