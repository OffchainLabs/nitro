//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ExecutionChallenge.sol";
import "./IExecutionChallengeFactory.sol";
import "./BlockChallenge.sol";
import "./IBlockChallengeFactory.sol";

import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ChallengeManager is IExecutionChallengeFactory, IBlockChallengeFactory {
    IOneStepProofEntry public osp;
    UpgradeableBeacon public executionBeacon;
    UpgradeableBeacon public blockBeacon;

    constructor(IOneStepProofEntry osp_) {
        // TODO: does the challenge manager need to be behind a proxy in case there is a need to upgrade it?
        // Instead, migrating to a new `challengeFactory` in the rollup might work.
        // For ongoing challenges, the admin can `forceResolveChallenge` if need be.
		osp = osp_;
        address execChallengeTemplate = address(new ExecutionChallenge());
        executionBeacon = new UpgradeableBeacon(execChallengeTemplate);
        executionBeacon.transferOwnership(msg.sender);

        address blockChallengeTemplate = address(new BlockChallenge());
        blockBeacon = new UpgradeableBeacon(blockChallengeTemplate);
        blockBeacon.transferOwnership(msg.sender);
    }

    function createBlockChallenge(
        ChallengeContracts calldata contractAddresses,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] calldata startAndEndMachineStatuses_,
        GlobalState[2] calldata startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external override returns (IChallenge) {
        address clone = address(new BeaconProxy(address(blockBeacon), ""));
        BlockChallenge(clone).initialize(
            IExecutionChallengeFactory(this),
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
        return IChallenge(clone);
    }

    function createExecChallenge(
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 challenge_length_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external override returns (IChallenge) {
        address clone = address(new BeaconProxy(address(executionBeacon), ""));
        ExecutionChallenge(clone).initialize(
            osp,
            resultReceiver_,
            execCtx_,
            startAndEndHashes,
            challenge_length_,
            asserter_,
            challenger_,
            asserterTimeLeft_,
            challengerTimeLeft_
        );
        return IChallenge(clone);
    }
}
