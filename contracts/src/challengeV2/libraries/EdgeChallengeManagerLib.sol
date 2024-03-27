// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "./UintUtilsLib.sol";
import "./MerkleTreeLib.sol";
import "./ChallengeEdgeLib.sol";
import "../../osp/IOneStepProofEntry.sol";
import "../../rollup/AssertionState.sol";
import "../../libraries/Constants.sol";
import "./ChallengeErrors.sol";

/// @notice An execution state and proof to show that it's valid
struct AssertionStateData {
    /// @notice An execution state
    AssertionState assertionState;
    /// @notice assertion Hash of the prev assertion
    bytes32 prevAssertionHash;
    /// @notice Inbox accumulator of the assertion
    bytes32 inboxAcc;
}

/// @notice Data for creating a layer zero edge
struct CreateEdgeArgs {
    /// @notice The level of edge to be created. Challenges are decomposed into multiple levels.
    ///         The first (level 0) being of type Block, followed by n (set by NUM_BIGSTEP_LEVEL) levels of type BigStep, and finally
    ///         followed by a single level of type SmallStep. Each level is bisected until an edge
    ///         of length one is reached before proceeding to the next level. The first edge in each level (the layer zero edge)
    ///         makes a claim about an assertion or assertion in the lower level.
    ///         Finally in the last level, a SmallStep edge is added that claims a lower level length one BigStep edge, and these
    ///         SmallStep edges are bisected until they reach length one. A length one small step edge
    ///         can then be directly executed using a one-step proof.
    uint8 level;
    /// @notice The end history root of the edge to be created
    bytes32 endHistoryRoot;
    /// @notice The end height of the edge to be created.
    /// @dev    End height is deterministic for different levels but supplying it here gives the
    ///         caller a bit of extra security that they are supplying data for the correct level of edge
    uint256 endHeight;
    /// @notice The edge, or assertion, that is being claimed correct by the newly created edge.
    bytes32 claimId;
    /// @notice Proof that the start history root commits to a prefix of the states that
    ///         end history root commits to
    bytes prefixProof;
    /// @notice Edge type specific data
    ///         For Block type edges this is the abi encoding of:
    ///         bytes32[]: Inclusion proof - proof to show that the end state is the last state in the end history root
    ///         AssertionStateData: the before state of the edge
    ///         AssertionStateData: the after state of the edge
    ///         bytes32 predecessorId: id of the prev assertion
    ///         bytes32 inboxAcc:  the inbox accumulator of the assertion
    ///         For BigStep and SmallStep edges this is the abi encoding of:
    ///         bytes32: Start state - first state the edge commits to
    ///         bytes32: End state - last state the edge commits to
    ///         bytes32[]: Claim start inclusion proof - proof to show the start state is the first state in the claim edge
    ///         bytes32[]: Claim end inclusion proof - proof to show the end state is the last state in the claim edge
    ///         bytes32[]: Inclusion proof - proof to show that the end state is the last state in the end history root
    bytes proof;
}

/// @notice Data parsed raw proof data
struct ProofData {
    /// @notice The first state being committed to by an edge
    bytes32 startState;
    /// @notice The last state being committed to by an edge
    bytes32 endState;
    /// @notice A proof that the end state is included in the edge
    bytes32[] inclusionProof;
}

/// @notice Stores all edges and their rival status
struct EdgeStore {
    /// @notice A mapping of edge id to edges. Edges are never deleted, only created, and potentially confirmed.
    mapping(bytes32 => ChallengeEdge) edges;
    /// @notice A mapping of mutualId to edge id. Rivals share the same mutual id, and here we
    ///         store the edge id of the second edge that was created with the same mutual id - the first rival
    ///         When only one edge exists for a specific mutual id then a special magic string hash is stored instead
    ///         of the first rival id, to signify that a single edge does exist with this mutual id
    mapping(bytes32 => bytes32) firstRivals;
    /// @notice A mapping of mutualId to the edge id of the confirmed rival with that mutualId
    /// @dev    Each group of rivals (edges sharing mutual id) can only have at most one confirmed edge
    mapping(bytes32 => bytes32) confirmedRivals;
}

/// @notice Input data to a one step proof
struct OneStepData {
    /// @notice The hash of the state that's being executed from
    bytes32 beforeHash;
    /// @notice Proof data to accompany the execution context
    bytes proof;
}

