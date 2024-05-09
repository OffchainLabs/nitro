// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";

import "../../src/rollup/RollupProxy.sol";

import "../../src/rollup/RollupCore.sol";
import "../../src/rollup/RollupUserLogic.sol";
import "../../src/rollup/RollupAdminLogic.sol";
import "../../src/rollup/RollupCreator.sol";

import "../../src/osp/OneStepProver0.sol";
import "../../src/osp/OneStepProverMemory.sol";
import "../../src/osp/OneStepProverMath.sol";
import "../../src/osp/OneStepProverHostIo.sol";
import "../../src/osp/OneStepProofEntry.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
import "../challengeV2/Utils.sol";

import "../../src/libraries/Error.sol";

import "../../src/mocks/TestWETH9.sol";
import "../../src/mocks/UpgradeExecutorMock.sol";


import "../../src/assertionStakingPool/AssertionStakingPool.sol";
import "../../src/assertionStakingPool/AssertionStakingPoolCreator.sol";

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

contract AssertionPoolTest is Test {
    address constant owner = address(1337);
    address constant sequencer = address(7331);

    bytes32 constant WASM_MODULE_ROOT = keccak256("WASM_MODULE_ROOT");
    uint256 constant BASE_STAKE = 10 ether;
    uint64 constant CONFIRM_PERIOD_BLOCKS = 100;
    uint256 constant MAX_DATA_SIZE = 117964;
    uint64 constant CHALLENGE_GRACE_PERIOD_BLOCKS = 10;

    bytes32 constant FIRST_ASSERTION_BLOCKHASH = keccak256("FIRST_ASSERTION_BLOCKHASH");
    bytes32 constant FIRST_ASSERTION_SENDROOT = keccak256("FIRST_ASSERTION_SENDROOT");

    IERC20 token;
    RollupUserLogic userRollup;
    RollupAdminLogic adminRollup;
    EdgeChallengeManager challengeManager;

    GlobalState emptyGlobalState;
    AssertionState emptyAssertionState = AssertionState(emptyGlobalState, MachineStatus.FINISHED, bytes32(0));
    bytes32 genesisHash = RollupLib.assertionHash({
        parentAssertionHash: bytes32(0),
        afterState: emptyAssertionState,
        inboxAcc: bytes32(0)
    });
    AssertionState firstState;

    AssertionStakingPool pool;

    AssertionStakingPoolCreator aspcreator;
    address staker1 = address(4000001);
    address staker2 = address(4000002);
    address excessStaker = address(4000003);
    address fullStaker = address(4000004);

    address rando = address(4000005);

    uint256 staker1Bal = 6 ether;
    uint256 staker2Bal = 4 ether;
    uint256 fullStakerBal = 10 ether;
    uint256 excessStakerBal = 1 ether;

    address rollupAddr;
    AssertionInputs assertionInputs;
    bytes32 assertionHash;
    AssertionState afterState;
    uint64 inboxcount;
    address upgradeExecutorAddr;

    event RollupCreated(
        address indexed rollupAddress,
        address indexed nativeToken,
        address inboxAddress,
        address outbox,
        address rollupEventInbox,
        address challengeManager,
        address adminProxy,
        address sequencerInbox,
        address bridge,
        address upgradeExecutor,
        address validatorWalletCreator
    );

    IReader4844 dummyReader4844 = IReader4844(address(137));
    BridgeCreator.BridgeTemplates ethBasedTemplates =
        BridgeCreator.BridgeTemplates({
            bridge: new Bridge(),
            sequencerInbox: new SequencerInbox(MAX_DATA_SIZE, dummyReader4844, false, false),
            delayBufferableSequencerInbox: new SequencerInbox(MAX_DATA_SIZE, dummyReader4844, false, true),
            inbox: new Inbox(MAX_DATA_SIZE),
            rollupEventInbox: new RollupEventInbox(),
            outbox: new Outbox()
        });
    BridgeCreator.BridgeTemplates erc20BasedTemplates =
        BridgeCreator.BridgeTemplates({
            bridge: new ERC20Bridge(),
            sequencerInbox: new SequencerInbox(MAX_DATA_SIZE, dummyReader4844, true, false),
            delayBufferableSequencerInbox: new SequencerInbox(MAX_DATA_SIZE, dummyReader4844, true, false),
            inbox: new ERC20Inbox(MAX_DATA_SIZE),
            rollupEventInbox: new ERC20RollupEventInbox(),
            outbox: new ERC20Outbox()
        });

    function setUp() public {
        OneStepProver0 oneStepProver = new OneStepProver0();
        OneStepProverMemory oneStepProverMemory = new OneStepProverMemory();
        OneStepProverMath oneStepProverMath = new OneStepProverMath();
        OneStepProverHostIo oneStepProverHostIo = new OneStepProverHostIo();
        OneStepProofEntry oneStepProofEntry = new OneStepProofEntry(
            oneStepProver,
            oneStepProverMemory,
            oneStepProverMath,
            oneStepProverHostIo
        );
        EdgeChallengeManager edgeChallengeManager = new EdgeChallengeManager();
        BridgeCreator bridgeCreator = new BridgeCreator(ethBasedTemplates, erc20BasedTemplates);
        RollupCreator rollupCreator = new RollupCreator();
        RollupAdminLogic rollupAdminLogicImpl = new RollupAdminLogic();
        RollupUserLogic rollupUserLogicImpl = new RollupUserLogic();
        DeployHelper deployHelper = new DeployHelper();
        IUpgradeExecutor upgradeExecutorLogic = new UpgradeExecutorMock();

        rollupCreator.setTemplates(
            bridgeCreator,
            oneStepProofEntry,
            edgeChallengeManager,
            rollupAdminLogicImpl,
            rollupUserLogicImpl,
            upgradeExecutorLogic,
            address(0),
            deployHelper
        );
        AssertionState memory emptyState =
            AssertionState(GlobalState([bytes32(0), bytes32(0)], [uint64(0), uint64(0)]), MachineStatus.FINISHED, bytes32(0));
        token = new TestWETH9("Test", "TEST");
        IWETH9(address(token)).deposit{value: 21 ether}();

        uint256[] memory miniStakeValues = new uint256[](4);
        miniStakeValues[0] = 1 ether;
        miniStakeValues[1] = 2 ether;
        miniStakeValues[2] = 3 ether;
        miniStakeValues[3] = 4 ether;
        Config memory config = Config({
            baseStake: BASE_STAKE,
            chainId: 0,
            chainConfig: "{}",
            confirmPeriodBlocks: uint64(CONFIRM_PERIOD_BLOCKS),
            owner: owner,
            sequencerInboxMaxTimeVariation: ISequencerInbox.MaxTimeVariation({
                delayBlocks: (60 * 60 * 24) / 15,
                futureBlocks: 12,
                delaySeconds: 60 * 60 * 24,
                futureSeconds: 60 * 60
            }),
            stakeToken: address(token),
            wasmModuleRoot: WASM_MODULE_ROOT,
            loserStakeEscrow: address(200001),
            genesisAssertionState: emptyState,
            genesisInboxCount: 0,
            miniStakeValues: miniStakeValues,
            layerZeroBlockEdgeHeight: 2 ** 5,
            layerZeroBigStepEdgeHeight: 2 ** 5,
            layerZeroSmallStepEdgeHeight: 2 ** 5,
            numBigStepLevel: 2,
            anyTrustFastConfirmer: address(300001),
            challengeGracePeriodBlocks: CHALLENGE_GRACE_PERIOD_BLOCKS,
            bufferConfig: BufferConfig({threshold: 600, max: 14400, replenishRateInBasis: 500})
        });

        vm.expectEmit(false, false, false, false);
        emit RollupCreated(address(0), address(0), address(0), address(0), address(0), 
                            address(0), address(0), address(0), address(0), address(0), address(0));
        RollupCreator.RollupDeploymentParams memory param = RollupCreator.RollupDeploymentParams({
            config: config,
            validators: new address[](0),
            maxDataSize: MAX_DATA_SIZE,
            nativeToken: address(0),
            deployFactoriesToL2: false,
            maxFeePerGasForRetryables: 0,
            batchPosters: new address[](0),
            batchPosterManager: address(0)
        });

        rollupAddr = rollupCreator.createRollup(param);
        userRollup = RollupUserLogic(address(rollupAddr));
        adminRollup = RollupAdminLogic(address(rollupAddr));
        challengeManager = EdgeChallengeManager(address(userRollup.challengeManager()));
        upgradeExecutorAddr = userRollup.owner();

        vm.startPrank(upgradeExecutorAddr);
        adminRollup.sequencerInbox().setIsBatchPoster(sequencer, true);
        vm.stopPrank();

        firstState.machineStatus = MachineStatus.FINISHED;
        firstState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        firstState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        firstState.globalState.u64Vals[0] = 1; // inbox count
        firstState.globalState.u64Vals[1] = 0; // pos in msg

        vm.roll(block.number + 75);

        inboxcount = uint64(_createNewBatch());
        AssertionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        assertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        assertionInputs = AssertionInputs({
            beforeStateData: BeforeStateData({
                sequencerBatchAcc: bytes32(0),
                prevPrevAssertionHash: bytes32(0),
                configData: ConfigData({
                    wasmModuleRoot: WASM_MODULE_ROOT,
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
                    nextInboxPosition: afterState.globalState.u64Vals[0]
                })
            }),
            beforeState: beforeState,
            afterState: afterState
        });
        aspcreator = new AssertionStakingPoolCreator();
        pool =
            AssertionStakingPool(aspcreator.createPoolForAssertion(address(rollupAddr), assertionInputs, assertionHash));

        token.transfer(staker1, staker1Bal);
        token.transfer(staker2, staker2Bal);
        token.transfer(fullStaker, fullStakerBal);

        token.transfer(excessStaker, excessStakerBal);

        vm.prank(staker1);
        token.approve(address(pool), type(uint256).max);

        vm.prank(staker2);
        token.approve(address(pool), type(uint256).max);

        vm.prank(fullStaker);
        token.approve(address(pool), type(uint256).max);

        vm.prank(excessStaker);
        token.approve(address(pool), type(uint256).max);

        vm.prank(upgradeExecutorAddr);
        adminRollup.setValidatorWhitelistDisabled(true);
    }

    function _createNewBatch() internal returns (uint256) {
        uint256 count = userRollup.bridge().sequencerMessageCount();
        vm.startPrank(sequencer);
        userRollup.sequencerInbox().addSequencerL2Batch({
            sequenceNumber: count,
            data: "",
            afterDelayedMessagesRead: 1,
            gasRefunder: IGasRefunder(address(0)),
            prevMessageCount: 0,
            newMessageCount: 0
        });
        vm.stopPrank();
        assertEq(userRollup.bridge().sequencerMessageCount(), ++count);
        return count;
    }

    function testGetPool() external {
        assertEq(
            address(pool),
            address(aspcreator.getPool(rollupAddr, assertionInputs, assertionHash)),
            "getPool returns created pool's expected address"
        );
    }

    function testgetRequiredStake() external {
        assertEq(pool.getRequiredStake(), BASE_STAKE, "required stake set");
    }

    function testOverDeposit() external {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);
        vm.prank(staker2);
        pool.depositIntoPool(staker2Bal);

        vm.startPrank(excessStaker);
        pool.depositIntoPool(excessStakerBal);
        pool.withdrawFromPool();
        vm.stopPrank();
        assertEq(token.balanceOf(excessStaker), excessStakerBal, "excess balance returned");
    }

    function testCanDepositAndWithdrawWhilePending() external {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);
        vm.prank(staker2);
        pool.depositIntoPool(staker2Bal);

        assertEq(token.balanceOf(address(pool)), staker1Bal + staker2Bal, "tokens depositted into pool");
        assertEq(token.balanceOf(address(staker1)), uint256(0), "tokens depositted into pool");
        assertEq(token.balanceOf(address(staker2)), uint256(0), "tokens depositted into pool");

        vm.prank(staker1);
        pool.withdrawFromPool();

        vm.prank(staker2);
        pool.withdrawFromPool();

        assertEq(token.balanceOf(address(pool)), uint256(0), "tokens withdrawn from pool");
        assertEq(token.balanceOf(address(staker1)), staker1Bal, "tokens withdrawn from pool");
        assertEq(token.balanceOf(address(staker2)), staker2Bal, "tokens withdrawn from pool");
    }

    function testCantAssertWithInsufficientStake() external {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);

        vm.expectRevert("ERC20: transfer amount exceeds balance");
        pool.createAssertion();
    }

    function testCantAssertTwice() external {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);
        vm.prank(staker2);
        pool.depositIntoPool(staker2Bal);

        pool.createAssertion();

        vm.expectRevert("ALREADY_STAKED");
        pool.createAssertion();
    }

    function testCantAssertTwiceAfterConfirmed() external {
        _createAndConfirmAssertion();
        pool.makeStakeWithdrawable();
        pool.withdrawStakeBackIntoPool();

        vm.expectRevert("EXPECTED_ASSERTION_SEEN");
        pool.createAssertion();
    }

    function _createAssertion() internal {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);
        vm.prank(staker2);
        pool.depositIntoPool(staker2Bal);

        vm.prank(rando);
        pool.createAssertion();
    }

    function _createAndConfirmAssertion() internal {
        _createAssertion();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        userRollup.confirmAssertion(
            assertionHash,
            genesisHash,
            firstState,
            bytes32(0),
            ConfigData({
                wasmModuleRoot: WASM_MODULE_ROOT,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
                nextInboxPosition: firstState.globalState.u64Vals[0]
            }),
            inboxAccs
        );
    }

    function testCanAssert() external {
        _createAssertion();
        assertEq(token.balanceOf(address(pool)), 0, "stake moved to rollup");
        assertEq(token.balanceOf(address(userRollup)), BASE_STAKE, "stake moved to rollup");
    }

    function testCanDepositInAssertedState() external {
        _createAssertion();
        vm.startPrank(excessStaker);
        pool.depositIntoPool(excessStakerBal);
        pool.withdrawFromPool();
        vm.stopPrank();

        assertEq(token.balanceOf(excessStaker), excessStakerBal, "excess balance returned");
    }

    function testPartialWithdraw() external {
        vm.prank(staker1);
        pool.depositIntoPool(staker1Bal);

        vm.startPrank(fullStaker);
        pool.depositIntoPool(fullStakerBal);
        pool.createAssertion();

        vm.expectRevert("ERC20: transfer amount exceeds balance");
        pool.withdrawFromPool();

        pool.withdrawFromPool(staker1Bal);
        assertEq(token.balanceOf(fullStaker), staker1Bal, "partial stake returned");

        vm.stopPrank();
    }

    function testReturnStake() external {
        _createAndConfirmAssertion();
        vm.prank(rando);
        pool.makeStakeWithdrawable();

        pool.withdrawStakeBackIntoPool();
        assertEq(token.balanceOf(address(pool)), BASE_STAKE, "tokens returned to pool");
        assertEq(token.balanceOf(address(userRollup)), 0, "tokens returned to pool");

        vm.prank(staker1);
        pool.withdrawFromPool();

        vm.prank(staker2);
        pool.withdrawFromPool();

        assertEq(token.balanceOf(address(pool)), 0, "tokens returned to users");
        assertEq(token.balanceOf(staker1), staker1Bal, "tokens returned to users");
        assertEq(token.balanceOf(staker2), staker2Bal, "tokens returned to users");
    }

    function testCantWithdrawTwice() external {
        _createAndConfirmAssertion();
        pool.makeStakeWithdrawable();
        pool.withdrawStakeBackIntoPool();

        vm.startPrank(staker1);
        pool.withdrawFromPool();
        vm.expectRevert(abi.encodeWithSelector(NoBalanceToWithdraw.selector, staker1));
        pool.withdrawFromPool();
        vm.stopPrank();
    }
}
