// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./DataEntities.sol";
import "./osp/IOneStepProofEntry.sol";

library ChallengeVertexLib {
    function newRoot(bytes32 challengeId, bytes32 historyCommitment) internal pure returns (ChallengeVertex memory) {
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
            lowestHeightSucessorId: 0,
            challengeId: challengeId
        });
    }

    function id(bytes32 challengeId, bytes32 historyCommitment, uint256 height) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeId, historyCommitment, height));
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
        require(has(vertices, vId), "Vertex does not exist for ps timer");
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
        require(!has(vertices, vId), "Successor already exists");
        require(has(vertices, predecessorId), "Predecessor does not already exist");

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
        require(has(vertices, vId), "Fork candidate vertex does not exist");

        // CHRIS: TODO: do we want to include this?
        // require(!vertices.hasConfirmablePsAt(predecessorId, challengePeriod), "Presumptive successor confirmable");

        require(has(vertices, vertices[vId].lowestHeightSucessorId), "No successors");

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

    // take from https://solidity-by-example.org/bitwise/
    // Find most significant bit using binary search
    function mostSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        // x >= 2 ** 128
        if (x >= 0x100000000000000000000000000000000) {
            x >>= 128;
            msb += 128;
        }
        // x >= 2 ** 64
        if (x >= 0x10000000000000000) {
            x >>= 64;
            msb += 64;
        }
        // x >= 2 ** 32
        if (x >= 0x100000000) {
            x >>= 32;
            msb += 32;
        }
        // x >= 2 ** 16
        if (x >= 0x10000) {
            x >>= 16;
            msb += 16;
        }
        // x >= 2 ** 8
        if (x >= 0x100) {
            x >>= 8;
            msb += 8;
        }
        // x >= 2 ** 4
        if (x >= 0x10) {
            x >>= 4;
            msb += 4;
        }
        // x >= 2 ** 2
        if (x >= 0x4) {
            x >>= 2;
            msb += 2;
        }
        // x >= 2 ** 1
        if (x >= 0x2) msb += 1;
    }

    // CHRIS: TODO: move this and the above out of here
    function mandatoryBisectionHeight(uint256 start, uint256 end) internal pure returns (uint256) {
        require(end - start >= 2, "Height different not two or more");
        if (end - start == 2) {
            return start + 1;
        }

        uint256 mostSignificantSharedBit = mostSignificantBit((end - 1) ^ start);
        uint256 mask = type(uint256).max << mostSignificantSharedBit;
        return (end - 1) & mask;
    }

    function bisectionHeight(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        require(has(vertices, vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(has(vertices, predecessorId), "Predecessor vertex does not exist");

        // CHRIS: TODO: look at the boundary conditions here
        return mandatoryBisectionHeight(vertices[predecessorId].height, vertices[vId].height);
    }
}

library ChallengeMappingLib {
    function has(mapping(bytes32 => Challenge) storage challenges, bytes32 challengeId) public view returns (bool) {
        // CHRIS: TODO: this doesnt work for root atm
        return challenges[challengeId].rootId != 0;
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
    using ChallengeMappingLib for mapping(bytes32 => Challenge);
    using ChallengeTypeLib for ChallengeType;

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
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 vId
    ) internal view {
        confirmationPreChecks(vertices, vId);

        // ensure only one type of confirmation is valid on this node and all it's siblings
        bytes32 successionChallenge = vertices[vertices[vId].predecessorId].successionChallenge;
        require(successionChallenge != 0, "Succession challenge does not exist");

        // now ensure that only one of the siblings is valid for this time of confirmation
        // here we ensure that since a succession challenge only declares one winner
        require(
            challenges[successionChallenge].winningClaim == vId,
            "Succession challenge did not declare this vertex the winner"
        );
    }

    // CHRIS: TODO: rename and move
    function calculateChallengeId(bytes32 challengeOriginId, ChallengeType cType) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeOriginId, cType));
    }

    function checkCreateChallenge(
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 assertionId,
        address assertionChain
    ) internal view returns (bytes32) {
        // CHRIS: TODO: use pre-existing rights model contracts
        require(msg.sender == address(assertionChain), "Only assertion chain can create challenges");

        // get the state hash of the challenge origin
        bytes32 challengeId = calculateChallengeId(assertionId, ChallengeType.Block);
        require(!challenges.has(challengeId), "Challenge already exists");

        return challengeId;
    }

    function checkCreateSubChallenge(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 vId,
        uint256 challengePeriod
    ) internal view returns (bytes32, ChallengeType) {
        vertices.checkAtOneStepFork(vId);

        require(challenges[vId].winningClaim == 0, "Winner already declared");

        // CHRIS: TODO: we should check this in every move?
        require(!vertices.hasConfirmablePsAt(vId, challengePeriod), "Presumptive successor confirmable");
        require(vertices[vId].successionChallenge == 0, "Challenge already exists");

        bytes32 challengeId = vertices[vId].challengeId;
        ChallengeType nextCType = challenges[challengeId].challengeType.nextType();

        // CHRIS: TODO: it should be impossible for two vertices to have the same id, even in different challenges
        // CHRIS: TODO: is this true for the root? no - the root can have the same id
        bytes32 newChallengeId = calculateChallengeId(challengeId, nextCType);
        require(!challenges.has(newChallengeId), "Challenge already exists");

        return (newChallengeId, nextCType);
    }

    function calculateBisectionVertex(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        require(vertices.has(vId), "Vertex does not exist");
        // CHRIS: TODO: put this together with the has confirmable ps check?
        bytes32 challengeId = vertices[vId].challengeId;
        require(challenges[challengeId].winningClaim == 0, "Winner already declared");

        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices.has(predecessorId), "Predecessor vertex does not exist");
        require(vertices[predecessorId].presumptiveSuccessorId != vId, "Cannot bisect presumptive successor");

        require(
            !vertices.hasConfirmablePsAt(predecessorId, challengePeriod), "Presumptive successor already confirmable"
        );

        uint256 bHeight = vertices.bisectionHeight(vId);
        require(
            HistoryCommitmentLib.hasPrefix(
                vertices[vId].historyCommitment, prefixHistoryCommitment, bHeight, prefixProof
            ),
            "Invalid prefix history"
        );

        return (ChallengeVertexLib.id(challengeId, prefixHistoryCommitment, bHeight), bHeight);
    }

    function checkBisect(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.calculateBisectionVertex(
            vertices, challenges, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        // CHRIS: redundant check?
        require(!vertices.has(bVId), "Bisection vertex already exists");

        return (bVId, bHeight);
    }

    function checkMerge(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof,
        uint256 challengePeriod
    ) internal view returns (bytes32, uint256) {
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.calculateBisectionVertex(
            vertices, challenges, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        require(vertices.has(bVId), "Bisection vertex does not already exist");
        // CHRIS: TODO: include a long comment about this
        require(!vertices[bVId].isLeaf(), "Cannot merge to a leaf");

        return (bVId, bHeight);
    }

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function checkAddLeaf(
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafArgs memory leafData,
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

        // CHRIS: TODO: do we need to pass in first state if we can derive it from the root id?
        require(
            HistoryCommitmentLib.hasState(
                leafData.historyCommitment, leafData.firstState, 0, leafData.firstStatehistoryProof
            ),
            "First state not in history"
        );

        // CHRIS: TODO: we dont know the root id - this is in the challenge itself?

        require(
            challenges[leafData.challengeId].rootId
                == ChallengeVertexLib.id(leafData.challengeId, leafData.firstState, 0),
            "First state is not the challenge root"
        );
    }

    function checkExecuteOneStep(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        IOneStepProofEntry oneStepProofEntry,
        bytes32 winnerVId,
        OneStepData calldata oneStepData,
        bytes calldata beforeHistoryInclusionProof,
        bytes calldata afterHistoryInclusionProof
    ) internal returns (bytes32) {
        require(vertices.has(winnerVId), "Vertex does not exist");
        bytes32 predecessorId = vertices[winnerVId].predecessorId;
        require(vertices.has(predecessorId), "Predecessor does not exist");

        bytes32 challengeId = vertices[predecessorId].successionChallenge;
        require(challengeId != 0, "Succession challenge does not exist");
        require(
            challenges[challengeId].challengeType == ChallengeType.OneStep,
            "Challenge is not at one step execution point"
        );

        // check that the before hash is the state has of the root id
        // the root id is challenge id combined with the history commitment and the height
        // bytes32 historyCommitment, bytes32 state, uint256 stateHeight, bytes memory proof
        require(
            HistoryCommitmentLib.hasState(
                vertices[predecessorId].historyCommitment,
                oneStepData.beforeHash,
                oneStepData.machineStep,
                beforeHistoryInclusionProof
            ),
            "Before state not in history"
        );

        // CHRIS: TODO: validate the execCtx?
        bytes32 afterHash = oneStepProofEntry.proveOneStep(
            oneStepData.execCtx, oneStepData.machineStep, oneStepData.beforeHash, oneStepData.proof
        );

        require(
            HistoryCommitmentLib.hasState(
                vertices[winnerVId].historyCommitment,
                afterHash,
                oneStepData.machineStep + 1,
                afterHistoryInclusionProof
            ),
            "After state not in history"
        );

        return challengeId;
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
        return bytes32(proof);
        // CHRIS: TODO:
        // 1. The assertion state hash contains all the info being asserted - including the block hash
        // 2. Extract the block hash from the assertion state hash using the claim proof and return it
    }

    function getInboxMsgProcessedCount(bytes32 assertionStateHash, bytes memory proof) internal returns (uint256) {
        return uint256(bytes32(bytes(proof)));
        // CHRIS: TODO:
        // 1. Unwrap the assertion state hash to find the number of inbox messages it processed
    }

    function addLeaf(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs, // CHRIS: TODO: better name
        IAssertionChain assertionChain
    ) public returns (bytes32) {
        {
            // check that the predecessor of this claim has registered this contract as it's succession challenge
            bytes32 predecessorId = assertionChain.getPredecessorId(leafLibArgs.leafData.claimId);
            require(
                assertionChain.getSuccessionChallenge(predecessorId) == leafLibArgs.leafData.challengeId,
                "Claim predecessor not linked to this challenge"
            );

            uint256 assertionHeight = assertionChain.getHeight(leafLibArgs.leafData.claimId);
            uint256 predecessorAssertionHeight = assertionChain.getHeight(predecessorId);

            uint256 leafHeight = assertionHeight - predecessorAssertionHeight;
            require(leafHeight == leafLibArgs.leafData.height, "Invalid height");

            bytes32 claimStateHash = assertionChain.getStateHash(leafLibArgs.leafData.claimId);
            require(
                getInboxMsgProcessedCount(claimStateHash, leafLibArgs.proof2)
                    == assertionChain.getInboxMsgCountSeen(predecessorId),
                "Invalid inbox messages processed"
            );

            require(
                getBlockHash(claimStateHash, leafLibArgs.proof1) == leafLibArgs.leafData.lastState,
                "Last state is not the assertion claim block hash"
            );

            ChallengeManagerLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }

        return vertices.addNewSuccessor(
            leafLibArgs.leafData.challengeId,
            challenges[leafLibArgs.leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafLibArgs.leafData.historyCommitment,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            // CHRIS: TODO: this has a nicer pattern by encapsulating the args, could we do the same?
            initialPsTime(leafLibArgs.leafData.claimId, assertionChain),
            leafLibArgs.challengePeriod
        );
    }

    // CHRIS: TODO: check exists whenever we access the challenges? also the vertices now have a challenge index
}

library BigStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

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
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs // CHRIS: TODO: better name
    ) internal returns (bytes32) {
        {
            // CHRIS: TODO: we should only have the special stuff in here, we can pass in the initial ps timer or something
            // CHRIS: TODO: rename challenge to challenge manager
            require(vertices[leafLibArgs.leafData.claimId].exists(), "Claim does not exist");
            bytes32 predecessorId = vertices[leafLibArgs.leafData.claimId].predecessorId;
            require(vertices[predecessorId].exists(), "Claim predecessor does not exist");
            require(
                vertices[leafLibArgs.leafData.claimId].height - vertices[predecessorId].height == 1,
                "Claim not height one above predecessor"
            );
            require(
                vertices[predecessorId].successionChallenge == leafLibArgs.leafData.challengeId,
                "Claim has invalid succession challenge"
            );

            // CHRIS: TODO: check challenge also exists

            // CHRIS: TODO: also check that the claim is a block hash?

            // in a bigstep challenge the states are wasm states, and the claims are block challenge vertices
            // check that the wasm state is a terminal state, and that it produces the blockhash that's in the claim
            bytes32 lastStateBlockHash =
                getBlockHashProducedByTerminalState(leafLibArgs.leafData.lastState, leafLibArgs.proof1);
            bytes32 claimBlockHash = getBlockHashFromClaim(leafLibArgs.leafData.claimId, leafLibArgs.proof2);

            require(claimBlockHash == lastStateBlockHash, "Claim inconsistent with state");

            ChallengeManagerLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }
        return vertices.addNewSuccessor(
            leafLibArgs.leafData.challengeId,
            challenges[leafLibArgs.leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafLibArgs.leafData.historyCommitment,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            vertices.getCurrentPsTimer(leafLibArgs.leafData.claimId),
            leafLibArgs.challengePeriod
        );
    }
}

library SmallStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    uint256 public constant MAX_STEPS = 2 << 19;

    function getProgramCounter(bytes32 state, bytes memory proof) public returns (uint256) {
        // CHRIS: TODO:
        // 1. hydrate the wavm state with the proof
        // 2. find the program counter and return it
        return uint256(bytes32(proof));
    }

    function addLeaf(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs
    ) internal returns (bytes32) {
        {
            require(vertices[leafLibArgs.leafData.claimId].exists(), "Claim does not exist");
            bytes32 predecessorId = vertices[leafLibArgs.leafData.claimId].predecessorId;
            require(vertices[predecessorId].exists(), "Claim predecessor does not exist");
            require(
                vertices[leafLibArgs.leafData.claimId].height - vertices[predecessorId].height == 1,
                "Claim not height one above predecessor"
            );
            require(
                vertices[predecessorId].successionChallenge == leafLibArgs.leafData.challengeId,
                "Claim has invalid succession challenge"
            );

            // CHRIS: TODO: should call it "claimChallengeId";

            // the wavm state of the last state should always be exactly the same as the wavm state of the claim
            // regardless of the height
            require(
                HistoryCommitmentLib.hasState(
                    vertices[leafLibArgs.leafData.claimId].historyCommitment,
                    leafLibArgs.leafData.lastState,
                    1,
                    leafLibArgs.proof1
                ),
                "Invalid claim state"
            );

            // CHRIS: TODO: document and align the proogs
            uint256 lastStateProgramCounter = getProgramCounter(leafLibArgs.leafData.lastState, leafLibArgs.proof2);
            uint256 predecessorSteps = vertices[predecessorId].height * MAX_STEPS;

            require(
                predecessorSteps + leafLibArgs.leafData.height == lastStateProgramCounter,
                "Inconsistent program counter"
            );

            // CHRIS: TODO: re-enable this leaf check
            // if (!ChallengeVertexLib.isLeaf(vertices[leafLibArgs.leafData.claimId])) {
            //     require(leafLibArgs.leafData.height == MAX_STEPS, "Invalid non-leaf steps");
            // } else {
            //     require(leafLibArgs.leafData.height <= MAX_STEPS, "Invalid leaf steps");
            // }

            ChallengeManagerLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }
        return vertices.addNewSuccessor(
            leafLibArgs.leafData.challengeId,
            challenges[leafLibArgs.leafData.challengeId].rootId,
            // CHRIS: TODO: move this struct out
            leafLibArgs.leafData.historyCommitment,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            // CHRIS: TODO: the naming is bad here
            vertices.getCurrentPsTimer(leafLibArgs.leafData.claimId),
            leafLibArgs.challengePeriod
        );
    }
}

