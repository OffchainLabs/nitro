//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./BlockChallenge.sol";
import "./IBlockChallengeFactory.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract BlockChallengeFactory is IBlockChallengeFactory {
    UpgradeableBeacon public beacon;
    IOneStepProofEntry public osp;

    constructor(IOneStepProofEntry osp_) {
        osp = osp_;
        address challengeTemplate = address(new BlockChallenge());
        beacon = new UpgradeableBeacon(challengeTemplate);
        beacon.transferOwnership(msg.sender);
    }

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
    ) external override returns (IChallenge) {
        address clone = address(new BeaconProxy(address(beacon), ""));
        BlockChallenge(clone).initialize(
            osp,
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
        emit ChallengeCreated(IChallenge(clone));
        return IChallenge(clone);
    }
}
