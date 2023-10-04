// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "../rollup/Assertion.sol";
import "./libraries/UintUtilsLib.sol";
import "./IAssertionChain.sol";
import "./libraries/EdgeChallengeManagerLib.sol";
import "../libraries/Constants.sol";
import "../state/Machine.sol";

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title EdgeChallengeManager interface
interface IEdgeChallengeManager {
    /// @notice Initialize the EdgeChallengeManager. EdgeChallengeManagers are upgradeable
    ///         so use the initializer paradigm
    /// @param _assertionChain              The assertion chain contract
    /// @param _challengePeriodBlocks       The amount of cumulative time an edge must spend unrivaled before it can be confirmed
    ///                                     This should be the censorship period + the cumulative amount of time needed to do any
    ///                                     offchain calculation. We currently estimate around 10 mins for each layer zero edge and 1
    ///                                     one minute for each other edge.
    /// @param _oneStepProofEntry           The one step proof logic
    /// @param layerZeroBlockEdgeHeight     The end height of layer zero edges of type Block
    /// @param layerZeroBigStepEdgeHeight   The end height of layer zero edges of type BigStep
    /// @param layerZeroSmallStepEdgeHeight The end height of layer zero edges of type SmallStep
    /// @param _stakeToken                  The token that stake will be provided in when creating zero layer block edges
    /// @param _stakeAmount                 The amount of stake (in units of stake token) required to create a block edge
    /// @param _excessStakeReceiver         The address that excess stake will be sent to when 2nd+ block edge is created
    /// @param _numBigStepLevel             The number of bigstep levels
    function initialize(
        IAssertionChain _assertionChain,
        uint64 _challengePeriodBlocks,
        IOneStepProofEntry _oneStepProofEntry,
        uint256 layerZeroBlockEdgeHeight,
        uint256 layerZeroBigStepEdgeHeight,
        uint256 layerZeroSmallStepEdgeHeight,
        IERC20 _stakeToken,
        uint256 _stakeAmount,
        address _excessStakeReceiver,
        uint8 _numBigStepLevel
    ) external;

    function challengePeriodBlocks() external view returns (uint64);

    /// @notice The one step proof resolver used to decide between rival SmallStep edges of length 1
    function oneStepProofEntry() external view returns (IOneStepProofEntry);

    /// @notice Performs necessary checks and creates a new layer zero edge
    /// @param args             Edge creation args
    function createLayerZeroEdge(CreateEdgeArgs calldata args) external returns (bytes32);