// CHRIS: TODO: check that all the lib functions have the correct visibility

library ChallengeTypeLib {
    function nextType(ChallengeType cType) internal pure returns (ChallengeType) {
        if (cType == ChallengeType.Block) {
            return ChallengeType.BigStep;
        } else if (cType == ChallengeType.BigStep) {
            return ChallengeType.SmallStep;
            // CHRIS: TODO: everywhere we have a switch we should check we have a revert for everything else
        } else if (cType == ChallengeType.SmallStep) {
            return ChallengeType.OneStep;
        } else {
            revert("Cannot get next challenge type for one step challenge");
        }
    }
}

contract ChallengeManager is IChallengeManager {
    // CHRIS: TODO: do this in a different way
    // ChallengeManagers internal challengeManagers;

    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeMappingLib for mapping(bytes32 => Challenge);
    using ChallengeTypeLib for ChallengeType;

    mapping(bytes32 => ChallengeVertex) public vertices;
    mapping(bytes32 => Challenge) public challenges;
    IAssertionChain public assertionChain;
    IOneStepProofEntry oneStepProofEntry;

    uint256 public immutable miniStakeValue;
    uint256 public immutable challengePeriod;

    constructor(
        IAssertionChain _assertionChain,
        uint256 _miniStakeValue,
        uint256 _challengePeriod,
        IOneStepProofEntry _oneStepProofEntry
    ) {
        assertionChain = _assertionChain;
        miniStakeValue = _miniStakeValue;
        challengePeriod = _challengePeriod;
        oneStepProofEntry = _oneStepProofEntry;
    }

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function addLeaf(AddLeafArgs calldata leafData, bytes calldata proof1, bytes calldata proof2)
        external
        payable
        override
        returns (bytes32)
    {
        if (challenges[leafData.challengeId].challengeType == ChallengeType.Block) {
            return BlockLeafAdder.addLeaf(
                vertices,
                challenges,
                AddLeafLibArgs({
                    miniStake: miniStakeValue,
                    challengePeriod: challengePeriod,
                    leafData: leafData,
                    proof1: proof1,
                    proof2: proof2
                }),
                assertionChain
            );
        } else if (challenges[leafData.challengeId].challengeType == ChallengeType.BigStep) {
            return BigStepLeafAdder.addLeaf(
                vertices,
                challenges,
                AddLeafLibArgs({
                    miniStake: miniStakeValue,
                    challengePeriod: challengePeriod,
                    leafData: leafData,
                    proof1: proof1,
                    proof2: proof2
                })
            );
        } else if (challenges[leafData.challengeId].challengeType == ChallengeType.SmallStep) {
            return SmallStepLeafAdder.addLeaf(
                vertices,
                challenges,
                AddLeafLibArgs({
                    miniStake: miniStakeValue,
                    challengePeriod: challengePeriod,
                    leafData: leafData,
                    proof1: proof1,
                    proof2: proof2
                })
            );
        } else {
            revert("Unexpected challenge type");
        }
    }

    /// @dev Confirms the vertex without doing any checks. Also sets the winning claim if the vertex
    ///      is a leaf.
    function setConfirmed(bytes32 vId) internal {
        vertices[vId].status = Status.Confirmed;
        bytes32 challengeId = vertices[vId].challengeId;
        if (vertices[vId].isLeaf()) {
            challenges[challengeId].winningClaim = vertices[vId].claimId;
        }
    }

    // CHRIS: TODO: better name for that predcessor id
    // CHRIS: TODO: any access management here? we shouldnt allow the challenge to be created by anyone as this affects the start timer - so we should has the id with teh creating address?
    function createChallenge(bytes32 assertionId) public returns (bytes32) {
        bytes32 challengeId = ChallengeManagerLib.checkCreateChallenge(challenges, assertionId, address(assertionChain));

        // CHRIS: TODO: we could be more consistent with the root here - it cannot be the same as a vertex id?

        // CHRIS: TODO: calling out to the assertion chain is weird because it makes us reliant on behaviour there, much better to not have to do that have the stuff injected here?
        // CHRIS: TODO: whenever we call an external function we should make a list of the assumptions we're making about the external contract

        // CHRIS: TODO: we should have an existance check
        bytes32 originStateHash = assertionChain.getStateHash(assertionId);
        bytes32 rootId = ChallengeVertexLib.id(challengeId, originStateHash, 0);

        vertices[rootId] = ChallengeVertexLib.newRoot(challengeId, originStateHash);
        challenges[challengeId] = Challenge({rootId: rootId, challengeType: ChallengeType.Block, winningClaim: 0});

        return challengeId;
    }

    /// @notice Confirm a vertex because it has been the presumptive successor for long enough
    /// @param vId The vertex id
    function confirmForPsTimer(bytes32 vId) public {
        ChallengeManagerLib.checkConfirmForPsTimer(vertices, vId, challengePeriod);
        setConfirmed(vId);
    }

    /// Confirm a vertex because it has won a succession challenge
    /// @param vId The vertex id
    function confirmForSucessionChallengeWin(bytes32 vId) public {
        ChallengeManagerLib.checkConfirmForSucessionChallengeWin(vertices, challenges, vId);
        setConfirmed(vId);
    }

    // CHRIS: TODO: the challengeid is stored in the children..

    function createSubChallenge(bytes32 vId) public returns (bytes32) {
        (bytes32 newChallengeId, ChallengeType newChallengeType) =
            ChallengeManagerLib.checkCreateSubChallenge(vertices, challenges, vId, challengePeriod);

        bytes32 originHistoryCommitment = vertices[vId].historyCommitment;
        bytes32 rootId = ChallengeVertexLib.id(newChallengeId, originHistoryCommitment, 0);

        // CHRIS: TODO: should we even add the root for the one step? probably not
        vertices[rootId] = ChallengeVertexLib.newRoot(newChallengeId, originHistoryCommitment);
        challenges[newChallengeId] = Challenge({rootId: rootId, challengeType: newChallengeType, winningClaim: 0});
        vertices[vId].successionChallenge = newChallengeId;

        // CHRIS: TODO: opening a challenge and confirming a winner vertex should have mutually exlusive checks
        // CHRIS: TODO: these should ensure this internally
        return newChallengeId;
    }

    // CHRIS: TODO: everywhere change commitment to root

    function executeOneStep(
        bytes32 winnerVId,
        OneStepData calldata oneStepData,
        bytes calldata beforeHistoryInclusionProof,
        bytes calldata afterHistoryInclusionProof
    ) public returns (bytes32) {
        bytes32 challengeId = ChallengeManagerLib.checkExecuteOneStep(
            vertices,
            challenges,
            oneStepProofEntry,
            winnerVId,
            oneStepData,
            beforeHistoryInclusionProof,
            afterHistoryInclusionProof
        );
        challenges[challengeId].winningClaim = winnerVId;
    }

    function bisect(bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof) external returns (bytes32) {
        // CHRIS: TODO: we calculate this again below when we call addnewsuccessor?
        (bytes32 bVId, uint256 bHeight) = ChallengeManagerLib.checkBisect(
            vertices, challenges, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        // CHRIS: TODO: the spec says we should stop the presumptive successor timer of the vId, but why?
        // CHRIS: TODO: is that because we only care about presumptive successors further down the chain?

        bytes32 predecessorId = vertices[vId].predecessorId;
        uint256 currentPsTimer = vertices.getCurrentPsTimer(vId);
        vertices.addNewSuccessor(
            vertices[vId].challengeId,
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
        vertices.connectVertices(bVId, vId, challengePeriod);

        return bVId;
    }

    function merge(bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof) external returns (bytes32) {
        (bytes32 bVId,) = ChallengeManagerLib.checkMerge(
            vertices, challenges, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        vertices.connectVertices(bVId, vId, challengePeriod);
        // setting the presumptive successor to itself will just cause the ps timer to be flushed
        vertices.setPresumptiveSuccessor(vertices[bVId].predecessorId, bVId, challengePeriod);
        // update the merge vertex if we have a higher ps time
        if (vertices[bVId].flushedPsTime < vertices[vId].flushedPsTime) {
            vertices[bVId].flushedPsTime = vertices[vId].flushedPsTime;
        }

        return bVId;
    }

    // EXTERNAL FUNCTIONS
    // --------------------
    // Functions that are not required internally but may be useful for external
    // callers. 
    // All functions below this point should be external, not just public.

    function winningClaim(bytes32 challengeId) external view returns (bytes32) {
        // CHRIS: TODO: check exists? or return the full struct?
        return challenges[challengeId].winningClaim;
    }

    function challengeExists(bytes32 challengeId) external view returns (bool) {
        // CHRIS: TODO: move to lib
        return challenges[challengeId].rootId != 0;
    }

    function getChallenge(bytes32 challengeId) external view returns (Challenge memory) {
        // CHRIS: TODO: move this into a lib - we should have a challengeMapping lib
        require(challenges.has(challengeId), "Challenge does not exist");
        return challenges[challengeId];
    }

    function vertexExists(bytes32 vId) external view returns (bool) {
        return vertices.has(vId);
    }

    function getVertex(bytes32 vId) external view returns (ChallengeVertex memory) {
        require(vertices.has(vId), "Vertex does not exist");
        return vertices[vId];
    }

    function getCurrentPsTimer(bytes32 vId) external view returns (uint256) {
        return vertices.getCurrentPsTimer(vId);
    }

    function hasConfirmedSibling(bytes32 vId) external view returns (bool) {
        // CHRIS: TODO: consider removal - or put in a lib. COuld be a nice chec in the confirms?

        require(vertices.has(vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices.has(predecessorId), "Predecessor does not exist");

        // sub challenge check
        bytes32 challengeId = vertices[predecessorId].successionChallenge;
        if (challengeId != 0) {
            bytes32 wClaim = challenges[challengeId].winningClaim;
            if (wClaim != 0) {
                // CHRIS: TODO: this should be an assert?
                require(vertices.has(wClaim), "Winning claim does not exist");
                if (wClaim == vId) return false;

                return vertices[wClaim].status == Status.Confirmed;
            }
        }

        // ps check
        bytes32 psId = vertices[predecessorId].presumptiveSuccessorId;
        if (psId != 0) {
            require(vertices.has(psId), "Presumptive successor does not exist");

            if (psId == vId) return false;
            return vertices[psId].status == Status.Confirmed;
        }

        return false;
    }

    function isAtOneStepFork(bytes32 vId) external view returns (bool) {
        // CHRIS: TODO: remove this function - it hides error messages
        try vertices.checkAtOneStepFork(vId) {
            return true;
        } catch {
            return false;
        }
    }
}
