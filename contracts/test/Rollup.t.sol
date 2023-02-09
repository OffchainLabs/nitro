// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";

import "../src/rollup/RollupProxy.sol";

import "../src/rollup/RollupCore.sol";
import "../src/rollup/RollupUserLogic.sol";
import "../src/rollup/RollupAdminLogic.sol";

contract RollupTest is Test {
    address owner = address(1);
    bytes32 wasmModuleRoot = keccak256("wasmModuleRoot");

    RollupProxy rollup;
    RollupUserLogic userRollup;
    RollupAdminLogic adminRollup;

    function setUp() public {
        Config memory config = Config({
            baseStake: 10,
            chainId: 0,
            confirmPeriodBlocks: 100,
            extraChallengeTimeBlocks: 100,
            owner: owner,
            sequencerInboxMaxTimeVariation: ISequencerInbox.MaxTimeVariation({
                delayBlocks: (60 * 60 * 24) / 15,
                futureBlocks: 12,
                delaySeconds: 60 * 60 * 24,
                futureSeconds: 60 * 60
            }),
            stakeToken: address(0),
            wasmModuleRoot: wasmModuleRoot,
            loserStakeEscrow: address(0),
            genesisBlockNum: 0
        });
        RollupUserLogic userLogic = new RollupUserLogic();
        RollupAdminLogic adminLogic = new RollupAdminLogic();
        ContractDependencies memory connectedContracts = ContractDependencies({
            oldChallengeManager: IOldChallengeManager(address(0)),
            bridge: IBridge(address(0)),
            inbox: IInbox(address(0)),
            outbox: IOutbox(address(0)),
            rollupAdminLogic: IRollupAdmin(adminLogic),
            rollupEventInbox: IRollupEventInbox(address(0)),
            rollupUserLogic: IRollupUser(userLogic),
            sequencerInbox: ISequencerInbox(address(0)),
            validatorUtils: address(0),
            validatorWalletCreator: address(0)
        });
        rollup = new RollupProxy(config, connectedContracts);
        userRollup = RollupUserLogic(address(rollup));
        adminRollup = RollupAdminLogic(address(rollup));

        vm.startPrank(owner);
        adminRollup.setValidatorWhitelistDisabled(true);
        vm.stopPrank();
    }
}
