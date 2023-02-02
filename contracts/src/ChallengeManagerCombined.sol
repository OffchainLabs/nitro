// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import {
    ChallengeVertex,
    Status,
    IChallengeManager,
    ChallengeManagers,
    IWinningClaim,
    Challenge,
    AddLeafArgs,
    ChallengeType,
    IAssertionChain
} from "./DataEntities.sol";

library ChallengeVertexLib {
    function newRoot(bytes32 historyCommitment) internal pure returns (ChallengeVertex memory) {
        // CHRIS: TODO: the root should have a height 1 and should inherit the state commitment from above right?
        return ChallengeVertex({
            predecessorId: 0,
            successionChallenge: 0,
            historyCommitment: historyCommitment, // CHRIS: TODO: this isnt correct - we should compute this from the claim apparently
            height: 0, // CHRIS: TODO: this should be 1 from the spec/paper - DIFF to paper - also in the id
            claimId: 0, // CHRIS: TODO: should this be a reference to the assertion on which this challenge is based? 2-way link?
            status: Status.Confirmed,
            staker: address(0),
            presumptiveSuccessorId: 0,
            presumptiveSuccessorLastUpdated: 0, // CHRIS: TODO: maybe we wanna update this? We should set it as the start time? or are we gonna do special stuff for root?
            flushedPsTime: 0, // always zero for the root
            lowestHeightSucessorId: 0
        });
    }

    function id(bytes32 historyCommitment, uint256 height) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(historyCommitment, height));
    }

    // CHRIS: TODO: duplication for storage/mem - we also dont need `has` AND vertexExists
    function exists(ChallengeVertex storage vertex) internal view returns (bool) {
        return vertex.historyCommitment != 0;
    }

    function existsMem(ChallengeVertex memory vertex) internal pure returns (bool) {
        return vertex.historyCommitment != 0;
    }

    function isLeaf(ChallengeVertex storage vertex) internal view returns (bool) {
        return exists(vertex) && vertex.staker != address(0);
    }

    function isLeafMem(ChallengeVertex memory vertex) internal pure returns (bool) {
        return existsMem(vertex) && vertex.staker != address(0);
    }
}

