// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./libraries/UintUtilsLib.sol";
import "./DataEntities.sol";
import "./libraries/MerkleTreeLib.sol";
import "../osp/IOneStepProofEntry.sol";

enum EdgeStatus {
    Pending, // This vertex is vertex is pending, it has yet to be confirmed
    Confirmed // This vertex has been confirmed, once confirmed it cannot be unconfirmed
}

enum EdgeType {
    Block,
    BigStep,
    SmallStep
}

struct ChallengeEdge {
    bytes32 originId;
    bytes32 startHistoryRoot;
    uint256 startHeight;
    bytes32 endHistoryRoot;
    uint256 endHeight;
    bytes32 lowerChildId; // start -> middle
    bytes32 upperChildId; // middle -> end
    uint256 createdWhen;
    bytes32 claimEdgeId; // only on layer zero edge. Claim must have same start point and challenge id as this edge
    address staker; // only on layer zero edge
    EdgeStatus status;
    EdgeType eType;
}

library ChallengeEdgeLib {
    function mutualIdComponent(
        EdgeType eType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(eType, originId, startHeight, startHistoryRoot, endHeight));
    }

    function mutualId(ChallengeEdge storage ce) internal view returns (bytes32) {
        return mutualIdComponent(ce.eType, ce.originId, ce.startHeight, ce.startHistoryRoot, ce.endHeight);
    }

    function idComponent(
        EdgeType eType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) internal pure returns (bytes32) {
        return keccak256(
            abi.encodePacked(
                mutualIdComponent(eType, originId, startHeight, startHistoryRoot, endHeight), endHistoryRoot
            )
        );
    }

    function id(ChallengeEdge memory edge) internal pure returns (bytes32) {
        // CHRIS: TODO: consider if we need to include the claim id here? that shouldnt be necessary if we have the correct checks in createZeroLayerEdge
        return idComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight, edge.endHistoryRoot
        );
    }

    function exists(ChallengeEdge storage edge) internal view returns (bool) {
        return edge.createdWhen != 0;
    }
}

struct EdgeStore {
    mapping(bytes32 => ChallengeEdge) edges;
    // CHRIS: TODO: explain better what we're doing with the firt rivals
    mapping(bytes32 => bytes32) firstRivals;
}

library EdgeStoreLib {
    bytes32 constant NO_RIVAL = keccak256(abi.encodePacked("NO_RIVAL"));

    using ChallengeEdgeLib for ChallengeEdge;

    function add(EdgeStore storage s, ChallengeEdge memory ce) internal {
        // add an edge to the store, if another edge exists with this challenge id then this
        // edge must be rival
        // could we instead identify the challenge by it's own base?
        // if we did that then we have a deterministic challenge id
        // we can check somewhere else that the rules are correct

        // could include the parent baseid in each edge
        // that way we dont need a challenge id - this base == parent base, not for upper child tho, yes since their parents
        // are rivals
        // so if we include the parent base id, what do we get? that would work, but what would be the point?

        // edges are rivals, but they share, a share is a part of something too
        // common also means plain

        // CHRIS: TODO: check that the children are empty?

        bytes32 eId = ce.id();

        require(!s.edges[eId].exists(), "Edge already exists");
        s.edges[eId] = ce;

        bytes32 mutualId =
            ChallengeEdgeLib.mutualIdComponent(ce.eType, ce.originId, ce.startHeight, ce.startHistoryRoot, ce.endHeight);
        bytes32 firstRival = s.firstRivals[mutualId];

        if (firstRival == 0) {
            s.firstRivals[mutualId] = NO_RIVAL;
        } else if (firstRival == NO_RIVAL) {
            s.firstRivals[mutualId] = eId;
        } else {
            // CHRIS: TODO: comment as to why we do nothing
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

        bytes32 mutualId = s.edges[edgeId].mutualId();
        bytes32 firstRival = s.firstRivals[mutualId];
        // CHRIS: TODO: this should be an assert? could do invariant testing for this?
        require(firstRival != 0, "Empty first rival");

        return firstRival == NO_RIVAL;
    }

    function isAtOneStepFork(EdgeStore storage s, bytes32 edgeId) internal view returns (bool) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        require(s.edges[edgeId].endHeight - s.edges[edgeId].startHeight == 1, "Edge is not length 1");

        require(!isPresumptive(s, edgeId), "Edge is presumptive, so cannot be at one step fork");

        return true;
    }

    function psTimer(EdgeStore storage s, bytes32 edgeId) internal view returns (uint256) {
        require(s.edges[edgeId].exists(), "Edge does not exist");

        bytes32 mutualId = s.edges[edgeId].mutualId();
        bytes32 firstRival = s.firstRivals[mutualId];
        // CHRIS: TODO: this should be an assert? could do invariant testing for this?
        require(firstRival != 0, "Empty rival record");

        if (firstRival == NO_RIVAL) {
            return block.timestamp - s.edges[edgeId].createdWhen;
        } else {
            // get the created when of the first rival
            require(s.edges[firstRival].exists(), "Rival edge does not exist");

            uint256 firstRivalCreatedWhen = s.edges[firstRival].createdWhen;
            uint256 edgeCreatedWhen = s.edges[edgeId].createdWhen;
            if (firstRivalCreatedWhen > edgeCreatedWhen) {
                return firstRivalCreatedWhen - edgeCreatedWhen;
            } else {
                return 0;
            }
        }
    }
}

