// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/GlobalState.sol";
import "../state/Machine.sol";
import "../osp/IOneStepProofEntry.sol";

enum AssertionStatus {
    // No assertion at this index
    NoAssertion,
    // Assertion is being computed
    Pending,
    // Assertion is confirmed
    Confirmed
}

struct AssertionNode {
    // The inbox position that the assertion that succeeds should process up to and including
    // TODO: HN: move this into configHash or not? we do have extra space in this struct but we can remove the below 2 fields
    uint64 nextInboxPosition;
    // Deadline at which this assertion can be confirmed
    // TODO: HN: remove this and derive from createdAtBlock?
    uint64 deadlineBlock;
    // Deadline at which a child of this assertion can be confirmed
    // TODO: HN: remove this and derive from first child?
    uint64 noChildConfirmedBeforeBlock;
    // This value starts at zero and is set to a value when the first child is created. After that it is constant until the assertion is destroyed or the owner destroys pending assertions
    uint64 firstChildBlock;
    // This value starts at zero and is set to a value when the second child is created. After that it is constant until the assertion is destroyed or the owner destroys pending assertions
    uint64 secondChildBlock;
    // The block number when this assertion was created
    uint64 createdAtBlock;
    // True if this assertion is the first child of its prev
    bool isFirstChild;
    // Status of the Assertion
    AssertionStatus status;
    // Id of the assertion previous to this one
    bytes32 prevId;
    // A hash of the context available at the time of this assertions creation. It should contain information that is not specific
    // to this assertion, but instead to the environment at the time of creation. This is necessary to store on the assertion
    // as this environment can change and we need to know what it was like at the time this assertion was created. An example
    // of this is the wasm module root which determines the state transition function on the L2. If the wasm module root
    // changes we need to know that previous assertions were made under a different root, so that we can understand that they
    // were valid at the time. So when resolving a challenge by one step, the edge challenge manager finds the wasm module root
    // that was recorded on the prev of the assertions being disputed and uses it to resolve the one step proof.
    bytes32 configHash;
}

struct BeforeStateData {
    // The assertion hash of the prev of the beforeState(prev)
    bytes32 prevPrevAssertionHash;
    // The sequencer inbox accumulator asserted by the beforeState(prev)
    bytes32 sequencerBatchAcc;
    // below are the components of config hash
    bytes32 wasmRoot;
    uint256 requiredStake;
    address challengeManager;
    uint64 confirmPeriodBlocks;
}

struct AssertionInputs {
    // Additional data used to validate the before state
    BeforeStateData beforeStateData;
    ExecutionState beforeState;
    ExecutionState afterState;
}

/**
 * @notice Utility functions for Assertion
 */
library AssertionNodeLib {
    /**
     * @notice Initialize a Assertion
     * @param _nextInboxPosition The inbox position that the assertion that succeeds should process up to and including
     * @param _prevId Initial value of prevId
     * @param _deadlineBlock Initial value of deadlineBlock
     */
    function createAssertion(
        uint64 _nextInboxPosition,
        bytes32 _prevId,
        uint64 _deadlineBlock,
        bool _isFirstChild,
        bytes32 _configHash
    ) internal view returns (AssertionNode memory) {
        AssertionNode memory assertion;
        assertion.nextInboxPosition = _nextInboxPosition;
        assertion.prevId = _prevId;
        assertion.deadlineBlock = _deadlineBlock;
        assertion.noChildConfirmedBeforeBlock = _deadlineBlock;
        assertion.createdAtBlock = uint64(block.number);
        assertion.isFirstChild = _isFirstChild;
        assertion.configHash = _configHash;
        assertion.status = AssertionStatus.Pending;
        return assertion;
    }

    /**
     * @notice Update child properties
     * @param confirmPeriodBlocks The confirmPeriodBlocks
     */
    function childCreated(AssertionNode storage self, uint64 confirmPeriodBlocks) internal {
        if (self.firstChildBlock == 0) {
            self.firstChildBlock = uint64(block.number);
            self.noChildConfirmedBeforeBlock = uint64(block.number) + confirmPeriodBlocks;
        } else if (self.secondChildBlock == 0) {
            self.secondChildBlock = uint64(block.number);
        }
    }

    /**
     * @notice Update the child confirmed deadline
     * @param deadline The new deadline to set
     */
    function newChildConfirmDeadline(AssertionNode storage self, uint64 deadline) internal {
        self.noChildConfirmedBeforeBlock = deadline;
    }

    /**
     * @notice Check whether the current block number has met or passed the assertion's deadline
     */
    function requirePastDeadline(AssertionNode memory self) internal view {
        require(block.number >= self.deadlineBlock, "BEFORE_DEADLINE");
    }

    /**
     * @notice Check whether the current block number has met or passed deadline for children of this assertion to be confirmed
     */
    function requirePastChildConfirmDeadline(AssertionNode memory self) internal view {
        require(block.number >= self.noChildConfirmedBeforeBlock, "CHILD_TOO_RECENT");
    }

    function requireMoreThanOneChild(AssertionNode memory self) internal pure {
        require(self.secondChildBlock > 0, "TOO_FEW_CHILD");
    }

    function requireExists(AssertionNode memory self) internal pure {
        require(self.status != AssertionStatus.NoAssertion, "ASSERTION_NOT_EXIST");
    }
}
