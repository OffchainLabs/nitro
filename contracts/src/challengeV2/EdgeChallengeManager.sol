// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./libraries/UintUtilsLib.sol";
import "./DataEntities.sol";
import "./libraries/EdgeChallengeManagerLib.sol";
import "../libraries/Constants.sol";

interface IEdgeChallengeManager {
    // Checks if an edge by ID exists.
    function edgeExists(bytes32 eId) external view returns (bool);

    function initialize(
        IAssertionChain _assertionChain,
        uint256 _challengePeriodSec,
        IOneStepProofEntry _oneStepProofEntry
    ) external;

    // // Checks if an edge by ID exists.
    // function edgeExists(bytes32 eId) external view returns (bool);
    // Gets an edge by ID.
    function getEdge(bytes32 eId) external view returns (ChallengeEdge memory);

    // Gets the current time unrivaled by edge ID. TODO: Needs more thinking.
    function timeUnrivaled(bytes32 eId) external view returns (uint256);

    // We define a mutual ID as hash(EdgeType  ++ originId ++ hash(startCommit ++ startHeight)) as a way
    // of checking if an edge has rivals. Rivals edges share the same mutual ID.
    function calculateMutualId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) external returns (bytes32);

    function calculateEdgeId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) external returns (bytes32);

    // Checks if an edge's mutual ID corresponds to multiple rivals and checks if a one step fork exists.
    function hasRival(bytes32 eId) external view returns (bool);

    // Checks if an edge's mutual ID corresponds to multiple rivals and checks if a one step fork exists.
    function hasLengthOneRival(bytes32 eId) external view returns (bool);

    // Creates a layer zero edge in a challenge.
    function createLayerZeroEdge(CreateEdgeArgs memory args, bytes calldata, bytes calldata)
        external
        payable
        returns (bytes32);

    // Bisects an edge. Emits both children's edge IDs in an event.
    function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32);

    // Checks if both children of an edge are already confirmed in order to confirm the edge.
    function confirmEdgeByChildren(bytes32 eId) external;

    // Confirms an edge by edge ID and an array of ancestor edges based on total time unrivaled
    function confirmEdgeByTime(bytes32 eId, bytes32[] memory ancestorIds) external;

    // If we have created a subchallenge, confirmed a layer 0 edge already, we can use a claim id to confirm edge ids.
    // All edges have two children, unless they only have a link to a claim id.
    function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) external;

    // when we reach a one step fork in a small step challenge we can confirm
    // the edge by executing a one step proof to show the edge is valid
    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) external;
}

struct CreateEdgeArgs {
    EdgeType edgeType;
    bytes32 startHistoryRoot;
    uint256 startHeight; // TODO: This isn't necessary because it's always 0. Do we want it?
    bytes32 endHistoryRoot;
    uint256 endHeight;
    bytes32 claimId;
}

// CHRIS: TODO: more examples in the merkle expansion
// CHRIS: TODO: explain that 0 represents the level

// CHRIS: TODO: invariants
// 1. edges are only created, never destroyed
// 2. all edges have at least one parent, or a claim id - other property invariants exist
// 3. all edges have a mutual id, and that mutual id must have an entry in firstRivals
// 4. all values of firstRivals are existing edges (must be in the edge mapping), or are the NO_RIVAL magic hash
// 5. where to check edge prefix proofs? in bisection, or in add?

