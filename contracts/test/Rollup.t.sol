// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";

import "../src/rollup/RollupProxy.sol";

import "../src/rollup/RollupCore.sol";
import "../src/rollup/RollupUserLogic.sol";
import "../src/rollup/RollupAdminLogic.sol";
import "../src/rollup/RollupCreator.sol";

import "../src/osp/OneStepProver0.sol";
import "../src/osp/OneStepProverMemory.sol";
import "../src/osp/OneStepProverMath.sol";
import "../src/osp/OneStepProverHostIo.sol";
import "../src/osp/OneStepProofEntry.sol";
import "../src/challengeV2/EdgeChallengeManager.sol";
import "./challengeV2/Utils.sol";

import "../src/libraries/Error.sol";

import "../src/mocks/TestWETH9.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

contract RollupTest is Test {
    address constant owner = address(1337);
    address constant sequencer = address(7331);

    address constant validator1 = address(100001);
    address constant validator2 = address(100002);
    address constant validator3 = address(100003);
    address constant loserStakeEscrow = address(200001);
    address constant anyTrustFastConfirmer = address(300001);

    bytes32 constant WASM_MODULE_ROOT = keccak256("WASM_MODULE_ROOT");
    uint256 constant BASE_STAKE = 10;
    uint256 constant MINI_STAKE_VALUE = 2;
    uint64 constant CONFIRM_PERIOD_BLOCKS = 100;

    bytes32 constant FIRST_ASSERTION_BLOCKHASH = keccak256("FIRST_ASSERTION_BLOCKHASH");
    bytes32 constant FIRST_ASSERTION_SENDROOT = keccak256("FIRST_ASSERTION_SENDROOT");

    uint256 constant LAYERZERO_BLOCKEDGE_HEIGHT = 2 ** 5;

    IERC20 token;
    RollupProxy rollup;
    RollupUserLogic userRollup;
    RollupAdminLogic adminRollup;
    EdgeChallengeManager challengeManager;
    Random rand = new Random();

    address[] validators;
    bool[] flags;

    GlobalState emptyGlobalState;
    ExecutionState emptyExecutionState = ExecutionState(emptyGlobalState, MachineStatus.FINISHED);
    bytes32 genesisHash = RollupLib.assertionHash({
        parentAssertionHash: bytes32(0),
        afterState: emptyExecutionState,
        inboxAcc: bytes32(0)
    });
    ExecutionState firstState;

    event RollupCreated(
        address indexed rollupAddress, address inboxAddress, address adminProxy, address sequencerInbox, address bridge
    );

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
        BridgeCreator bridgeCreator = new BridgeCreator();
        RollupCreator rollupCreator = new RollupCreator();
        RollupAdminLogic rollupAdminLogicImpl = new RollupAdminLogic();
        RollupUserLogic rollupUserLogicImpl = new RollupUserLogic();

        rollupCreator.setTemplates(
            bridgeCreator,
            oneStepProofEntry,
            edgeChallengeManager,
            rollupAdminLogicImpl,
            rollupUserLogicImpl,
            address(0)
        );

        ExecutionState memory emptyState =
            ExecutionState(GlobalState([bytes32(0), bytes32(0)], [uint64(0), uint64(0)]), MachineStatus.FINISHED);
        token = new TestWETH9("Test", "TEST");
        IWETH9(address(token)).deposit{value: 10 ether}();

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
            loserStakeEscrow: loserStakeEscrow,
            genesisExecutionState: emptyState,
            genesisInboxCount: 0,
            miniStakeValue: MINI_STAKE_VALUE,
            layerZeroBlockEdgeHeight: 2 ** 5,
            layerZeroBigStepEdgeHeight: 2 ** 5,
            layerZeroSmallStepEdgeHeight: 2 ** 5,
            anyTrustFastConfirmer: anyTrustFastConfirmer
        });

        vm.expectEmit(false, false, false, false);
        emit RollupCreated(address(0), address(0), address(0), address(0), address(0));
        address rollupAddr = rollupCreator.createRollup(config);

        userRollup = RollupUserLogic(address(rollupAddr));
        adminRollup = RollupAdminLogic(address(rollupAddr));
        challengeManager = EdgeChallengeManager(address(userRollup.challengeManager()));

        vm.startPrank(owner);
        validators.push(validator1);
        validators.push(validator2);
        validators.push(validator3);
        flags.push(true);
        flags.push(true);
        flags.push(true);
        adminRollup.setValidator(address[](validators), flags);
        adminRollup.sequencerInbox().setIsBatchPoster(sequencer, true);
        vm.stopPrank();

        firstState.machineStatus = MachineStatus.FINISHED;
        firstState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        firstState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        firstState.globalState.u64Vals[0] = 1; // inbox count
        firstState.globalState.u64Vals[1] = 0; // pos in msg

        // TODO: determine if challengeManager should be permissionless at the stage
        token.approve(address(challengeManager), type(uint256).max);

        token.transfer(validator1, 1 ether);
        vm.deal(validator1, 1 ether);
        vm.prank(validator1);
        token.approve(address(userRollup), type(uint256).max);
        vm.prank(validator1);
        token.approve(address(challengeManager), type(uint256).max);

        token.transfer(validator2, 1 ether);
        vm.deal(validator2, 1 ether);
        vm.prank(validator2);
        token.approve(address(userRollup), type(uint256).max);
        vm.prank(validator2);
        token.approve(address(challengeManager), type(uint256).max);

        token.transfer(validator3, 1 ether);
        vm.deal(validator3, 1 ether);
        vm.prank(validator3);
        token.approve(address(userRollup), type(uint256).max);
        vm.prank(validator3);
        token.approve(address(challengeManager), type(uint256).max);

        vm.deal(sequencer, 1 ether);

        vm.roll(block.number + 75);
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

    function testGenesisAssertionConfirmed() external {
        bytes32 latestConfirmed = userRollup.latestConfirmed();
        assertEq(latestConfirmed, genesisHash);
        assertEq(userRollup.getAssertion(latestConfirmed).status == AssertionStatus.Confirmed, true);
    }

    function testSuccessPause() public {
        vm.prank(owner);
        adminRollup.pause();
    }

    function testConfirmAssertionWhenPaused() public {
        (bytes32 assertionHash, ExecutionState memory state, uint64 inboxcount) = testSuccessCreateAssertion();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(owner);
        adminRollup.pause();
        vm.prank(validator1);
        vm.expectRevert("Pausable: paused");
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

    function testSuccessPauseResume() public {
        testSuccessPause();
        vm.prank(owner);
        adminRollup.resume();
    }

    function testSuccessOwner() public {
        assertEq(userRollup.owner(), owner);
    }

    function testSuccessRemoveWhitelistAfterFork() public {
        vm.chainId(313377);
        userRollup.removeWhitelistAfterFork();
    }

    function testRevertRemoveWhitelistAfterFork() public {
        vm.expectRevert("CHAIN_ID_NOT_CHANGED");
        userRollup.removeWhitelistAfterFork();
    }

    function testRevertRemoveWhitelistAfterForkAgain() public {
        testSuccessRemoveWhitelistAfterFork();
        vm.expectRevert("WHITELIST_DISABLED");
        userRollup.removeWhitelistAfterFork();
    }

    function testSuccessCreateAssertion() public returns (bytes32, ExecutionState memory, uint64) {
        uint64 inboxcount = uint64(_createNewBatch());
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        bytes32 expectedAssertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: expectedAssertionHash
        });

        return (expectedAssertionHash, afterState, inboxcount);
    }

    function testSuccessGetStaker() public {
        assertEq(userRollup.stakerCount(), 0);
        testSuccessCreateAssertion();
        assertEq(userRollup.stakerCount(), 1);
        assertEq(userRollup.getStakerAddress(userRollup.getStaker(validator1).index), validator1);
    }

    function testSuccessCreateErroredAssertions() public returns (bytes32, ExecutionState memory, uint64) {
        uint64 inboxcount = uint64(_createNewBatch());
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.ERRORED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        bytes32 expectedAssertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: expectedAssertionHash
        });

        return (expectedAssertionHash, afterState, inboxcount);
    }

    function testRevertIdenticalAssertions() public {
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: bytes32(0)
        });

        vm.prank(validator2);
        vm.expectRevert("ASSERTION_SEEN");
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: bytes32(0)
        });
    }

    function testRevertInvalidPrev() public {
        uint64 inboxcount = uint64(_createNewBatch());
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        bytes32 expectedAssertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: expectedAssertionHash
        });

        ExecutionState memory afterState2;
        afterState2.machineStatus = MachineStatus.FINISHED;
        afterState2.globalState.u64Vals[0] = inboxcount;
        bytes32 expectedAssertionHash2 = RollupLib.assertionHash({
            parentAssertionHash: expectedAssertionHash,
            afterState: afterState2,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(1) // 1 because we moved the position within message
        });
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);

        // set the wrong before state
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_SENDROOT;

        vm.roll(block.number + 75);
        vm.prank(validator1);
        vm.expectRevert("ASSERTION_NOT_EXIST");
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
                    configData: ConfigData({
                        wasmModuleRoot: WASM_MODULE_ROOT,
                        requiredStake: BASE_STAKE,
                        challengeManager: address(challengeManager),
                        confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
                        nextInboxPosition: afterState2.globalState.u64Vals[0]
                    })
                }),
                beforeState: afterState,
                afterState: afterState2
            }),
            expectedAssertionHash: expectedAssertionHash2
        });
    }

    function testSuccessCreateSecondChild()
        public
        returns (
            ExecutionState memory,
            ExecutionState memory,
            ExecutionState memory,
            uint256,
            uint256,
            bytes32,
            bytes32
        )
    {
        uint256 genesisInboxCount = 1;
        uint64 newInboxCount = uint64(_createNewBatch());
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        bytes32 expectedAssertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
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
            }),
            expectedAssertionHash: expectedAssertionHash
        });

        ExecutionState memory afterState2;
        afterState2.machineStatus = MachineStatus.FINISHED;
        afterState2.globalState.bytes32Vals[0] = keccak256(abi.encodePacked(FIRST_ASSERTION_BLOCKHASH)); // blockhash
        afterState2.globalState.bytes32Vals[1] = keccak256(abi.encodePacked(FIRST_ASSERTION_SENDROOT)); // sendroot
        afterState2.globalState.u64Vals[0] = 1; // inbox count
        afterState2.globalState.u64Vals[1] = 0; // modify the state

        bytes32 expectedAssertionHash2 = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState2,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });
        vm.prank(validator2);
        userRollup.newStakeOnNewAssertion({
            tokenAmount: BASE_STAKE,
            assertion: AssertionInputs({
                beforeState: beforeState,
                beforeStateData: BeforeStateData({
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    configData: ConfigData({
                        wasmModuleRoot: WASM_MODULE_ROOT,
                        requiredStake: BASE_STAKE,
                        challengeManager: address(challengeManager),
                        confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
                        nextInboxPosition: afterState2.globalState.u64Vals[0]
                    })
                }),
                afterState: afterState2
            }),
            expectedAssertionHash: expectedAssertionHash2
        });

        assertEq(userRollup.getAssertion(genesisHash).secondChildBlock, block.number);

        return (
            beforeState,
            afterState,
            afterState2,
            genesisInboxCount,
            newInboxCount,
            expectedAssertionHash,
            expectedAssertionHash2
        );
    }

    function testRevertConfirmWrongInput() public {
        (bytes32 assertionHash1,,) = testSuccessCreateAssertion();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        vm.expectRevert("CONFIRM_DATA");
        userRollup.confirmAssertion(
            assertionHash1,
            genesisHash,
            emptyExecutionState,
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

    function testSuccessConfirmUnchallengedAssertions() public returns (bytes32, ExecutionState memory, uint64) {
        (bytes32 assertionHash, ExecutionState memory state, uint64 inboxcount) = testSuccessCreateAssertion();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
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
        return (assertionHash, state, inboxcount);
    }

    function testSuccessRemoveWhitelistAfterValidatorAfk() public {
        (bytes32 assertionHash,,) = testSuccessConfirmUnchallengedAssertions();
        vm.roll(userRollup.getAssertion(assertionHash).createdAtBlock + userRollup.VALIDATOR_AFK_BLOCKS() + 1);
        userRollup.removeWhitelistAfterValidatorAfk();
    }

    function testRevertRemoveWhitelistAfterValidatorAfk() public {
        vm.expectRevert("VALIDATOR_NOT_AFK");
        userRollup.removeWhitelistAfterValidatorAfk();
    }

    function testRevertConfirmSiblingedAssertions() public {
        (,,,,, bytes32 assertionHash,) = testSuccessCreateSecondChild();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);

        vm.expectRevert(abi.encodeWithSelector(EdgeNotExists.selector, bytes32(0)));
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

    struct SuccessCreateChallengeData {
        ExecutionState beforeState;
        uint256 genesisInboxCount;
        ExecutionState afterState1;
        ExecutionState afterState2;
        uint256 newInboxCount;
        bytes32 e1Id;
        bytes32 assertionHash;
        bytes32 assertionHash2;
    }

    function testSuccessCreateChallenge() public returns (SuccessCreateChallengeData memory data) {
        (
            data.beforeState,
            data.afterState1,
            data.afterState2,
            data.genesisInboxCount,
            data.newInboxCount,
            data.assertionHash,
            data.assertionHash2
        ) = testSuccessCreateSecondChild();

        bytes32[] memory states;
        {
            IOneStepProofEntry osp = userRollup.challengeManager().oneStepProofEntry();
            bytes32 h0 = osp.getMachineHash(data.beforeState);
            bytes32 h1 = osp.getMachineHash(data.afterState1);
            states = fillStatesInBetween(h0, h1, LAYERZERO_BLOCKEDGE_HEIGHT + 1);
        }

        bytes32 root = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, LAYERZERO_BLOCKEDGE_HEIGHT + 1));

        data.e1Id = challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: EdgeType.Block,
                endHistoryRoot: root,
                endHeight: LAYERZERO_BLOCKEDGE_HEIGHT,
                claimId: data.assertionHash,
                prefixProof: abi.encode(
                    ProofUtils.expansionFromLeaves(states, 0, 1),
                    ProofUtils.generatePrefixProof(1, ArrayUtilsLib.slice(states, 1, states.length))
                    ),
                proof: abi.encode(
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1),
                    ExecutionStateData(data.beforeState, bytes32(0), bytes32(0)),
                    ExecutionStateData(data.afterState1, genesisHash, userRollup.bridge().sequencerInboxAccs(0))
                    )
            })
        );
    }

    function testSuccessCreate2Edge() public returns (bytes32, bytes32) {
        SuccessCreateChallengeData memory data = testSuccessCreateChallenge();
        require(data.genesisInboxCount == 1, "A");
        require(data.newInboxCount == 2, "B");

        bytes32[] memory states;
        {
            IOneStepProofEntry osp = userRollup.challengeManager().oneStepProofEntry();
            bytes32 h0 = osp.getMachineHash(data.beforeState);
            bytes32 h1 = osp.getMachineHash(data.afterState2);
            states = fillStatesInBetween(h0, h1, LAYERZERO_BLOCKEDGE_HEIGHT + 1);
        }

        bytes32 root = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, LAYERZERO_BLOCKEDGE_HEIGHT + 1));

        bytes32 e2Id = challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: EdgeType.Block,
                endHistoryRoot: root,
                endHeight: LAYERZERO_BLOCKEDGE_HEIGHT,
                claimId: data.assertionHash2,
                prefixProof: abi.encode(
                    ProofUtils.expansionFromLeaves(states, 0, 1),
                    ProofUtils.generatePrefixProof(1, ArrayUtilsLib.slice(states, 1, states.length))
                    ),
                proof: abi.encode(
                    ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1),
                    ExecutionStateData(data.beforeState, bytes32(0), bytes32(0)),
                    ExecutionStateData(data.afterState2, genesisHash, userRollup.bridge().sequencerInboxAccs(0))
                    )
            })
        );

        return (data.e1Id, e2Id);
    }

    function fillStatesInBetween(bytes32 start, bytes32 end, uint256 totalCount) internal returns (bytes32[] memory) {
        bytes32[] memory innerStates = rand.hashes(totalCount - 2);

        bytes32[] memory states = new bytes32[](totalCount);
        states[0] = start;
        for (uint256 i = 0; i < innerStates.length; i++) {
            states[i + 1] = innerStates[i];
        }
        states[totalCount - 1] = end;

        return states;
    }

    function testSuccessConfirmEdgeByTime() public returns (bytes32) {
        SuccessCreateChallengeData memory data = testSuccessCreateChallenge();

        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        vm.warp(block.timestamp + CONFIRM_PERIOD_BLOCKS * 15);
        userRollup.challengeManager().confirmEdgeByTime(
            data.e1Id,
            new bytes32[](0),
            ExecutionStateData(firstState, genesisHash, userRollup.bridge().sequencerInboxAccs(0))
        );
        bytes32 inboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        userRollup.confirmAssertion(
            data.assertionHash,
            genesisHash,
            firstState,
            data.e1Id,
            ConfigData({
                wasmModuleRoot: WASM_MODULE_ROOT,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
                nextInboxPosition: firstState.globalState.u64Vals[0]
            }),
            inboxAcc
        );
        return data.e1Id;
    }

    function testRevertWithdrawStake() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        vm.expectRevert("NO_FUNDS_TO_WITHDRAW");
        userRollup.withdrawStakerFunds();
    }

    function testSuccessWithdrawStake() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        userRollup.returnOldDeposit();
        assertGt(userRollup.withdrawableFunds(validator1), 0);
        vm.prank(validator1);
        userRollup.withdrawStakerFunds();
    }

    function testRevertWithdrawActiveStake() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator2);
        vm.expectRevert("STAKE_ACTIVE");
        userRollup.returnOldDeposit();
    }

    function testSuccessWithdrawExcessStake() public {
        testSuccessCreateSecondChild();
        vm.prank(loserStakeEscrow);
        userRollup.withdrawStakerFunds();
    }

    function testRevertWithdrawNoExcessStake() public {
        testSuccessCreateAssertion();
        vm.prank(loserStakeEscrow);
        vm.expectRevert("NO_FUNDS_TO_WITHDRAW");
        userRollup.withdrawStakerFunds();
    }

    function testSuccessReduceDeposit() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        userRollup.reduceDeposit(1);
    }

    function testRevertReduceDepositActive() public {
        testSuccessCreateAssertion();
        vm.prank(validator1);
        vm.expectRevert("STAKE_ACTIVE");
        userRollup.reduceDeposit(1);
    }

    function testSuccessAddToDeposit() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        userRollup.addToDeposit(validator1, 1);
    }

    function testRevertAddToDepositNotValidator() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(sequencer);
        vm.expectRevert("NOT_VALIDATOR");
        userRollup.addToDeposit(validator1, 1);
    }

    function testRevertAddToDepositNotStaker() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        vm.expectRevert("NOT_STAKED");
        userRollup.addToDeposit(sequencer, 1);
    }

    function testSuccessCreateSecondAssertion() public returns (bytes32, bytes32, ExecutionState memory, bytes32) {
        (bytes32 prevHash, ExecutionState memory beforeState, uint64 prevInboxCount) = testSuccessCreateAssertion();

        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.u64Vals[0] = prevInboxCount;
        bytes32 inboxAcc = userRollup.bridge().sequencerInboxAccs(1); // 1 because we moved the position within message
        bytes32 expectedAssertionHash2 =
            RollupLib.assertionHash({parentAssertionHash: prevHash, afterState: afterState, inboxAcc: inboxAcc});
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.roll(block.number + 75);
        vm.prank(validator1);
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
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
            }),
            expectedAssertionHash: expectedAssertionHash2
        });
        return (prevHash, expectedAssertionHash2, afterState, inboxAcc);
    }

    function testRevertCreateChildReducedStake() public {
        (bytes32 prevHash, ExecutionState memory beforeState, uint64 prevInboxCount) =
            testSuccessConfirmUnchallengedAssertions();

        vm.prank(validator1);
        userRollup.reduceDeposit(1);

        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.u64Vals[0] = prevInboxCount;
        bytes32 expectedAssertionHash2 = RollupLib.assertionHash({
            parentAssertionHash: prevHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(1) // 1 because we moved the position within message
        });
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.roll(block.number + 75);
        vm.prank(validator1);
        vm.expectRevert("INSUFFICIENT_STAKE");
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
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
            }),
            expectedAssertionHash: expectedAssertionHash2
        });
    }

    function testSuccessFastConfirmNext() public {
        (bytes32 assertionHash,,) = testSuccessCreateAssertion();
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        assertEq(userRollup.latestConfirmed(), genesisHash);
        vm.prank(anyTrustFastConfirmer);
        userRollup.fastConfirmAssertion(assertionHash, genesisHash, firstState, inboxAccs);
        assertEq(userRollup.latestConfirmed(), assertionHash);
    }

    function testSuccessFastConfirmSkipOne() public {
        (bytes32 prevHash, bytes32 assertionHash, ExecutionState memory afterState, bytes32 inboxAcc) =
            testSuccessCreateSecondAssertion();
        assertEq(userRollup.latestConfirmed() != prevHash, true);
        vm.prank(anyTrustFastConfirmer);
        userRollup.fastConfirmAssertion(assertionHash, prevHash, afterState, inboxAcc);
        assertEq(userRollup.latestConfirmed(), assertionHash);
    }

    function testRevertFastConfirmNotPending() public {
        (bytes32 assertionHash,,) = testSuccessConfirmUnchallengedAssertions();
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.expectRevert("NOT_PENDING");
        vm.prank(anyTrustFastConfirmer);
        userRollup.fastConfirmAssertion(assertionHash, genesisHash, firstState, inboxAccs);
    }

    function testRevertFastConfirmNotConfirmer() public {
        (bytes32 assertionHash,,) = testSuccessCreateAssertion();
        bytes32 inboxAccs = userRollup.bridge().sequencerInboxAccs(0);
        vm.expectRevert("NOT_FAST_CONFIRMER");
        userRollup.fastConfirmAssertion(assertionHash, genesisHash, firstState, inboxAccs);
    }

    function _testFastConfirmNewAssertion(address by, string memory err, bool isCreated)
        internal
        returns (AssertionInputs memory, bytes32)
    {
        uint64 inboxcount = uint64(_createNewBatch());
        ExecutionState memory beforeState;
        beforeState.machineStatus = MachineStatus.FINISHED;
        ExecutionState memory afterState;
        afterState.machineStatus = MachineStatus.FINISHED;
        afterState.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState.globalState.u64Vals[0] = 1; // inbox count
        afterState.globalState.u64Vals[1] = 0; // pos in msg

        bytes32 expectedAssertionHash = RollupLib.assertionHash({
            parentAssertionHash: genesisHash,
            afterState: afterState,
            inboxAcc: userRollup.bridge().sequencerInboxAccs(0)
        });

        AssertionInputs memory assertion = AssertionInputs({
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

        if (isCreated) {
            vm.prank(validator1);
            userRollup.newStakeOnNewAssertion({
                tokenAmount: BASE_STAKE,
                assertion: assertion,
                expectedAssertionHash: expectedAssertionHash
            });
        }

        if (bytes(err).length > 0) {
            vm.expectRevert(bytes(err));
        }
        vm.prank(by);
        userRollup.fastConfirmNewAssertion({assertion: assertion, expectedAssertionHash: expectedAssertionHash});
        if (bytes(err).length == 0) {
            assertEq(userRollup.latestConfirmed(), expectedAssertionHash);
        }
        return (assertion, expectedAssertionHash);
    }

    function testSuccessFastConfirmNewAssertion() public {
        _testFastConfirmNewAssertion(anyTrustFastConfirmer, "", false);
    }

    function testRevertFastConfirmNewAssertionNotConfirmer() public {
        _testFastConfirmNewAssertion(validator1, "NOT_FAST_CONFIRMER", false);
    }

    function testSuccessFastConfirmNewAssertionPending() public {
        _testFastConfirmNewAssertion(anyTrustFastConfirmer, "", true);
    }

    function testRevertFastConfirmNewAssertionConfirmed() public {
        (AssertionInputs memory assertion, bytes32 expectedAssertionHash) =
            _testFastConfirmNewAssertion(anyTrustFastConfirmer, "", true);
        vm.expectRevert("NOT_PENDING");
        vm.prank(anyTrustFastConfirmer);
        userRollup.fastConfirmNewAssertion({assertion: assertion, expectedAssertionHash: expectedAssertionHash});
    }

    bytes32 constant _IMPLEMENTATION_PRIMARY_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
    bytes32 constant _IMPLEMENTATION_SECONDARY_SLOT = 0x2b1dbce74324248c222f0ec2d5ed7bd323cfc425b336f0253c5ccfda7265546d;

    // should only allow admin to upgrade primary logic
    function testRevertUpgradeNotAdmin() public {
        RollupAdminLogic newAdminLogicImpl = new RollupAdminLogic();
        vm.expectRevert();
        adminRollup.upgradeTo(address(newAdminLogicImpl));
    }

    function testRevertUpgradeNotUUPS() public {
        vm.prank(owner);
        vm.expectRevert();
        adminRollup.upgradeTo(address(rollup));
    }

    function testRevertUpgradePrimaryAsSecondary() public {
        RollupAdminLogic newAdminLogicImpl = new RollupAdminLogic();
        vm.prank(owner);
        vm.expectRevert("ERC1967Upgrade: unsupported secondary proxiableUUID");
        adminRollup.upgradeSecondaryTo(address(newAdminLogicImpl));
    }

    function testRevertUpgradeSecondaryAsPrimary() public {
        RollupUserLogic newUserLogicImpl = new RollupUserLogic();
        vm.prank(owner);
        vm.expectRevert("ERC1967Upgrade: unsupported proxiableUUID");
        adminRollup.upgradeTo(address(newUserLogicImpl));
    }

    function testSuccessUpgradePrimary() public {
        address ori_secondary_impl =
            address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_SECONDARY_SLOT))));

        RollupAdminLogic newAdminLogicImpl = new RollupAdminLogic();
        vm.prank(owner);
        adminRollup.upgradeTo(address(newAdminLogicImpl));

        address new_primary_impl = address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_PRIMARY_SLOT))));
        address new_secondary_impl =
            address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_SECONDARY_SLOT))));

        assertEq(address(newAdminLogicImpl), new_primary_impl);
        assertEq(ori_secondary_impl, new_secondary_impl);
    }

    function testSuccessUpgradePrimaryAndCall() public {
        address ori_secondary_impl =
            address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_SECONDARY_SLOT))));

        RollupAdminLogic newAdminLogicImpl = new RollupAdminLogic();
        vm.prank(owner);
        adminRollup.upgradeToAndCall(address(newAdminLogicImpl), abi.encodeCall(adminRollup.pause, ()));
        assertEq(adminRollup.paused(), true);

        address new_primary_impl = address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_PRIMARY_SLOT))));
        address new_secondary_impl =
            address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_SECONDARY_SLOT))));

        assertEq(address(newAdminLogicImpl), new_primary_impl);
        assertEq(ori_secondary_impl, new_secondary_impl);
    }

    function testSuccessUpgradeSecondary() public {
        address ori_primary_impl = address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_PRIMARY_SLOT))));

        RollupUserLogic newUserLogicImpl = new RollupUserLogic();
        vm.prank(owner);
        adminRollup.upgradeSecondaryTo(address(newUserLogicImpl));

        address new_primary_impl = address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_PRIMARY_SLOT))));
        address new_secondary_impl =
            address(uint160(uint256(vm.load(address(userRollup), _IMPLEMENTATION_SECONDARY_SLOT))));

        assertEq(ori_primary_impl, new_primary_impl);
        assertEq(address(newUserLogicImpl), new_secondary_impl);
    }

    function testRevertInitAdminLogicDirectly() public {
        RollupAdminLogic newAdminLogicImpl = new RollupAdminLogic();
        Config memory c;
        ContractDependencies memory cd;
        vm.expectRevert("Function must be called through delegatecall");
        newAdminLogicImpl.initialize(c, cd);
    }

    function testRevertInitUserLogicDirectly() public {
        RollupUserLogic newUserLogicImpl = new RollupUserLogic();
        vm.expectRevert("Function must be called through delegatecall");
        newUserLogicImpl.initialize(address(token));
    }

    function testRevertInitTwice() public {
        Config memory c;
        ContractDependencies memory cd;
        vm.prank(owner);
        vm.expectRevert("Initializable: contract is already initialized");
        adminRollup.initialize(c, cd);
    }

    function testRevertChainIDFork() public {
        ISequencerInbox sequencerInbox = userRollup.sequencerInbox();
        vm.expectRevert(NotForked.selector);
        sequencerInbox.removeDelayAfterFork();
    }

    function testRevertNotBatchPoster() public {
        ISequencerInbox sequencerInbox = userRollup.sequencerInbox();
        vm.expectRevert(NotBatchPoster.selector);
        sequencerInbox.addSequencerL2Batch(0, "0x", 0, IGasRefunder(address(0)), 0, 0);
    }

    function testSuccessSetChallengeManager() public {
        vm.prank(owner);
        adminRollup.setChallengeManager(address(0xdeadbeef));
        assertEq(address(userRollup.challengeManager()), address(0xdeadbeef));
    }

    function testRevertSetChallengeManager() public {
        vm.expectRevert();
        adminRollup.setChallengeManager(address(0xdeadbeef));
    }
}
