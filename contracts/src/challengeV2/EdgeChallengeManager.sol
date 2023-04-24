// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../rollup/Assertion.sol";
import "./libraries/UintUtilsLib.sol";
import "./DataEntities.sol";
import "./libraries/EdgeChallengeManagerLib.sol";
import "../libraries/Constants.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

/// @title EdgeChallengeManager interface
interface IEdgeChallengeManager {
    /// @notice Initialize the EdgeChallengeManager. EdgeChallengeManagers are upgradeable
    ///         so use the initializer paradigm
    function initialize(
        IAssertionChain _assertionChain,
        uint256 _challengePeriodBlocks,
        IOneStepProofEntry _oneStepProofEntry
    ) external;

    /// @notice Performs necessary checks and creates a new layer zero edge
    /// @param args             Edge data
    /// @param prefixProof      Proof that the start history root commits to a prefix of the states that
    ///                         end history root commits to
    /// @param proof            Additional proof data
    ///                         For Block type edges this is the abi encoding of:
    ///                         bytes32[]: Inclusion proof - proof to show that the end state is the last state in the end history root
    ///                         For BigStep and SmallStep edges this is the abi encoding of:
    ///                         bytes32: Start state - first state the edge commits to
    ///                         bytes32: End state - last state the edge commits to
    ///                         bytes32[]: Claim start inclusion proof - proof to show the start state is the first state in the claim edge
    ///                         bytes32[]: Claim end inclusion proof - proof to show the end state is the last state in the claim edge
    ///                         bytes32[]: Inclusion proof - proof to show that the end state is the last state in the end history root
    function createLayerZeroEdge(CreateEdgeArgs memory args, bytes calldata prefixProof, bytes calldata proof)
        external
        payable
        returns (bytes32);

