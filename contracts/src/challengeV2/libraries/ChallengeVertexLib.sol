// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

enum VertexStatus {
    Pending, // This vertex is vertex is pending, it has yet to be confirmed
    Confirmed // This vertex has been confirmed, once confirmed it cannot be unconfirmed
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
    /// @dev    Always zero for leaf vertices as they have no successors.
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
