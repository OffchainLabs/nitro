//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./BlockChallenge.sol";
import "./IBlockChallengeFactory.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract BlockChallengeFactory is IBlockChallengeFactory {
	IExecutionChallengeFactory public executionChallengeFactory;
    UpgradeableBeacon public beacon;

    constructor(IExecutionChallengeFactory executionChallengeFactory_) {
		executionChallengeFactory = executionChallengeFactory_;
        address challengeTemplate = address(new BlockChallenge());
        beacon = new UpgradeableBeacon(challengeTemplate);
        beacon.transferOwnership(msg.sender);
    }

    function createChallenge(
        IChallengeResultReceiver resultReceiver_,
        bytes32 wasmModuleRoot_,
        MachineStatus[2] memory startAndEndMachineStatuses_,
        GlobalState[2] memory startAndEndGlobalStates_,
        uint64 numBlocks,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external override returns (IChallenge) {
        address clone = address(new BeaconProxy(address(beacon), ""));
        BlockChallenge(clone).initialize(
            executionChallengeFactory,
            resultReceiver_,
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
}
