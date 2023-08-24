// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "./Enums.sol";
import "./ChallengeErrors.sol";

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
    ///         For a Block edge the origin id is the assertion hash of the assertion that is the root of the challenge - all edges in this challenge agree
    ///         that that assertion hash is valid.
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
    /// @notice Set to true when the staker has been refunded. Can only be set to true if the status is Confirmed
    ///         and the staker is non zero.
    bool refunded;
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
        if (originId == 0) {
            revert EmptyOriginId();
        }
        if (endHeight <= startHeight) {
            revert InvalidHeights(startHeight, endHeight);
        }
        if (startHistoryRoot == 0) {
            revert EmptyStartRoot();
        }
        if (endHistoryRoot == 0) {
            revert EmptyEndRoot();
        }
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
        if (staker == address(0)) {
            revert EmptyStaker();
        }
        if (claimId == 0) {
            revert EmptyClaimId();
        }

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
            eType: eType,
            refunded: false
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
            eType: eType,
            refunded: false
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
        if (len == 0) {
            revert EdgeNotExists(ChallengeEdgeLib.id(edge));
        }
        return len;
    }

    /// @notice Set the children of an edge
    /// @dev    Children can only be set once
    function setChildren(ChallengeEdge storage edge, bytes32 lowerChildId, bytes32 upperChildId) internal {
        if (edge.lowerChildId != 0 || edge.upperChildId != 0) {
            revert ChildrenAlreadySet(ChallengeEdgeLib.id(edge), edge.lowerChildId, edge.upperChildId);
        }
        edge.lowerChildId = lowerChildId;
        edge.upperChildId = upperChildId;
    }

    /// @notice Set the status of an edge to Confirmed
    /// @dev    Only Pending edges can be confirmed
    function setConfirmed(ChallengeEdge storage edge) internal {
        if (edge.status != EdgeStatus.Pending) {
            revert EdgeNotPending(ChallengeEdgeLib.id(edge), edge.status);
        }
        edge.status = EdgeStatus.Confirmed;
    }

    /// @notice Is the edge a layer zero edge.
    function isLayerZero(ChallengeEdge storage edge) internal view returns (bool) {
        return edge.claimId != 0 && edge.staker != address(0);
    }

    /// @notice Set the refunded flag of an edge
    /// @dev    Checks internally that edge is confirmed, layer zero edge and hasnt been refunded already
    function setRefunded(ChallengeEdge storage edge) internal {
        if (edge.status != EdgeStatus.Confirmed) {
            revert EdgeNotConfirmed(ChallengeEdgeLib.id(edge), edge.status);
        }
        if (!isLayerZero(edge)) {
            revert EdgeNotLayerZero(ChallengeEdgeLib.id(edge), edge.staker, edge.claimId);
        }
        if (edge.refunded == true) {
            revert EdgeAlreadyRefunded(ChallengeEdgeLib.id(edge));
        }

        edge.refunded = true;
    }
}
