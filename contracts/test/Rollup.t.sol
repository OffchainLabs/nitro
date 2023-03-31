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
import "../src/challengeV2/ChallengeManagerImpl.sol";
import "./challengeV2/Utils.sol";

contract RollupTest is Test {
    address constant owner = address(1337);
    address constant sequencer = address(7331);

    address constant validator1 = address(100001);
    address constant validator2 = address(100002);
    address constant validator3 = address(100003);

    bytes32 constant WASM_MODULE_ROOT = keccak256("WASM_MODULE_ROOT");
    uint256 constant BASE_STAKE = 10;
    uint256 constant CONFIRM_PERIOD_BLOCKS = 100;

    bytes32 constant FIRST_ASSERTION_BLOCKHASH = keccak256("FIRST_ASSERTION_BLOCKHASH");
    bytes32 constant FIRST_ASSERTION_SENDROOT = keccak256("FIRST_ASSERTION_SENDROOT");

    RollupProxy rollup;
    RollupUserLogic userRollup;
    RollupAdminLogic adminRollup;
    ChallengeManagerImpl challengeManager;
    Random rand = new Random();

    address[] validators;
    bool[] flags;

    event RollupCreated(
        address indexed rollupAddress,
        address inboxAddress,
        address adminProxy,
        address sequencerInbox,
        address bridge
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
        ChallengeManagerImpl challengeManagerImpl = new ChallengeManagerImpl({
            _assertionChain: IAssertionChain(address(0)),
            _miniStakeValue: 0,
            _challengePeriodSec: 0,
            _oneStepProofEntry: IOneStepProofEntry(address(0))
        });
        BridgeCreator bridgeCreator = new BridgeCreator();
        RollupCreator rollupCreator = new RollupCreator();
        RollupAdminLogic rollupAdminLogicImpl = new RollupAdminLogic();
        RollupUserLogic rollupUserLogicImpl = new RollupUserLogic();

        rollupCreator.setTemplates(
            bridgeCreator,
            oneStepProofEntry,
            challengeManagerImpl,
            rollupAdminLogicImpl,
            rollupUserLogicImpl,
            address(0),
            address(0)
        );

        Config memory config = Config({
            baseStake: BASE_STAKE,
            chainId: 0,
            confirmPeriodBlocks: uint64(CONFIRM_PERIOD_BLOCKS),
            extraChallengeTimeBlocks: 100,
            owner: owner,
            sequencerInboxMaxTimeVariation: ISequencerInbox.MaxTimeVariation({
                delayBlocks: (60 * 60 * 24) / 15,
                futureBlocks: 12,
                delaySeconds: 60 * 60 * 24,
                futureSeconds: 60 * 60
            }),
            stakeToken: address(0),
            wasmModuleRoot: WASM_MODULE_ROOT,
            loserStakeEscrow: address(0),
            genesisBlockNum: 0,
            challengePeriodSeconds: 100,
            miniStakeValue: 1
        });

        address expectedRollupAddr = address(
            uint160(
                uint256(
                    keccak256(
                        abi.encodePacked(
                            bytes1(0xd6),
                            bytes1(0x94),
                            address(rollupCreator),
                            bytes1(0x03)
                        )
                    )
                )
            )
        );

        vm.expectEmit(true, true, false, false);
        emit RollupCreated(expectedRollupAddr, address(0), address(0), address(0), address(0));
        rollupCreator.createRollup(config, expectedRollupAddr);

        userRollup = RollupUserLogic(address(expectedRollupAddr));
        adminRollup = RollupAdminLogic(address(expectedRollupAddr));
        challengeManager = ChallengeManagerImpl(address(userRollup.challengeManager()));

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

        payable(validator1).transfer(1 ether);
        payable(validator2).transfer(1 ether);
        payable(validator3).transfer(1 ether);

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

    function testSuccessCreateAssertions() public {
        uint64 inboxcount = uint64(_createNewBatch());
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
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });

        ExecutionState memory afterState2;
        afterState2.machineStatus = MachineStatus.FINISHED;
        afterState2.globalState.u64Vals[0] = inboxcount;
        vm.roll(block.number + 75);
        vm.prank(validator1);
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeState: afterState,
                afterState: afterState2,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: inboxcount
        });
    }

    function testRevertIdenticalAssertions() public {
        uint64 inboxcount = uint64(_createNewBatch());
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
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });

        vm.prank(validator2);
        vm.expectRevert("ASSERTION_SEEN");
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });
    }

    function testRevertAssertWrongBranch() public {
        uint64 inboxcount = uint64(_createNewBatch());
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
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });

        vm.expectRevert("PREV_STATE_HASH");
        afterState.globalState.u64Vals[1] = 1; // modify the state
        vm.roll(block.number + 75);
        vm.prank(validator1);
        userRollup.stakeOnNewAssertion({
            assertion: AssertionInputs({
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });
    }

    function testSuccessCreateSecondChild()
        public
        returns (
            ExecutionState memory,
            ExecutionState memory,
            ExecutionState memory,
            uint256
        )
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

        // record some genesis state for later use
        uint256 genesisInboxCount = userRollup.bridge().sequencerMessageCount();

        vm.prank(validator1);
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeState: beforeState,
                afterState: afterState,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });


        ExecutionState memory afterState2;
        afterState2.machineStatus = MachineStatus.FINISHED;
        afterState2.globalState.bytes32Vals[0] = FIRST_ASSERTION_BLOCKHASH; // blockhash
        afterState2.globalState.bytes32Vals[1] = FIRST_ASSERTION_SENDROOT; // sendroot
        afterState2.globalState.u64Vals[0] = 1; // inbox count
        afterState2.globalState.u64Vals[1] = 1; // modify the state
        vm.prank(validator2);
        userRollup.newStakeOnNewAssertion{value: BASE_STAKE}({
            assertion: AssertionInputs({
                beforeState: beforeState,
                afterState: afterState2,
                numBlocks: 8
            }),
            expectedAssertionHash: bytes32(0),
            prevAssertionInboxMaxCount: 1
        });

        assertEq(userRollup.getAssertion(0).secondChildBlock, block.number);

        return (beforeState, afterState, afterState2, genesisInboxCount);
    }

    function testRevertConfirmWrongInput() public {
        testSuccessCreateAssertions();
        vm.roll(userRollup.getAssertion(0).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        vm.prank(validator1);
        vm.expectRevert("CONFIRM_DATA");
        userRollup.confirmNextAssertion(bytes32(0), bytes32(0));
    }

    function testSuccessConfirmUnchallengedAssertions() public {
        testSuccessCreateAssertions();
        vm.roll(userRollup.getAssertion(0).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        vm.prank(validator1);
        userRollup.confirmNextAssertion(FIRST_ASSERTION_BLOCKHASH, FIRST_ASSERTION_SENDROOT);
    }

    function testRevertConfirmSiblingedAssertions() public {
        testSuccessCreateSecondChild();
        vm.roll(userRollup.getAssertion(0).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        vm.prank(validator1);
        vm.expectRevert("NO_CHAL");
        userRollup.confirmNextAssertion(FIRST_ASSERTION_BLOCKHASH, FIRST_ASSERTION_SENDROOT);
    }

    function testRevertCreateChallengeSingleChild() public {
        testSuccessCreateAssertions();
        vm.prank(validator1);
        vm.expectRevert("NO_SECOND_CHILD");
        userRollup.createChallenge({
            assertionNum: 0
        });
    }

    function testSuccessCreateChallenge() public {
        testSuccessCreateSecondChild();
        vm.prank(validator1);
        userRollup.createChallenge({
            assertionNum: 0
        });
    }

    function fillStatesInBetween(bytes32 start, bytes32 end, uint256 totalCount) internal returns(bytes32[] memory) {
        bytes32[] memory innerStates = rand.hashes(totalCount - 2);

        bytes32[] memory states = new bytes32[](totalCount);
        states[0] = start;
        for(uint i = 0; i < innerStates.length; i++) {
            states[i + 1] = innerStates[i];
        }
        states[totalCount - 1] = end;

        return states;
    }

    function testSuccessConfirmForPsTimer() public {
        (,ExecutionState memory afterState,,uint256 genesisInboxCount) = testSuccessCreateSecondChild();
        vm.prank(validator1);
        bytes32 challengeId = userRollup.createChallenge({
            assertionNum: 0
        });
        vm.roll(userRollup.getAssertion(0).firstChildBlock + CONFIRM_PERIOD_BLOCKS + 1);
        vm.warp(block.timestamp + CONFIRM_PERIOD_BLOCKS * 15);
        bytes32 h0 = userRollup.getStateHash(userRollup.getAssertionId(0));
        bytes32 h1 = userRollup.getStateHash(userRollup.getAssertionId(1));

        bytes32[] memory states = fillStatesInBetween(h0, h1, 9);
        bytes32 root = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(states, 0, 9));

        bytes32 v1Id = challengeManager.addLeaf{value: 1}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: userRollup.getAssertionId(1),
                height: 8,
                historyRoot: root,
                firstState: h0,
                firstStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), 0),
                lastState: states[states.length - 1],
                lastStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1)
            }),
            abi.encodePacked(h1),
            abi.encode(afterState.globalState, genesisInboxCount, afterState.machineStatus)
        );
        userRollup.challengeManager().confirmForPsTimer(v1Id);
        vm.prank(validator1);
        userRollup.confirmNextAssertion(FIRST_ASSERTION_BLOCKHASH, FIRST_ASSERTION_SENDROOT);
    }

    function testSuccessRejection() public {
        testSuccessConfirmForPsTimer();
        vm.prank(validator1);
        userRollup.rejectNextAssertion(validator2);
    }

    function testRevertRejectionTooRecent() public {
        testSuccessCreateSecondChild();
        vm.prank(validator1);
        vm.expectRevert("CHILD_TOO_RECENT");
        userRollup.rejectNextAssertion(validator2);
    }

    function testRevertRejectionNoUnresolved() public {
        vm.prank(validator1);
        vm.expectRevert("NO_UNRESOLVED");
        userRollup.rejectNextAssertion(validator2);
    }
}
