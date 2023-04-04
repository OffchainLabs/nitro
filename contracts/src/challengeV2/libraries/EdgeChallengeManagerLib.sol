// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "./UintUtilsLib.sol";
import "./MerkleTreeLib.sol";
import "./ChallengeEdgeLib.sol";

/// @notice Stores all edges and their rival status
struct EdgeStore {
    /// @dev A mapping of edge id to edges. Edges are never deleted, only created, and potentially confirmed.
    mapping(bytes32 => ChallengeEdge) edges;
    /// @dev A mapping of mutualId to edge id. Rivals share the same mutual id, and here we
    ///      store the edge id of the second edge that was created with the same mutual id - the first rival
    ///      When only one edge exists for a specific mutual id then a special magic string hash is stored instead
    ///      of the first rival id, to signify that a single edge does exist with this mutual id
    mapping(bytes32 => bytes32) firstRivals;
}

/// @title  Core functionality for the Edge Challenge Manager
/// @notice The edge manager library allows edges to be added and bisected, and keeps track of the amount
///         of time an edge remained unrivaled.
library EdgeChallengeManagerLib {
    /// @dev Magic string hash to represent that an edge has no rivals
    bytes32 constant NO_RIVAL = keccak256(abi.encodePacked("NO_RIVAL"));

    using ChallengeEdgeLib for ChallengeEdge;

    /// @notice Get an edge from the store
    /// @dev    Throws if the edge does not exist in the store
    /// @param store    The edge store to fetch an id from
    /// @param edgeId   The id of the edge to fetch
    function get(EdgeStore storage store, bytes32 edgeId) internal view returns (ChallengeEdge storage) {
        require(store.edges[edgeId].exists(), "Edge does not exist");
        return store.edges[edgeId];
    }

    /// @notice Adds a new edge to the store
    /// @dev    Updates first rival info for later use in calculating time unrivaled
    /// @param store    The store to add the edge to
    /// @param edge     The edge to add
    function add(EdgeStore storage store, ChallengeEdge memory edge) internal {
        // add the edge if it doesnt exist already
        bytes32 eId = edge.id();
        require(!store.edges[eId].exists(), "Edge already exists");
        store.edges[eId] = edge;

        // edges that are rivals share the same mutual id
        // we use records of whether a mutual id has ever been added to decide if
        // the new edge is a rival. This will later allow us to calculate time an edge
        // stayed unrivaled
        bytes32 mutualId = ChallengeEdgeLib.mutualIdComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight
        );
        bytes32 firstRival = store.firstRivals[mutualId];

        // the first time we add a mutual id we store a magic string hash against it
        // We do this to distinguish from there being no edges
        // with this mutual. And to distinguish it from the first rival, where we
        // will use an actual edge id so that we can look up the created when time
        // of the first rival, and use it for calculating time unrivaled
        if (firstRival == 0) {
            store.firstRivals[mutualId] = NO_RIVAL;
        } else if (firstRival == NO_RIVAL) {
            store.firstRivals[mutualId] = eId;
        } else {
            // after we've stored the first rival we dont need to keep a record of any
            // other rival edges - they will all have a zero time unrivaled
        }
    }

    /// @notice Does this edge currently have one or more rivals
    ///         Rival edges share the same startHeight, startHistoryCommitment and the same endHeight,
    ///         but they have a different endHistoryRoot. Rival edges have the same mutualId
    /// @param store    The edge store containing the edge
    /// @param edgeId   The edge if to test if it is unrivaled
    function hasRival(EdgeStore storage store, bytes32 edgeId) internal view returns (bool) {
        require(store.edges[edgeId].exists(), "Edge does not exist");

        // rivals have the same mutual id
        bytes32 mutualId = store.edges[edgeId].mutualId();
        bytes32 firstRival = store.firstRivals[mutualId];
        // Sanity check: it should never be possible to create an edge without having an entry in firstRivals
        require(firstRival != 0, "Empty first rival");

        // can only have no rival is the firstRival is the NO_RIVAL magic hash
        return firstRival != NO_RIVAL;
    }

    /// @notice Is the edge a single step in length, and does it have at least one rival.
    /// @param store    The edge store containing the edge
    /// @param edgeId   The edge id to test for single step and rivaled
    function hasLengthOneRival(EdgeStore storage store, bytes32 edgeId) internal view returns (bool) {
        require(store.edges[edgeId].exists(), "Edge does not exist");

        // must be length 1 and have rivals
        return (store.edges[edgeId].length() == 1 && hasRival(store, edgeId));
    }

    /// @notice The amount of time this edge has spent without rivals
    ///         This value is increasing whilst an edge is unrivaled, once a rival is created
    ///         it is fixed. If an edge has rivals from the moment it is created then it will have
    ///         a zero time unrivaled
    function timeUnrivaled(EdgeStore storage store, bytes32 edgeId) internal view returns (uint256) {
        require(store.edges[edgeId].exists(), "Edge does not exist");

        bytes32 mutualId = store.edges[edgeId].mutualId();
        bytes32 firstRival = store.firstRivals[mutualId];
        // Sanity check: it's not possible to have a 0 first rival for an edge that exists
        require(firstRival != 0, "Empty rival record");

        // this edge has no rivals, the time is still going up
        // we give the current amount of time unrivaled
        if (firstRival == NO_RIVAL) {
            return block.timestamp - store.edges[edgeId].createdWhen;
        } else {
            // Sanity check: it's not possible an edge does not exist for a first rival record
            require(store.edges[firstRival].exists(), "Rival edge does not exist");

            // rivals exist for this edge
            uint256 firstRivalCreatedWhen = store.edges[firstRival].createdWhen;
            uint256 edgeCreatedWhen = store.edges[edgeId].createdWhen;
            if (firstRivalCreatedWhen > edgeCreatedWhen) {
                // if this edge was created before the first rival then we return the difference
                // in createdWhen times
                return firstRivalCreatedWhen - edgeCreatedWhen;
            } else {
                // if this was created at the same time as, or after the the first rival
                // then we return 0
                return 0;
            }
        }
    }

    /// @notice Given a start and an endpoint determine the bisection height
    /// @dev    Returns the highest power of 2 in the differing lower bits of start and end
    function mandatoryBisectionHeight(uint256 start, uint256 end) internal pure returns (uint256) {
        require(end - start >= 2, "Height difference not two or more");
        if (end - start == 2) {
            return start + 1;
        }

        uint256 diff = (end - 1) ^ start;
        uint256 mostSignificantSharedBit = UintUtilsLib.mostSignificantBit(diff);
        uint256 mask = type(uint256).max << mostSignificantSharedBit;
        return ((end - 1) & mask);
    }

    /// @notice Bisect and edge. This creates two child edges:
    ///         lowerChild: has the same start root and height as this edge, but a different end root and height
    ///         upperChild: has the same end root and height as this edge, but a different start root and height
    ///         The lower child end root and height are equal to the upper child start root and height. This height
    ///         is the mandatoryBisectionHeight
    /// @param store                The edge store containing the edge to bisect
    /// @param edgeId               Edge to bisect
    /// @param bisectionHistoryRoot The new history root to be used in the lower and upper children
    /// @param prefixProof          A proof to show that the bisectionHistoryRoot commits to a prefix of the current endHistoryRoot
    /// @return lowerChildId        The id of the newly created lower child edge
    /// @return upperChildId        The id of the newly created upper child edge
    function bisectEdge(EdgeStore storage store, bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        internal
        returns (bytes32, bytes32)
    {
        require(hasRival(store, edgeId), "Cannot bisect an unrivaled edge");

        // cannot bisect an edge twice
        ChallengeEdge memory ce = get(store, edgeId);
        require(
            store.edges[edgeId].lowerChildId == 0 && store.edges[edgeId].upperChildId == 0, "Edge already has children"
        );

        // bisections occur at deterministic heights, this ensures that
        // rival edges bisect at the same height, and create the same child if they agree
        uint256 middleHeight = mandatoryBisectionHeight(ce.startHeight, ce.endHeight);
        {
            (bytes32[] memory preExpansion, bytes32[] memory proof) = abi.decode(prefixProof, (bytes32[], bytes32[]));
            MerkleTreeLib.verifyPrefixProof(
                bisectionHistoryRoot, middleHeight + 1, ce.endHistoryRoot, ce.endHeight + 1, preExpansion, proof
            );
        }

        // midpoint proof it valid, create and store the children
        ChallengeEdge memory lowerChild = ChallengeEdgeLib.newChildEdge(
            ce.originId, ce.startHistoryRoot, ce.startHeight, bisectionHistoryRoot, middleHeight, ce.eType
        );

        ChallengeEdge memory upperChild = ChallengeEdgeLib.newChildEdge(
            ce.originId, bisectionHistoryRoot, middleHeight, ce.endHistoryRoot, ce.endHeight, ce.eType
        );

        bytes32 lowerChildId = lowerChild.id();
        bytes32 upperChildId = upperChild.id();

        // it's possible that the store already has the lower child if it was created by a rival
        // (aka a merge move)
        if (!store.edges[lowerChildId].exists()) {
            add(store, lowerChild);
        }

        // Sanity check: it's not possible that the upper child already exists, for this to be the case
        // the edge would have to have been bisected already.
        require(!store.edges[upperChildId].exists(), "Store contains upper child");
        add(store, upperChild);

        store.edges[edgeId].setChildren(lowerChildId, upperChildId);
        return (lowerChildId, upperChildId);
    }
}
