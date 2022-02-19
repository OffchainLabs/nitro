//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ExecutionChallenge.sol";
import "./IExecutionChallengeFactory.sol";
import "./BlockChallenge.sol";
import "./IBlockChallengeFactory.sol";

import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ChallengeManager {
    using BlockChallengeLib for BlockChallengeState;
    IOneStepProofEntry public osp;

    struct ChallengeTracker {
        BlockChallengeState challengeState;
        IChallengeResultReceiver resultReceiver;
    }
    uint256 challengeCounter;
    mapping(uint256 => ChallengeTracker) public challenges;

    // TODO: flatten execution challenge outside of block challenge? makes it easier to initialise it
    // TODO: expose user functionality in manager
    // TODO: think through challenge counter and different aggregates useful to surface (ie total challenges per user)

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
        newChall.challengeState.createBlockChallenge(
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
        challengeCounter = currChallId++;
        return currChallId;
    }
}
