//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ExecutionChallenge.sol";
import "./IExecutionChallengeFactory.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ExecutionChallengeFactory is IExecutionChallengeFactory {
	IOneStepProofEntry public osp;
    UpgradeableBeacon public beacon;

    constructor(IOneStepProofEntry osp_) {
		osp = osp_;
        address challengeTemplate = address(new ExecutionChallenge());
        beacon = new UpgradeableBeacon(challengeTemplate);
        beacon.transferOwnership(msg.sender);
    }

    function createChallenge(
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 challenge_length_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external override returns (IExecutionChallenge) {
        address clone = address(new BeaconProxy(address(beacon), ""));
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
        return IExecutionChallenge(clone);
    }
}
