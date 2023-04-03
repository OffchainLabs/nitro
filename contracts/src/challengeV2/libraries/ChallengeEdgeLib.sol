// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

// CHRIS: TODO: move this and type elsewhere
enum EdgeStatus
/// @dev This vertex is vertex is pending, it has yet to be confirmed. Not all edges can be confirmed.
{
    Pending,
    /// @dev This vertex has been confirmed, once confirmed it cannot transition back to pending
    Confirmed
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
    bytes32 claimId; // only on layer zero edge. Claim must have same start point and challenge id as this edge
    address staker; // only on layer zero edge
    EdgeStatus status;
    EdgeType eType;
}

library ChallengeEdgeLib {
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
            createdWhen: block.timestamp,
            claimId: claimId,
            staker: staker,
            status: EdgeStatus.Pending,
            eType: eType
        });
    }

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
            createdWhen: block.timestamp,
            claimId: 0,
            staker: address(0),
            status: EdgeStatus.Pending,
            eType: eType
        });
    }

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

    // CHRIS: TODO: this is forcing the whole struct to get loaded just to compute the id
    function id(ChallengeEdge memory edge) internal pure returns (bytes32) {
        // CHRIS: TODO: consider if we need to include the claim id here? that shouldnt be necessary if we have the correct checks in createZeroLayerEdge
        return idComponent(
            edge.eType, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight, edge.endHistoryRoot
        );
    }

    function exists(ChallengeEdge storage edge) internal view returns (bool) {
        return edge.createdWhen != 0;
    }

    function length(ChallengeEdge storage edge) internal view returns (uint256) {
        return edge.endHeight - edge.startHeight;
    }

    function setChildren(ChallengeEdge storage edge, bytes32 lowerChildId, bytes32 upperChildId) internal {
        // CHRIS: TODO: we dont need this if it's storage rigth?
        require(exists(edge), "Edge does not exist");

        edge.lowerChildId = lowerChildId;
        edge.upperChildId = upperChildId;
    }

    function setConfirmed(ChallengeEdge storage edge) internal {
        // CHRIS: TODO: we dont need this if it's storage rigth?
        require(exists(edge), "Edge does not exist");

        edge.status = EdgeStatus.Confirmed;
    }
}