struct CreateEdgeArgs {
    EdgeType edgeType;
    bytes32 startHistoryRoot;
    uint256 startHeight;
    bytes32 endHistoryRoot;
    uint256 endHeight;
    bytes32 claimId;
}

interface IEdgeChallengeManager {
    function initialize(
        IAssertionChain _assertionChain,
        uint256 _challengePeriodSec,
        IOneStepProofEntry _oneStepProofEntry
    ) external;
    // // Gets the winning claim ID for a challenge. TODO: Needs more thinking.
    // function winningClaim(bytes32 challengeId) external view returns (bytes32);
    // // Checks if an edge by ID exists.
    // function edgeExists(bytes32 eId) external view returns (bool);
    // Gets an edge by ID.
    function getEdge(bytes32 eId) external view returns (ChallengeEdge memory);
    // Gets the current ps timer by edge ID. TODO: Needs more thinking.
    // Flushed ps time vs total current ps time needs differentiation
    function getCurrentPsTimer(bytes32 eId) external view returns (uint256);
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
    function isAtOneStepFork(bytes32 eId) external view returns (bool);
    // Creates a layer zero edge in a challenge.
    function createLayerZeroEdge(CreateEdgeArgs memory args, bytes calldata, bytes calldata)
        external
        payable
        returns (bytes32);
    // // Creates a subchallenge on an edge. Emits the challenge ID in an event.
    // function createSubChallenge(bytes32 eId) external returns (bytes32);
    // Bisects an edge. Emits both children's edge IDs in an event.
    function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32);
    // Checks if both children of an edge are already confirmed in order to confirm the edge.
    function confirmEdgeByChildren(bytes32 eId) external;
    // Confirms an edge by edge ID and an array of ancestor edges based on timers.
    function confirmEdgeByTimer(bytes32 eId, bytes32[] memory ancestorIds) external;
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

