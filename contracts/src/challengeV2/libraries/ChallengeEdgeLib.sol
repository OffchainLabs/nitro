// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

/// @notice The status of the edge
/// - Pending: Yet to be confirmed. Not all edges can be confirmed.
/// - Confirmed: Once confirmed it cannot transition back to pending
enum EdgeStatus {
    Pending,
    Confirmed
}

/// @notice The type of the edge. Challenges are decomposed into 3 types of subchallenge
///         represented here by the edge type. Edges are initially created of type Block
///         and are then bisected until they have length one. After that new BigStep edges are
///         added that claim a Block type edge, and are then bisected until they have length one.
///         Then a SmallStep edge is added that claims a length one BigStep edge, and these
///         SmallStep edges are bisected until they reach length one. A length one small step edge
///         can then be directly executed using a one-step proof.
enum EdgeType {
    Block,
    BigStep,
    SmallStep
}

/// @notice An edge committing to a range of states. These edges will be bisected, slowly
///         reducing them in length until they reach length one. At that point new edges of a different
///         type will be added that claim the result of this edge, or a one step proof will be calculated
///         if the edge type is already SmallStep.
struct ChallengeEdge {
    /// @notice The origin id is a link from the edge to an edge or assertion at a higher type. The types
    ///         of edge are Block, BigStep and SmallStep.
    ///         Intuitively all edges with the same origin id agree on the information committed to in the origin id
    ///         For a SmallStep edge the origin id is the 'mutual' id of the length one BigStep edge being claimed by the zero layer ancestors of this edge
    ///         For a BigStep edge the origin id is the 'mutual' id of the length one Block edge being claimed by the zero layer ancestors of this edge
    ///         For a Block edge the origin id is the assertion id of the assertion that is the root of the challenge - all edges in this challenge agree
    ///         that that assertion id is valid.
    ///         The purpose of the origin id is to ensure that only edges that agree on a common start position
    ///         are being compared against one another.
    bytes32 originId;
    /// @notice A root of all the states in the history up to the startHeight
    bytes32 startHistoryRoot;
    /// @notice The number of states (+1 for 0 index) that the startHistoryRoot commits to
    uint256 startHeight;
    /// @notice A root of all the states in the history up to the endHeight. Since endHeight > startHeight, the startHistoryRoot must
    ///         commit to a prefix of the states committed to by the endHistoryRoot
    bytes32 endHistoryRoot;
    /// @notice The number of states (+1 for 0 index) that the endHistoryRoot commits to
    uint256 endHeight;
    /// @notice Edges can be bisected into two children. If this edge has been bisected the id of the
    ///         lower child is populated here, until that time this value is 0. The lower child has startHistoryRoot and startHeight
    ///         equal to this edge, but endHistoryRoot and endHeight equal to some prefix of the endHistoryRoot of this edge
    bytes32 lowerChildId;
    /// @notice Edges can be bisected into two children. If this edge has been bisected the id of the
    ///         upper child is populated here, until that time this value is 0. The upper child has startHistoryRoot and startHeight
    ///         equal to some prefix of the endHistoryRoot of this edge, and endHistoryRoot and endHeight equal to this edge
    bytes32 upperChildId;
    /// @notice The block number when this edge was created
    uint256 createdAtBlock;
    /// @notice The edge or assertion in the upper level that this edge claims to be true.
    ///         Only populated on zero layer edges
    bytes32 claimId;
    /// @notice The entity that supplied a mini-stake accompanying this edge
    ///         Only populated on zero layer edges
    address staker;
    /// @notice Current status of this edge. All edges are created Pending, and may be updated to Confirmed
    ///         Once Confirmed they cannot transition back to Pending
    EdgeStatus status;
    /// @notice The type of edge Block, BigStep or SmallStep that this edge is.
    EdgeType eType;
}

