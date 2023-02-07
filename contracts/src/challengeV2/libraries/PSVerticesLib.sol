// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";
import "./ChallengeVertexLib.sol";

// CHRIS: TODO: define the remit of this lib
// CHRIS: TODO: defines properties of linked challenge vertices, and how they can be updated
// CHRIS: TODO: we should rename this then, since it isnt just a vertex mapping, it's specific type
// of mapping. 
// CHRIS: TODO: we dont need to put lib in the name here?
library PSVerticesLib {
    using ChallengeVertexLib for ChallengeVertex;

    function hasConfirmablePsAt(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) public view returns (bool) {
        require(vertices[vId].exists(), "Predecessor vertex does not exist");

        // we dont allow presumptive successor to be set to 0 if one is confirmable
        // therefore if it is at 0 we must not have any confirmable presumptive successors
        // or this is a new vertex, so also no confirmable ps
        if (vertices[vId].presumptiveSuccessorId == 0) {
            return false;
        }

        // CHRIS: TODO: rework this to question if we are confirmable
        return getCurrentPsTimer(vertices, vertices[vId].presumptiveSuccessorId) > challengePeriod;
    }

    function getCurrentPsTimer(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        // CHRIS: TODO: is it necessary to check exists everywhere? shoudlnt we just do that in the base? ideally we'd do it here, but it's expensive
        require(vertices[vId].exists(), "Vertex does not exist for ps timer");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");

        bytes32 presumptiveSuccessorId = vertices[predecessorId].presumptiveSuccessorId;
        uint256 flushedPsTimer = vertices[vId].flushedPsTime;
        if (presumptiveSuccessorId == vId) {
            return (block.timestamp - vertices[predecessorId].presumptiveSuccessorLastUpdated) + flushedPsTimer;
        } else {
            return flushedPsTimer;
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
    ) public returns (bytes32) {
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
            presumptiveSuccessorLastUpdated: 0,
            flushedPsTime: successorInitialPsTime,
            lowestHeightSucessorId: 0
        });

        connectVertices(vertices, predecessorId, vId, challengePeriod);

        return vId;
    }

    // CHRIS: TODO: rather than checking if prev exists we could explicitly disallow root?

    // CHRIS: TODO: make all lib functions internal

    function setPresumptiveSuccessor(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        bytes32 presumptiveSuccessorId,
        uint256 challengePeriod
    ) public {
        // CHRIS: TODO: check that this is not a leaf - we cant set the presumptive successor on a leaf
        require(!hasConfirmablePsAt(vertices, vId, challengePeriod), "Presumptive successor already confirmable");

        if (vertices[vId].presumptiveSuccessorId != 0) {
            uint256 timeToAdd = block.timestamp - vertices[vId].presumptiveSuccessorLastUpdated;
            vertices[vertices[vId].presumptiveSuccessorId].flushedPsTime += timeToAdd;
        }
        vertices[vId].presumptiveSuccessorLastUpdated = block.timestamp;
        // CHRIS: TODO: invariants testing here lowest height successor = presumptiveSuccessorId, or presumptiveSuccessorId = 0

        vertices[vId].presumptiveSuccessorId = presumptiveSuccessorId;
        if (presumptiveSuccessorId != 0 && presumptiveSuccessorId != vertices[vId].lowestHeightSucessorId) {
            require(
                vertices[vId].lowestHeightSucessorId == 0
                    || vertices[presumptiveSuccessorId].height < vertices[vertices[vId].lowestHeightSucessorId].height,
                "New height not lower"
            );
            vertices[vId].lowestHeightSucessorId = presumptiveSuccessorId;
        }
    }

    function checkAtOneStepFork(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) public view {
        require(vertices[vId].exists(), "Fork candidate vertex does not exist");

        // CHRIS: TODO: do we want to include this?
        // require(!vertices.hasConfirmablePsAt(predecessorId, challengePeriod), "Presumptive successor confirmable");

        require(vertices[vertices[vId].lowestHeightSucessorId].exists(), "No successors");

        uint256 lowestHeightSuccessorHeight = vertices[vertices[vId].lowestHeightSucessorId].height;
        require(
            lowestHeightSuccessorHeight - vertices[vId].height == 1, "Lowest height not one above the current height"
        );

        require(vertices[vId].presumptiveSuccessorId == 0, "Has presumptive successor");
    }

    // dont allow updates if the challenge has a winner?
    // CHRIS: TODO: require winning claim == 0

    function connectVertices(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 startVertexId,
        bytes32 endVertexId,
        uint256 challengePeriod
    ) public {
        require(vertices[startVertexId].exists(), "Predecessor vertex does not exist");
        require(vertices[endVertexId].exists(), "Successor does not exist");
        
        require(vertices[endVertexId].predecessorId != startVertexId, "Vertices already connected");

        // CHRIS: TODO comments and assertions in here
        // eg. assert that presumptive successor id is also 0 if lowest height = 0

        vertices[endVertexId].predecessorId = startVertexId;
        if (vertices[startVertexId].lowestHeightSucessorId == 0) {
            // no lowest height successor, means no successors at all, so we can set this vertex as the presumptive successor
            setPresumptiveSuccessor(vertices, startVertexId, endVertexId, challengePeriod);
            return;
        }

        uint256 height = vertices[endVertexId].height;
        uint256 lowestHeightSuccessorHeight = vertices[vertices[startVertexId].lowestHeightSucessorId].height;
        if (height < lowestHeightSuccessorHeight) {
            setPresumptiveSuccessor(vertices, startVertexId, endVertexId, challengePeriod);
            return;
        }

        if (height == lowestHeightSuccessorHeight) {
            // if we are at the same height as the ps, then flush the ps and 0 the ps
            setPresumptiveSuccessor(vertices, startVertexId, 0, challengePeriod);
            return;
        }
    }
}
