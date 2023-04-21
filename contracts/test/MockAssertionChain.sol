// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import {IAssertionChain} from "../src/challengeV2/DataEntities.sol";
import { IEdgeChallengeManager } from "../src/challengeV2/EdgeChallengeManager.sol";
import "../src/bridge/IBridge.sol";
import "../src/rollup/RollupLib.sol";
import "./challengeV2/StateTools.sol";

struct MockAssertion {
    bytes32 predecessorId;
    uint256 height;
    uint256 inboxMsgCountSeen;
    bytes32 stateHash;
    bytes32 successionChallenge;
    uint256 firstChildCreationBlock;
    uint256 secondChildCreationBlock;
    bool isFirstChild;
    bool isPending;
}

contract MockAssertionChain is IAssertionChain {
    mapping(bytes32 => MockAssertion) assertions;
    IBridge public bridge; // TODO: set bridge in this mock
    bytes32 public wasmModuleRoot;

    function assertionExists(bytes32 assertionId) public view returns (bool) {
        return assertions[assertionId].stateHash != 0;
    }

    function getPredecessorId(bytes32 assertionId) public view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].predecessorId;
    }

    function getHeight(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].height;
    }

    function proveInboxMsgCountSeen(bytes32 assertionId, uint256 inboxMsgCountSeen, bytes memory proof) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        require(
            RollupLib.stateHashMem(abi.decode(proof, (ExecutionState)), inboxMsgCountSeen) ==
                assertions[assertionId].stateHash,
            "Inbox msg count proof does not match assertion"
        );
        return inboxMsgCountSeen;
    }

    function getStateHash(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].stateHash;
    }

    function hasSibling(bytes32 assertionId) external view returns (bool) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return (assertions[getPredecessorId(assertionId)].secondChildCreationBlock != 0);
    }

    function getFirstChildCreationBlock(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].firstChildCreationBlock;
    }

    function proveWasmModuleRoot(bytes32 assertionId, bytes32 root, bytes memory proof) external view returns (bytes32){
        (bytes32 lastHash, bytes32 assertionExecHash, bytes32 inboxAcc) = abi.decode(proof, (bytes32, bytes32, bytes32));
        require(
            RollupLib.assertionHash({
                lastHash: lastHash,
                assertionExecHash: assertionExecHash,
                inboxAcc: inboxAcc,
                wasmModuleRoot: root
            }) == assertionId,
            "Wasm module root proof does not match assertion"
        );
        return root;
    }

    function isFirstChild(bytes32 assertionId) external view returns (bool) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].isFirstChild;
    }

    function isPending(bytes32 assertionId) external view returns (bool) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].isPending;
    }

    function calculateAssertionId(
        bytes32 predecessorId, 
        State memory beforeState,
        State memory afterState
    )
        public
        view
        returns (bytes32)
    {
        bytes32 executionHash = RollupLib.executionHash(AssertionInputs({
            beforeState: beforeState.es,
            afterState: afterState.es
        }));
        return RollupLib.assertionHash({
            lastHash: predecessorId,
            assertionExecHash: executionHash,
            inboxAcc: keccak256(abi.encode(afterState.es.globalState.u64Vals[0])), // mock accumulator based on inbox count 
            wasmModuleRoot: wasmModuleRoot
        });
    }

    function childCreated(bytes32 assertionId) internal {
        if (assertions[assertionId].firstChildCreationBlock == 0) {
            assertions[assertionId].firstChildCreationBlock = block.number;
        } else if (assertions[assertionId].secondChildCreationBlock == 0) {
            assertions[assertionId].secondChildCreationBlock = block.number;
        }
    }

    function addAssertionUnsafe(
        bytes32 predecessorId,
        uint256 height,
        uint256 inboxMsgCountSeen,
        State memory beforeState,
        State memory afterState,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 afterStateHash = StateToolsLib.hash(afterState);
        bytes32 assertionId = calculateAssertionId(predecessorId, beforeState, afterState);
        assertions[assertionId] = MockAssertion({
            predecessorId: predecessorId,
            height: height,
            inboxMsgCountSeen: inboxMsgCountSeen,
            stateHash: afterStateHash,
            successionChallenge: successionChallenge,
            firstChildCreationBlock: 0,
            secondChildCreationBlock: 0,
            isFirstChild: assertions[predecessorId].firstChildCreationBlock == 0,
            isPending: true
        });
        childCreated(predecessorId);
        return assertionId;
    }

    function addAssertion(
        bytes32 predecessorId,
        uint256 height,
        uint256 inboxMsgCountSeen,
        State memory beforeState,
        State memory afterState,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 beforeStateHash = StateToolsLib.hash(beforeState);
        bytes32 afterStateHash = StateToolsLib.hash(afterState);
        bytes32 assertionId = calculateAssertionId(predecessorId, beforeState, afterState);
        require(!assertionExists(assertionId), "Assertion already exists");
        require(assertionExists(predecessorId), "Predecessor does not exists");
        require(height > assertions[predecessorId].height, "Height too low");
        require(inboxMsgCountSeen >= assertions[predecessorId].inboxMsgCountSeen, "Inbox count seen too low");
        require(beforeStateHash == assertions[predecessorId].stateHash, "Before state hash does not match predecessor");

        return addAssertionUnsafe(predecessorId, height, inboxMsgCountSeen, beforeState, afterState, successionChallenge);
    }
}