library ChallengeEdgeLib {
    /// @notice Common checks to do when adding an edge
    function newEdgeChecks(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight
    ) internal pure {
        require(originId != 0, "Empty origin id");
        require(endHeight - startHeight > 0, "Invalid heights");
        require(startHistoryRoot != 0, "Empty start history root");
        require(endHistoryRoot != 0, "Empty end history root");
    }

    /// @notice Create a new layer zero edge. These edges make claims about length one edges in the level
    ///         (edge type) above. Creating a layer zero edge also requires placing a mini stake, so information
    ///         about that staker is also stored on this edge.
    function newLayerZeroEdge(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight,
        bytes32 claimId,
        address staker,
        EdgeType eType
    ) internal view returns (ChallengeEdge memory) {
        require(staker != address(0), "Empty staker");
        require(claimId != 0, "Empty claim id");

        newEdgeChecks(originId, startHistoryRoot, startHeight, endHistoryRoot, endHeight);

        return ChallengeEdge({
            originId: originId,
            startHeight: startHeight,
            startHistoryRoot: startHistoryRoot,
            endHeight: endHeight,
            endHistoryRoot: endHistoryRoot,
            lowerChildId: 0,
            upperChildId: 0,
            createdAtBlock: block.number,
            claimId: claimId,
            staker: staker,
            status: EdgeStatus.Pending,
            eType: eType
        });
    }

    /// @notice Creates a new child edge. All edges except layer zero edges are child edges.
    ///         These are edges that are created by bisection, and have parents rather than claims.
    function newChildEdge(
        bytes32 originId,
        bytes32 startHistoryRoot,
        uint256 startHeight,
        bytes32 endHistoryRoot,
        uint256 endHeight,
        EdgeType eType
    ) internal view returns (ChallengeEdge memory) {
        newEdgeChecks(originId, startHistoryRoot, startHeight, endHistoryRoot, endHeight);

        return ChallengeEdge({
            originId: originId,
            startHeight: startHeight,
            startHistoryRoot: startHistoryRoot,
            endHeight: endHeight,
            endHistoryRoot: endHistoryRoot,
            lowerChildId: 0,
            upperChildId: 0,
            createdAtBlock: block.number,
            claimId: 0,
            staker: address(0),
            status: EdgeStatus.Pending,
            eType: eType
        });
    }

    /// @notice The "mutualId" of an edge. A mutual id is a hash of all the data that is shared by rivals.
    ///         Rivals have the same start height, start history root and end height. They also have the same origin id and type.
    ///         The difference between rivals is that they have a different endHistoryRoot, so that information
    ///         is not included in this hash.
    function mutualIdComponent(
        EdgeType eType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(eType, originId, startHeight, startHistoryRoot, endHeight));
    }

    /// @notice The "mutualId" of an edge. A mutual id is a hash of all the data that is shared by rivals.
    ///         Rivals have the same start height, start history root and end height. They also have the same origin id and type.
    ///         The difference between rivals is that they have a different endHistoryRoot, so that information
    ///         is not included in this hash.
    function mutualId(ChallengeEdge storage ce) internal view returns (bytes32) {
        return mutualIdComponent(ce.eType, ce.originId, ce.startHeight, ce.startHistoryRoot, ce.endHeight);
    }

    /// @notice The id of an edge. Edges are uniquely identified by their id, and commit to the same information
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

    /// @notice The id of an edge. Edges are uniquely identified by their id, and commit to the same information
    /// @dev    This separate idMem method is to be explicit about when ChallengeEdges are copied into memory. It is
    ///         possible to pass a storage edge to this method and the id be computed correctly, but that would load
    ///         the whole struct into memory, so we're explicit here that this should be used for edges already in memory.
    function idMem(ChallengeEdge memory edge) internal pure returns (bytes32) {
        return idComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight, edge.endHistoryRoot
        );
    }

    /// @notice The id of an edge. Edges are uniquely identified by their id, and commit to the same information
    function id(ChallengeEdge storage edge) internal view returns (bytes32) {
        return idComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight, edge.endHistoryRoot
        );
    }

    /// @notice Does this edge exist in storage
    function exists(ChallengeEdge storage edge) internal view returns (bool) {
        // All edges have a createdAtBlock number
        return edge.createdAtBlock != 0;
    }

    /// @notice The length of this edge - difference between the start and end heights
    function length(ChallengeEdge storage edge) internal view returns (uint256) {
        uint256 len = edge.endHeight - edge.startHeight;
        // It's impossible for a zero length edge to exist
        require(len > 0, "Edge does not exist");
        return len;
    }

    /// @notice Set the children of an edge
    /// @dev    Children can only be set once
    function setChildren(ChallengeEdge storage edge, bytes32 lowerChildId, bytes32 upperChildId) internal {
        require(edge.lowerChildId == 0 && edge.upperChildId == 0, "Children already set");
        edge.lowerChildId = lowerChildId;
        edge.upperChildId = upperChildId;
    }

    /// @notice Set the status of an edge to Confirmed
    /// @dev    Only Pending edges can be confirmed
    function setConfirmed(ChallengeEdge storage edge) internal {
        require(edge.status == EdgeStatus.Pending, "Only Pending edges can be Confirmed");
        edge.status = EdgeStatus.Confirmed;
    }
}
