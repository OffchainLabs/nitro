// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";

library ChallengeVertexLib {
    function newRoot(bytes32 challengeId, bytes32 historyRoot) internal pure returns (ChallengeVertex memory) {
        // CHRIS: TODO: the root should have a height 1 and should inherit the state commitment from above right?
        return ChallengeVertex({
            challengeId: challengeId,
            predecessorId: 0, // always zero for root
            successionChallenge: 0,
            historyRoot: historyRoot,
            height: 0,
            claimId: 0, // CHRIS: TODO: should this be a reference to the assertion on which this challenge is based? 2-way link?
            status: VertexStatus.Confirmed, // root starts off as confirmed
            staker: address(0), // always zero for non leaf
            psId: 0, // initially 0 - updated during connection
            psLastUpdated: 0, // initially 0 - updated during connection
            flushedPsTime: 0, // always zero for the root
            lowestHeightSucessorId: 0 // initially 0 - updated during connection
        });
    }

    function newLeaf(
        bytes32 challengeId,
        bytes32 historyRoot,
        uint256 height,
        bytes32 claimId,
        address staker,
        uint256 initialPsTime
    ) internal pure returns (ChallengeVertex memory) {
        require(challengeId != 0, "Zero challenge id");
        require(historyRoot != 0, "Zero history root");
        require(height != 0, "Zero height");
        require(claimId != 0, "Zero claim id");
        require(staker != address(0), "Zero staker address");

        return ChallengeVertex({
            challengeId: challengeId,
            predecessorId: 0, // vertices are always created with a zero predecessor then connected after
            successionChallenge: 0, // always zero for leaf
            historyRoot: historyRoot,
            height: height,
            claimId: claimId,
            status: VertexStatus.Pending,
            staker: staker,
            psId: 0, // always zero for leaf
            psLastUpdated: 0, // always zero for leaf
            flushedPsTime: initialPsTime,
            lowestHeightSucessorId: 0 // always zero for leaf
        });
    }

    function newVertex(bytes32 challengeId, bytes32 historyRoot, uint256 height, uint256 initialPsTime)
        internal
        pure
        returns (ChallengeVertex memory)
    {
        // CHRIS: TODO: check non-zero in all these things
        require(challengeId != 0, "Zero challenge id");
        require(historyRoot != 0, "Zero history root");
        require(height != 0, "Zero height");

        return ChallengeVertex({
            challengeId: challengeId,
            predecessorId: 0, // vertices are always created with a zero predecessor then connected after
            successionChallenge: 0, // vertex cannot be created with an existing challenge
            historyRoot: historyRoot,
            height: height,
            claimId: 0, // non leaves have no claim
            status: VertexStatus.Pending,
            staker: address(0), // non leaves have no staker
            psId: 0, // initially 0 - updated during connection
            psLastUpdated: 0, // initially 0 - updated during connection
            flushedPsTime: initialPsTime,
            lowestHeightSucessorId: 0 // initially 0 - updated during connection
        });
    }

    function id(bytes32 challengeId, bytes32 historyRoot, uint256 height) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeId, historyRoot, height));
    }

    function exists(ChallengeVertex storage vertex) internal view returns (bool) {
        return vertex.historyRoot != 0;
    }

    function isLeaf(ChallengeVertex storage vertex) internal view returns (bool) {
        return exists(vertex) && vertex.staker != address(0);
    }
}
