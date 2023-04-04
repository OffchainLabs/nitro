// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./libraries/UintUtilsLib.sol";
import {IAssertionChain} from "./DataEntities.sol";
import "./libraries/EdgeChallengeManagerLib.sol";

interface IEdgeChallengeManager {
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
    uint256 startHeight;
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
    event LevelZeroEdgeAdded(bytes32 edgeId);

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

    function createLayerZeroEdge(
        CreateEdgeArgs memory args,
        bytes calldata, // CHRIS: TODO: not yet implemented
        bytes calldata
    ) external payable returns (bytes32) {
        bytes32 originId;
        if (args.edgeType == EdgeType.Block) {
            // CHRIS: TODO: check that the assertion chain is in a fork

            // challenge id is the assertion which is the root of challenge
            originId = assertionChain.getPredecessorId(args.claimId);
        } else if (args.edgeType == EdgeType.BigStep) {
            require(store.get(args.claimId).eType == EdgeType.Block, "Claim challenge type is not Block");

            originId = store.get(args.claimId).mutualId();
        } else if (args.edgeType == EdgeType.SmallStep) {
            require(store.get(args.claimId).eType == EdgeType.BigStep, "Claim challenge type is not BigStep");

            originId = store.get(args.claimId).mutualId();
        } else {
            revert("Unexpected challenge type");
        }

        // CHRIS: TODO: sub challenge specific checks, also start and end consistency checks, and claim consistency checks
        // CHRIS: TODO: check the ministake was provided
        // CHRIS: TODO: also prove that the the start root is a prefix of the end root
        // CHRIS: TODO: we had inclusion proofs before?

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

        emit LevelZeroEdgeAdded(ce.id());

        return ce.id();
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
        store.confirmEdgeByOneStepProof(
            edgeId, oneStepProofEntry, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof
        );
    }

    // CHRIS: TODO: remove these?
    ///////////////////////////////////////////////
    ///////////// VIEW FUNCS ///////////////

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