contract EdgeChallengeManager is IEdgeChallengeManager {
    using EdgeStoreLib for EdgeStore;
    using ChallengeEdgeLib for ChallengeEdge;

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

    function createLayerZeroEdge(
        CreateEdgeArgs memory args,
        bytes calldata, // CHRIS: TODO: not yet implemented
        bytes calldata // CHRIS: TODO: not yet implemented
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

        ChallengeEdge memory ce = ChallengeEdge({
            originId: originId,
            startHistoryRoot: args.startHistoryRoot,
            startHeight: args.startHeight,
            endHistoryRoot: args.endHistoryRoot,
            endHeight: args.endHeight,
            createdWhen: block.timestamp,
            claimEdgeId: args.claimId,
            staker: msg.sender,
            lowerChildId: 0,
            upperChildId: 0,
            status: EdgeStatus.Pending,
            eType: args.edgeType
        });

        store.add(ce);

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
            originId: ce.originId,
            startHistoryRoot: ce.startHistoryRoot,
            startHeight: ce.startHeight,
            endHistoryRoot: middleHistoryRoot,
            endHeight: middleHeight,
            createdWhen: block.timestamp,
            status: EdgeStatus.Pending,
            claimEdgeId: 0,
            staker: address(0),
            lowerChildId: 0,
            upperChildId: 0,
            eType: ce.eType
        });

        ChallengeEdge memory upperChild = ChallengeEdge({
            originId: ce.originId,
            startHistoryRoot: middleHistoryRoot,
            startHeight: middleHeight,
            endHistoryRoot: ce.endHistoryRoot,
            endHeight: ce.endHeight,
            createdWhen: block.timestamp,
            status: EdgeStatus.Pending,
            claimEdgeId: 0,
            staker: address(0),
            lowerChildId: 0,
            upperChildId: 0,
            eType: ce.eType
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
        require(store.edges[edgeId].status == EdgeStatus.Pending, "Edge not pending");

        bytes32 lowerChildId = store.edges[edgeId].lowerChildId;
        require(store.edges[lowerChildId].exists(), "Lower child does not exist");

        bytes32 upperChildId = store.edges[edgeId].upperChildId;
        require(store.edges[upperChildId].exists(), "Upper child does not exist");

        require(store.edges[lowerChildId].status == EdgeStatus.Confirmed, "Lower child not confirmed");
        require(store.edges[upperChildId].status == EdgeStatus.Confirmed, "Upper child not confirmed");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    function nextEdgeType(EdgeType eType) internal returns (EdgeType) {
        if (eType == EdgeType.Block) {
            return EdgeType.BigStep;
        } else if (eType == EdgeType.BigStep) {
            return EdgeType.SmallStep;
        } else if (eType == EdgeType.SmallStep) {
            revert("No next type after SmallStep");
        } else {
            revert("Unexpected edge type");
        }
    }

    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");
        require(store.edges[edgeId].status == EdgeStatus.Pending, "Edge not pending");
        require(store.edges[claimingEdgeId].exists(), "Claiming edge does not exist");

        // CHRIS: TODO: this may not be necessary if we have the correct checks in add zero layer edge
        // CHRIS: TODO: infact it wont be an exact equality like this - we're probably going to wrap this up together
        require(store.edges[edgeId].mutualId() == store.edges[claimingEdgeId].originId, "Origin id-mutual id mismatch");
        // CHRIS: TODO: this also may be unnecessary
        require(
            nextEdgeType(store.edges[edgeId].eType) == store.edges[claimingEdgeId].eType,
            "Edge type does not match claiming edge type"
        );

        require(edgeId == store.edges[claimingEdgeId].claimEdgeId, "Claim does not match edge");

        require(store.edges[claimingEdgeId].status == EdgeStatus.Confirmed, "Claiming edge not confirmed");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    function confirmEdgeByTimer(bytes32 edgeId, bytes32[] memory ancestorEdges) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");
        require(store.edges[edgeId].status == EdgeStatus.Pending, "Edge not pending");

        // loop through the ancestors chain summing ps timers as we go
        bytes32 currentEdge = edgeId;
        uint256 psTime = store.psTimer(edgeId);
        for (uint256 i = 0; i < ancestorEdges.length; i++) {
            ChallengeEdge storage e = store.get(ancestorEdges[i]);
            require(
                // direct child check
                e.lowerChildId == currentEdge || e.upperChildId == currentEdge
                // check accross sub challenge boundary
                || store.edges[currentEdge].claimEdgeId == ancestorEdges[i],
                "Current is not a child of ancestor"
            );

            psTime += store.psTimer(e.id());
            currentEdge = ancestorEdges[i];
        }

        require(psTime > challengePeriodSec, "Ps timer not greater than challenge period");

        // CHRIS: TODO: only use setters on the edge lib
        store.edges[edgeId].status = EdgeStatus.Confirmed;
    }

    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) public {
        require(store.edges[edgeId].exists(), "Edge does not exist");
        require(store.edges[edgeId].status == EdgeStatus.Pending, "Edge not pending");

        require(store.edges[edgeId].eType == EdgeType.SmallStep, "Edge is not a small step");
        require(store.isAtOneStepFork(edgeId), "Edge is not at one step fork");

        require(
            MerkleTreeLib.verifyInclusionProof(
                store.edges[edgeId].startHistoryRoot,
                oneStepData.beforeHash,
                oneStepData.machineStep,
                beforeHistoryInclusionProof
            ),
            "Before state not in history"
        );

        bytes32 afterHash = oneStepProofEntry.proveOneStep(
            oneStepData.execCtx, oneStepData.machineStep, oneStepData.beforeHash, oneStepData.proof
        );

        require(
            MerkleTreeLib.verifyInclusionProof(
                store.edges[edgeId].endHistoryRoot, afterHash, oneStepData.machineStep + 1, afterHistoryInclusionProof
            ),
            "After state not in history"
        );

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
}
