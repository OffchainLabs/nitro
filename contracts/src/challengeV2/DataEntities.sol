// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../osp/IOneStepProofEntry.sol";

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

// CHRIS: TODO: move this to data entities?
struct OneStepData {
    ExecutionContext execCtx;
    uint256 machineStep;
    bytes32 beforeHash;
    bytes proof;
}

struct AddLeafArgs {
    bytes32 challengeId;
    bytes32 claimId;
    uint256 height;
    bytes32 historyRoot;
    bytes32 firstState;
    bytes firstStatehistoryProof;
    bytes32 lastState;
    bytes lastStatehistoryProof;
}

struct AddLeafLibArgs {
    uint256 miniStake;
    uint256 challengePeriod;
    AddLeafArgs leafData;
    bytes proof1;
    bytes proof2;
}

interface IChallengeManagerExternalView {
    function challengeExists(bytes32 challengeId) external view returns (bool);

    function getChallenge(bytes32 challengeId) external view returns (Challenge memory);

    function winningClaim(bytes32 challengeId) external view returns (bytes32);

    function vertexExists(bytes32 vId) external view returns (bool);

    function getVertex(bytes32 vId) external view returns (ChallengeVertex memory);

    function getCurrentPsTimer(bytes32 vId) external view returns (uint256);

    function hasConfirmedSibling(bytes32 vId) external view returns (bool);

    function isAtOneStepFork(bytes32 vId) external view returns (bool);
}

interface IChallengeManagerCore {
    function confirmForPsTimer(bytes32 vId) external;

    function confirmForSucessionChallengeWin(bytes32 vId) external;

    function createChallenge(bytes32 assertionId) external returns (bytes32);

    function createSubChallenge(bytes32 vId) external returns (bytes32);

    function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32);

    function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes memory prefixProof) external returns (bytes32);

    function addLeaf(AddLeafArgs calldata leafData, bytes calldata proof1, bytes calldata proof2)
        external
        payable
        returns (bytes32);
}

interface IChallengeManager is IChallengeManagerCore, IChallengeManagerExternalView {}

enum VertexStatus {
    /// @notice This vertex is vertex is pending, it has yet to be confirmed
    Pending,
    /// @notive This vertex has been confirmed, once confirmed it cannot be unconfirmed
    Confirmed
}

/// @notice A challenge vertex represents history root at specific height in a specific challenge. Vertices
///         form a tree linked by predecessor id.
struct ChallengeVertex {
    /// @notice The challenge, or sub challenge, that this vertex is part of
    bytes32 challengeId;
    
    /// @notice The history root is the merkle root of all the states from the root vertex up to the height of this vertex
    ///         It is a commitment to the full history of state since the root
    bytes32 historyRoot;
    
    /// @notice The height of this vertex - the number of "steps" since the root vertex. Steps are defined
    ///         different for different challenge types. A step in a BlockChallenge is a whole block, a step in
    ///         BigStepChallenge is a 2^20 WASM operations (or less if the vertex is a leaf), a step in a SmallStepChallenge
    ///         is a single WASM operation
    uint256 height;
    
    /// @notice Is there a challenge open to decide the successor to this vertex. The winner of that challenge will be a leaf
    ///         vertex whose claim decides which vertex succeeds this one.
    /// @dev    Leaf vertices cannot have a succession challenge as they have no successors.
    bytes32 successionChallenge;
    
    /// @notice The predecessor vertex of this challenge. Predecessors always contain a history root which is a root of a sub-history
    ///         of the history root of this vertex. That is in order to connect two vertices, it must be proven
    ///         that the predecessor commits to a sub-history of the correct height.
    /// @dev    All vertices except the root have a predecessor
    bytes32 predecessorId;
    
    /// @notice When a leaf is created it makes contains a reference to a vertex in a higher level, or a top level assertion,
    ///         that can be confirmed if this leaf is confirmed - the claim id is that reference.
    /// @dev    Only leaf vertices have claim ids. CHRIS: TODO: also put this on the root for consistency?
    bytes32 claimId;
    
    /// @notice In order to create a leaf a mini-stake must be placed. The placer of this stake is record so that they can be refunded
    ///         in the event that they win the challenge.
    /// @dev    Only leaf vertices have a populated staker
    address staker;
    
    /// @notice The current status of this vertex. There is no Rejected status as vertices are implicitly rejected if they can no longer be confirmed
    /// @dev    The root vertex is created in the Confirmed status, all other vertices are created as Pending, and may later transition to Confirmed
    VertexStatus status;
    
    /// @notice The id of the current presumptive successor (ps) vertex to this vertex. A successor vertex is one who has a predecessorId property
    ///         equal to id of this vertex. The presumptive successor is the one with the unique lowest height distance from this vertex.
    ///         If multiple vertices have the lowest height distance from this vertex then neither is the presumptive successor. 
    ///         Successors can become presumptive by reducing their height using bisect and merge moves.
    ///         Whilst a successor is presumptive it's ps timer is ticking up, if the ps timer becomes greater than the challenge period
    ///         then this vertex can be confirmed
    /// @dev    Always zero on leaf vertices as have no successors
    bytes32 psId;
    
    /// @notice The last time the psId was updated, or the flushedPsTime of the ps was updated.
    ///         Used to record the amount of time the current ps has spent as ps, when the ps is changed
    ///         this time is then flushed onto the ps before updating the ps id.
    /// @dev    Always zero on leaf vertices as have no successors
    uint256 psLastUpdated;
    
    /// @notice The flushed amount of time this vertex has spent as ps. This may not be the total amount
    ///         of time if this vertex is current the ps on its predecessor. For this reason do not use this
    ///         property to get the amount of time this vertex has been ps, instead use the PsVertexLib.getPsTimer function
    /// @dev    Always zero on the root vertex as it is not the successor to anything.
    uint256 flushedPsTime;
    
    /// @notice The id of the of successor with the lowest height. Zero if this vertex has no successors
    /// @dev    This is used to decide whether the ps is at the unique lowest height.
    ///         Always zero for leaf vertices as they have no successors.
    bytes32 lowestHeightSucessorId;
}

enum ChallengeType {
    Block,
    BigStep,
    SmallStep,
    OneStep
}

struct Challenge {
    bytes32 rootId;
    // CHRIS: TODO: we could the leaf id here instead and just lookup the claim from the leaf
    bytes32 winningClaim;
    ChallengeType challengeType; // CHRIS: TODO: can use the keyword 'type' here?
}

// CHRIS: TODO: one step proof test just here for structure test
contract OneStepProofManager {
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
