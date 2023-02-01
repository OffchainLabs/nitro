// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

enum Status {
    Pending,
    Confirmed,
    Rejected
}

interface IAssertionChain {
    function getPredecessorId(bytes32 assertionId) external view returns (bytes32);
    function getHeight(bytes32 assertionId) external view returns (uint256);
    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256);
    function getStateHash(bytes32 assertionId) external view returns (bytes32);
    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32);
    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256);
    function isFirstChild(bytes32 assertionId) external view returns (bool);
}

// CHRIS: TODO: remove this?
interface IWinningClaim {
    function winningClaim(bytes32 challengeId) external view returns (bytes32);
}

interface IChallengeManager is IWinningClaim {
    function createChallenge(bytes32 startId) external returns (bytes32);
    function vertexExists(bytes32 challengeId, bytes32 vId) external view returns (bool);
    function getVertex(bytes32 challengeId, bytes32 vId) external view returns (ChallengeVertex memory);
    function getCurrentPsTimer(bytes32 challengeId, bytes32 vId) external view returns (uint256);
    function confirmForPsTimer(bytes32 challengeId, bytes32 vId) external;
    function confirmForSucessionChallengeWin(bytes32 challengeId, bytes32 vId) external;
    function createSubChallenge(bytes32 challengeId, bytes32 child1Id, bytes32 child2Id) external;
    function bisect(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        external;
    function merge(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        external;
    function addLeaf(
        bytes32 challengeId,
        bytes32 claimId,
        uint256 height,
        bytes32 historyCommitment,
        bytes32 lastState,
        bytes memory lastStatehistoryProof,
        bytes memory proof1,
        bytes memory proof2
    ) external;
}

struct ChallengeVertex {
    bytes32 predecessorId;
    bytes32 successionChallenge;
    bytes32 historyCommitment;
    uint256 height; // CHRIS: TODO: are heights zero indexed or from 1?
    bytes32 claimId; // CHRIS: TODO: aka tag; only on a leaf
    address staker; // CHRIS: TODO: only on a leaf
    // CHRIS: TODO: use a different status for the vertices since they never transition to rejected?
    Status status;
    // the presumptive successor to this vertex
    bytes32 presumptiveSuccessorId;
    // CHRIS: TODO: we should have a staker in here to decide what do in the event of a win/loss?
    // the last time the presumptive successor to this vertex changed
    uint256 presumptiveSuccessorLastUpdated;
    // the amount of time this vertex has spent as the presumptive successor
    /// @notice DO NOT USE TO GET PS TIME! Instead use a getter function which takes into account unflushed ps time as well.
    ///         This is the amount of time that this vertex is recorded to have been the presumptive successor
    ///         However this may not be the total amount of time being the presumptive successor, as this vertex may currently
    ///         be the ps, and so may have some time currently being record on the predecessor.
    uint256 flushedPsTime;
    // the id of the successor with the lowest height. Zero if this vertex has no successors.
    bytes32 lowestHeightSucessorId;
}

// CHRIS: TODO: one step proof test just here for structure test
contract OneStepProofManager is IWinningClaim {
    mapping(bytes32 => bytes32) public winningClaims;

    function winningClaim(bytes32 challengeId) public view returns (bytes32) {
        return winningClaims[challengeId];
    }

    function createOneStepProof(bytes32 startState) public returns (bytes32) {
        revert("NOT_IMPLEMENTED");
    }

    function setWinningClaim(bytes32 startState, bytes32 _winner) public {
        winningClaims[startState] = _winner;
    }
}

contract ChallengeManagers {
    IChallengeManager public blockChallengeManager;
    IChallengeManager public bigStepChallengeManager;
    IChallengeManager public smallStepChallengeManager;
    IAssertionChain public assertionChain;
    OneStepProofManager public oneStepProofManager;
}
