// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";
import "./ChallengeVertexLib.sol";

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
        require(!vertices[vId].isLeaf(), "Leaf can never be a fork candidate");

        // if this vertex has no successor at all, it cannot be the root of a one step fork
        require(vertices[vertices[vId].lowestHeightSuccessorId].exists(), "No successors");

        // the lowest height must be the root height + 1 at a one step fork
        uint256 lowestHeightSuccessorHeight = vertices[vertices[vId].lowestHeightSuccessorId].height;
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
    /// @param challengePeriodSec The challenge period that the ps timer must exceed
    function psExceedsChallengePeriod(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriodSec
    ) internal view returns (bool) {
        require(vertices[vId].exists(), "Predecessor vertex does not exist");

        // we dont allow presumptive successor to be updated if the ps has a timer that exceeds the challenge period
        // therefore if it is at 0 we must non of the successor must have a high enough timer,
        // or this is a new vertex so it doesnt have any successors, and therefore no high enough ps
        if (vertices[vId].psId == 0) {
            return false;
        }

        return getCurrentPsTimer(vertices, vertices[vId].psId) > challengePeriodSec;
    }

    /// @notice The amount of time (seconds) this vertex has spent as the presumptive successor.
    ///         Use this function instead of the flushPsTime since this function also takes into account unflushed time
    /// @dev    We record ps time using the psLastUpdatedTimestamp on the predecessor vertex, and flush it onto the target it vertex
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
            return (block.timestamp - vertices[predecessorId].psLastUpdatedTimestamp) + vertices[vId].flushedPsTimeSec;
        } else {
            return vertices[vId].flushedPsTimeSec;
        }
    }

    /// @notice Flush the psLastUpdatedTimestamp of a vertex onto the current ps, and record that this occurred.
    ///         Once flushed will also check that the final flushed time is at least the provided minimum
    /// @param vertices The ps vertices
    /// @param vId The id of the vertex on which to update psLastUpdatedTimestamp
    /// @param minFlushedTimeSec A minimum amount to set the flushed ps time to.
    function flushPs(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId, uint256 minFlushedTimeSec)
        internal
    {
        require(vertices[vId].exists(), "Vertex does not exist");
        // leaves should never have a ps, so we cant flush here
        require(!vertices[vId].isLeaf(), "Cannot flush leaf as it will never have a PS");

        // if a presumptive successor already exists we flush it
        if (vertices[vId].psId != 0) {
            uint256 timeToAdd = block.timestamp - vertices[vId].psLastUpdatedTimestamp;
            uint256 timeToSet = vertices[vertices[vId].psId].flushedPsTimeSec + timeToAdd;

            // CHRIS: TODO: we're updating flushed time here! this could accidentally take us above the expected amount
            // CHRIS: TODO: we should check that it's not confirmable
            if (timeToSet < minFlushedTimeSec) {
                timeToSet = minFlushedTimeSec;
            }

            vertices[vertices[vId].psId].setFlushedPsTimeSec(timeToSet);
        }
        // every time we update the ps we record when it happened so that we can flush in the future
        vertices[vId].setPsLastUpdatedTimestamp(block.timestamp);
    }

    /// @notice Connect two existing vertices. The connection is made by setting the predecessor of the end vertex to
    ///         be the start vertex. When the connection is made ps timers, and lowest heigh successor, are updated
    ///         if relevant.
    /// @param vertices The collection of vertices
    /// @param startVertexId The start vertex to connect to
    /// @param endVertexId The end vertex to connect from
    /// @param challengePeriodSec The challenge period - used for checking valid ps timers
    function connect(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 startVertexId,
        bytes32 endVertexId,
        uint256 challengePeriodSec
    ) internal {
        require(vertices[startVertexId].exists(), "Start vertex does not exist");
        // by definition of a leaf no connection can occur if the leaf is a start vertex
        require(!vertices[startVertexId].isLeaf(), "Cannot connect a successor to a leaf");
        require(vertices[endVertexId].exists(), "End vertex does not exist");
        require(vertices[endVertexId].predecessorId != startVertexId, "Vertices already connected");
        require(vertices[startVertexId].height < vertices[endVertexId].height, "Start height not lower than end height");
        // cannot connect vertices that are in different challenges
        require(
            vertices[startVertexId].challengeId == vertices[endVertexId].challengeId,
            "Predecessor and successor are in different challenges"
        );

        // are we newly connecting, then predecessor should be 0

        // first make the connection, then update ps
        vertices[endVertexId].setPredecessor(startVertexId);

        // the current vertex has no successors, in this case the new successor will certainly
        // be the ps
        if (vertices[startVertexId].lowestHeightSuccessorId == 0) {
            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].setPsId(endVertexId);
            return;
        }

        uint256 height = vertices[endVertexId].height;
        uint256 lowestHeightSuccessorHeight = vertices[vertices[startVertexId].lowestHeightSuccessorId].height;
        // we're connect a successor that is lower than the current lowest height, so this new successor
        // will become the ps. Set the ps.
        if (height < lowestHeightSuccessorHeight) {
            // never allow a ps with a timer greater than the challenge period to be replaced
            require(
                !psExceedsChallengePeriod(vertices, startVertexId, challengePeriodSec),
                "Start vertex has ps with timer greater than challenge period, cannot set lower ps"
            );

            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].setPsId(endVertexId);
            return;
        }

        // we're connecting a sibling to the current lowest height, that means that there will be more than
        // one successor at the same lowest height, in this case we set non of the successors to be the ps
        if (height == lowestHeightSuccessorHeight) {
            // never allow a ps with a timer greater than the challenge period to be replaced
            require(
                !psExceedsChallengePeriod(vertices, startVertexId, challengePeriodSec),
                "Start vertex has ps with timer greater than challenge period, cannot set same height ps"
            );

            flushPs(vertices, startVertexId, 0);
            vertices[startVertexId].setPsId(0);
            return;
        }
    }

    /// @notice Adds a vertex to the collection, and connects it to the provided predecessor
    /// @param vertices The vertex collection
    /// @param vertex The vertex to add
    /// @param predecessorId The predecessor this vertex will become a successor to
    /// @param challengePeriodSec The challenge period - used for checking ps timers
    function addVertex(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        ChallengeVertex memory vertex,
        bytes32 predecessorId,
        uint256 challengePeriodSec
    ) internal returns (bytes32) {
        bytes32 vId = vertex.id();
        require(!vertices[vId].exists(), "Vertex already exists");
        vertices[vId] = vertex;

        // connect the newly stored vertex to an existing vertex
        connect(vertices, predecessorId, vId, challengePeriodSec);

        return vId;
    }
}