    /// @notice Bisect an edge. This creates two child edges:
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
    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes calldata prefixProof)
        external
        returns (bytes32, bytes32);

    /// @notice Confirm an edge if both its children are already confirmed
    function confirmEdgeByChildren(bytes32 edgeId) external;

    /// @notice An edge can be confirmed if the total amount of time it and a single chain of its direct ancestors
    ///         has spent unrivaled is greater than the challenge period.
    /// @dev    Edges inherit time from their parents, so the sum of unrivaled timers is compared against the threshold.
    ///         Given that an edge cannot become unrivaled after becoming rivaled, once the threshold is passed
    ///         it will always remain passed. The direct ancestors of an edge are linked by parent-child links for edges
    ///         of the same level, and claimId-edgeId links for zero layer edges that claim an edge in the level below.
    ///         This method also includes the amount of time the assertion being claimed spent without a sibling
    /// @param edgeId                   The id of the edge to confirm
    /// @param ancestorEdgeIds          The ids of the direct ancestors of an edge. These are ordered from the parent first, then going to grand-parent,
    ///                                 great-grandparent etc. The chain can extend only as far as the zero layer edge of type Block.
    function confirmEdgeByTime(
        bytes32 edgeId,
        bytes32[] calldata ancestorEdgeIds,
        ExecutionStateData calldata claimStateData
    ) external;

    /// @notice If a confirmed edge exists whose claim id is equal to this edge, then this edge can be confirmed
    /// @dev    When zero layer edges are created they reference an edge, or assertion, in the level below. If a zero layer
    ///         edge is confirmed, it becomes possible to also confirm the edge that it claims
    /// @param edgeId           The id of the edge to confirm
    /// @param claimingEdgeId   The id of the edge which has a claimId equal to edgeId
    function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) external;

    /// @notice Confirm an edge by executing a one step proof
    /// @dev    One step proofs can only be executed against edges that have length one and of type SmallStep
    /// @param edgeId                       The id of the edge to confirm
    /// @param oneStepData                  Input data to the one step proof
    /// @param prevConfig                     Data about the config set in prev
    /// @param beforeHistoryInclusionProof  Proof that the state which is the start of the edge is committed to by the startHistoryRoot
    /// @param afterHistoryInclusionProof   Proof that the state which is the end of the edge is committed to by the endHistoryRoot
    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        ConfigData calldata prevConfig,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) external;

    /// @notice When zero layer block edges are created a stake is also provided
    ///         The stake on this edge can be refunded if the edge is confirme
    function refundStake(bytes32 edgeId) external;

    /// @notice Zero layer edges have to be a fixed height.
    ///         This function returns the end height for a given edge type
    function getLayerZeroEndHeight(EdgeType eType) external view returns (uint256);

    /// @notice Calculate the unique id of an edge
    /// @param level            The level of the edge
    /// @param originId         The origin id of the edge
    /// @param startHeight      The start height of the edge
    /// @param startHistoryRoot The start history root of the edge
    /// @param endHeight        The end height of the edge
    /// @param endHistoryRoot   The end history root of the edge
    function calculateEdgeId(
        uint8 level,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) external pure returns (bytes32);

    /// @notice Calculate the mutual id of the edge
    ///         Edges that are rivals share the same mutual id
    /// @param level            The level of the edge
    /// @param originId         The origin id of the edge
    /// @param startHeight      The start height of the edge
    /// @param startHistoryRoot The start history root of the edge
    /// @param endHeight        The end height of the edge
    function calculateMutualId(
        uint8 level,
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
    function timeUnrivaled(bytes32 edgeId) external view returns (uint64);

    /// @notice Get the id of the prev assertion that this edge is originates from
    /// @dev    Uses the parent chain to traverse upwards SmallStep->BigStep->Block->Assertion
    ///         until it gets to the origin assertion
    function getPrevAssertionHash(bytes32 edgeId) external view returns (bytes32);

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
    using SafeERC20 for IERC20;

    /// @notice A new edge has been added to the challenge manager
    /// @param edgeId       The id of the newly added edge
    /// @param mutualId     The mutual id of the added edge - all rivals share the same mutual id
    /// @param originId     The origin id of the added edge - origin ids link an edge to the level below
    /// @param hasRival     Does the newly added edge have a rival upon creation
    /// @param length       The length of the new edge
    /// @param level        The level of the new edge
    /// @param isLayerZero  Whether the new edge was added at layer zero - has a claim and a staker
    event EdgeAdded(
        bytes32 indexed edgeId,
        bytes32 indexed mutualId,
        bytes32 indexed originId,
        bytes32 claimId,
        uint256 length,
        uint8 level,
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

    /// @notice An edge can be confirmed if the cumulative time (in blocks) unrivaled of it and a direct chain of ancestors is greater than a threshold
    /// @param edgeId               The edge that was confirmed
    /// @param mutualId             The mutual id of the confirmed edge
    /// @param totalTimeUnrivaled   The cumulative amount of time (in blocks) this edge spent unrivaled
    event EdgeConfirmedByTime(bytes32 indexed edgeId, bytes32 indexed mutualId, uint64 totalTimeUnrivaled);

    /// @notice An edge can be confirmed if a zero layer edge in the level below claims this edge
    /// @param edgeId           The edge that was confirmed
    /// @param mutualId         The mutual id of the confirmed edge
    /// @param claimingEdgeId   The id of the zero layer edge that claimed this edge
    event EdgeConfirmedByClaim(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 claimingEdgeId);

    /// @notice A SmallStep edge of length 1 can be confirmed via a one step proof
    /// @param edgeId   The edge that was confirmed
    /// @param mutualId The mutual id of the confirmed edge
    event EdgeConfirmedByOneStepProof(bytes32 indexed edgeId, bytes32 indexed mutualId);

    /// @notice A stake has been refunded for a confirmed layer zero block edge
    /// @param edgeId       The edge that was confirmed
    /// @param mutualId     The mutual id of the confirmed edge
    /// @param stakeToken   The ERC20 being refunded
    /// @param stakeAmount  The amount of tokens being refunded
    event EdgeRefunded(bytes32 indexed edgeId, bytes32 indexed mutualId, address stakeToken, uint256 stakeAmount);

    /// @dev Store for all edges and rival data
    ///      All edges, including edges from different challenges, are stored together in the same store
    ///      Since edge ids include the origin id, which is unique for each challenge, we can be sure that
    ///      edges from different challenges cannot have the same id, and so can be stored in the same store
    EdgeStore internal store;

    /// @notice When creating a zero layer block edge a stake must be supplied. However since we know that only
    ///         one edge in a group of rivals can ever be confirmed, we only need to keep one stake in this contract
    ///         to later refund for that edge. Other stakes can immediately be sent to an excess stake receiver.
    ///         This excess stake receiver can then choose to refund the gas of participants who aided in the confirmation
    ///         of the winning edge
    address public excessStakeReceiver;

    /// @notice The token to supply stake in
    IERC20 public stakeToken;

    /// @notice The amount of stake token to be supplied when creating a zero layer block edge
    uint256 public stakeAmount;

    /// @notice The number of blocks accumulated on an edge before it can be confirmed by time
    uint64 public challengePeriodBlocks;

    /// @notice The assertion chain about which challenges are created
    IAssertionChain public assertionChain;

    /// @inheritdoc IEdgeChallengeManager
    IOneStepProofEntry public override oneStepProofEntry;

    /// @notice The end height of layer zero Block edges
    uint256 public LAYERZERO_BLOCKEDGE_HEIGHT;
    /// @notice The end height of layer zero BigStep edges
    uint256 public LAYERZERO_BIGSTEPEDGE_HEIGHT;
    /// @notice The end height of layer zero SmallStep edges
    uint256 public LAYERZERO_SMALLSTEPEDGE_HEIGHT;
    /// @notice The number of big step levels configured for this challenge manager
    ///         There is 1 block level, 1 small step level and N big step levels
    uint8 public NUM_BIGSTEP_LEVEL;

    constructor() {
        _disableInitializers();
    }

    /// @inheritdoc IEdgeChallengeManager
    function initialize(
        IAssertionChain _assertionChain,
        uint64 _challengePeriodBlocks,
        IOneStepProofEntry _oneStepProofEntry,
        uint256 layerZeroBlockEdgeHeight,
        uint256 layerZeroBigStepEdgeHeight,
        uint256 layerZeroSmallStepEdgeHeight,
        IERC20 _stakeToken,
        uint256 _stakeAmount,
        address _excessStakeReceiver,
        uint8 _numBigStepLevel
    ) public initializer {
        if (address(_assertionChain) == address(0)) {
            revert EmptyAssertionChain();
        }
        assertionChain = _assertionChain;
        if (address(_oneStepProofEntry) == address(0)) {
            revert EmptyOneStepProofEntry();
        }
        oneStepProofEntry = _oneStepProofEntry;
        if (_challengePeriodBlocks == 0) {
            revert EmptyChallengePeriod();
        }
        challengePeriodBlocks = _challengePeriodBlocks;

        stakeToken = _stakeToken;
        stakeAmount = _stakeAmount;
        if (_excessStakeReceiver == address(0)) {
            revert EmptyStakeReceiver();
        }
        excessStakeReceiver = _excessStakeReceiver;

        if (!EdgeChallengeManagerLib.isPowerOfTwo(layerZeroBlockEdgeHeight)) {
            revert NotPowerOfTwo(layerZeroBlockEdgeHeight);
        }
        LAYERZERO_BLOCKEDGE_HEIGHT = layerZeroBlockEdgeHeight;
        if (!EdgeChallengeManagerLib.isPowerOfTwo(layerZeroBigStepEdgeHeight)) {
            revert NotPowerOfTwo(layerZeroBigStepEdgeHeight);
        }
        LAYERZERO_BIGSTEPEDGE_HEIGHT = layerZeroBigStepEdgeHeight;
        if (!EdgeChallengeManagerLib.isPowerOfTwo(layerZeroSmallStepEdgeHeight)) {
            revert NotPowerOfTwo(layerZeroSmallStepEdgeHeight);
        }
        LAYERZERO_SMALLSTEPEDGE_HEIGHT = layerZeroSmallStepEdgeHeight;

        // ensure that there is at least on of each type of level
        if (_numBigStepLevel == 0) {
            revert ZeroBigStepLevels();
        }
        // ensure there's also space for the block level and the small step level
        // in total level parameters
        if (_numBigStepLevel > 253) {
            revert BigStepLevelsTooMany(_numBigStepLevel);
        }
        NUM_BIGSTEP_LEVEL = _numBigStepLevel;
    }

    /////////////////////////////
    // STATE MUTATING SECTIION //
    /////////////////////////////

    /// @inheritdoc IEdgeChallengeManager
    function createLayerZeroEdge(CreateEdgeArgs calldata args) external returns (bytes32) {
        EdgeAddedData memory edgeAdded;
        EdgeType eType = ChallengeEdgeLib.levelToType(args.level, NUM_BIGSTEP_LEVEL);
        uint256 expectedEndHeight = getLayerZeroEndHeight(eType);
        AssertionReferenceData memory ard;

        if (eType == EdgeType.Block) {
            // for block type edges we need to provide some extra assertion data context
            if (args.proof.length == 0) {
                revert EmptyEdgeSpecificProof();
            }
            (, ExecutionStateData memory predecessorStateData, ExecutionStateData memory claimStateData) =
                abi.decode(args.proof, (bytes32[], ExecutionStateData, ExecutionStateData));

            assertionChain.validateAssertionHash(
                args.claimId, claimStateData.executionState, claimStateData.prevAssertionHash, claimStateData.inboxAcc
            );

            assertionChain.validateAssertionHash(
                claimStateData.prevAssertionHash,
                predecessorStateData.executionState,
                predecessorStateData.prevAssertionHash,
                predecessorStateData.inboxAcc
            );

            ard = AssertionReferenceData(
                args.claimId,
                claimStateData.prevAssertionHash,
                assertionChain.isPending(args.claimId),
                assertionChain.getSecondChildCreationBlock(claimStateData.prevAssertionHash) > 0,
                predecessorStateData.executionState,
                claimStateData.executionState
            );

            edgeAdded = store.createLayerZeroEdge(args, ard, oneStepProofEntry, expectedEndHeight, NUM_BIGSTEP_LEVEL);
        } else {
            edgeAdded = store.createLayerZeroEdge(args, ard, oneStepProofEntry, expectedEndHeight, NUM_BIGSTEP_LEVEL);
        }

        IERC20 st = stakeToken;
        uint256 sa = stakeAmount;
        // when a zero layer edge is created it must include stake amount. Each time a zero layer
        // edge is created it forces the honest participants to do some work, so we want to disincentive
        // their creation. The amount should also be enough to pay for the gas costs incurred by the honest
        // participant. This can be arranged out of bound by the excess stake receiver.
        // The contract initializer can disable staking by setting zeros for token or amount, to change
        // this a new challenge manager needs to be deployed and its address updated in the assertion chain
        if (address(st) != address(0) && sa != 0) {
            // since only one edge in a group of rivals can ever be confirmed, we know that we
            // will never need to refund more than one edge. Therefore we can immediately send
            // all stakes provided after the first one to an excess stake receiver.
            address receiver = edgeAdded.hasRival ? excessStakeReceiver : address(this);
            st.safeTransferFrom(msg.sender, receiver, sa);
        }

        emit EdgeAdded(
            edgeAdded.edgeId,
            edgeAdded.mutualId,
            edgeAdded.originId,
            edgeAdded.claimId,
            edgeAdded.length,
            edgeAdded.level,
            edgeAdded.hasRival,
            edgeAdded.isLayerZero
        );
        return edgeAdded.edgeId;
    }

    /// @inheritdoc IEdgeChallengeManager
    function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes calldata prefixProof)
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
                lowerChildAdded.level,
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
            upperChildAdded.level,
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
        store.confirmEdgeByClaim(edgeId, claimingEdgeId, NUM_BIGSTEP_LEVEL);

        emit EdgeConfirmedByClaim(edgeId, store.edges[edgeId].mutualId(), claimingEdgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByTime(
        bytes32 edgeId,
        bytes32[] memory ancestorEdges,
        ExecutionStateData calldata claimStateData
    ) public {
        // if there are no ancestors provided, then the top edge is the edge we're confirming itself
        bytes32 lastEdgeId = ancestorEdges.length > 0 ? ancestorEdges[ancestorEdges.length - 1] : edgeId;
        ChallengeEdge storage topEdge = store.get(lastEdgeId);
        EdgeType topLevelType = ChallengeEdgeLib.levelToType(topEdge.level, NUM_BIGSTEP_LEVEL);

        if (topLevelType != EdgeType.Block) {
            revert EdgeTypeNotBlock(topEdge.level);
        }
        if (!topEdge.isLayerZero()) {
            revert EdgeNotLayerZero(topEdge.id(), topEdge.staker, topEdge.claimId);
        }

        uint64 assertionBlocks;
        // if the assertion being claiming against was the first child of its predecessor
        // then we are able to count the time between the first and second child as time towards
        // the this edge
        bool isFirstChild = assertionChain.isFirstChild(topEdge.claimId);
        if (isFirstChild) {
            assertionChain.validateAssertionHash(
                topEdge.claimId,
                claimStateData.executionState,
                claimStateData.prevAssertionHash,
                claimStateData.inboxAcc
            );
            assertionBlocks = assertionChain.getSecondChildCreationBlock(claimStateData.prevAssertionHash)
                - assertionChain.getFirstChildCreationBlock(claimStateData.prevAssertionHash);
        } else {
            // if the assertion being claimed is not the first child, then it had siblings from the moment
            // it was created, so it has no time unrivaled
            assertionBlocks = 0;
        }

        uint64 totalTimeUnrivaled =
            store.confirmEdgeByTime(edgeId, ancestorEdges, assertionBlocks, challengePeriodBlocks, NUM_BIGSTEP_LEVEL);

        emit EdgeConfirmedByTime(edgeId, store.edges[edgeId].mutualId(), totalTimeUnrivaled);
    }

    /// @inheritdoc IEdgeChallengeManager
    function confirmEdgeByOneStepProof(
        bytes32 edgeId,
        OneStepData calldata oneStepData,
        ConfigData calldata prevConfig,
        bytes32[] calldata beforeHistoryInclusionProof,
        bytes32[] calldata afterHistoryInclusionProof
    ) public {
        bytes32 prevAssertionHash = store.getPrevAssertionHash(edgeId, NUM_BIGSTEP_LEVEL);

        assertionChain.validateConfig(prevAssertionHash, prevConfig);

        ExecutionContext memory execCtx = ExecutionContext({
            maxInboxMessagesRead: prevConfig.nextInboxPosition,
            bridge: assertionChain.bridge(),
            initialWasmModuleRoot: prevConfig.wasmModuleRoot
        });

        store.confirmEdgeByOneStepProof(
            edgeId,
            oneStepProofEntry,
            oneStepData,
            execCtx,
            beforeHistoryInclusionProof,
            afterHistoryInclusionProof,
            NUM_BIGSTEP_LEVEL
        );

        emit EdgeConfirmedByOneStepProof(edgeId, store.edges[edgeId].mutualId());
    }

    /// @inheritdoc IEdgeChallengeManager
    function refundStake(bytes32 edgeId) public {
        ChallengeEdge storage edge = store.get(edgeId);
        // setting refunded also do checks that the edge cannot be refunded twice
        edge.setRefunded();

        IERC20 st = stakeToken;
        uint256 sa = stakeAmount;
        // no need to refund with the token or amount where zero'd out
        if (address(st) != address(0) && sa != 0) {
            st.safeTransfer(edge.staker, sa);
        }

        emit EdgeRefunded(edgeId, store.edges[edgeId].mutualId(), address(st), sa);
    }

    ///////////////////////
    // VIEW ONLY SECTION //
    ///////////////////////

    /// @inheritdoc IEdgeChallengeManager
    function getLayerZeroEndHeight(EdgeType eType) public view returns (uint256) {
        if (eType == EdgeType.Block) {
            return LAYERZERO_BLOCKEDGE_HEIGHT;
        } else if (eType == EdgeType.BigStep) {
            return LAYERZERO_BIGSTEPEDGE_HEIGHT;
        } else if (eType == EdgeType.SmallStep) {
            return LAYERZERO_SMALLSTEPEDGE_HEIGHT;
        } else {
            revert("Unrecognised edge type");
        }
    }

    /// @inheritdoc IEdgeChallengeManager
    function calculateEdgeId(
        uint8 level,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight,
        bytes32 endHistoryRoot
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.idComponent(level, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot);
    }

    /// @inheritdoc IEdgeChallengeManager
    function calculateMutualId(
        uint8 level,
        bytes32 originId,
        uint256 startHeight,
        bytes32 startHistoryRoot,
        uint256 endHeight
    ) public pure returns (bytes32) {
        return ChallengeEdgeLib.mutualIdComponent(level, originId, startHeight, startHistoryRoot, endHeight);
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
    function timeUnrivaled(bytes32 edgeId) public view returns (uint64) {
        return store.timeUnrivaled(edgeId);
    }

    /// @inheritdoc IEdgeChallengeManager
    function getPrevAssertionHash(bytes32 edgeId) public view returns (bytes32) {
        return store.getPrevAssertionHash(edgeId, NUM_BIGSTEP_LEVEL);
    }

    /// @inheritdoc IEdgeChallengeManager
    function firstRival(bytes32 edgeId) public view returns (bytes32) {
        return store.firstRivals[edgeId];
    }
}