contract EdgeChallengeManager is IEdgeChallengeManager {
    using EdgeChallengeManagerLib for EdgeStore;
    using ChallengeEdgeLib for ChallengeEdge;

    event Bisected(bytes32 bisectedEdgeId);
    event LayerZeroEdgeAdded(bytes32 edgeId);

    EdgeStore internal store;

    uint256 public challengePeriodSec;
    IAssertionChain internal assertionChain;
    IOneStepProofEntry oneStepProofEntry;

    constructor(IAssertionChain _assertionChain, uint256 _challengePeriodSec, IOneStepProofEntry _oneStepProofEntry) {
        // HN: TODO: remove constructor?
        initialize(_assertionChain, _challengePeriodSec, _oneStepProofEntry);
    }

    function initialize(
        IAssertionChain _assertionChain,
        uint256 _challengePeriodSec,
        IOneStepProofEntry _oneStepProofEntry
    ) public {
        require(address(assertionChain) == address(0), "ALREADY_INIT");
        assertionChain = _assertionChain;
        challengePeriodSec = _challengePeriodSec;
        oneStepProofEntry = _oneStepProofEntry;
    }

    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32)
    {
        return store.bisectEdge(edgeId, bisectionHistoryRoot, prefixProof);
    }

    function createLayerZeroEdge(CreateEdgeArgs memory args, bytes calldata prefixProof, bytes calldata proof)
        external
        payable
        returns (bytes32)
    {
        bytes32 originId;
        require(args.startHeight == 0, "Start height is not 0");
        if (args.edgeType == EdgeType.Block) {
            // origin id is the assertion which is the root of challenge
            originId = assertionChain.getPredecessorId(args.claimId);
            // HN: TODO: check if prev is rejected
            require(assertionChain.isPending(args.claimId), "Claim assertion is not pending");
            require(assertionChain.getSuccessionChallenge(originId) != 0, "Assertion is not in a fork");

            require(args.endHeight == LAYERZERO_BLOCKEDGE_HEIGHT, "Invalid block edge end height");

            // check that the start history root is the hash of the previous assertion
            require(
                args.startHistoryRoot == keccak256(abi.encodePacked(assertionChain.getStateHash(originId))),
                "Start history root does not match previous assertion"
            );

            // check that the end history root is consistent with the claim
            require(proof.length > 0, "Block edge specific proof is empty");
            bytes32[] memory inclusionProof = abi.decode(proof, (bytes32[]));
            MerkleTreeLib.verifyInclusionProof(
                args.endHistoryRoot,
                assertionChain.getStateHash(args.claimId),
                LAYERZERO_BLOCKEDGE_HEIGHT,
                inclusionProof
            );

            // HN: TODO: do we want to enforce this here? if no block edge is created the rollup cannot confirm by timer on its own
            // HN: TODO: spec said 2 challenge period, should we change it to 1?
            // check if the top level challenge has reached the end time
            require(
                block.timestamp - assertionChain.getFirstChildCreationTime(originId) < 2 * challengePeriodSec,
                "Challenge period has expired"
            );
        } else {
            // common logics for sub-challenges with a higher level claim
            ChallengeEdge storage claimEdge = store.get(args.claimId);
            // origin id is the mutual id of the claim
            originId = claimEdge.mutualId();
            require(claimEdge.status == EdgeStatus.Pending, "Claim is not pending");
            require(store.hasLengthOneRival(args.claimId), "Claim does not have length 1 rival");

            require(proof.length > 0, "Edge type specific proof is empty");
            (
                bytes32 startState,
                bytes32 endState,
                bytes32[] memory claimStartInclusionProof,
                bytes32[] memory claimEndInclusionProof,
                bytes32[] memory edgeInclusionProof
            ) = abi.decode(proof, (bytes32, bytes32, bytes32[], bytes32[], bytes32[]));

            // if the start and end states are consistent with both the claim the roots in the arguments, then the roots in the arguments are consistent with the claim
            // check the states are consistent with the claims
            MerkleTreeLib.verifyInclusionProof(
                claimEdge.startHistoryRoot, startState, claimEdge.startHeight, claimStartInclusionProof
            );
            MerkleTreeLib.verifyInclusionProof(
                claimEdge.endHistoryRoot, endState, claimEdge.endHeight, claimEndInclusionProof
            );
            // check that the start state is consistent with the root in the argument
            require(
                args.startHistoryRoot == keccak256(abi.encodePacked(startState)),
                "Start history root does not match mutual startHistoryRoot"
            );
            // we check that the end state is consistent with the roots in the arguments below

            ChallengeEdge storage topLevelEdge;
            if (args.edgeType == EdgeType.BigStep) {
                require(claimEdge.eType == EdgeType.Block, "Claim challenge type is not Block");
                require(args.endHeight == LAYERZERO_BIGSTEPEDGE_HEIGHT, "Invalid bigstep edge end height");

                // check the endState is consistent with the endHistoryRoot
                MerkleTreeLib.verifyInclusionProof(
                    args.endHistoryRoot, endState, LAYERZERO_BIGSTEPEDGE_HEIGHT, edgeInclusionProof
                );

                topLevelEdge = claimEdge;
            } else if (args.edgeType == EdgeType.SmallStep) {
                require(claimEdge.eType == EdgeType.BigStep, "Claim challenge type is not BigStep");
                require(args.endHeight == LAYERZERO_SMALLSTEPEDGE_HEIGHT, "Invalid smallstep edge end height");

                // check the endState is consistent with the endHistoryRoot
                MerkleTreeLib.verifyInclusionProof(
                    args.endHistoryRoot, endState, LAYERZERO_SMALLSTEPEDGE_HEIGHT, edgeInclusionProof
                );

                // origin of the smallstep edge is the mutual id of block edge
                // TODO: make a getter in EdgeChallengeManagerLib instead of reading store.firstRivals directly
                topLevelEdge = store.get(store.firstRivals[claimEdge.originId]);
            } else {
                revert("Unexpected challenge type");
            }

            // check if the top level challenge has reached the end time
            require(block.timestamp - topLevelEdge.createdWhen < challengePeriodSec, "Challenge period has expired");
        }

        // prove that the start root is a prefix of the end root
        {
            require(prefixProof.length > 0, "Prefix proof is empty");
            (bytes32[] memory preExpansion, bytes32[] memory preProof) = abi.decode(prefixProof, (bytes32[], bytes32[]));
            MerkleTreeLib.verifyPrefixProof(
                args.startHistoryRoot,
                args.startHeight + 1,
                args.endHistoryRoot,
                args.endHeight + 1,
                preExpansion,
                preProof
            );
        }

        // CHRIS: TODO: sub challenge specific checks, also start and end consistency checks, and claim consistency checks
        // CHRIS: TODO: check the ministake was provided
        // CHRIS: TODO: we had inclusion proofs before?

        // CHRIS: TODO: currently the claim id is not part of the edge id hash, this means that two edges with the same id cannot have a different claim id
        // CHRIS: TODO: this method needs to enforce that this is not possible by tying the end state to the claim id somehow, to ensure that it's not logically
        // CHRIS: TODO: possible to have the same endHistoryRoot but a different claim id. This needs to be done for all edge types

        ChallengeEdge memory ce = ChallengeEdgeLib.newLayerZeroEdge(
            originId,
            args.startHistoryRoot,
            args.startHeight,
            args.endHistoryRoot,
            args.endHeight,
            args.claimId,
            msg.sender,
            args.edgeType
        );

        store.add(ce);

        emit LayerZeroEdgeAdded(ce.idMem());

        return ce.idMem();
    }

    function confirmEdgeByChildren(bytes32 edgeId) public {
        store.confirmEdgeByChildren(edgeId);
    }

    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) public {
        store.confirmEdgeByClaim(edgeId, claimingEdgeId);
    }

    function confirmEdgeByTime(bytes32 edgeId, bytes32[] memory ancestorEdges) public {
        store.confirmEdgeByTime(edgeId, ancestorEdges, challengePeriodSec);
    }

    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) public {
        bytes32 prevAssertionId = store.getPrevAssertionId(edgeId);
        ExecutionContext memory execCtx = ExecutionContext({
            maxInboxMessagesRead: assertionChain.getInboxMsgCountSeen(prevAssertionId),
            bridge: assertionChain.bridge(),
            initialWasmModuleRoot: assertionChain.getWasmModuleRoot(prevAssertionId)
        });

        store.confirmEdgeByOneStepProof(
            edgeId, oneStepProofEntry, oneStepData, execCtx, beforeHistoryInclusionProof, afterHistoryInclusionProof
        );
    }

    // CHRIS: TODO: remove these?
    ///////////////////////////////////////////////
    ///////////// VIEW FUNCS ///////////////

    function getPrevAssertionId(bytes32 edgeId) public view returns (bytes32) {
        return store.getPrevAssertionId(edgeId);
    }

    function hasRival(bytes32 edgeId) public view returns (bool) {
        return store.hasRival(edgeId);
    }

    function timeUnrivaled(bytes32 edgeId) public view returns (uint256) {
        return store.timeUnrivaled(edgeId);
    }

    function hasLengthOneRival(bytes32 edgeId) public view returns (bool) {
        return store.hasLengthOneRival(edgeId);
    }

    function calculateEdgeId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) public pure returns (bytes32) {
        return
            ChallengeEdgeLib.idComponent(edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot);
    }

    function calculateMutualId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.mutualIdComponent(edgeType, originId, startHeight, startHistoryRoot, endHeight);
    }

    function edgeExists(bytes32 edgeId) public view returns (bool) {
        return store.edges[edgeId].exists();
    }

    function getEdge(bytes32 edgeId) public view returns (ChallengeEdge memory) {
        return store.get(edgeId);
    }

    function firstRival(bytes32 edgeId) public view returns (bytes32) {
        return store.firstRivals[edgeId];
    }

    function edgeLength(bytes32 edgeId) public view returns (uint256) {
        return store.get(edgeId).length();
    }
}
