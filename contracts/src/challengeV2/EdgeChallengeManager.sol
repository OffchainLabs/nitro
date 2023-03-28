// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./libraries/UintUtilsLib.sol";
import "./DataEntities.sol";
import "./libraries/MerkleTreeLib.sol";

enum EdgeStatus {
    Pending, // This vertex is vertex is pending, it has yet to be confirmed
    Confirmed // This vertex has been confirmed, once confirmed it cannot be unconfirmed
}

struct EChallenge {
    bytes32 baseId;
    ChallengeType cType;
}

struct ChallengeEdge {
    bytes32 challengeId;
    bytes32 startHistoryRoot;
    uint256 startHeight;
    bytes32 endHistoryRoot;
    uint256 endHeight;
    bytes32 lowerChildId; // start -> middle
    bytes32 upperChildId; // middle -> end
    uint256 createdWhen;
    EdgeStatus status;
    bytes32 claimEdgeId; // only on layer zero edge. Claim must have same start point and challenge id as this edge
    address staker; // only on layer zero edge
}

library ChallengeEdgeLib {
    function baseIdComponent(bytes32 challengeId, bytes32 startHistoryRoot, uint256 startHeight, uint256 endHeight)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(challengeId, startHistoryRoot, startHeight, endHeight));
    }
    // CHRIS: TODO: merge these two functions - provide one with accepts the specific args, then call internally

    function baseId(ChallengeEdge storage ce) internal view returns (bytes32) {
        return baseIdComponent(ce.challengeId, ce.startHistoryRoot, ce.startHeight, ce.endHeight);
    }

    function idComponent(
        bytes32 challengeId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight
    ) internal pure returns (bytes32) {
        // CHRIS: TODO: consider if we need to include the claim id here? that shouldnt be necessary if we have the correct checks in createZeroLayerEdge
        return keccak256(abi.encodePacked(challengeId, startHistoryRoot, startHeight, endHistoryRoot, endHeight));
    }

    function id(ChallengeEdge memory edge) internal pure returns (bytes32) {
        // CHRIS: TODO: consider if we need to include the claim id here? that shouldnt be necessary if we have the correct checks in createZeroLayerEdge
        return
            idComponent(edge.challengeId, edge.startHistoryRoot, edge.startHeight, edge.endHistoryRoot, edge.endHeight);
    }

    function exists(ChallengeEdge storage edge) internal view returns (bool) {
        return edge.createdWhen != 0;
    }
}

library EChallengeLib {
    function id(EChallenge memory c) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(c.cType, c.baseId));
    }
}

struct EdgeStore {
    mapping(bytes32 => ChallengeEdge) edges;
    // CHRIS: TODO: explain better what we're doing with the base records
    mapping(bytes32 => bytes32) baseRecords;
}