/// @notice Data about a recently added edge
struct EdgeAddedData {
    bytes32 edgeId;
    bytes32 mutualId;
    bytes32 originId;
    bytes32 claimId;
    uint256 length;
    uint8 level;
    bool hasRival;
    bool isLayerZero;
}

/// @notice Data about an assertion that is being claimed by an edge
/// @dev    This extra information that is needed in order to verify that a block edge can be created
struct AssertionReferenceData {
    /// @notice The id of the assertion - will be used in a sanity check
    bytes32 assertionHash;
    /// @notice The predecessor of the assertion
    bytes32 predecessorId;
    /// @notice Is the assertion pending
    bool isPending;
    /// @notice Does the assertion have a sibling
    bool hasSibling;
    /// @notice The execution state of the predecessor assertion
    AssertionState startState;
    /// @notice The execution state of the assertion being claimed
    AssertionState endState;
}

/// @title  Core functionality for the Edge Challenge Manager
/// @notice The edge manager library allows edges to be added and bisected, and keeps track of the amount
///         of time an edge remained unrivaled.
library EdgeChallengeManagerLib {
    using ChallengeEdgeLib for ChallengeEdge;
    using GlobalStateLib for GlobalState;
    using AssertionStateLib for AssertionState;

    /// @dev Magic string hash to represent that a edges with a given mutual id have no rivals
    bytes32 public constant UNRIVALED = keccak256(abi.encodePacked("UNRIVALED"));

    /// @notice Get an edge from the store
    /// @dev    Throws if the edge does not exist in the store
    /// @param store    The edge store to fetch an id from
    /// @param edgeId   The id of the edge to fetch
    function get(EdgeStore storage store, bytes32 edgeId) internal view returns (ChallengeEdge storage) {
        if (!store.edges[edgeId].exists()) {
            revert EdgeNotExists(edgeId);
        }
        return store.edges[edgeId];
    }

    /// @notice Gets an edge from the store without checking if it exists
    /// @dev    Useful where you already know the edge exists in the store - avoid a storage lookup
    /// @param store    The edge store to fetch an id from
    /// @param edgeId   The id of the edge to fetch
    function getNoCheck(EdgeStore storage store, bytes32 edgeId) internal view returns (ChallengeEdge storage) {
        return store.edges[edgeId];
    }

    /// @notice Adds a new edge to the store
    /// @dev    Updates first rival info for later use in calculating time unrivaled
    /// @param store    The store to add the edge to
    /// @param edge     The edge to add
    function add(EdgeStore storage store, ChallengeEdge memory edge) internal returns (EdgeAddedData memory) {
        bytes32 eId = edge.idMem();
        // add the edge if it doesnt exist already
        if (store.edges[eId].exists()) {
            revert EdgeAlreadyExists(eId);
        }
        store.edges[eId] = edge;

        // edges that are rivals share the same mutual id
        // we use records of whether a mutual id has ever been added to decide if
        // the new edge is a rival. This will later allow us to calculate time an edge
        // stayed unrivaled
        bytes32 mutualId = ChallengeEdgeLib.mutualIdComponent(
            edge.level, edge.originId, edge.startHeight, edge.startHistoryRoot, edge.endHeight
        );
        bytes32 firstRival = store.firstRivals[mutualId];

        // the first time we add a mutual id we store a magic string hash against it
        // We do this to distinguish from there being no edges
        // with this mutual. And to distinguish it from the first rival, where we
        // will use an actual edge id so that we can look up the created when time
        // of the first rival, and use it for calculating time unrivaled
        if (firstRival == 0) {
            store.firstRivals[mutualId] = UNRIVALED;
        } else if (firstRival == UNRIVALED) {
            store.firstRivals[mutualId] = eId;
        } else {
            // after we've stored the first rival we dont need to keep a record of any
            // other rival edges - they will all have a zero time unrivaled
        }

        return EdgeAddedData(
            eId,
            mutualId,
            edge.originId,
            edge.claimId,
            store.edges[eId].length(),
            edge.level,
            firstRival != 0,
            edge.claimId != 0
        );
    }

    /// @notice Conduct checks that are specific to the edge type.
    /// @dev    Since different edge types also require different proofs, we also include the specific
    ///         proof parsing logic and return the common parts for later use.
    /// @param store                The store containing current edges
    /// @param args                 The edge creation args
    /// @param ard                  Data about the assertion data is is also need to when creating a Block edge type
    ///                             The created edge must be shown to be consistent with the states in the assertion chain
    ///                             Empty for non block edge type edges
    /// @param oneStepProofEntry    The one step proof contract that defines how machine states are hashed
    /// @return                     Data parsed from the proof, or fetched from elsewhere. Also the origin id for the edge to be created.
    function layerZeroTypeSpecificChecks(
        EdgeStore storage store,
        CreateEdgeArgs calldata args,
        AssertionReferenceData memory ard,
        IOneStepProofEntry oneStepProofEntry,
        uint8 numBigStepLevel
    ) private view returns (ProofData memory, bytes32) {
        if (ChallengeEdgeLib.levelToType(args.level, numBigStepLevel) == EdgeType.Block) {
            // origin id is the assertion which is the root of challenge
            // all rivals and their children share the same origin id - it is a link to the information
            // they agree on
            bytes32 originId = ard.predecessorId;

            // Sanity check: The assertion reference data should be related to the claim
            // Of course the caller can provide whatever args they wish, so this is really just a helpful
            // check to avoid mistakes
            if (ard.assertionHash == 0) {
                revert AssertionHashEmpty();
            }
            if (ard.assertionHash != args.claimId) {
                revert AssertionHashMismatch(ard.assertionHash, args.claimId);
            }

            // if the assertion is already confirmed or rejected then it cant be referenced as a claim
            if (!ard.isPending) {
                revert AssertionNotPending();
            }

            // if the claim doesnt have a sibling then it is undisputed, there's no need
            // to open challenge edges for it
            if (!ard.hasSibling) {
                revert AssertionNoSibling();
            }

            // parse the inclusion proof for later use
            if (args.proof.length == 0) {
                revert EmptyEdgeSpecificProof();
            }
            (bytes32[] memory inclusionProof,,) =
                abi.decode(args.proof, (bytes32[], AssertionStateData, AssertionStateData));

            // check the start and end execution states exist, the block hash entry should be non zero
            if (ard.startState.machineStatus == MachineStatus.RUNNING) {
                revert EmptyStartMachineStatus();
            }
            if (ard.endState.machineStatus == MachineStatus.RUNNING) {
                revert EmptyEndMachineStatus();
            }

            // Create machine hashes out of the proven state
            bytes32 startStateHash = oneStepProofEntry.getMachineHash(ard.startState.toExecutionState());
            bytes32 endStateHash = oneStepProofEntry.getMachineHash(ard.endState.toExecutionState());

            return (ProofData(startStateHash, endStateHash, inclusionProof), originId);
        } else {
            // Claim must be length one. If it is unrivaled then its unrivaled time is ticking up, so there's
            // no need to create claims against it
            if (!hasLengthOneRival(store, args.claimId)) {
                revert ClaimEdgeNotLengthOneRival(args.claimId);
            }

            // hasLengthOneRival checks existance, so we can use getNoCheck
            ChallengeEdge storage claimEdge = getNoCheck(store, args.claimId);

            // origin id is the mutual id of the claim
            // all rivals and their children share the same origin id - it is a link to the information
            // they agree on
            bytes32 originId = claimEdge.mutualId();

            // once a claim is confirmed it's status can never become pending again, so there is no point
            // opening a challenge that references it
            if (claimEdge.status != EdgeStatus.Pending) {
                revert ClaimEdgeNotPending();
            }

            // the edge must be a level up from the claim
            if (args.level != nextEdgeLevel(claimEdge.level, numBigStepLevel)) {
                revert ClaimEdgeInvalidLevel(args.level, claimEdge.level);
            }

            // parse the proofs
            if (args.proof.length == 0) {
                revert EmptyEdgeSpecificProof();
            }
            (
                bytes32 startState,
                bytes32 endState,
                bytes32[] memory claimStartInclusionProof,
                bytes32[] memory claimEndInclusionProof,
                bytes32[] memory edgeInclusionProof
            ) = abi.decode(args.proof, (bytes32, bytes32, bytes32[], bytes32[], bytes32[]));

            // if the start and end states are consistent with the claim edge
            // this guarantees that the edge we're creating is a 'continuation' of the claim edge, it is
            // a commitment to the states that between start and end states of the claim
            MerkleTreeLib.verifyInclusionProof(
                claimEdge.startHistoryRoot, startState, claimEdge.startHeight, claimStartInclusionProof
            );

            // it's doubly important to check the end state since if the end state since the claim id is
            // not part of the edge id, so we need to ensure that it's not possible to create two edges of the
            // same id, but with different claim id. Ensuring that the end state is linked to the claim,
            // and later ensuring that the end state is part of the history commitment of the new edge ensures
            // that the end history root of the new edge will be different for different claim ids, and therefore
            // the edge ids will be different
            MerkleTreeLib.verifyInclusionProof(
                claimEdge.endHistoryRoot, endState, claimEdge.endHeight, claimEndInclusionProof
            );

            return (ProofData(startState, endState, edgeInclusionProof), originId);
        }
    }

    /// @notice Check that a uint is a power of 2
    function isPowerOfTwo(uint256 x) internal pure returns (bool) {
        // zero is not a power of 2
        if (x == 0) {
            return false;
        }

        // if x is a power of 2, then this will be 0111111
        uint256 y = x - 1;

        // if x is a power of 2 then y will share no bits with x
        return ((x & y) == 0);
    }

    /// @notice Common checks that apply to all layer zero edges
    /// @param proofData            Data extracted from supplied proof
    /// @param args                 The edge creation args
    /// @param expectedEndHeight    Edges have a deterministic end height dependent on their level
    function layerZeroCommonChecks(ProofData memory proofData, CreateEdgeArgs calldata args, uint256 expectedEndHeight)
        private
        pure
        returns (bytes32)
    {
        // since zero layer edges have a start height of zero, we know that they are a size
        // one tree containing only the start state. We can then compute the history root directly
        bytes32 startHistoryRoot = MerkleTreeLib.root(MerkleTreeLib.appendLeaf(new bytes32[](0), proofData.startState));

        // all end heights are expected to be a power of 2, the specific power is defined by the
        // edge challenge manager itself
        if (!isPowerOfTwo(expectedEndHeight)) {
            revert NotPowerOfTwo(expectedEndHeight);
        }

        // It isnt strictly necessary to pass in the end height, we know what it
        // should be so we could just use the end height that we get from getLayerZeroEndHeight
        // However it's a nice sanity check for the calling code to check that their local edge
        // will have the same height as the one created here
        if (args.endHeight != expectedEndHeight) {
            revert InvalidEndHeight(args.endHeight, expectedEndHeight);
        }

        // the end state is checked/determined as part of the specific edge level
        // We then ensure that that same end state is part of the end history root we're creating
        // This ensures continuity of states between levels - the state is present in both this
        // level and the one below
        MerkleTreeLib.verifyInclusionProof(
            args.endHistoryRoot, proofData.endState, args.endHeight, proofData.inclusionProof
        );

        // start root must always be a prefix of end root, we ensure that
        // this new edge adheres to this. Future bisections will ensure that this
        // property is conserved
        if (args.prefixProof.length == 0) {
            revert EmptyPrefixProof();
        }
        (bytes32[] memory preExpansion, bytes32[] memory preProof) =
            abi.decode(args.prefixProof, (bytes32[], bytes32[]));
        MerkleTreeLib.verifyPrefixProof(
            startHistoryRoot, 1, args.endHistoryRoot, args.endHeight + 1, preExpansion, preProof
        );

        return (startHistoryRoot);
    }

    /// @notice Creates a new layer zero edges from edge creation args
    function toLayerZeroEdge(bytes32 originId, bytes32 startHistoryRoot, CreateEdgeArgs calldata args)
        private
        view
        returns (ChallengeEdge memory)
    {
        return ChallengeEdgeLib.newLayerZeroEdge(
            originId, startHistoryRoot, 0, args.endHistoryRoot, args.endHeight, args.claimId, msg.sender, args.level
        );
    }

    /// @notice Performs necessary checks and creates a new layer zero edge
    /// @param store                The store containing existing edges
    /// @param args                 Edge data
    /// @param ard                  If the edge being added is of Block type then additional assertion data is required
    ///                             to check if the edge can be added. Empty if edge is not of type Block.
    ///                             The supplied assertion data must be related to the assertion that is being claimed
    ///                             by the supplied edge args
    /// @param oneStepProofEntry    The one step proof contract that defines how machine states are hashed
    /// @param expectedEndHeight    The expected end height of an edge. Layer zero block edges have predefined heights.
    /// @param numBigStepLevel      The number of big step levels in this challenge
    function createLayerZeroEdge(
        EdgeStore storage store,
        CreateEdgeArgs calldata args,
        AssertionReferenceData memory ard,
        IOneStepProofEntry oneStepProofEntry,
        uint256 expectedEndHeight,
        uint8 numBigStepLevel
    ) internal returns (EdgeAddedData memory) {
        // each edge type requires some specific checks
        (ProofData memory proofData, bytes32 originId) =
            layerZeroTypeSpecificChecks(store, args, ard, oneStepProofEntry, numBigStepLevel);
        // all edge types share some common checks
        (bytes32 startHistoryRoot) = layerZeroCommonChecks(proofData, args, expectedEndHeight);
        // we only wrap the struct creation in a function as doing so with exceeds the stack limit
        ChallengeEdge memory ce = toLayerZeroEdge(originId, startHistoryRoot, args);
        return add(store, ce);
    }

    /// @notice From any given edge, get the id of the previous assertion
    /// @param edgeId           The edge to get the prev assertion hash
    function getPrevAssertionHash(EdgeStore storage store, bytes32 edgeId) internal view returns (bytes32) {
        ChallengeEdge storage edge = get(store, edgeId);
        while (edge.level > 0) {
            // the origin id gives us a link to the lower level
            // we know a first rival must exist, since otherwise we would not have had a one step fork
            // and we wouldnt be able to go to the next level
            // we can use the first rival to get an edge id, and from there get the next origin id
            bytes32 lowerLevelId = store.firstRivals[edge.originId];
            edge = get(store, lowerLevelId);
        }

        // For Block type edges the origin id is the assertion hash of claim prev
        return edge.originId;
    }

    /// @notice Does this edge currently have one or more rivals
    ///         Rival edges share the same startHeight, startHistoryCommitment and the same endHeight,
    ///         but they have a different endHistoryRoot. Rival edges have the same mutualId
    /// @param store    The edge store containing the edge
    /// @param edgeId   The edge if to test if it is unrivaled
    function hasRival(EdgeStore storage store, bytes32 edgeId) internal view returns (bool) {
        if (!store.edges[edgeId].exists()) {
            revert EdgeNotExists(edgeId);
        }

        // rivals have the same mutual id
        bytes32 mutualId = store.edges[edgeId].mutualId();
        bytes32 firstRival = store.firstRivals[mutualId];
        // Sanity check: it should never be possible to create an edge without having an entry in firstRivals
        if (firstRival == 0) {
            revert EmptyFirstRival();
        }

        // can only have no rival if the firstRival is the UNRIVALED magic hash
        return firstRival != UNRIVALED;
    }

    /// @notice Is the edge a single step in length, and does it have at least one rival.
    /// @param store    The edge store containing the edge
    /// @param edgeId   The edge id to test for single step and rivaled
    function hasLengthOneRival(EdgeStore storage store, bytes32 edgeId) internal view returns (bool) {
        // must be length 1 and have rivals - all rivals have the same length
        return (hasRival(store, edgeId) && store.edges[edgeId].length() == 1);
    }

    function timeUnrivaledTotal(EdgeStore storage store, bytes32 edgeId) internal view returns (uint256) {
        uint256 totalTimeUnrivaled = timeUnrivaled(store, edgeId);
        if (store.edges[edgeId].lowerChildId != bytes32(0)) {
            uint256 lowerTimer = store.edges[store.edges[edgeId].lowerChildId].totalTimeUnrivaledCache;
            uint256 upperTimer = store.edges[store.edges[edgeId].upperChildId].totalTimeUnrivaledCache;
            totalTimeUnrivaled += lowerTimer < upperTimer ? lowerTimer : upperTimer;
        }
        return totalTimeUnrivaled;
    }

    function updateTimerCache(EdgeStore storage store, bytes32 edgeId, uint256 newValue) internal returns (bool) {
        uint256 currentAccuTimer = store.edges[edgeId].totalTimeUnrivaledCache;
        newValue = newValue > type(uint64).max ? type(uint64).max : newValue;
        // only update when increased
        if (newValue > currentAccuTimer) {
            store.edges[edgeId].totalTimeUnrivaledCache = uint64(newValue);
            return true;
        }
        return false;
    }

    function updateTimerCacheByChildren(EdgeStore storage store, bytes32 edgeId) internal {
        updateTimerCache(store, edgeId, timeUnrivaledTotal(store, edgeId));
    }

    function updateTimerCacheByClaim(
        EdgeStore storage store,
        bytes32 edgeId,
        bytes32 claimingEdgeId,
        uint8 numBigStepLevel
    ) internal {
        // calculate the time unrivaled without inheritance
        uint256 totalTimeUnrivaled = timeUnrivaled(store, edgeId);
        checkClaimIdLink(store, edgeId, claimingEdgeId, numBigStepLevel);
        totalTimeUnrivaled += store.edges[claimingEdgeId].totalTimeUnrivaledCache;
        updateTimerCache(store, edgeId, totalTimeUnrivaled);
    }

    /// @notice The amount of time (in blocks) this edge has spent without rivals
    ///         This value is increasing whilst an edge is unrivaled, once a rival is created
    ///         it is fixed. If an edge has rivals from the moment it is created then it will have
    ///         a zero time unrivaled
    function timeUnrivaled(EdgeStore storage store, bytes32 edgeId) internal view returns (uint256) {
        if (!store.edges[edgeId].exists()) {
            revert EdgeNotExists(edgeId);
        }

        bytes32 mutualId = store.edges[edgeId].mutualId();
        bytes32 firstRival = store.firstRivals[mutualId];
        // Sanity check: it's not possible to have a 0 first rival for an edge that exists
        if (firstRival == 0) {
            revert EmptyFirstRival();
        }

        // this edge has no rivals, the time is still going up
        // we give the current amount of time unrivaled
        if (firstRival == UNRIVALED) {
            return block.number - store.edges[edgeId].createdAtBlock;
        } else {
            // Sanity check: it's not possible an edge does not exist for a first rival record
            if (!store.edges[firstRival].exists()) {
                revert EdgeNotExists(firstRival);
            }

            // rivals exist for this edge
            uint256 firstRivalCreatedAtBlock = store.edges[firstRival].createdAtBlock;
            uint256 edgeCreatedAtBlock = store.edges[edgeId].createdAtBlock;
            if (firstRivalCreatedAtBlock > edgeCreatedAtBlock) {
                // if this edge was created before the first rival then we return the difference
                // in createdAtBlock number
                return firstRivalCreatedAtBlock - edgeCreatedAtBlock;
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
        if (end - start < 2) {
            revert HeightDiffLtTwo(start, end);
        }
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
    ///         is the mandatoryBisectionHeight.
    ///         The lower child may already exist, however it's not possible for the upper child to exist as that would
    ///         mean that the edge has already been bisected
    /// @param store                The edge store containing the edge to bisect
    /// @param edgeId               Edge to bisect
    /// @param bisectionHistoryRoot The new history root to be used in the lower and upper children
    /// @param prefixProof          A proof to show that the bisectionHistoryRoot commits to a prefix of the current endHistoryRoot
    /// @return lowerChildId        The id of the newly created lower child edge
    /// @return lowerChildAdded     Data about the lower child edge, empty if the lower child already existed
    /// @return upperChildAdded     Data about the upper child edge, never empty
    function bisectEdge(EdgeStore storage store, bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        internal
        returns (bytes32, EdgeAddedData memory, EdgeAddedData memory)
    {
        if (store.edges[edgeId].status != EdgeStatus.Pending) {
            revert EdgeNotPending(edgeId, store.edges[edgeId].status);
        }
        if (!hasRival(store, edgeId)) {
            revert EdgeUnrivaled(edgeId);
        }

        // cannot bisect an edge twice
        // has rival above checks the edge - so no need to check again
        ChallengeEdge memory ce = getNoCheck(store, edgeId);

        // bisections occur at deterministic heights, this ensures that
        // rival edges bisect at the same height, and create the same child if they agree
        uint256 middleHeight = mandatoryBisectionHeight(ce.startHeight, ce.endHeight);
        {
            (bytes32[] memory preExpansion, bytes32[] memory proof) = abi.decode(prefixProof, (bytes32[], bytes32[]));
            MerkleTreeLib.verifyPrefixProof(
                bisectionHistoryRoot, middleHeight + 1, ce.endHistoryRoot, ce.endHeight + 1, preExpansion, proof
            );
        }

        bytes32 lowerChildId;
        EdgeAddedData memory lowerChildAdded;
        {
            // midpoint proof it valid, create and store the children
            ChallengeEdge memory lowerChild = ChallengeEdgeLib.newChildEdge(
                ce.originId, ce.startHistoryRoot, ce.startHeight, bisectionHistoryRoot, middleHeight, ce.level
            );
            lowerChildId = lowerChild.idMem();
            // it's possible that the store already has the lower child if it was created by a rival
            // (aka a merge move)
            if (!store.edges[lowerChildId].exists()) {
                lowerChildAdded = add(store, lowerChild);
            }
        }

        EdgeAddedData memory upperChildAdded;
        {
            ChallengeEdge memory upperChild = ChallengeEdgeLib.newChildEdge(
                ce.originId, bisectionHistoryRoot, middleHeight, ce.endHistoryRoot, ce.endHeight, ce.level
            );

            // add checks existence and throws if the id already exists
            upperChildAdded = add(store, upperChild);
        }

        store.edges[edgeId].setChildren(lowerChildId, upperChildAdded.edgeId);

        return (lowerChildId, lowerChildAdded, upperChildAdded);
    }

    /// @notice Store that an edge has been confirmed. Recorded against the mutual id
    ///         so that rivals can look up rival confirmed edges
    /// @dev    Checks that a rival edge has not already been confirmed
    function setConfirmedRival(EdgeStore storage store, bytes32 edgeId) internal {
        bytes32 mutualId = store.edges[edgeId].mutualId();
        bytes32 confirmedRivalId = store.confirmedRivals[mutualId];
        if (confirmedRivalId != bytes32(0)) {
            revert RivalEdgeConfirmed(edgeId, confirmedRivalId);
        }
        store.confirmedRivals[mutualId] = edgeId;
    }

    /// @notice Returns the sub edge level of the provided edge level
    /// @param level            The edge level to fetch the next of
    /// @param numBigStepLevel  The number of big step levels in this challenge
    function nextEdgeLevel(uint8 level, uint8 numBigStepLevel) internal pure returns (uint8) {
        uint8 nextLevel = level + 1;

        // levelToType throws an error when level is not a valid type
        ChallengeEdgeLib.levelToType(nextLevel, numBigStepLevel);

        return nextLevel;
    }

    /// @notice Check that the originId of a claiming edge matched the mutualId() of a supplied edge
    /// @dev    Does some additional sanity checks to ensure that the claim id link is valid
    /// @param store            The store containing all edges and rivals
    /// @param edgeId           The edge being claimed
    /// @param claimingEdgeId   The edge with a claim id equal to edge id
    /// @param numBigStepLevel  The number of big step levels in this challenge
    function checkClaimIdLink(EdgeStore storage store, bytes32 edgeId, bytes32 claimingEdgeId, uint8 numBigStepLevel)
        private
        view
    {
        // we do some extra checks that edge being claimed is eligible to be claimed by the claiming edge
        // these shouldn't be necessary since it should be impossible to add layer zero edges that do not
        // satisfy the checks below, but we conduct these checks anyway for double safety

        // the origin id of an edge should be the mutual id of the edge in the level below
        if (store.edges[edgeId].mutualId() != store.edges[claimingEdgeId].originId) {
            revert OriginIdMutualIdMismatch(store.edges[edgeId].mutualId(), store.edges[claimingEdgeId].originId);
        }
        // the claiming edge must be exactly one level below
        if (nextEdgeLevel(store.edges[edgeId].level, numBigStepLevel) != store.edges[claimingEdgeId].level) {
            revert EdgeLevelInvalid(
                edgeId,
                claimingEdgeId,
                nextEdgeLevel(store.edges[edgeId].level, numBigStepLevel),
                store.edges[claimingEdgeId].level
            );
        }
    }

    /// @notice An edge can be confirmed if the total amount of time (in blocks) it and a single chain of its direct ancestors
    ///         has spent unrivaled is greater than the challenge period.
    /// @dev    Edges inherit time from their parents, so the sum of unrivaled timer is compared against the threshold.
    ///         Given that an edge cannot become unrivaled after becoming rivaled, once the threshold is passed
    ///         it will always remain passed. The direct ancestors of an edge are linked by parent-child links for edges
    ///         of the same level, and claimId-edgeId links for zero layer edges that claim an edge in the level below.
    /// @param store                            The edge store containing all edges and rival data
    /// @param edgeId                           The id of the edge to confirm
    /// @param claimedAssertionUnrivaledBlocks  The number of blocks that the assertion ultimately being claimed by this edge spent unrivaled
    /// @param confirmationThresholdBlock       The number of blocks that the total unrivaled time of an ancestor chain needs to exceed in
    ///                                         order to be confirmed
    function confirmEdgeByTime(
        EdgeStore storage store,
        bytes32 edgeId,
        uint64 claimedAssertionUnrivaledBlocks,
        uint64 confirmationThresholdBlock
    ) internal returns (uint256) {
        if (!store.edges[edgeId].exists()) {
            revert EdgeNotExists(edgeId);
        }

        uint256 totalTimeUnrivaled = timeUnrivaledTotal(store, edgeId);

        // since sibling assertions have the same predecessor, they can be viewed as
        // rival edges. Adding the assertion unrivaled time allows us to start the confirmation
        // timer from the moment the first assertion is made, rather than having to wait until the
        // second assertion is made.
        totalTimeUnrivaled += claimedAssertionUnrivaledBlocks;

        if (totalTimeUnrivaled < confirmationThresholdBlock) {
            revert InsufficientConfirmationBlocks(totalTimeUnrivaled, confirmationThresholdBlock);
        }

        // also checks that no other rival has been confirmed
        setConfirmedRival(store, edgeId);

        // we also check the edge is pending in setConfirmed()
        store.edges[edgeId].setConfirmed();

        return totalTimeUnrivaled;
    }

    /// @notice Confirm an edge by executing a one step proof
    /// @dev    One step proofs can only be executed against edges that have length one and of type SmallStep
    /// @param store                        The edge store containing all edges and rival data
    /// @param edgeId                       The id of the edge to confirm
    /// @param oneStepProofEntry            The one step proof contract
    /// @param oneStepData                  Input data to the one step proof
    /// @param execCtx                      The execution context to be supplied to the one step proof entry
    /// @param beforeHistoryInclusionProof  Proof that the state which is the start of the edge is committed to by the startHistoryRoot
    /// @param afterHistoryInclusionProof   Proof that the state which is the end of the edge is committed to by the endHistoryRoot
    /// @param numBigStepLevel              The number of big step levels in this challenge
    /// @param bigStepHeight                The height of the zero layer levels of big step type
    /// @param smallStepHeight              The height of the zero layer levels of big step type
    function confirmEdgeByOneStepProof(
        EdgeStore storage store,
        bytes32 edgeId,
        IOneStepProofEntry oneStepProofEntry,
        OneStepData calldata oneStepData,
        ExecutionContext memory execCtx,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof,
        uint8 numBigStepLevel,
        uint256 bigStepHeight,
        uint256 smallStepHeight
    ) internal {
        if (!store.edges[edgeId].exists()) {
            revert EdgeNotExists(edgeId);
        }

        // edge must of type SmallStep
        if (ChallengeEdgeLib.levelToType(store.edges[edgeId].level, numBigStepLevel) != EdgeType.SmallStep) {
            revert EdgeTypeNotSmallStep(store.edges[edgeId].level);
        }

        // edge must be length one to execute one step proofs against
        if (store.edges[edgeId].length() != 1) {
            revert EdgeNotLengthOne(store.edges[edgeId].length());
        }

        // Get the machine step that corresponds to the start height of this edge
        // To do this we sum the machine steps of the edges in each of the preceeding levels.
        // We do not include the block height, since each step at the block level is a new block
        // and new blocks reset the machine step to 0.
        uint256 machineStep = store.edges[edgeId].startHeight;
        {
            bytes32 cursor = edgeId;
            uint256 stepSize = smallStepHeight;
            while (store.edges[cursor].level > 1) {
                bytes32 nextEdgeId = store.edges[cursor].originId;
                // We can traverse to previous levels using the origin id
                cursor = store.firstRivals[nextEdgeId];
                // sum the stepSize * offset from 0 at this level
                machineStep += stepSize * store.edges[cursor].startHeight;
                // the step size at each level is the product of the heights at all succeeding levels
                stepSize *= bigStepHeight;
            }
        }

        // the state in the onestep data must be committed to by the startHistoryRoot
        MerkleTreeLib.verifyInclusionProof(
            store.edges[edgeId].startHistoryRoot, oneStepData.beforeHash, machineStep, beforeHistoryInclusionProof
        );

        // execute the single step to produce the after state
        bytes32 afterHash =
            oneStepProofEntry.proveOneStep(execCtx, machineStep, oneStepData.beforeHash, oneStepData.proof);

        // check that the after state was indeed committed to by the endHistoryRoot
        MerkleTreeLib.verifyInclusionProof(
            store.edges[edgeId].endHistoryRoot, afterHash, machineStep + 1, afterHistoryInclusionProof
        );

        // also checks that no other rival has been confirmed
        setConfirmedRival(store, edgeId);

        // we also check the edge is pending in setConfirmed()
        store.edges[edgeId].setConfirmed();
        store.edges[edgeId].totalTimeUnrivaledCache = type(uint64).max;
    }
}
