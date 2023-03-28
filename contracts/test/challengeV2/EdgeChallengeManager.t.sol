// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/DataEntities.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
// import "../src/osp/IOneStepProofEntry.sol";
import "./Utils.sol";
import "./StateTools.sol";
// import "../src/state/GlobalState.sol";
// import "../src/state/Machine.sol";

contract EdgeChallengeManagerTest is Test {
    Random rand = new Random();
    bytes32 genesisBlockHash = rand.hash();
    State genesisState = StateToolsLib.randomState(rand, 7, genesisBlockHash, MachineStatus.RUNNING);
    bytes32 genesisStateHash = StateToolsLib.hash(genesisState);

    function genesisStates() internal view returns (bytes32[] memory) {
        bytes32[] memory genStates = new bytes32[](1);
        genStates[0] = genesisStateHash;
        return genStates;
    }

    uint256 genesisHeight = 2;
    uint256 inboxMsgCountGenesis = 7;
    uint256 inboxMsgCountAssertion = 12;

    bytes32 h1 = rand.hash();
    bytes32 h2 = rand.hash();
    uint256 height1 = 18;

    uint256 miniStakeVal = 1 ether;
    uint256 challengePeriodSec = 1000;

    function appendRandomStates(bytes32[] memory currentStates, uint256 numStates)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory newStates = rand.hashes(numStates);
        bytes32[] memory full = ArrayUtilsLib.concat(currentStates, newStates);
        bytes32[] memory exp = ProofUtils.expansionFromLeaves(full, 0, full.length);

        return (full, exp);
    }

    function deploy() internal returns (MockAssertionChain, EdgeChallengeManager, bytes32) {
        MockAssertionChain assertionChain = new MockAssertionChain();
        EdgeChallengeManager challengeManager = new EdgeChallengeManager(assertionChain, challengePeriodSec);

        bytes32 genesis = assertionChain.addAssertionUnsafe(0, genesisHeight, inboxMsgCountGenesis, genesisStateHash, 0);
        return (assertionChain, challengeManager, genesis);
    }

    struct EdgeInitData {
        MockAssertionChain assertionChain;
        EdgeChallengeManager challengeManager;
        bytes32 genesis;
        bytes32 a1;
        bytes32 a2;
        State a1State;
        State a2State;
    }

    function deployAndInit() internal returns (EdgeInitData memory) {
        (MockAssertionChain assertionChain, EdgeChallengeManager challengeManager, bytes32 genesis) = deploy();

        State memory a1State =
            StateToolsLib.randomState(rand, GlobalStateLib.getInboxPosition(genesisState.gs), h1, MachineStatus.RUNNING);
        State memory a2State =
            StateToolsLib.randomState(rand, GlobalStateLib.getInboxPosition(genesisState.gs), h2, MachineStatus.RUNNING);

        // add one since heights are zero indexed in the history states
        bytes32 a1 = assertionChain.addAssertion(
            genesis, genesisHeight + height1, inboxMsgCountAssertion, StateToolsLib.hash(a1State), 0
        );
        bytes32 a2 = assertionChain.addAssertion(
            genesis, genesisHeight + height1, inboxMsgCountAssertion, StateToolsLib.hash(a2State), 0
        );

        return EdgeInitData({
            assertionChain: assertionChain,
            challengeManager: challengeManager,
            genesis: genesis,
            a1: a1,
            a2: a2,
            a1State: a1State,
            a2State: a2State
        });
    }

    function testCanConfirmPs() public {
        EdgeInitData memory ei = deployAndInit();

        (, bytes32[] memory exp) = appendRandomStates(genesisStates(), height1);

        bytes32 edgeId = ei.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeChallengeType: ChallengeType.Block,
                startHistoryRoot: MerkleTreeLib.root(genesisStates()),
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp),
                endHeight: height1,
                claimId: ei.a1
            }),
            "",
            ""
        );

        vm.warp(challengePeriodSec + 2);

        bytes32[] memory ancestorEdges = new bytes32[](0);
        ei.challengeManager.confirmEdgeByTimer(edgeId, ancestorEdges);

        assertTrue(ei.challengeManager.getEdge(edgeId).status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function testCanConfirmByChildren() public {
        EdgeInitData memory ei = deployAndInit();

        (bytes32[] memory states, bytes32[] memory exp1) = appendRandomStates(genesisStates(), height1);

        bytes32 edge1Id = ei.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeChallengeType: ChallengeType.Block,
                startHistoryRoot: MerkleTreeLib.root(genesisStates()),
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp1),
                endHeight: height1,
                claimId: ei.a1
            }),
            "",
            ""
        );

        vm.warp(block.timestamp + 1);

        assertEq(ei.challengeManager.getCurrentPsTimer(edge1Id), 1, "Edge1 timer");
        {


        (, bytes32[] memory exp2) = appendRandomStates(genesisStates(), height1);
        bytes32 edge2Id = ei.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeChallengeType: ChallengeType.Block,
                startHistoryRoot: MerkleTreeLib.root(genesisStates()),
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp2),
                endHeight: height1,
                claimId: ei.a1
            }),
            "",
            ""
        );

        vm.warp(block.timestamp + 2);
        assertEq(ei.challengeManager.getCurrentPsTimer(edge1Id), 1, "Edge1 timer");
        assertEq(ei.challengeManager.getCurrentPsTimer(edge2Id), 0, "Edge2 timer");

        }
        bytes32[] memory middleExp = ProofUtils.expansionFromLeaves(states, 0, 16);

        (bytes32 lowerChildId, bytes32 upperChildId) = ei.challengeManager.bisectEdge(
            edge1Id,
            MerkleTreeLib.root(middleExp),
            abi.encode(middleExp, ProofUtils.generatePrefixProof(16, ArrayUtilsLib.slice(states, 16, states.length)))
        );

        vm.warp(challengePeriodSec + 5);

        bytes32[] memory ancestors = new bytes32[](1);
        ancestors[0] = edge1Id;
        ei.challengeManager.confirmEdgeByTimer(lowerChildId, ancestors);
        ei.challengeManager.confirmEdgeByTimer(upperChildId, ancestors);
        ei.challengeManager.confirmEdgeByChildren(edge1Id);
    }
}
