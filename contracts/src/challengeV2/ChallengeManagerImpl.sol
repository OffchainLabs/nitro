// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./DataEntities.sol";
import "../osp/IOneStepProofEntry.sol";
import "./libraries/ChallengeVertexLib.sol";
import "./libraries/PsVerticesLib.sol";
import "./libraries/ChallengeStructLib.sol";
import "./libraries/HistoryCommitmentLib.sol";
import "./libraries/ChallengeTypeLib.sol";
import "./libraries/LeafAdderLib.sol";

library ChallengeManagerLib {
    using ChallengeVertexLib for ChallengeVertex;
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeTypeLib for ChallengeType;
    using ChallengeStructLib for Challenge;

    function confirmationPreChecks(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) internal view {
        // basic checks
        
        require(vertices[vId].exists(), "Vertex does not exist");
        require(vertices[vId].status == Status.Pending, "Vertex is not pending");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");

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

    function checkCreateChallenge(
        mapping(bytes32 => Challenge) storage challenges,
        bytes32 assertionId,
        address assertionChain
    ) internal view returns (bytes32) {
        // CHRIS: TODO: use pre-existing rights model contracts
        require(msg.sender == address(assertionChain), "Only assertion chain can create challenges");

        // get the state hash of the challenge origin
        bytes32 challengeId = ChallengeStructLib.id(assertionId, ChallengeType.Block);
        require(!challenges[challengeId].exists(), "Challenge already exists");

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
        // CHRIS: TODO: in every move we should check confirmable behaviour - not just ps
        require(!vertices.psExceedsChallengePeriod(vId, challengePeriod), "Presumptive successor confirmable");
        require(vertices[vId].successionChallenge == 0, "Challenge already exists");

        // we never want to update a vertex if it's already confirmable?
        // we can update it's successor stuff, but not the rest
        // we can but does this mean we can't merge to it? yes, we should be able to
        // but not if the merged node has a confirmable sibling or is confirmable itself?
        // no, we should be able to merge to it, but then what happens if we update


        // if it has a sibling that is confirmable, then we dont allow merge to it
        // if is confirmable itself, we do allow merge to it, and we do allow it to be updated

        bytes32 challengeId = vertices[vId].challengeId;
        ChallengeType nextCType = challenges[challengeId].challengeType.nextType();

        // CHRIS: TODO: it should be impossible for two vertices to have the same id, even in different challenges
        // CHRIS: TODO: is this true for the root? no - the root can have the same id
        // CHRIS: TODO: check that this is the correct challenge origin passing in - we need to better define this
        bytes32 newChallengeId = ChallengeStructLib.id(vId, nextCType);
        require(!challenges[newChallengeId].exists(), "Challenge already exists");

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
        require(vertices[vId].exists(), "Vertex does not exist");
        // CHRIS: TODO: put this together with the has confirmable ps check?
        bytes32 challengeId = vertices[vId].challengeId;
        require(challenges[challengeId].winningClaim == 0, "Winner already declared");

        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");
        require(vertices[predecessorId].presumptiveSuccessorId != vId, "Cannot bisect presumptive successor");

        uint256 bHeight = ChallengeManagerLib.bisectionHeight(vertices, vId);
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
        require(!vertices[bVId].exists(), "Bisection vertex already exists");

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

        require(vertices[bVId].exists(), "Bisection vertex does not already exist");

        // CHRIS: TODO: include a long comment about this - it's actually covered by the connect vertices I think
        require(!vertices[bVId].isLeaf(), "Cannot merge to a leaf");

        return (bVId, bHeight);
    }

    // CHRIS: TODO: this should be view really?
    function checkExecuteOneStep(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        IOneStepProofEntry oneStepProofEntry,
        bytes32 winnerVId,
        OneStepData calldata oneStepData,
        bytes calldata beforeHistoryInclusionProof,
        bytes calldata afterHistoryInclusionProof
    ) internal returns (bytes32) {
        require(vertices[winnerVId].exists(), "Vertex does not exist");
        bytes32 predecessorId = vertices[winnerVId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor does not exist");

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
        require(vertices[vId].exists(), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor vertex does not exist");

        // CHRIS: TODO: look at the boundary conditions here
        return mandatoryBisectionHeight(vertices[predecessorId].height, vertices[vId].height);
    }
}


contract ChallengeManagerImpl is IChallengeManager {
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeVertexLib for ChallengeVertex;
    using ChallengeTypeLib for ChallengeType;
    using ChallengeStructLib for Challenge;

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

    function bisect(bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        external
        returns (bytes32)
    {
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
        // CHRIS: TODO: rename to just `connect`
        vertices.connectVertices(bVId, vId, challengePeriod);

        return bVId;
    }

    function merge(bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof) external returns (bytes32) {
        (bytes32 bVId,) = ChallengeManagerLib.checkMerge(
            vertices, challenges, vId, prefixHistoryCommitment, prefixProof, challengePeriod
        );

        vertices.connectVertices(bVId, vId, challengePeriod);
        vertices.flushPs(vertices[bVId].predecessorId);
        // update the merge vertex if we have a higher ps time
        // CHRIS: TODO: should we allow flushed time to be updated whenever we want like this? or have a lib for this that compares and updates max
        // CHRIS: TODO: we're updating flushed time here! this could accidentally take us above the expected amount
        // CHRIS: TODO: we should check that it's not confirmable
        if (vertices[bVId].flushedPsTime < vertices[vId].flushedPsTime) {
            vertices[bVId].flushedPsTime = vertices[vId].flushedPsTime;
        }

        return bVId;
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

    // EXTERNAL VIEW FUNCTIONS
    // --------------------
    // Functions that are not required internally, and do not update state, but may be useful
    // for external callers.
    // All functions below this point should be external, not just public, and view and not
    // called within this contract.

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
        require(challenges[challengeId].exists(), "Challenge does not exist");
        return challenges[challengeId];
    }

    function vertexExists(bytes32 vId) external view returns (bool) {
        return vertices[vId].exists();
    }

    function getVertex(bytes32 vId) external view returns (ChallengeVertex memory) {
        require(vertices[vId].exists(), "Vertex does not exist");
        return vertices[vId];
    }

    function getCurrentPsTimer(bytes32 vId) external view returns (uint256) {
        return vertices.getCurrentPsTimer(vId);
    }

    // CHRIS: TODO: move to lib?
    function hasConfirmedSibling(bytes32 vId) external view returns (bool) {
        // CHRIS: TODO: consider removal - or put in a lib. COuld be a nice chec in the confirms?

        require(vertices[vId].exists(), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices[predecessorId].exists(), "Predecessor does not exist");

        // sub challenge check
        bytes32 challengeId = vertices[predecessorId].successionChallenge;
        if (challengeId != 0) {
            bytes32 wClaim = challenges[challengeId].winningClaim;
            if (wClaim != 0) {
                // CHRIS: TODO: this should be an assert?
                require(vertices[wClaim].exists(), "Winning claim does not exist");
                if (wClaim == vId) return false;

                return vertices[wClaim].status == Status.Confirmed;
            }
        }

        // ps check
        bytes32 psId = vertices[predecessorId].presumptiveSuccessorId;
        if (psId != 0) {
            require(vertices[psId].exists(), "Presumptive successor does not exist");

            if (psId == vId) return false;
            return vertices[psId].status == Status.Confirmed;
        }

        return false;
    }

    function isAtOneStepFork(bytes32 vId) external view returns (bool) {
        // CHRIS: TODO: remove this function - it hides error messages
        vertices.checkAtOneStepFork(vId);
        return true;   
    }
}