library EdgeStoreLib {
    bytes32 constant IS_PRESUMPTIVE = keccak256(abi.encodePacked("IS PRESUMPTIVE"));

    using ChallengeEdgeLib for ChallengeEdge;

    function add(EdgeStore storage s, ChallengeEdge memory ce) internal {
        // CHRIS: TODO: check that the children are empty?

        bytes32 eId = ce.id();

        require(!s.edges[eId].exists(), "Edge already exists");
        s.edges[eId] = ce;

        bytes32 baseId =
            ChallengeEdgeLib.baseIdComponent(ce.challengeId, ce.startHistoryRoot, ce.startHeight, ce.endHeight);
        bytes32 baseRecord = s.baseRecords[baseId];
        if (baseRecord == 0) {
            s.baseRecords[baseId] = IS_PRESUMPTIVE;
        } else if (baseRecord == IS_PRESUMPTIVE) {
            s.baseRecords[baseId] = eId;
        }
    }

    function get(EdgeStore storage s, bytes32 edgeId) internal view returns (ChallengeEdge storage) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        return s.edges[edgeId];
    }

    function has(EdgeStore storage s, bytes32 edgeId) internal view returns (bool) {
        return s.edges[edgeId].exists();
    }

    function setChildren(EdgeStore storage s, bytes32 edgeId, bytes32 lowerChildId, bytes32 upperChildId) internal {
        require(s.edges[edgeId].exists(), "Edge does not exist");
        // CHRIS: TODO: check the ids arent zero?
        require(s.edges[lowerChildId].exists(), "Lower does not exist");
        require(s.edges[upperChildId].exists(), "Upper does not exist");
        require(s.edges[edgeId].lowerChildId == 0 && s.edges[edgeId].upperChildId == 0, "Non empty children");

        s.edges[edgeId].lowerChildId = lowerChildId;
        s.edges[edgeId].upperChildId = upperChildId;
    }

    function isPresumptive(EdgeStore storage s, bytes32 edgeId) internal view returns (bool) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        bytes32 baseRecord = s.edges[edgeId].baseId();
        // CHRIS: TODO: this should be an assert? could do invariant testing for this?
        require(baseRecord != 0, "Empty base record");

        return baseRecord == IS_PRESUMPTIVE;
    }

    function isAtOneStepFork(EdgeStore storage s, bytes32 edgeId) internal view returns (bool) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        require(s.edges[edgeId].endHeight - s.edges[edgeId].startHeight == 1, "Edge is not length 1");

        require(!isPresumptive(s, edgeId), "Edge is presumptive, so cannot be at one step fork");

        return true;
    }

    function psTimer(EdgeStore storage s, bytes32 edgeId) internal view returns (uint256) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        bytes32 baseId = s.edges[edgeId].baseId();
        bytes32 baseRecord = s.baseRecords[baseId];
        // CHRIS: TODO: this should be an assert? could do invariant testing for this?
        require(baseRecord != 0, "Empty base record");

        if (baseRecord == IS_PRESUMPTIVE) {
            return block.timestamp - s.edges[edgeId].createdWhen;
        } else {
            // get the created when of the first rival
            require(s.edges[baseRecord].exists(), "Base record edge does not exist");

            // CHRIS: TODO: rename base record to first rival?
            uint256 baseRecordCreatedWhen = s.edges[baseRecord].createdWhen;
            uint256 edgeCreatedWhen = s.edges[edgeId].createdWhen;
            if (baseRecordCreatedWhen > edgeCreatedWhen) {
                return baseRecordCreatedWhen - edgeCreatedWhen;
            } else {
                return 0;
            }
        }
    }
}

struct CreateEdgeArgs {
    ChallengeType edgeChallengeType;
    bytes32 startHistoryRoot;
    uint256 startHeight;
    bytes32 endHistoryRoot;
    uint256 endHeight;
    bytes32 claimId;
}

interface IEdgeChallengeManager {
    // // Checks if a challenge by ID exists.
    // function challengeExists(bytes32 challengeId) external view returns (bool);
    // Fetches a challenge object by ID.
    function getChallenge(bytes32 challengeId) external view returns (EChallenge memory);
    // // Gets the winning claim ID for a challenge. TODO: Needs more thinking.
    // function winningClaim(bytes32 challengeId) external view returns (bytes32);
    // // Checks if an edge by ID exists.
    // function edgeExists(bytes32 eId) external view returns (bool);
    // Gets an edge by ID.
    function getEdge(bytes32 eId) external view returns (ChallengeEdge memory);
    // Gets the current ps timer by edge ID. TODO: Needs more thinking.
    // Flushed ps time vs total current ps time needs differentiation
    function getCurrentPsTimer(bytes32 eId) external view returns (uint256);
    // // We define a base ID as hash(challengeType  ++ hash(startCommit ++ startHeight)) as a way
    // // of checking if an edge has rivals. Edges can share the same base ID.
    // function calculateBaseIdForEdge(bytes32 edgeId) external returns (bytes32);
    // Checks if an edge's base ID corresponds to multiple rivals and checks if a one step fork exists.
    function isAtOneStepFork(bytes32 eId) external view returns (bool);
    // Creates a layer zero edge in a challenge.
    function createLayerZeroEdge(CreateEdgeArgs memory args,
        bytes calldata,
        bytes calldata )
        external
        payable
        returns (bytes32);
    // // Creates a subchallenge on an edge. Emits the challenge ID in an event.
    // function createSubChallenge(bytes32 eId) external returns (bytes32);
    // Bisects an edge. Emits both children's edge IDs in an event.
    function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes memory prefixProof) external returns (bytes32, bytes32);
    // Checks if both children of an edge are already confirmed in order to confirm the edge.
    function confirmEdgeByChildren(bytes32 eId) external;
    // Confirms an edge by edge ID and an array of ancestor edges based on timers.
    function confirmEdgeByTimer(bytes32 eId, bytes32[] memory ancestorIds) external;
    // If we have created a subchallenge, confirmed a layer 0 edge already, we can use a claim id to confirm edge ids.
    // All edges have two children, unless they only have a link to a claim id.
    function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) external;
}

