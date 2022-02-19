//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ExecutionChallenge.sol";
import "./BlockChallenge.sol";
import "./ChallengeCore.sol";
import "./IChallengeManager.sol";

import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ChallengeManager is IChallengeManager {
    using BlockChallengeLib for BlockChallengeState;
    using ChallengeCoreLib for BisectableChallengeState;

    IOneStepProofEntry public override osp;

    enum ChallengeTrackerState {
        PendingBlockChallenge,
        PendingExecutionChallenge,
        Complete
    }

    // TODO: bubble up asserter/challenger and only store it once instead of twice
    struct ChallengeTracker {
        ChallengeTrackerState trackerState;
        BlockChallengeState blockChallState;
        ExecutionChallengeState execChallState;
        IChallengeResultReceiver resultReceiver;
        IOneStepProofEntry osp;
        ChallengeWinner winner;
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
    function createChallenge(
        ChallengeContracts calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external override returns (uint256) {
        // we never use chall id of 0 to avoid mistakes with mapping default value
        uint256 currChallId = challengeCounter + 1;
        challengeCounter = currChallId;
        ChallengeTracker storage currTrckr = challenges[currChallId];
        BlockChallengeLib.createBlockChallenge(
            currTrckr.blockChallState,
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
        currTrckr.resultReceiver = contractAddresses.resultReceiver;
        currTrckr.trackerState = ChallengeTrackerState.PendingBlockChallenge;
        currTrckr.osp = osp;
        currTrckr.winner = ChallengeWinner.NoWinner;
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
        
        ChallengeWinner winner = bisectionState.timeout();
        
        bisectionState.turn = Turn.NO_CHALLENGE;
        challenges[challengeId].winner = winner;
        challenges[challengeId].trackerState = ChallengeTrackerState.Complete;
    }

    function getCurrentBisectionState(uint256 challengeId)
        internal
        view
        returns (BisectableChallengeState storage)
    {
        ChallengeTracker storage currTrckr = challenges[challengeId];
        ChallengeTrackerState currState = currTrckr.trackerState;
        require(currState != ChallengeTrackerState.Complete, "CHALLENGE_ALREADY_COMPLETE");
        
        return
            currState == ChallengeTrackerState.PendingBlockChallenge
            ? currTrckr.blockChallState.bisectionState
            : currTrckr.execChallState.bisectionState;
    }

    function challengeWinner(uint256 challengeId) external view override returns (address winner) {
        ChallengeTracker storage currTrckr = challenges[challengeId];
        require(currTrckr.trackerState == ChallengeTrackerState.Complete, "NOT_COMPLETE");
        return
            currTrckr.winner == ChallengeWinner.AsserterWin
            ? currTrckr.blockChallState.bisectionState.asserter
            : currTrckr.blockChallState.bisectionState.challenger;
    }

    function clearChallenge(uint256 challengeId) external override {
        ChallengeTracker storage currTrckr = challenges[challengeId];
        require(msg.sender == address(currTrckr.resultReceiver), "NOT_RES_RECEIVER");
        delete challenges[challengeId];
    }
}
