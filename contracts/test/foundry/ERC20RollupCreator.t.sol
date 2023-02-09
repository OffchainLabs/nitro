// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "./AbsRollupCreator.t.sol";
import "../../src/rollup/ERC20RollupCreator.sol";
import "../../src/rollup/ERC20BridgeCreator.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract ERC20RollupCreatorTest is AbsRollupCreatorTest {
    address public nativeToken;

    function setUp() public {
        vm.prank(deployer);
        nativeToken = address(
            new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this))
        );
    }

    /* solhint-disable func-name-mixedcase */
    function test_createRollup() public {
        vm.startPrank(deployer);

        ERC20RollupCreator rollupCreator = new ERC20RollupCreator();

        (
            IOneStepProofEntry ospEntry,
            IChallengeManager challengeManager,
            IRollupAdmin rollupAdmin,
            IRollupUser rollupUser,
            ISequencerInbox.MaxTimeVariation memory timeVars,
            address expectedRollupAddr
        ) = _prepareRollupDeployment(address(rollupCreator));

        //// deployBridgeCreator
        IBridgeCreator bridgeCreator = new ERC20BridgeCreator();

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
                chainId: chainId,
                genesisBlockNum: 15000000,
                sequencerInboxMaxTimeVariation: timeVars
            }),
            expectedRollupAddr,
            nativeToken
        );

        vm.stopPrank();

        /// common checks
        _checkRollupIsSetUp(rollupCreator, rollupAddress, rollupAdmin, rollupUser);

        // native token check
        IBridge bridge = RollupCore(address(rollupAddress)).bridge();
        // assertEq(
        //     IERC20Bridge(address(bridge)).nativeToken(),
        //     nativeToken,
        //     "Invalid native token ref"
        // );
    }
}
