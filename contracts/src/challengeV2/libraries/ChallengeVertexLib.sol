// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";

library ChallengeVertexLib {
    function newRoot(bytes32 challengeId, bytes32 historyCommitment) internal pure returns (ChallengeVertex memory) {
        // CHRIS: TODO: the root should have a height 1 and should inherit the state commitment from above right?
        return ChallengeVertex({
            predecessorId: 0,
            successionChallenge: 0,
            historyCommitment: historyCommitment, // CHRIS: TODO: this isnt correct - we should compute this from the claim apparently
            height: 0, // CHRIS: TODO: this should be 1 from the spec/paper - DIFF to paper - also in the id
            claimId: 0, // CHRIS: TODO: should this be a reference to the assertion on which this challenge is based? 2-way link?
            status: Status.Confirmed,
            staker: address(0),
            presumptiveSuccessorId: 0,
            psLastUpdated: 0, // CHRIS: TODO: maybe we wanna update this? We should set it as the start time? or are we gonna do special stuff for root?
            flushedPsTime: 0, // always zero for the root
            lowestHeightSucessorId: 0,
            challengeId: challengeId
        });
    }

    function id(bytes32 challengeId, bytes32 historyCommitment, uint256 height) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeId, historyCommitment, height));
    }

    // CHRIS: TODO: duplication for storage/mem - we also dont need `has` AND vertexExists
    function exists(ChallengeVertex storage vertex) internal view returns (bool) {
        return vertex.historyCommitment != 0;
    }
    
    function isLeaf(ChallengeVertex storage vertex) internal view returns (bool) {
        return exists(vertex) && vertex.staker != address(0);
    }
}