library ChallengeVertexMappingLib {
    using ChallengeVertexLib for ChallengeVertex;

    function has(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) public view returns (bool) {
        // CHRIS: TODO: this doesnt work for root atm
        return vertices[vId].historyCommitment != 0;
    }

    function hasConfirmablePsAt(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) public view returns (bool) {
        require(has(vertices, vId), "Predecessor vertex does not exist");

        // CHRIS: TODO: rework this to question if we are confirmable
        return getCurrentPsTimer(vertices, vertices[vId].presumptiveSuccessorId) > challengePeriod;
    }

    function getCurrentPsTimer(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        // CHRIS: TODO: is it necessary to check exists everywhere? shoudlnt we just do that in the base? ideally we'd do it here, but it's expensive
        require(has(vertices, vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(has(vertices, predecessorId), "Predecessor vertex does not exist");

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
        bytes32 predecessorId,
        bytes32 successorHistoryCommitment,
        uint256 successorHeight,
        bytes32 successorClaimId,
        address successorStaker,
        uint256 successorInitialPsTime,
        uint256 challengePeriod
    ) public {
        bytes32 vId = ChallengeVertexLib.id(successorHistoryCommitment, successorHeight);
        require(!has(vertices, vId), "Successor already exists");
        require(has(vertices, predecessorId), "Predecessor does not already exist");

        vertices[vId] = ChallengeVertex({
            predecessorId: predecessorId,
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

    // update a successor by
    // 1. flush the ps if required

    // 2. setting a new predecessor
    // 3. setting new lowest height
    // dont allow updates if the challenge has a winner?
    // CHRIS: TODO: require winning claim == 0

    function connectVertices(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 startVertexId,
        bytes32 endVertexId,
        uint256 challengePeriod
    ) public {
        require(has(vertices, startVertexId), "Predecessor vertex does not exist");
        require(has(vertices, endVertexId), "Successor already exists exist");

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

    function bisectionHeight(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        require(has(vertices, vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(has(vertices, predecessorId), "Predecessor vertex does not exist");

        require(vertices[vId].height - vertices[predecessorId].height >= 2, "Height different not two or more");
        // CHRIS: TODO: look at the boundary conditions here
        // CHRIS: TODO: update this to use the correct formula from the paper
        return 10; // placeholder
    }
}

library HistoryCommitmentLib {
    function hasState(bytes32 historyCommitment, bytes32 state, uint256 stateHeight, bytes memory proof)
        internal
        pure
        returns (bool)
    {
        // CHRIS: TODO: do a merkle proof check
        return true;
    }

    function hasPrefix(
        bytes32 historyCommitment,
        bytes32 prefixHistoryCommitment,
        uint256 prefixHistoryHeight,
        bytes memory proof
    ) internal pure returns (bool) {
        // CHRIS: TODO:
        // prove that the sequence of states commited to by prefixHistoryCommitment is a prefix
        // of the sequence of state commited to by the historyCommitment
        return true;
    }
}

library ChallengeManagerLib {
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    function confirmationPreChecks(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) internal view {
        // basic checks
        require(vertices.has(vId), "Vertex does not exist");
        require(vertices[vId].status == Status.Pending, "Vertex is not pending");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices.has(predecessorId), "Predecessor vertex does not exist");

        // for a vertex to be confirmed its predecessor must be confirmed
        // this ensures an unbroken chain of confirmation from the root eventually up to one the leaves
        require(vertices[predecessorId].status == Status.Confirmed, "Predecessor not confirmed");
    }

    // CHRIS: TODO: consider moving this and the other check to the challenge lib
    /// @notice Checks if the vertex is eligible to be confirmed because it has a high enought ps timer
    /// @param vertices The tree of vertices
    /// @param vId The vertex to be confirmed
    /// @param challengePeriod One challenge period in seconds
    function checkConfirmForPsTimer(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) internal view {
        confirmationPreChecks(vertices, vId);

        // ensure only one type of confirmation is valid on this node and all it's siblings
        require(vertices[vertices[vId].predecessorId].successionChallenge == 0, "Succession challenge already opened");

        // now ensure that only one of the siblings is valid for this time of confirmation
        // here we ensure that because only one vertex can ever have a ps timer greater than the challenge period, before the end time
        require(vertices.getCurrentPsTimer(vId) > challengePeriod, "PsTimer not greater than challenge period");
    }

    /// @notice Checks if the vertex is eligible to be confirmed because it has been declared a winner in a succession challenge
    function checkConfirmForSucessionChallengeWin(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 challengeId,
        bytes32 vId
    ) internal view {
        confirmationPreChecks(vertices[challengeId], vId);

        // ensure only one type of confirmation is valid on this node and all it's siblings
        bytes32 successionChallenge =
            vertices[challengeId][vertices[challengeId][vId].predecessorId].successionChallenge;
        require(successionChallenge != 0, "Succession challenge does not exist");

        // now ensure that only one of the siblings is valid for this time of confirmation
        // here we ensure that since a succession challenge only declares one winner
        require(
            challenges[challengeId].winningClaim == vId, "Succession challenge did not declare this vertex the winner"
        );
    }

    // CHRIS: TODO: this func has too many args, cant we simplify it?
    function checkCreateSubChallenge(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 challengeId,
        bytes32 child1Id,
        bytes32 child2Id,
        uint256 challengePeriod
    ) internal view {
        // CHRIS: TODO: should also check the challenge exists?
        require(vertices[challengeId].has(child1Id), "Child 1 does not exist");
        require(vertices[challengeId].has(child2Id), "Child 2 does not exist");

        require(child1Id != child2Id, "Children are not different");

        bytes32 predecessorId = vertices[challengeId][child1Id].predecessorId;
        require(
            predecessorId == vertices[challengeId][child2Id].predecessorId, "Children do not have the same predecessor"
        );

        uint256 predecessorHeight = vertices[challengeId][predecessorId].height;
        require(
            vertices[challengeId][child1Id].height - predecessorHeight == 1,
            "Child 1 is not one step from it's predecessor"
        );
        require(
            vertices[challengeId][child2Id].height - predecessorHeight == 1,
            "Child 2 is not one step from it's predecessor"
        );

        require(challenges[predecessorId].winningClaim == 0, "Winner already declared");

        // CHRIS: TODO: we should check this in every move?
        require(
            !vertices[challengeId].hasConfirmablePsAt(predecessorId, challengePeriod),
            "Presumptive successor confirmable"
        );
        require(vertices[challengeId][predecessorId].successionChallenge == 0, "Challenge already exists");
    }

    // CHRIS: TODO: could use this? and pass it in, but then we may disconnect fron the challenge id
    // mapping(bytes32 => ChallengeVertex) storage v = vertices[challengeId];

    function calculateBisectionVertex(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 challengeId,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        // CHRIS: TODO: put this together with the has confirmable ps check?
        require(challenges[challengeId].winningClaim == 0, "Winner already declared");

        require(vertices[challengeId].has(vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[challengeId][vId].predecessorId;
        require(vertices[challengeId].has(predecessorId), "Predecessor vertex does not exist");
        require(
            vertices[challengeId][predecessorId].presumptiveSuccessorId != vId, "Cannot bisect presumptive successor"
        );

        require(
            !vertices[challengeId].hasConfirmablePsAt(predecessorId, challengePeriod),
            "Presumptive successor already confirmable"
        );

        uint256 bHeight = vertices[challengeId].bisectionHeight(vId);
        require(
            HistoryCommitmentLib.hasPrefix(
                vertices[challengeId][vId].historyCommitment, prefixHistoryCommitment, bHeight, prefixProof
            ),
            "Invalid prefix history"
        );

        return (ChallengeVertexLib.id(prefixHistoryCommitment, bHeight), bHeight);
    }

    function checkBisect(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 challengeId,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.calculateBisectionVertex(
            vertices, challenges, challengeId, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        // CHRIS: redundant check?
        require(!vertices[challengeId].has(bVId), "Bisection vertex already exists");

        return (bVId, bHeight);
    }

    function checkMerge(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 challengeId,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.calculateBisectionVertex(
            vertices, challenges, challengeId, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        require(vertices[challengeId].has(bVId), "Bisection vertex does not already exist");
        // CHRIS: TODO: include a long comment about this
        require(!vertices[challengeId][bVId].isLeaf(), "Cannot merge to a leaf");

        return (bVId, bHeight);
    }

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function checkAddLeaf(
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafArgs calldata leafData,
        uint256 miniStake
    ) internal view {
        require(leafData.claimId != 0, "Empty claimId");
        require(leafData.historyCommitment != 0, "Empty historyCommitment");
        // CHRIS: TODO: we should also prove that the height is greater than 1 if we set the root heigt to 1
        require(leafData.height != 0, "Empty height");

        // CHRIS: TODO: comment on why we need the mini stake
        // CHRIS: TODO: also are we using this to refund moves in real-time? would be more expensive if so, but could be necessary?
        // CHRIS: TODO: this can apparently be moved directly to the public goods fund
        // CHRIS: TODO: we need to record who was on the winning leaf
        require(msg.value == miniStake, "Incorrect mini-stake amount");

        // CHRIS: TODO: require that this challenge hasnt declared a winner
        require(challenges[leafData.challengeId].winningClaim == 0, "Winner already declared");

        // CHRIS: TODO: also check the root is in the history at height 0/1?
        require(
            HistoryCommitmentLib.hasState(
                leafData.historyCommitment, leafData.lastState, leafData.height, leafData.lastStatehistoryProof
            ),
            "Last state not in history"
        );

        require(
            HistoryCommitmentLib.hasState(
                leafData.historyCommitment, leafData.firstState, 0, leafData.firstStatehistoryProof
            ),
            "First state not in history"
        );

        // CHRIS: TODO: check that this is how we form the history commitment for the root

        // we dont know the root id - this is in the challenge itself?
        require(challenges[leafData.challengeId].rootId == ChallengeVertexLib.id(leafData.firstState, 0));
    }
}

contract ChallengeManager is IChallengeManager {
    // CHRIS: TODO: do this in a different way
    // ChallengeManagers internal challengeManagers;

    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeVertexLib for ChallengeVertex;

    mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) public vertices;
    mapping(bytes32 => Challenge) public challenges;
    IAssertionChain public assertionChain;

    uint256 public immutable miniStake = 1 ether; // CHRIS: TODO: fill with value
    uint256 public immutable challengePeriod = 10; // CHRIS: TODO: how to set this, and compare to end time?

    constructor(IAssertionChain _assertionChain) {
        assertionChain = _assertionChain;
    }

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function addLeaf(AddLeafArgs calldata leafData, bytes calldata proof1, bytes calldata proof2) external {
        if (challenges[leafData.challengeId].challengeType == ChallengeType.Block) {
            BlockLeafAdder.addLeaf(
                vertices, challenges, miniStake, challengePeriod, leafData, proof1, proof2, assertionChain
            );
        } else if (challenges[leafData.challengeId].challengeType == ChallengeType.BigStep) {
            BigStepLeafAdder.addLeaf(vertices, challenges, miniStake, challengePeriod, leafData, proof1, proof2);
        } else if (challenges[leafData.challengeId].challengeType == ChallengeType.SmallStep) {
            SmallStepLeafAdder.addLeaf(vertices, challenges, miniStake, challengePeriod, leafData, proof1, proof2);
        } else {
            revert("Unexpected challenge type");
        }
    }

    /// @dev Confirms the vertex without doing any checks. Also sets the winning claim if the vertex
    ///      is a leaf.
    function setConfirmed(bytes32 challengeId, bytes32 vId) internal {
        vertices[challengeId][vId].status = Status.Confirmed;
        if (vertices[challengeId][vId].isLeaf()) {
            challenges[challengeId].winningClaim = vertices[challengeId][vId].claimId;
        }
    }

    function winningClaim(bytes32 challengeId) public view returns (bytes32) {
        // CHRIS: TODO: check exists? or return the full struct?
        return challenges[challengeId].winningClaim;
    }

    function vertexExists(bytes32 challengeId, bytes32 vId) public view returns (bool) {
        return vertices[challengeId].has(vId);
    }

    function challengeExists(bytes32 challengeId) public view returns (bool) {
        // CHRIS: TODO: move to lib
        return challenges[challengeId].rootId != 0;
    }

    function getVertex(bytes32 challengeId, bytes32 vId) public view returns (ChallengeVertex memory) {
        require(vertices[challengeId][vId].exists(), "Vertex does not exist");

        return vertices[challengeId][vId];
    }

    function getCurrentPsTimer(bytes32 challengeId, bytes32 vId) public view returns (uint256) {
        return vertices[challengeId].getCurrentPsTimer(vId);
    }

    // CHRIS: TODO: rename
    function calculateChallengeId(bytes32 challengeOriginId, ChallengeType cType) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeOriginId, cType));
    }

    // CHRIS: TODO: better name for that predcessor id
    // CHRIS: TODO: any access management here? we shouldnt allow the challenge to be created by anyone as this affects the start timer - so we should has the id with teh creating address?
    function createChallenge(bytes32 assertionId) public returns (bytes32) {
        require(msg.sender == address(assertionChain), "Only assertion chain can create challenges");

        // get the state hash of the challenge origin
        bytes32 challengeId = calculateChallengeId(assertionId, ChallengeType.Block);
        require(!challengeExists(challengeId), "Challenge already exists");

        // CHRIS: TODO: we could be more consistent with the root here - it cannot be the same as a vertex id?

        // CHRIS: TODO: calling out to the assertion chain is weird because it makes us reliant on behaviour there, much better to not have to do that have the stuff injected here?

        bytes32 originStateHash = assertionChain.getStateHash(assertionId);
        bytes32 rootId = ChallengeVertexLib.id(originStateHash, 0);
        vertices[challengeId][rootId] = ChallengeVertexLib.newRoot(originStateHash);

        challenges[challengeId] = Challenge({rootId: rootId, challengeType: ChallengeType.Block, winningClaim: 0});
        return challengeId;
    }

    /// @notice Confirm a vertex because it has been the presumptive successor for long enough
    /// @param challengeId The challenge to confirm the vertex in
    /// @param vId The vertex id
    function confirmForPsTimer(bytes32 challengeId, bytes32 vId) public {
        ChallengeManagerLib.checkConfirmForPsTimer(vertices[challengeId], vId, challengePeriod);
        setConfirmed(challengeId, vId);
    }

    /// Confirm a vertex because it has won a succession challenge
    /// @param challengeId The challenge to confirm the vertex in
    /// @param vId The vertex id
    function confirmForSucessionChallengeWin(bytes32 challengeId, bytes32 vId) public {
        ChallengeManagerLib.checkConfirmForSucessionChallengeWin(vertices, challenges, challengeId, vId);
        setConfirmed(challengeId, vId);
    }

    function nextSubChallengeType(ChallengeType cType) internal pure returns (ChallengeType) {
        if (cType == ChallengeType.Block) {
            return ChallengeType.BigStep;
        } else if (cType == ChallengeType.BigStep) {
            return ChallengeType.SmallStep;
            // CHRIS: TODO: everywhere we have a switch we should check we have a revert for everything else
        } else if (cType == ChallengeType.SmallStep) {
            return ChallengeType.OneStep;
        } else {
            revert("Cannot get sub challenge type for one step challenge");
        }
    }

    function createSubChallenge(bytes32 challengeId, bytes32 child1Id, bytes32 child2Id) public {
        ChallengeManagerLib.checkCreateSubChallenge(
            vertices, challenges, challengeId, child1Id, child2Id, challengePeriod
        );

        // CHRIS: TODO: the stuff below should go in a lib or something?

        bytes32 predecessorId = vertices[challengeId][child1Id].predecessorId;
        ChallengeType nextCType = nextSubChallengeType(challenges[challengeId].challengeType);

        // CHRIS: TODO: it should be impossible for two vertices to have the same id, even in different challenges
        // CHRIS: TODO: is this true for the root? no - the root can have the same id
        bytes32 newChallengeId = calculateChallengeId(challengeId, nextCType);
        require(!challengeExists(newChallengeId), "Challenge already exists");

        require(vertices[challengeId][predecessorId].exists(), "Origin Vertex does not exist");
        bytes32 originHistoryCommitment = vertices[challengeId][predecessorId].historyCommitment;

        bytes32 rootId = ChallengeVertexLib.id(originHistoryCommitment, 0);

        // CHRIS: TODO: should we even add the root for the one step? probably not
        vertices[newChallengeId][rootId] = ChallengeVertexLib.newRoot(originHistoryCommitment);
        challenges[newChallengeId] = Challenge({rootId: rootId, challengeType: nextCType, winningClaim: 0});
        vertices[challengeId][predecessorId].successionChallenge = newChallengeId;

        // CHRIS: TODO: opening a challenge and confirming a winner vertex should have mutually exlusive checks
        // CHRIS: TODO: these should ensure this internally
    }

    function bisect(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        public
    {
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.checkBisect(
            vertices, challenges, challengeId, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        // CHRIS: TODO: the spec says we should stop the presumptive successor timer of the vId, but why?
        // CHRIS: TODO: is that because we only care about presumptive successors further down the chain?

        bytes32 predecessorId = vertices[challengeId][vId].predecessorId;
        uint256 currentPsTimer = vertices[challengeId].getCurrentPsTimer(vId);
        vertices[challengeId].addNewSuccessor(
            predecessorId,
            prefixHistoryCommitment,
            bHeight,
            0,
            address(0),
            // CHRIS: TODO: double check the timer updates in here and merge - they're a bit tricky to reason about
            currentPsTimer,
            challengePeriod
        );
        // CHRIS: TODO: check these two successor updates really do conform to the spec
        vertices[challengeId].connectVertices(bVId, vId, challengePeriod);
    }

    function merge(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        public
    {
        (bytes32 bVId,) = ChallengeManagerLib.checkMerge(
            vertices, challenges, challengeId, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        vertices[challengeId].connectVertices(bVId, vId, challengePeriod);
        // setting the presumptive successor to itself will just cause the ps timer to be flushed
        vertices[challengeId].setPresumptiveSuccessor(vertices[challengeId][bVId].predecessorId, bVId, challengePeriod);
        // update the merge vertex if we have a higher ps time
        if (vertices[challengeId][bVId].flushedPsTime < vertices[challengeId][vId].flushedPsTime) {
            vertices[challengeId][bVId].flushedPsTime = vertices[challengeId][vId].flushedPsTime;
        }
    }
}

library BlockLeafAdder {
    // CHRIS: TODO: not all these libs are used
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    function initialPsTime(bytes32 claimId, IAssertionChain assertionChain) internal view returns (uint256) {
        bool isFirstChild = assertionChain.isFirstChild(claimId);

        if (isFirstChild) {
            bytes32 predecessorId = assertionChain.getPredecessorId(claimId);
            uint256 firstChildCreationTime = assertionChain.getFirstChildCreationTime(predecessorId);

            return block.timestamp - firstChildCreationTime;
        } else {
            return 0;
        }
    }

    function getBlockHash(bytes32 assertionStateHash, bytes memory proof) internal returns (bytes32) {
        // CHRIS: TODO:
        // 1. The assertion state hash contains all the info being asserted - including the block hash
        // 2. Extract the block hash from the assertion state hash using the claim proof and return it
    }

    function getInboxMsgProcessedCount(bytes32 assertionStateHash, bytes memory proof) internal returns (uint256) {
        // CHRIS: TODO:
        // 1. Unwrap the assertion state hash to find the number of inbox messages it processed
    }

    function addLeaf(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        uint256 miniStake,
        uint256 challengePeriod,
        AddLeafArgs calldata leafData,
        bytes calldata blockHashProof,
        bytes calldata inboxMsgProcessedProof,
        IAssertionChain assertionChain
    ) public {
        // check that the predecessor of this claim has registered this contract as it's succession challenge
        bytes32 predecessorId = assertionChain.getPredecessorId(leafData.claimId);
        require(
            assertionChain.getSuccessionChallenge(predecessorId) == leafData.challengeId,
            "Claim predecessor not linked to this challenge"
        );

        uint256 assertionHeight = assertionChain.getHeight(leafData.claimId);
        uint256 predecessorAssertionHeight = assertionChain.getHeight(predecessorId);

        uint256 leafHeight = assertionHeight - predecessorAssertionHeight;
        require(leafHeight == leafData.height, "Invalid height");

        bytes32 claimStateHash = assertionChain.getStateHash(leafData.claimId);
        require(
            getInboxMsgProcessedCount(claimStateHash, inboxMsgProcessedProof)
                == assertionChain.getInboxMsgCountSeen(predecessorId),
            "Invalid inbox messages processed"
        );

        require(
            getBlockHash(claimStateHash, blockHashProof) == leafData.lastState,
            "Last state is not the assertion claim block hash"
        );

        ChallengeManagerLib.checkAddLeaf(challenges, leafData, miniStake);

        vertices[leafData.challengeId].addNewSuccessor(
            challenges[leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafData.historyCommitment,
            leafData.height,
            leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            // CHRIS: TODO: this has a nicer pattern by encapsulating the args, could we do the same?
            initialPsTime(leafData.claimId, assertionChain),
            challengePeriod
        );
    }

    // CHRIS: TODO: check exists whenever we access the challenges? also the vertices now have a challenge index
}

library BigStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    function initialPsTime(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        bytes32 challengeOriginId,
        bytes32 challengeId,
        bytes32 claimId
    ) internal view returns (uint256) {
        // CHRIS: TODO: do this validation at a higher layer
        bytes32 originPredecessorId = vertices[challengeOriginId][claimId].predecessorId;
        require(
            vertices[challengeOriginId][originPredecessorId].successionChallenge == challengeId,
            "Invalid challenge origin"
        );

        // CHRIS: TODO: we point upwards with the claim id! that's correct!

        return vertices[challengeId].getCurrentPsTimer(claimId);
    }

    function getBlockHashFromClaim(bytes32 claimId, bytes memory claimProof) internal returns (bytes32) {
        // CHRIS: TODO:
        // 1. Get the history commitment for this claim
        // 2. Unwrap the last state of the claim using the proof
        // 3. Also use the proof to extract the block hash from the last state
        // 4. Return the block hash
    }

    function getBlockHashProducedByTerminalState(bytes32 state, bytes memory stateProof) internal returns (bytes32) {
        // 1. Hydrate the state using the state proof
        // 2. Show that the state is terminal
        // 3. Extract the block hash that is being produced by this terminal state
    }

    function addLeaf(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        uint256 miniStake,
        uint256 challengePeriod,
        AddLeafArgs calldata leafData,
        bytes calldata claimBlockHashProof,
        bytes calldata stateBlockHashProof
    ) internal {
        // CHRIS: TODO: we should only have the special stuff in here, we can pass in the initial ps timer or something

        // CHRIS: TODO: rename challenge to challenge manager
        require(vertices[leafData.parentChallengeId][leafData.claimId].exists(), "Claim does not exist");
        bytes32 predecessorId = vertices[leafData.parentChallengeId][leafData.claimId].predecessorId;
        require(vertices[leafData.parentChallengeId][predecessorId].exists(), "Claim predecessor does not exist");
        require(
            vertices[leafData.parentChallengeId][leafData.claimId].height
                - vertices[leafData.parentChallengeId][predecessorId].height == 1,
            "Claim not height one above predecessor"
        );
        require(
            vertices[leafData.parentChallengeId][predecessorId].successionChallenge == leafData.challengeId,
            "Claim has invalid succession challenge"
        );

        // CHRIS: TODO: check challenge also exists

        // CHRIS: TODO: also check that the claim is a block hash?

        // in a bigstep challenge the states are wasm states, and the claims are block challenge vertices
        // check that the wasm state is a terminal state, and that it produces the blockhash that's in the claim
        bytes32 lastStateBlockHash = getBlockHashProducedByTerminalState(leafData.lastState, stateBlockHashProof);
        bytes32 claimBlockHash = getBlockHashFromClaim(leafData.claimId, claimBlockHashProof);

        require(claimBlockHash == lastStateBlockHash, "Claim inconsistent with state");

        ChallengeManagerLib.checkAddLeaf(challenges, leafData, miniStake);

        vertices[leafData.challengeId].addNewSuccessor(
            challenges[leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafData.historyCommitment,
            leafData.height,
            leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            initialPsTime(vertices, leafData.parentChallengeId, leafData.challengeId, leafData.claimId),
            challengePeriod
        );
    }
}

library SmallStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    uint256 public constant MAX_STEPS = 2 << 19;

    function initialPsTime(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        bytes32 challengeOriginId,
        bytes32 challengeId,
        bytes32 claimId
    ) internal view returns (uint256) {
        // CHRIS: TODO: do this validation at a higher layer
        bytes32 originPredecessorId = vertices[challengeOriginId][claimId].predecessorId;
        require(
            vertices[challengeOriginId][originPredecessorId].successionChallenge == challengeId,
            "Invalid challenge origin"
        );

        // CHRIS: TODO: we point upwards with the claim id! that's correct!

        return vertices[challengeId].getCurrentPsTimer(claimId);
    }

    function getProgramCounter(bytes32 state, bytes memory proof) public returns (uint256) {
        // CHRIS: TODO:
        // 1. hydrate the wavm state with the proof
        // 2. find the program counter and return it
    }

    function addLeaf(
        mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        uint256 miniStake,
        uint256 challengePeriod,
        AddLeafArgs calldata leafData,
        // CHRIS: TODO: these, and in the other add leaves, should be calldata
        bytes calldata claimHistoryProof,
        bytes calldata programCounterProof
    ) internal {
        require(vertices[leafData.parentChallengeId][leafData.claimId].exists(), "Claim does not exist");
        bytes32 predecessorId = vertices[leafData.parentChallengeId][leafData.claimId].predecessorId;
        require(vertices[leafData.parentChallengeId][predecessorId].exists(), "Claim predecessor does not exist");
        require(
            vertices[leafData.parentChallengeId][leafData.claimId].height
                - vertices[leafData.parentChallengeId][predecessorId].height == 1,
            "Claim not height one above predecessor"
        );
        require(
            vertices[leafData.parentChallengeId][predecessorId].successionChallenge == leafData.challengeId,
            "Claim has invalid succession challenge"
        );

        // CHRIS: TODO: should call it "claimChallengeId";

        // the wavm state of the last state should always be exactly the same as the wavm state of the claim
        // regardless of the height
        require(
            HistoryCommitmentLib.hasState(
                vertices[leafData.parentChallengeId][leafData.claimId].historyCommitment,
                leafData.lastState,
                1,
                claimHistoryProof
            ),
            "Invalid claim state"
        );

        uint256 lastStateProgramCounter = getProgramCounter(leafData.lastState, programCounterProof);
        uint256 predecessorSteps = vertices[leafData.parentChallengeId][predecessorId].height * MAX_STEPS;

        require(predecessorSteps + leafData.height == lastStateProgramCounter, "Inconsistent program counter");

        if (ChallengeVertexLib.isLeaf(vertices[leafData.parentChallengeId][predecessorId])) {
            require(leafData.height == MAX_STEPS, "Invalid non-leaf steps");
        } else {
            require(leafData.height <= MAX_STEPS, "Invalid leaf steps");
        }

        ChallengeManagerLib.checkAddLeaf(challenges, leafData, miniStake);

        vertices[leafData.challengeId].addNewSuccessor(
            challenges[leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafData.historyCommitment,
            leafData.height,
            leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            initialPsTime(vertices, leafData.parentChallengeId, leafData.challengeId, leafData.claimId),
            challengePeriod
        );
    }
}
