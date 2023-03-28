// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../challengeV2/DataEntities.sol";
import "../challengeV2/libraries/UintUtilsLib.sol";

contract SpecChallengeManager is ISpecChallengeManager {
    // Checks if a challenge by ID exists.
    function challengeExists(bytes32 challengeId) external view returns (bool) {
        return true;
    }
    // Fetches a challenge object by ID.
    function getChallenge(bytes32 challengeId) external view returns (Challenge memory) {
        return Challenge({
            rootId: bytes32(0),
            winningClaim: bytes32(0),
            challengeType: ChallengeType.Block,
            challenger: address(0)
        });
    }
    // Gets the winning claim ID for a challenge. TODO: Needs more thinking.
    function winningClaim(bytes32 challengeId) external view returns (bytes32) {
        return bytes32(0);
    }
    // Checks if an edge by ID exists.
    function edgeExists(bytes32 eId) external view returns (bool) {
        return true;
    }
    // Gets an edge by ID.
    function getEdge(bytes32 eId) external view returns (ChallengeEdge memory) {
        return ChallengeEdge({
            challengeId: bytes32(0),
            startHistoryRoot: bytes32(0),
            startHeight: 0,
            endHistoryRoot: bytes32(0),
            endHeight: 0,
            lowerChildId: bytes32(0),
            upperChildId: bytes32(0),
            createdWhen: 0,
            status: EdgeStatus.Pending,
            claimEdgeId: bytes32(0),
            staker: address(0)
        });
    }
    // Gets the current ps timer by edge ID. TODO: Needs more thinking.
    // Flushed ps time vs total current ps time needs differentiation
    function getCurrentPsTimer(bytes32 eId) external view returns (uint256) {
        return 0;
    }
    // We define a base ID as hash(challengeType  ++ hash(startCommit ++ startHeight)) as a way
    // of checking if an edge has rivals. Edges can share the same base ID.
    function calculateBaseIdForEdge(bytes32 edgeId) external returns (bytes32) {
        return bytes32(0);
    }
    // Checks if an edge's base ID corresponds to multiple rivals and checks if a one step fork exists.
    function isOneStepForkSource(bytes32 eId) external view returns (bool) {
        return true;
    }
    // Creates a layer zero edge in a challenge.
    function createLayerZeroEdge(AddLeafArgs calldata leafData, bytes calldata proof1, bytes calldata proof2)
        external
        payable
        returns (bytes32) {
        return bytes32(0);
    }
    // Creates a subchallenge on an edge. Emits the challenge ID in an event.
    function createSubChallenge(bytes32 eId) external returns (bytes32) {
        return bytes32(0);
    }
    // Bisects an edge. Emits both children's edge IDs in an event.
    function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes memory prefixProof) external returns (bytes32, bytes32) {
        return (bytes32(0), bytes32(0));
    }
    // Checks if both children of an edge are already confirmed in order to confirm the edge.
    function confirmEdgeByChildren(bytes32 eId) external {
        return;
    }
    // Confirms an edge by edge ID and an array of ancestor edges based on timers.
    function confirmEdgeByTimer(bytes32 eId, bytes32[] memory ancestorIds) external {
        return;
    }
    // If we have created a subchallenge, confirmed a layer 0 edge already, we can use a claim id to confirm edge ids.
    // All edges have two children, unless they only have a link to a claim id.
    function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) external {
        return;
    }
}