// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";
import "./ChallengeVertexLib.sol";

// Glossary of terms - we shouldnt need these if they're documented well? also we can find this stuff in the paper?
// successor
// PS
// Lowest height successor
// vertex
// confirmation

// CHRIS: TODO: define the remit of this lib
// CHRIS: TODO: defines properties of linked challenge vertices, and how they can be updated
// CHRIS: TODO: we should rename this then, since it isnt just a vertex mapping, it's specific type
// of mapping.
// CHRIS: TODO: we dont need to put lib in the name here?

// CHRIS: TODO: rather than checking if prev exists we could explicitly disallow root?

// CHRIS: TODO: make all lib functions internal

// CHRIS: TODO: how should we capitalise PS?

// CHRIS: TODO: add this back in maybe?
// function existsAndPredecessor(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
//     public
//     returns (bytes32)
// {
//     // CHRIS: TODO: is it necessary to check exists everywhere? shoudlnt we just do that in the base?
//     // ideally we'd do it at each boundary, but it's expensive. What if we only do it when accessing the vertices
//     // and what if we only access them via this class? is that true? no, because we grab and set other info all over the place
//     // eg succession challenge id
//     require(vertices[vId].exists(), "Vertex does not exist for ps timer");
//     bytes32 predecessorId = vertices[vId].predecessorId;
//     require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");
// }

// dont allow updates if the challenge has a winner? should this be a check at the challenge level?
// CHRIS: TODO: require winning claim == 0

// CHRIS: TODO comments and assertions in here
// eg. assert that presumptive successor id is also 0 if lowest height = 0
// CHRIS: TODO: check all the places we do existance checks - it doesnt seem necessary every where
// CHRIS: TODO: use unique messages if we're checking vertex exists in multiple places
// CHRIS: TODO: check isLeaf when trying to set a predecessor
// CHRIS: TODO: invariants testing here lowest height successor = presumptiveSuccessorId, or presumptiveSuccessorId = 0
// CHRIS: TODO: we could have getters and setters on the vertex props - that we know
// CHRIS: TODO:

// CHRIS: TODO: keep the psExceedsChallengePeriod in here? it gives us the security that we will never
// CHRIS: TODO: connect a vertex it the predecessor has a confirmable ps, but it means we need
// CHRIS: TODO: to always pass that information down, and it doesnt really belong here

// CHRIS: TODO: remove psExceeds from here? - we should have it elsewhere since it has challenge concepts in it
// CHRIS: TODO: however we cant put it in the challenge lib since it's used in the leaf adders
// CHRIS: TODO: it belongs in some base lib but we really want to keep the challenge lib with the challenge
// CHRIS: TODO: since they're so similar? It might become a pain in the ass otherwise to jump around

// CHRIS: TODO: wherever we talk about time we should include the word seconds for clarity

// CHRIS: TODO: we can never be made presumptive successor again
// CHRIS: TODO: this is an invariant we should try to test / assert