    /// @notice Bisect and edge. This creates two child edges:
    ///         lowerChild: has the same start root and height as this edge, but a different end root and height
    ///         upperChild: has the same end root and height as this edge, but a different start root and height
    ///         The lower child end root and height are equal to the upper child start root and height. This height
    ///         is the mandatoryBisectionHeight.
    ///         The lower child may already exist, however it's not possible for the upper child to exist as that would
    ///         mean that the edge has already been bisected
    /// @param edgeId               Edge to bisect
    /// @param bisectionHistoryRoot The new history root to be used in the lower and upper children
    /// @param prefixProof          A proof to show that the bisectionHistoryRoot commits to a prefix of the current endHistoryRoot
    /// @return lowerChildId        The id of the newly created lower child edge
    /// @return upperChildId        The id of the newly created upper child edge
    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32);

    /// @notice Confirm an edge if both its children are already confirmed
    function confirmEdgeByChildren(bytes32 edgeId) external;

    /// @notice An edge can be confirmed if the total amount of time it and a single chain of its direct ancestors
    ///         has spent unrivaled is greater than the challenge period.
    /// @dev    Edges inherit time from their parents, so the sum of unrivaled timers is compared against the threshold.
    ///         Given that an edge cannot become unrivaled after becoming rivaled, once the threshold is passed
    ///         it will always remain passed. The direct ancestors of an edge are linked by parent-child links for edges
    ///         of the same edgeType, and claimId-edgeid links for zero layer edges that claim an edge in the level above.
    /// @param edgeId                   The id of the edge to confirm
    /// @param ancestorEdgeIds          The ids of the direct ancestors of an edge. These are ordered from the parent first, then going to grand-parent,
    ///                                 great-grandparent etc. The chain can extend only as far as the zero layer edge of type Block.
    function confirmEdgeByTime(bytes32 edgeId, bytes32[] memory ancestorEdgeIds) external;

    /// @notice If a confirmed edge exists whose claim id is equal to this edge, then this edge can be confirmed
    /// @dev    When zero layer edges are created they reference an edge, or assertion, in the level above. If a zero layer
    ///         edge is confirmed, it becomes possible to also confirm the edge that it claims
    /// @param edgeId           The id of the edge to confirm
    /// @param claimingEdgeId   The id of the edge which has a claimId equal to edgeId
    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) external;

    /// @notice Confirm an edge by executing a one step proof
    /// @dev    One step proofs can only be executed against edges that have length one and of type SmallStep
    /// @param edgeId                       The id of the edge to confirm
    /// @param oneStepData                  Input data to the one step proof
    /// @param beforeHistoryInclusionProof  Proof that the state which is the start of the edge is committed to by the startHistoryRoot
    /// @param afterHistoryInclusionProof   Proof that the state which is the end of the edge is committed to by the endHistoryRoot
    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) external;

    /// @notice Calculate the unique id of an edge
    /// @param edgeType         The type of edge
    /// @param originId         The origin id of the edge
    /// @param startHeight      The start height of the edge
    /// @param startHistoryRoot The start history root of the edge
    /// @param endHeight        The end height of the edge
    /// @param endHistoryRoot   The end history root of the edge
    function calculateEdgeId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) external pure returns (bytes32);

    /// @notice Calculate the mutual id of the edge
    ///         Edges that are rivals share the same mutual id
    /// @param edgeType         The type of the edge
    /// @param originId         The origin id of the edge
    /// @param startHeight      The start height of the edge
    /// @param startHistoryRoot The start history root of the edge
    /// @param endHeight        The end height of the edge
    function calculateMutualId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) external pure returns (bytes32);

    /// @notice Has the edge already been stored in the manager
    function edgeExists(bytes32 edgeId) external view returns (bool);

    /// @notice Get full edge data for an edge
    function getEdge(bytes32 edgeId) external view returns (ChallengeEdge memory);

    /// @notice The length of the edge, from start height to end height
    function edgeLength(bytes32 edgeId) external view returns (uint256);

    /// @notice Does this edge currently have one or more rivals
    ///         Rival edges share the same mutual id
    function hasRival(bytes32 edgeId) external view returns (bool);

    /// @notice Does the edge have at least one rival, and it has length one
    function hasLengthOneRival(bytes32 edgeId) external view returns (bool);

    /// @notice The amount of time this edge has spent without rivals
    ///         This value is increasing whilst an edge is unrivaled, once a rival is created
    ///         it is fixed. If an edge has rivals from the moment it is created then it will have
    ///         a zero time unrivaled
    function timeUnrivaled(bytes32 edgeId) external view returns (uint256);

    /// @notice Get the id of the prev assertion that this edge is originates from
    /// @dev    Uses the parent chain to traverse upwards SmallStep->BigStep->Block->Assertion
    ///         until it gets to the origin assertion
    function getPrevAssertionId(bytes32 edgeId) external view returns (bytes32);

    /// @notice Fetch the raw first rival record for this edge
    /// @dev    Returns 0 if the edge does not exist
    ///         Returns a magic string if the edge exists but is unrivaled
    ///         Returns the id of the second edge created with the same mutual id as this edge, if a rival exists
    function firstRival(bytes32 edgeId) external view returns (bytes32);
}

