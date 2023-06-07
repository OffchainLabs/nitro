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

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

contract RollupTest is Test {
    address constant owner = address(1337);
    address constant sequencer = address(7331);

    address constant validator1 = address(100001);
    address constant validator2 = address(100002);
    address constant validator3 = address(100003);
    address constant loserStakeEscrow = address(200001);

    bytes32 constant WASM_MODULE_ROOT = keccak256("WASM_MODULE_ROOT");
    uint256 constant BASE_STAKE = 10;
    uint64 constant CONFIRM_PERIOD_BLOCKS = 100;

    bytes32 constant FIRST_ASSERTION_BLOCKHASH = keccak256("FIRST_ASSERTION_BLOCKHASH");
    bytes32 constant FIRST_ASSERTION_SENDROOT = keccak256("FIRST_ASSERTION_SENDROOT");

    uint256 constant LAYERZERO_BLOCKEDGE_HEIGHT = 2 ** 5;

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
            address(0),
            address(0)
        );

        Config memory config = Config({
            baseStake: BASE_STAKE,
            chainId: 0,
            confirmPeriodBlocks: uint64(CONFIRM_PERIOD_BLOCKS),
            owner: owner,
            sequencerInboxMaxTimeVariation: ISequencerInbox.MaxTimeVariation({
                delayBlocks: (60 * 60 * 24) / 15,
                futureBlocks: 12,
                delaySeconds: 60 * 60 * 24,
                futureSeconds: 60 * 60
            }),
            stakeToken: address(0),
            wasmModuleRoot: WASM_MODULE_ROOT,
            loserStakeEscrow: loserStakeEscrow,
            genesisBlockNum: 0,
            miniStakeValue: 0,
            layerZeroBlockEdgeHeight: 2 ** 5,
            layerZeroBigStepEdgeHeight: 2 ** 5,
            layerZeroSmallStepEdgeHeight: 2 ** 5
        });

        address expectedRollupAddr = address(
            uint160(
                uint256(keccak256(abi.encodePacked(bytes1(0xd6), bytes1(0x94), address(rollupCreator), bytes1(0x03))))
            )
        );

        vm.expectEmit(true, true, false, false);
        emit RollupCreated(expectedRollupAddr, address(0), address(0), address(0), address(0));
        rollupCreator.createRollup(config, expectedRollupAddr);

        userRollup = RollupUserLogic(address(expectedRollupAddr));
        adminRollup = RollupAdminLogic(address(expectedRollupAddr));
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

        vm.deal(validator1, 1 ether);
        vm.deal(validator2, 1 ether);
        vm.deal(validator3, 1 ether);
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

    function testSucessPause() public {
        vm.prank(owner);
        adminRollup.pause();
    }

    function testConfirmAssertionWhenPaused() public {
        (bytes32 assertionHash, ExecutionState memory state, uint64 inboxcount) = testSuccessCreateAssertions();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 prevPrevAssertionHash = genesisHash;
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(owner);
        adminRollup.pause();
        vm.prank(validator1);
        vm.expectRevert("Pausable: paused");
        userRollup.confirmAssertion(
            assertionHash,
            firstState,
            bytes32(0),
            BeforeStateData({
                wasmRoot: WASM_MODULE_ROOT,
                sequencerBatchAcc: prevInboxAcc,
                prevPrevAssertionHash: prevPrevAssertionHash,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
            })
        );
    }

    function testSucessPauseResume() public {
        testSucessPause();
        vm.prank(owner);
        adminRollup.resume();
    }

    function testSucessERC20Disabled() public {
        assertEq(userRollup.owner(), owner);
        assertEq(userRollup.isERC20Enabled(), false);
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

    function testSuccessCreateAssertions() public returns (bytes32, ExecutionState memory, uint64) {
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
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        testSuccessCreateAssertions();
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
            inboxAcc: userRollup.bridge().sequencerInboxAccs(1) // was 0, move forward 1 on errored state
        });

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
                }),
                beforeState: beforeState,
                afterState: afterState
            }),
            expectedAssertionHash: bytes32(0)
        });

        vm.prank(validator2);
        vm.expectRevert("ASSERTION_SEEN");
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeState: beforeState,
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: bytes32(0),
                    prevPrevAssertionHash: bytes32(0),
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
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
        (bytes32 assertionHash1,,) = testSuccessCreateAssertions();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 prevPrevAssertionHash = genesisHash;
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        vm.expectRevert("CONFIRM_DATA");
        userRollup.confirmAssertion(
            assertionHash1,
            emptyExecutionState,
            bytes32(0),
            BeforeStateData({
                wasmRoot: WASM_MODULE_ROOT,
                sequencerBatchAcc: prevInboxAcc,
                prevPrevAssertionHash: prevPrevAssertionHash,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
            })
        );
    }

    function testSuccessConfirmUnchallengedAssertions() public returns (bytes32, ExecutionState memory, uint64) {
        (bytes32 assertionHash, ExecutionState memory state, uint64 inboxcount) = testSuccessCreateAssertions();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 prevPrevAssertionHash = genesisHash;
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        userRollup.confirmAssertion(
            assertionHash,
            firstState,
            bytes32(0),
            BeforeStateData({
                wasmRoot: WASM_MODULE_ROOT,
                sequencerBatchAcc: prevInboxAcc,
                prevPrevAssertionHash: prevPrevAssertionHash,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
            })
        );
        return (assertionHash, state, inboxcount);
    }

    function testSuccessRemoveWhitelistAfterValidatorAfk(uint256 afkBlocks) public {
        (bytes32 assertionHash,,) = testSuccessConfirmUnchallengedAssertions();
        vm.roll(userRollup.getAssertion(assertionHash).createdAtBlock + userRollup.VALIDATOR_AFK_BLOCKS() + 1);
        userRollup.removeWhitelistAfterValidatorAfk();
    }

    function testRevertRemoveWhitelistAfterValidatorAfk(uint256 afkBlocks) public {
        vm.expectRevert("VALIDATOR_NOT_AFK");
        userRollup.removeWhitelistAfterValidatorAfk();
    }

    function testRevertConfirmSiblingedAssertions() public {
        (,,,,, bytes32 assertionHash,) = testSuccessCreateSecondChild();
        vm.roll(userRollup.getAssertion(genesisHash).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        bytes32 prevPrevAssertionHash = genesisHash;
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        vm.expectRevert("Edge does not exist"); // If there is a sibling, you need to supply a winning edge
        userRollup.confirmAssertion(
            assertionHash,
            firstState,
            bytes32(0),
            BeforeStateData({
                wasmRoot: WASM_MODULE_ROOT,
                sequencerBatchAcc: prevInboxAcc,
                prevPrevAssertionHash: prevPrevAssertionHash,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
            })
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
                    ExecutionStateData(data.beforeState, abi.encode(bytes32(0), bytes32(0), WASM_MODULE_ROOT)),
                    ExecutionStateData(
                        data.afterState1,
                        abi.encode(genesisHash, userRollup.bridge().sequencerInboxAccs(0), WASM_MODULE_ROOT)
                    )
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
                    ExecutionStateData(data.beforeState, abi.encode(bytes32(0), bytes32(0), WASM_MODULE_ROOT)),
                    ExecutionStateData(
                        data.afterState2,
                        abi.encode(genesisHash, userRollup.bridge().sequencerInboxAccs(0), WASM_MODULE_ROOT)
                    )
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
        userRollup.challengeManager().confirmEdgeByTime(data.e1Id, new bytes32[](0));
        bytes32 prevPrevAssertionHash = genesisHash;
        bytes32 prevInboxAcc = userRollup.bridge().sequencerInboxAccs(0);
        vm.prank(validator1);
        userRollup.confirmAssertion(
            data.assertionHash,
            firstState,
            data.e1Id,
            BeforeStateData({
                wasmRoot: WASM_MODULE_ROOT,
                sequencerBatchAcc: prevInboxAcc,
                prevPrevAssertionHash: prevPrevAssertionHash,
                requiredStake: BASE_STAKE,
                challengeManager: address(challengeManager),
                confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
            })
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
        testSuccessCreateAssertions();
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
        testSuccessCreateAssertions();
        vm.prank(validator1);
        vm.expectRevert("STAKE_ACTIVE");
        userRollup.reduceDeposit(1);
    }

    function testSuccessAddToDeposit() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        userRollup.addToDeposit{value: 1}(validator1);
    }

    function testRevertAddToDepositNotValidator() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(sequencer);
        vm.expectRevert("NOT_VALIDATOR");
        userRollup.addToDeposit{value: 1}(validator1);
    }

    function testRevertAddToDepositNotStaker() public {
        testSuccessConfirmEdgeByTime();
        vm.prank(validator1);
        vm.expectRevert("NOT_STAKED");
        userRollup.addToDeposit{value: 1}(sequencer);
    }

    function testSuccessCreateChild() public {
        (bytes32 prevHash, ExecutionState memory beforeState, uint64 prevInboxCount) = testSuccessCreateAssertions();

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
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeStateData: BeforeStateData({
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
                }),
                beforeState: beforeState,
                afterState: afterState
            }),
            expectedAssertionHash: expectedAssertionHash2
        });
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
                    wasmRoot: WASM_MODULE_ROOT,
                    sequencerBatchAcc: prevInboxAcc,
                    prevPrevAssertionHash: genesisHash,
                    requiredStake: BASE_STAKE,
                    challengeManager: address(challengeManager),
                    confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS
                }),
                beforeState: beforeState,
                afterState: afterState
            }),
            expectedAssertionHash: expectedAssertionHash2
        });
    }
}
