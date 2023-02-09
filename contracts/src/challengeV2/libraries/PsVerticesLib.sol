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
// CHRIS: TODO: invariants testing here lowest height successor = psId, or psId = 0
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

// CHRIS: TODO: should we also not allow connection if another vertex is confirmed, or if this start vertex
// has a chosen winner of a succession challenge?

// CHRIS: TODO: think about what happens if we add a new vertex with a high initial ps

// CHRIS: TODO: some docs to put somewhere
// include this in a doc - on the struct
// a set of ps vertices has the following rules
// a vertex may have 0 or 1 presumptive successor
// if the ps exists it is also the lowest height successor
// if no successors exist there is no ps
// if only one successor exists at the lowest height, then it is the ps
// if more than one successor exists at the lowest height, then there is no ps

// if a vertex is a leaf it will never have a ps

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
        require(vertices[vId].psId == 0, "Has presumptive successor");
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
        // or this is a new vertex so it doesnt have any successors, and therefore no high enough ps
        if (vertices[vId].psId == 0) {
            return false;
        }

        return getCurrentPsTimer(vertices, vertices[vId].psId) > challengePeriod;
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

        if (vertices[predecessorId].psId == vId) {
            // if the vertex is currently the presumptive one we add the flushed time and the unflushed time
            return (block.timestamp - vertices[predecessorId].psLastUpdated) + vertices[vId].flushedPsTime;
        } else {
            return vertices[vId].flushedPsTime;
        }
    }

    /// @notice Flush the psLastUpdated of a vertex onto the current ps, and record that this occurred.
    ///         Once flushed will also check that the final flushed time is at least the provided minimum
    /// @param vertices The ps vertices
    /// @param vId The id of the vertex on which to update psLastUpdated
    /// @param minFlushedTime A minimum amount to set the flushed ps time to.
    function flushPs(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId, uint256 minFlushedTime)
        internal
    {
        require(vertices[vId].exists(), "Vertex does not exist");
        // leaves should never have a ps, so we cant flush here
        require(!vertices[vId].isLeaf(), "Cannot flush leaf as it will never have a PS");

        // if a presumptive successor already exists we flush it
        if (vertices[vId].psId != 0) {
            uint256 timeToAdd = block.timestamp - vertices[vId].psLastUpdated;
            vertices[vertices[vId].psId].flushedPsTime += timeToAdd;

            // CHRIS: TODO: we're updating flushed time here! this could accidentally take us above the expected amount
            // CHRIS: TODO: we should check that it's not confirmable
            if (vertices[vertices[vId].psId].flushedPsTime < minFlushedTime) {
                vertices[vertices[vId].psId].flushedPsTime = minFlushedTime;
            }
        }
        // every time we update the ps we record when it happened so that we can flush in the future
        vertices[vId].psLastUpdated = block.timestamp;
    }

    /// @notice Connect two existing vertices. The connection is made by setting the predecessor of the end vertex to
    ///         be the start vertex. When the connection is made ps timers, and lowest heigh successor, are updated
    ///         if relevant.
    /// @param vertices The collection of vertices
    /// @param startVertexId The start vertex to connect to
    /// @param endVertexId The end vertex to connect from
    /// @param challengePeriod The challenge period - used for checking valid ps timers
    function connectVertices(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 startVertexId,
        bytes32 endVertexId,
        uint256 challengePeriod
    ) internal {
        require(vertices[startVertexId].exists(), "Start vertex does not exist");
        // by definition of a leaf no connection can occur if the leaf is a start vertex
        require(!vertices[startVertexId].isLeaf(), "Cannot connect a successor to a leaf");
        require(vertices[endVertexId].exists(), "End vertex does not exist");
        require(vertices[endVertexId].predecessorId != startVertexId, "Vertices already connected");
        // cannot connect vertices that are in different challenges
        require(
            vertices[startVertexId].challengeId == vertices[endVertexId].challengeId,
            "Predecessor and successor are in different challenges"
        );

        // if the start vertex has a ps that exceeds the challenge period then we dont allow any connection
        // if that vertex as the start. This is because any connected vertex would be rejectable by default
        require(
            !psExceedsChallengePeriod(vertices, startVertexId, challengePeriod),
            "The start vertex has a presumptive successor whose ps time has exceeded the challenge period"
        );

        // first make the connection
        vertices[endVertexId].predecessorId = startVertexId;

        // now we may need to update ps and the lowest height successor
        if (vertices[startVertexId].lowestHeightSucessorId == 0) {
            // no lowest height successor, means no successors at all,
            // so we can set this vertex as the ps and as the lowest height successor
            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].psId = endVertexId;
            vertices[startVertexId].lowestHeightSucessorId = endVertexId;
            return;
        }

        uint256 height = vertices[endVertexId].height;
        uint256 lowestHeightSuccessorHeight = vertices[vertices[startVertexId].lowestHeightSucessorId].height;
        if (height < lowestHeightSuccessorHeight) {
            // new successor has height lower than the current lowest height
            // so we can set the PS and the lowest height successor
            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].psId = endVertexId;
            vertices[startVertexId].lowestHeightSucessorId = endVertexId;
            return;
        }

        if (height == lowestHeightSuccessorHeight) {
            // same height as the lowest height successor, we should zero out the PS
            // no update to lowest height successor required
            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].psId = 0;
            return;
        }
    }

    /// @notice Adds a vertex to the collection, and connects it to the provided predecessor
    /// @param vertices The vertex collection
    /// @param vertex The vertex to add
    /// @param predecessorId The predecessor this vertex will become a successor to
    /// @param challengePeriod The challenge period - used for checking ps timers
    function addVertex(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        ChallengeVertex memory vertex,
        bytes32 predecessorId,
        uint256 challengePeriod
    ) internal returns (bytes32) {
        bytes32 vId = ChallengeVertexLib.id(vertex.challengeId, vertex.historyRoot, vertex.height);
        require(!vertices[vId].exists(), "Vertex already exists");

        vertices[vId] = vertex;
        connectVertices(vertices, predecessorId, vId, challengePeriod);

        return vId;
    }
}
