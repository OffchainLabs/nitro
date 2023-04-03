// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import {IAssertionChain} from "../src/challengeV2/DataEntities.sol";
import { IEdgeChallengeManager } from "../src/challengeV2/EdgeChallengeManager.sol";

struct MockAssertion {
    bytes32 predecessorId;
    uint256 height;
    uint256 inboxMsgCountSeen;
    bytes32 stateHash;
    bytes32 successionChallenge;
    uint256 firstChildCreationTime;
    bool isFirstChild;
}

contract MockAssertionChain is IAssertionChain {
    mapping(bytes32 => MockAssertion) assertions;

    function assertionExists(bytes32 assertionId) public view returns (bool) {
        return assertions[assertionId].stateHash != 0;
    }

    function getPredecessorId(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].predecessorId;
    }

    function getHeight(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].height;
    }

    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].inboxMsgCountSeen;
    }

    function getStateHash(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].stateHash;
    }

    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].successionChallenge;
    }

    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].firstChildCreationTime;
    }

    function isFirstChild(bytes32 assertionId) external view returns (bool) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].isFirstChild;
    }

    function calculateAssertionId(bytes32 predecessorId, uint256 height, bytes32 stateHash)
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(predecessorId, height, stateHash));
    }

    function addAssertionUnsafe(
        bytes32 predecessorId,
        uint256 height,
        uint256 inboxMsgCountSeen,
        bytes32 stateHash,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 assertionId = calculateAssertionId(predecessorId, height, stateHash);
        assertions[assertionId] = MockAssertion({
            predecessorId: predecessorId,
            height: height,
            inboxMsgCountSeen: inboxMsgCountSeen,
            stateHash: stateHash,
            successionChallenge: successionChallenge,
            firstChildCreationTime: 0,
            isFirstChild: assertions[predecessorId].firstChildCreationTime != 0
        });
        return assertionId;
    }

    function addAssertion(
        bytes32 predecessorId,
        uint256 height,
        uint256 inboxMsgCountSeen,
        bytes32 stateHash,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 assertionId = calculateAssertionId(predecessorId, height, stateHash);
        require(!assertionExists(assertionId), "Assertion already exists");
        require(assertionExists(predecessorId), "Predecessor does not exists");
        require(height > assertions[predecessorId].height, "Height too low");
        require(inboxMsgCountSeen >= assertions[predecessorId].inboxMsgCountSeen, "Inbox count seen too low");
        require(stateHash != 0, "Empty state hash");

        return addAssertionUnsafe(predecessorId, height, inboxMsgCountSeen, stateHash, successionChallenge);
    }
}