contract EdgeChallengeManager is IEdgeChallengeManager { 
    using EdgeStoreLib for EdgeStore;
    using ChallengeEdgeLib for ChallengeEdge;
    using EChallengeLib for EChallenge;

    EdgeStore internal store;
    mapping(bytes32 => EChallenge) challenges;

    uint256 public challengePeriodSec;
    IAssertionChain internal assertionChain;

    constructor(IAssertionChain _assertionChain, uint256 _challengePeriodSec) {
        challengePeriodSec = _challengePeriodSec;
        assertionChain = _assertionChain;
    }

    function createLayerZeroEdge(
        CreateEdgeArgs memory args,
        bytes calldata, // CHRIS: TODO: not yet implemented
        bytes calldata // CHRIS: TODO: not yet implemented
    ) external payable returns (bytes32) {
        bytes32 challengeBaseId;
        if (args.edgeChallengeType == ChallengeType.Block) {
            // CHRIS: TODO: check that the assertion chain is in a fork

            // challenge id is the assertion which is the root of challenge
            challengeBaseId = assertionChain.getPredecessorId(args.claimId);
        } else if (args.edgeChallengeType == ChallengeType.BigStep) {
            // challenge id is the base id of claim edge
            // all the claims in this sub challenge will agree on this base id
            bytes32 claimChallengeId = store.get(args.claimId).challengeId;
            require(challenges[claimChallengeId].cType == ChallengeType.Block, "Claim challenge type is not Block");

            challengeBaseId = store.get(args.claimId).baseId();
        } else if (args.edgeChallengeType == ChallengeType.SmallStep) {
            bytes32 claimChallengeId = store.get(args.claimId).challengeId;
            require(challenges[claimChallengeId].cType == ChallengeType.BigStep, "Claim challenge type is not BigStep");

            challengeBaseId = store.get(args.claimId).baseId();
        } else {
            revert("Unexpected challenge type");
        }

        EChallenge memory challenge = EChallenge({baseId: challengeBaseId, cType: args.edgeChallengeType});
        bytes32 challengeId = challenge.id();
        // challenge id is the id of the vertex we're based off + the type of challenge

        // CHRIS: TODO: sub challenge specific checks, also start and end consistency checks, and claim consistency checks
        // CHRIS: TODO: check the ministake was provided
        // CHRIS: TODO: also prove that the the start root is a prefix of the end root
        // CHRIS: TODO: we had inclusion proofs before?

        ChallengeEdge memory ce = ChallengeEdge({
            challengeId: challengeId,
            startHistoryRoot: args.startHistoryRoot,
            startHeight: args.startHeight,
            endHistoryRoot: args.endHistoryRoot,
            endHeight: args.endHeight,
            createdWhen: block.timestamp,
            status: EdgeStatus.Pending,
            claimEdgeId: args.claimId,
            staker: msg.sender,
            lowerChildId: 0,
            upperChildId: 0
        });

        store.add(ce);

        if (challenges[challengeId].baseId == 0) challenges[challengeId] = challenge;

        return ce.id();
    }

    // 1. the claim id specifies the parent that will succeed if we win
    // 2. however we know that the end and start state must match the claim
    // 3. so we can identify the claim right?
    // 4. we can do this

    function mandatoryBisectionHeight(uint256 start, uint256 end) internal pure returns (uint256) {
        require(end - start >= 2, "Height different not two or more");
        if (end - start == 2) {
            return start + 1;
        }

        uint256 mostSignificantSharedBit = UintUtilsLib.mostSignificantBit((end - 1) ^ start);
        uint256 mask = type(uint256).max << mostSignificantSharedBit;
        return ((end - 1) & mask) - 1;
    }

    function bisectEdge(bytes32 edgeId, bytes32 middleHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32)
    {
        require(!store.isPresumptive(edgeId), "Cannot bisect presumptive edge");

        ChallengeEdge memory ce = store.get(edgeId);
        require(ce.lowerChildId == 0, "Edge already has children");

        // CHRIS: TODO: can we bisect if the challenge has a winner?

        uint256 middleHeight = mandatoryBisectionHeight(ce.startHeight, ce.endHeight);
        (bytes32[] memory preExpansion, bytes32[] memory proof) = abi.decode(prefixProof, (bytes32[], bytes32[]));
        MerkleTreeLib.verifyPrefixProof(
            middleHistoryRoot, middleHeight + 1, ce.endHistoryRoot, ce.endHeight + 1, preExpansion, proof
        );

        // CHRIS: TODO: use the same naming as in the paper for lower and upper
        ChallengeEdge memory lowerChild = ChallengeEdge({
            challengeId: ce.challengeId,
            startHistoryRoot: ce.startHistoryRoot,
            startHeight: ce.startHeight,
            endHistoryRoot: middleHistoryRoot,
            endHeight: middleHeight,
            createdWhen: block.timestamp,
            status: EdgeStatus.Pending,
            claimEdgeId: 0,
            staker: address(0),
            lowerChildId: 0,
            upperChildId: 0
        });

        ChallengeEdge memory upperChild = ChallengeEdge({
            challengeId: ce.challengeId,
            startHistoryRoot: middleHistoryRoot,
            startHeight: middleHeight,
            endHistoryRoot: ce.endHistoryRoot,
            endHeight: ce.endHeight,
            createdWhen: block.timestamp,
            status: EdgeStatus.Pending,
            claimEdgeId: 0,
            staker: address(0),
            lowerChildId: 0,
            upperChildId: 0
        });

        // it's possible that the store already has the lower child if it was created by a rival
        if (!store.has(lowerChild.id())) {
            store.add(lowerChild);
        }

        // it's never possible for the store to contract the upper child

        // CHRIS: TODO: INVARIANT
        require(!store.has(upperChild.id()), "Store contains upper child");

        store.add(upperChild);

        store.setChildren(edgeId, lowerChild.id(), upperChild.id());

        // CHRIS: TODO: buffer the id
        return (lowerChild.id(), upperChild.id());
    }

    function confirmEdgeByChildren(bytes32 edgeId) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");

        bytes32 lowerChildId = store.edges[edgeId].lowerChildId;
        require(store.edges[lowerChildId].exists(), "Lower child does not exist");

        bytes32 upperChildId = store.edges[edgeId].upperChildId;
        require(store.edges[upperChildId].exists(), "Upper child does not exist");

        require(store.edges[lowerChildId].status == EdgeStatus.Confirmed, "Lower child not confirmed");
        require(store.edges[upperChildId].status == EdgeStatus.Confirmed, "Upper child not confirmed");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");
        require(store.edges[claimingEdgeId].exists(), "Claiming edge does not exist");

        // CHRIS: TODO: this may not be necessary if we have the correct checks in add zero layer edge
        // CHRIS: TODO: infact it wont be an exact equality like this - we're probably going to wrap this up together
        require(store.edges[edgeId].baseId() == store.edges[claimingEdgeId].challengeId, "Invalid claim ids");

        require(edgeId == store.edges[claimingEdgeId].claimEdgeId, "Claim does not match edge");

        require(store.edges[claimingEdgeId].status == EdgeStatus.Confirmed, "Claiming edge not confirmed");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    function confirmEdgeByTimer(bytes32 edgeId, bytes32[] memory ancestorEdges) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");

        // loop through the ancestors chain summing ps timers as we go
        bytes32 currentEdge = edgeId;
        uint256 psTime = store.psTimer(edgeId);
        for (uint256 i = 0; i < ancestorEdges.length; i++) {
            ChallengeEdge storage e = store.get(ancestorEdges[i]);
            require(
                // direct child check
                e.lowerChildId == currentEdge || e.upperChildId == currentEdge
                // check accross sub challenge boundary
                || store.edges[edgeId].claimEdgeId == ancestorEdges[i],
                "Current is not a child of ancestor"
            );

            psTime += store.psTimer(e.id());
        }

        require(psTime > challengePeriodSec, "Ps timer not greater than challenge period");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    // CHRIS: TODO: remove these?
    ///////////////////////////////////////////////
    ///////////// VIEW FUNCS ///////////////

    function isPresumptive(bytes32 edgeId) public view returns (bool) {
        return store.isPresumptive(edgeId);
    }

    function getCurrentPsTimer(bytes32 edgeId) public view returns (uint256) {
        return store.psTimer(edgeId);
    }

    function isAtOneStepFork(bytes32 edgeId) public view returns (bool) {
        return store.isAtOneStepFork(edgeId);
    }

    function calculateEdgeId(
        bytes32 challengeId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.idComponent(challengeId, startHistoryRoot, startHeight, endHistoryRoot, endHeight);
    }

    function calculateChallengeId(bytes32 baseId, ChallengeType cType) public pure returns (bytes32) {
        EChallenge memory eChallenge = EChallenge({baseId: baseId, cType: cType});

        return EChallengeLib.id(eChallenge);
    }

    function getEdge(bytes32 edgeId) public view returns (ChallengeEdge memory) {
        return store.get(edgeId);
    }

    function getChallenge(bytes32 challengeId) public view returns (EChallenge memory) {
        return challenges[challengeId];
    }

    function baseRecord(bytes32 edgeId) public view returns (bytes32) {
        return store.baseRecords[edgeId];
    }
}
