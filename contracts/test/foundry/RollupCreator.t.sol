// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "./AbsRollupCreator.t.sol";
import "../../src/rollup/BridgeCreator.sol";
import "../../src/rollup/RollupCreator.sol";

contract RollupCreatorTest is AbsRollupCreatorTest {
    function setUp() public {}

    /* solhint-disable func-name-mixedcase */

    function test_createRollup() public {
        vm.startPrank(deployer);

        RollupCreator rollupCreator = new RollupCreator();

        (
            IOneStepProofEntry ospEntry,
            IChallengeManager challengeManager,
            IRollupAdmin rollupAdmin,
            IRollupUser rollupUser,
            ISequencerInbox.MaxTimeVariation memory timeVars,
            address expectedRollupAddr
        ) = _prepareRollupDeployment(address(rollupCreator));
        //// deployBridgeCreator
        IBridgeCreator bridgeCreator = new BridgeCreator();

        //// deploy creator and set logic
        rollupCreator.setTemplates(
            bridgeCreator,
            ospEntry,
            challengeManager,
            rollupAdmin,
            rollupUser,
            address(new ValidatorUtils()),
            address(new ValidatorWalletCreator())
        );

        // deployment params
        bytes32 wasmModuleRoot = keccak256("wasm");
        uint256 chainId = 1337;

        // expect deployment events
        _expectEvents(rollupCreator, bridgeCreator, expectedRollupAddr, wasmModuleRoot, chainId);

        /// deploy rollup
        address rollupAddress = rollupCreator.createRollup(
            Config({
                confirmPeriodBlocks: 20,
                extraChallengeTimeBlocks: 200,
                stakeToken: address(0),
                baseStake: 1000,
                wasmModuleRoot: keccak256("0"),
                owner: rollupOwner,
                loserStakeEscrow: address(200),
                chainId: 1337,
                genesisBlockNum: 15000000,
                sequencerInboxMaxTimeVariation: timeVars
            }),
            expectedRollupAddr
        );

        vm.stopPrank();

        /// common checks
        _checkRollupIsSetUp(rollupCreator, rollupAddress, rollupAdmin, rollupUser);
    }
}