/// @title Presumptive Successor Vertices library
/// @author Offchain Labs
/// @notice A collection of challenge vertices linked by: predecessorId, psId and lowestHeightSuccessorId
///         This library allows vertices to be connected and these ids updated only in ways that preserve
///         presumptive successor behaviour
library PsVerticesLib {
    using ChallengeVertexLib for ChallengeVertex;

    /// @notice Check that the vertex is the root of a one step fork. A one step fork is where 2 or more
    ///         vertices are successors to this vertex, and have a height exactly one greater than the height of this vertex.
    /// @param vertices The vertices collection
    /// @param vId      The one step fork root to check
    function checkAtOneStepFork(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) internal view {
        require(vertices[vId].exists(), "Fork candidate vertex does not exist");

        // if this vertex has no successor at all, it cannot be the root of a one step fork
        require(vertices[vertices[vId].lowestHeightSucessorId].exists(), "No successors");

        // the lowest height must be the root height + 1 at a one step fork
        uint256 lowestHeightSuccessorHeight = vertices[vertices[vId].lowestHeightSucessorId].height;
        require(
            lowestHeightSuccessorHeight - vertices[vId].height == 1, "Lowest height not one above the current height"
        );

        // if 2 ore more successors are at the lowest height then the presumptive successor id is 0
        // therefore if the lowest height is 1 greater, and the presumptive successor id is 0 then we have
        // 2 or more successors at a height 1 greater than the root - so the root is a one step fork
        require(vertices[vId].presumptiveSuccessorId == 0, "Has presumptive successor");
    }

    /// @notice Does the presumptive successor of the supplied vertex have a ps timer greater than the provided time
    /// @param vertices The vertices collection
    /// @param vId The vertex whose presumptive successor we are checking
    /// @param challengePeriod The challenge period that the ps timer must exceed
    function psExceedsChallengePeriod(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) internal view returns (bool) {
        require(vertices[vId].exists(), "Predecessor vertex does not exist");

        // we dont allow presumptive successor to be updated if the ps has a timer that exceeds the challenge period
        // therefore if it is at 0 we must non of the successor must have a high enough timer,
        // or this is a new vertex, so also no confirmable ps
        if (vertices[vId].presumptiveSuccessorId == 0) {
            return false;
        }

        return getCurrentPsTimer(vertices, vertices[vId].presumptiveSuccessorId) > challengePeriod;
    }

    /// @notice The amount of time this vertex has spent as the presumptive successor.
    ///         Use this function instead of the flushPsTime since this function also takes into account unflushed time
    /// @dev    We record ps time using the psLastUpdated on the predecessor vertex, and flush it onto the target it vertex
    ///         This means that the flushPsTime does not represent the total ps time where the vertex in question is currently the ps
    /// @param vertices The collection of vertices
    /// @param vId The vertex whose ps timer we want to get
    function getCurrentPsTimer(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        require(vertices[vId].exists(), "Vertex does not exist for ps timer");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");

        if (vertices[predecessorId].presumptiveSuccessorId == vId) {
            return (block.timestamp - vertices[predecessorId].psLastUpdated) + vertices[vId].flushedPsTime;
        } else {
            return vertices[vId].flushedPsTime;
        }
    }

    // include this in a doc
    // a set of ps vertices has the following rules
    // a vertex may have 0 or 1 presumptive successor
    // if the ps exists it is also the lowest height successor
    // if no successors exist there is no ps
    // if only one successor exists at the lowest height, then it is the ps
    // if more than one successor exists at the lowest height, then there is no ps

    // if a vertex is a leaf it will never have a ps

    // CHRIS: TODO: decide on a natspec style

    /// @notice Flush the psLastUpdated time on a node, and record that this occurred
    /// @param vertices The PS vertices
    /// @param vId The id of the vertex on which to update psLastUpdated
    function flushPs(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) internal {
        require(vertices[vId].exists(), "Vertex does not exist");
        require(!vertices[vId].isLeaf(), "Cannot flush leaf as it will never have a PS");

        // if a presumptive successor already exists we flush it
        if (vertices[vId].presumptiveSuccessorId != 0) {
            uint256 timeToAdd = block.timestamp - vertices[vId].psLastUpdated;
            vertices[vertices[vId].presumptiveSuccessorId].flushedPsTime += timeToAdd;
        }
        // every time we update the ps we record when it happened so that we can flush in the future
        vertices[vId].psLastUpdated = block.timestamp;
    }

    function connectVertices(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 startVertexId,
        bytes32 endVertexId,
        uint256 challengePeriod
    ) internal {
        require(vertices[startVertexId].exists(), "Start vertex does not exist");
        require(!vertices[startVertexId].isLeaf(), "Cannot connect a successor to a leaf");
        require(vertices[endVertexId].exists(), "End vertex does not exist");
        require(vertices[endVertexId].predecessorId != startVertexId, "Vertices already connected");

        require(
            !psExceedsChallengePeriod(vertices, startVertexId, challengePeriod),
            "The start vertex has a presumptive successor whose ps time has exceeded the challenge period"
        );

        vertices[endVertexId].predecessorId = startVertexId;
        if (vertices[startVertexId].lowestHeightSucessorId == 0) {
            // no lowest height successor, means no successors at all,
            // so we can set this vertex as the PS and as the lowest height successor
            flushPs(vertices, startVertexId);
            vertices[startVertexId].presumptiveSuccessorId = endVertexId;
            vertices[startVertexId].lowestHeightSucessorId = endVertexId;
            return;
        }

        uint256 height = vertices[endVertexId].height;
        uint256 lowestHeightSuccessorHeight = vertices[vertices[startVertexId].lowestHeightSucessorId].height;
        if (height < lowestHeightSuccessorHeight) {
            // new successor has height lower than the current lowest height
            // so we can set the PS and the lowest height successor
            flushPs(vertices, startVertexId);
            vertices[startVertexId].presumptiveSuccessorId = endVertexId;
            vertices[startVertexId].lowestHeightSucessorId = endVertexId;
            return;
        }

        if (height == lowestHeightSuccessorHeight) {
            // same height as the lowest height successor, we should zero out the PS
            // no update to lowest height successor required
            flushPs(vertices, startVertexId);
            vertices[startVertexId].presumptiveSuccessorId = 0;
            return;
        }
    }

    function addNewSuccessor(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 challengeId,
        bytes32 predecessorId,
        bytes32 successorHistoryCommitment,
        uint256 successorHeight,
        bytes32 successorClaimId,
        address successorStaker,
        uint256 successorInitialPsTime,
        uint256 challengePeriod
    ) internal returns (bytes32) {
        bytes32 vId = ChallengeVertexLib.id(challengeId, successorHistoryCommitment, successorHeight);

        require(!vertices[vId].exists(), "Successor already exists");
        require(vertices[predecessorId].exists(), "Predecessor does not already exist");

        vertices[vId] = ChallengeVertex({
            challengeId: challengeId,
            predecessorId: 0, // CHRIS: TODO: this is a bit weird - it will get set when we connect the vertices below
            successionChallenge: 0,
            historyCommitment: successorHistoryCommitment,
            height: successorHeight,
            claimId: successorClaimId,
            staker: successorStaker,
            status: Status.Pending,
            presumptiveSuccessorId: 0,
            psLastUpdated: 0,
            flushedPsTime: successorInitialPsTime,
            lowestHeightSucessorId: 0
        });

        connectVertices(vertices, predecessorId, vId, challengePeriod);

        return vId;
    }
}
