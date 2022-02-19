//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ExecutionChallenge.sol";
import "./IExecutionChallengeFactory.sol";
import "./BlockChallenge.sol";
import "./IBlockChallengeFactory.sol";
import "./ChallengeCore.sol";

import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ChallengeManager {
    using BlockChallengeLib for BlockChallengeState;
    using ChallengeCoreLib for BisectableChallengeState;

    IOneStepProofEntry public osp;

    // TODO: is complete state useful? we delete when complete
    enum ChallengeTrackerState {
        PendingBlockChallenge,
        PendingExecutionChallenge,
        Complete
    }

    struct ChallengeTracker {
        ChallengeTrackerState trackerState;
        BlockChallengeState blockChallState;
        ExecutionChallengeState execChallState;
        IChallengeResultReceiver resultReceiver;
        IOneStepProofEntry osp;
    }
    uint256 challengeCounter;
    mapping(uint256 => ChallengeTracker) public challenges;

    constructor(IOneStepProofEntry osp_) {
        // TODO: does the challenge manager need to be behind a proxy in case there is a need to upgrade it?
        // Instead, migrating to a new `challengeFactory` in the rollup might work.
        // For ongoing challenges, the admin can `forceResolveChallenge` if need be.
		osp = osp_;
    }

    /// @dev this is called by the rollup
    function createBlockChallenge(
        IBlockChallengeFactory.ChallengeContracts calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (uint256) {
        uint256 currChallId = challengeCounter;
        ChallengeTracker storage newChall = challenges[currChallId];
        newChall.blockChallState.createBlockChallenge(
            contractAddresses,
            wasmModuleRoot_,
            startAndEndMachineStatuses_,
            startAndEndGlobalStates_,
            numBlocks,
            asserter_,
            challenger_,
            asserterTimeLeft_,
            challengerTimeLeft_
        );
        newChall.resultReceiver = contractAddresses.resultReceiver;
        newChall.trackerState = ChallengeTrackerState.PendingBlockChallenge;
        newChall.osp = osp;
        challengeCounter = currChallId++;
        // TODO: should we emit an event here?
        return currChallId;
    }

    function getStartGlobalState(uint256 challengeId) external view returns (GlobalState memory) {
        return challenges[challengeId].blockChallState.startAndEndGlobalStates[0];
    }

    function getEndGlobalState(uint256 challengeId) external view returns (GlobalState memory) {
        return challenges[challengeId].blockChallState.startAndEndGlobalStates[1];
    }

    function getStartAndEndGlobalStates(uint256 challengeId) external view returns (GlobalState[2] memory) {
        return challenges[challengeId].blockChallState.startAndEndGlobalStates;
    }

    function challengeExecution(
        uint256 challengeId,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        MachineStatus[2] calldata machineStatuses,
        bytes32[2] calldata globalStateHashes,
        uint256 numSteps
    ) external {
        BlockChallengeLib.challengeExecution(
            challenges,
            challengeId,
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition,
            machineStatuses,
            globalStateHashes,
            numSteps
        );
    }

    function oneStepProveExecution(
        uint256 challengeId,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes calldata proof
    ) external {
        ExecutionChallengeLib.oneStepProveExecution(
            challenges,
            challengeId,
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition,
            proof
        );
    }

    /**
     * @notice Initiate the next round in the bisection by objecting to execution correctness with a bisection
     * of an execution segment with the same length but a different endpoint. This is either the initial move
     * or follows another execution objection
     */
    function bisectExecution(
        uint256 challengeId,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes32[] calldata newSegments
    ) external {
        BisectableChallengeState storage bisectionState = getCurrentBisectionState(challengeId);
        bisectionState.bisectExecution(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition,
            newSegments
        );
    }

    function timeout(uint256 challengeId) external {
        BisectableChallengeState storage bisectionState = getCurrentBisectionState(challengeId);
        bisectionState.timeout();
    }

    function getCurrentBisectionState(uint256 challengeId)
        internal
        view
        returns (BisectableChallengeState storage)
    {
        ChallengeManager.ChallengeTracker storage currTrckr = challenges[challengeId];
        ChallengeTrackerState currState = currTrckr.trackerState;
        require(currState != ChallengeTrackerState.Complete, "CHALLENGE_ALREADY_COMPLETE");
        
        return
            currState == ChallengeTrackerState.PendingBlockChallenge
            ? currTrckr.blockChallState.bisectionState
            : currTrckr.execChallState.bisectionState;
    }

    function clearChallenge(uint256 challengeId) external {
        require(msg.sender == address(challenges[challengeId].resultReceiver), "NOT_RES_RECEIVER");
        delete challenges[challengeId];
    }

}