/// @title  A challenge manager that uses edge structures to decide between Assertions
/// @notice When two assertions are created that have the same predecessor the protocol needs to decide which of the two is correct
///         This challenge manager allows the staker who has created the valid assertion to enforce that it will be confirmed, and all
///         other rival assertions will be rejected. The challenge is all-vs-all in that all assertions with the same
///         predecessor will vie for succession against each other. Stakers compete by creating edges that reference the assertion they
///         believe in. These edges are then bisected, reducing the size of the disagreement with each bisection, and narrowing in on the
///         exact point of disagreement. Eventually, at step size 1, the step can be proved on-chain directly proving that the related assertion
///         must be invalid.
contract EdgeChallengeManager is IEdgeChallengeManager, Initializable {
    using EdgeChallengeManagerLib for EdgeStore;
    using ChallengeEdgeLib for ChallengeEdge;

    /// @notice A new edge has been added to the challenge manager
    /// @param edgeId       The id of the newly added edge
    /// @param mutualId     The mutual id of the added edge - all rivals share the same mutual id
    /// @param originId     The origin id of the added edge - origin ids link an edge to the level above
    /// @param hasRival     Does the newly added edge have a rival upon creation
    /// @param length       The length of the new edge
    /// @param eType        The type of the new edge
    /// @param isLayerZero  Whether the new edge was added at layer zero - has a claim and a staker
    event EdgeAdded(
        bytes32 indexed edgeId,
        bytes32 indexed mutualId,
        bytes32 indexed originId,
        bytes32 claimId,
        uint256 length,
        EdgeType eType,
        bool hasRival,
        bool isLayerZero
    );

    /// @notice An edge has been bisected
    /// @param edgeId                   The id of the edge that was bisected
    /// @param lowerChildId             The id of the lower child created during bisection
    /// @param upperChildId             The id of the upper child created during bisection
    /// @param lowerChildAlreadyExists  When an edge is bisected the lower child may already exist - created by a rival.
    event EdgeBisected(
        bytes32 indexed edgeId, bytes32 indexed lowerChildId, bytes32 indexed upperChildId, bool lowerChildAlreadyExists
    );

    /// @notice An edge can be confirmed if both of its children were already confirmed.
    /// @param edgeId   The edge that was confirmed
    /// @param mutualId The mutual id of the confirmed edge
    event EdgeConfirmedByChildren(bytes32 indexed edgeId, bytes32 indexed mutualId);

    /// @notice An edge can be confirmed if the cumulative time unrivaled of it and a direct chain of ancestors is greater than a threshold
    /// @param edgeId               The edge that was confirmed
    /// @param mutualId             The mutual id of the confirmed edge
    /// @param totalTimeUnrivaled   The cumulative amount of time this edge spent unrivaled
    event EdgeConfirmedByTime(bytes32 indexed edgeId, bytes32 indexed mutualId, uint256 totalTimeUnrivaled);

    /// @notice An edge can be confirmed if a zero layer edge in the level below claims this edge
    /// @param edgeId           The edge that was confirmed
    /// @param mutualId         The mutual id of the confirmed edge
    /// @param claimingEdgeId   The id of the zero layer edge that claimed this edge
    event EdgeConfirmedByClaim(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 claimingEdgeId);

    /// @notice A SmallStep edge of length 1 can be confirmed via a one step proof
    /// @param edgeId   The edge that was confirmed
    /// @param mutualId The mutual id of the confirmed edge
    event EdgeConfirmedByOneStepProof(bytes32 indexed edgeId, bytes32 indexed mutualId);

    /// @dev Store for all edges and rival data
    ///      All edges, including edges from different challenges, are stored together in the same store
    ///      Since edge ids include the origin id, which is unique for each challenge, we can be sure that
    ///      edges from different challenges cannot have the same id, and so can be stored in the same store
    EdgeStore internal store;

    uint256 public challengePeriodBlock;

    /// @notice The assertion chain about which challenges are created
    IAssertionChain public assertionChain;
    /// @notice The one step proof resolver used to decide between rival SmallStep edges of length 1
    IOneStepProofEntry public oneStepProofEntry;

    constructor() {
        _disableInitializers();
    }

    /// @inheritdoc IEdgeChallengeManager
    function initialize(
        IAssertionChain _assertionChain,
        uint256 _challengePeriodBlocks,
        IOneStepProofEntry _oneStepProofEntry
    ) public initializer {
        require(address(assertionChain) == address(0), "ALREADY_INIT");
        assertionChain = _assertionChain;
        challengePeriodBlock = _challengePeriodBlocks;
        oneStepProofEntry = _oneStepProofEntry;
    }

    /////////////////////////////
    // STATE MUTATING SECTIION //
    /////////////////////////////

    /// @inheritdoc IEdgeChallengeManager
    function createLayerZeroEdge(CreateEdgeArgs memory args, bytes calldata prefixProof, bytes calldata proof)
        external
        payable
        returns (bytes32)
    {
        EdgeAddedData memory edgeAdded = store.createLayerZeroEdge(assertionChain, args, prefixProof, proof);
        emit EdgeAdded(
            edgeAdded.edgeId,
            edgeAdded.mutualId,
            edgeAdded.originId,
            edgeAdded.claimId,
            edgeAdded.length,
            edgeAdded.eType,
            edgeAdded.hasRival,
            edgeAdded.isLayerZero
        );
        return edgeAdded.edgeId;
    }

    /// @inheritdoc IEdgeChallengeManager
    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes memory prefixProof)
        external
        returns (bytes32, bytes32)
    {
        (bytes32 lowerChildId, EdgeAddedData memory lowerChildAdded, EdgeAddedData memory upperChildAdded) =
            store.bisectEdge(edgeId, bisectionHistoryRoot, prefixProof);

        bool lowerChildAlreadyExists = lowerChildAdded.edgeId == 0;
        // the lower child might already exist, if it didnt then a new
        // edge was added
        if (!lowerChildAlreadyExists) {
            emit EdgeAdded(
                lowerChildAdded.edgeId,
                lowerChildAdded.mutualId,
                lowerChildAdded.originId,
                lowerChildAdded.claimId,
                lowerChildAdded.length,
                lowerChildAdded.eType,
                lowerChildAdded.hasRival,
                lowerChildAdded.isLayerZero
            );
        }
        // upper child is always added
        emit EdgeAdded(
            upperChildAdded.edgeId,
            upperChildAdded.mutualId,
            upperChildAdded.originId,
            upperChildAdded.claimId,
            upperChildAdded.length,
            upperChildAdded.eType,
            upperChildAdded.hasRival,
            upperChildAdded.isLayerZero
        );

        emit EdgeBisected(edgeId, lowerChildId, upperChildAdded.edgeId, lowerChildAlreadyExists);

        return (lowerChildId, upperChildAdded.edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByChildren(bytes32 edgeId) public {
        store.confirmEdgeByChildren(edgeId);

        emit EdgeConfirmedByChildren(edgeId, store.edges[edgeId].mutualId());
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) public {
        store.confirmEdgeByClaim(edgeId, claimingEdgeId);

        emit EdgeConfirmedByClaim(edgeId, store.edges[edgeId].mutualId(), claimingEdgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByTime(bytes32 edgeId, bytes32[] memory ancestorEdges) public {
        uint256 totalTimeUnrivaled = store.confirmEdgeByTime(edgeId, ancestorEdges, challengePeriodBlock);

        emit EdgeConfirmedByTime(edgeId, store.edges[edgeId].mutualId(), totalTimeUnrivaled);
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) public {
        bytes32 prevAssertionId = store.getPrevAssertionId(edgeId);
        ExecutionContext memory execCtx = ExecutionContext({
            maxInboxMessagesRead: assertionChain.proveInboxMsgCountSeen(
                prevAssertionId, oneStepData.inboxMsgCountSeen, oneStepData.inboxMsgCountSeenProof
                ),
            bridge: assertionChain.bridge(),
            initialWasmModuleRoot: assertionChain.proveWasmModuleRoot(
                prevAssertionId, oneStepData.wasmModuleRoot, oneStepData.wasmModuleRootProof
                )
        });

        store.confirmEdgeByOneStepProof(
            edgeId, oneStepProofEntry, oneStepData, execCtx, beforeHistoryInclusionProof, afterHistoryInclusionProof
        );

        emit EdgeConfirmedByOneStepProof(edgeId, store.edges[edgeId].mutualId());
    }

    ///////////////////////
    // VIEW ONLY SECTION //
    ///////////////////////

    /// @inheritdoc IEdgeChallengeManager
    function calculateEdgeId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) public pure returns (bytes32) {
        return
            ChallengeEdgeLib.idComponent(edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot);
    }

    /// @inheritdoc IEdgeChallengeManager
    function calculateMutualId(
        EdgeType edgeType,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.mutualIdComponent(edgeType, originId, startHeight, startHistoryRoot, endHeight);
    }

    /// @inheritdoc IEdgeChallengeManager
    function edgeExists(bytes32 edgeId) public view returns (bool) {
        return store.edges[edgeId].exists();
    }

    /// @inheritdoc IEdgeChallengeManager
    function getEdge(bytes32 edgeId) public view returns (ChallengeEdge memory) {
        return store.get(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function edgeLength(bytes32 edgeId) public view returns (uint256) {
        return store.get(edgeId).length();
    }

    /// @inheritdoc IEdgeChallengeManager
    function hasRival(bytes32 edgeId) public view returns (bool) {
        return store.hasRival(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function hasLengthOneRival(bytes32 edgeId) public view returns (bool) {
        return store.hasLengthOneRival(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function timeUnrivaled(bytes32 edgeId) public view returns (uint256) {
        return store.timeUnrivaled(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function getPrevAssertionId(bytes32 edgeId) public view returns (bytes32) {
        return store.getPrevAssertionId(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function firstRival(bytes32 edgeId) public view returns (bytes32) {
        return store.firstRivals[edgeId];
    }
}
