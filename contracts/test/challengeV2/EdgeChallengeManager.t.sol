// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/challengeV2/DataEntities.sol";
import "./Utils.sol";
import "../MockAssertionChain.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";
// import "../src/osp/IOneStepProofEntry.sol";
import "./StateTools.sol";
// import "../src/state/GlobalState.sol";
// import "../src/state/Machine.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    function proveOneStep(ExecutionContext calldata, uint256, bytes32, bytes calldata proof)
        external
        view
        returns (bytes32 afterHash)
    {
        return bytes32(proof);
    }
}

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

    bytes32 genesisRoot = MerkleTreeLib.root(ProofUtils.expansionFromLeaves(genesisStates(), 0, 1));

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
        EdgeChallengeManager challengeManager =
            new EdgeChallengeManager(assertionChain, challengePeriodSec, new MockOneStepProofEntry());

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
                edgeType: EdgeType.Block,
                startHistoryRoot: genesisRoot,
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
        ei.challengeManager.confirmEdgeByTime(edgeId, ancestorEdges);

        assertTrue(ei.challengeManager.getEdge(edgeId).status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function testCanConfirmByChildren() public {
        EdgeInitData memory ei = deployAndInit();

        (bytes32[] memory states, bytes32[] memory exp1) = appendRandomStates(genesisStates(), height1);

        bytes32 edge1Id = ei.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: EdgeType.Block,
                startHistoryRoot: genesisRoot,
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp1),
                endHeight: height1,
                claimId: ei.a1
            }),
            "",
            ""
        );

        vm.warp(block.timestamp + 1);

        assertEq(ei.challengeManager.getCurrentTimeUnrivaled(edge1Id), 1, "Edge1 timer");
        {
            (, bytes32[] memory exp2) = appendRandomStates(genesisStates(), height1);
            bytes32 edge2Id = ei.challengeManager.createLayerZeroEdge(
                CreateEdgeArgs({
                    edgeType: EdgeType.Block,
                    startHistoryRoot: genesisRoot,
                    startHeight: 0,
                    endHistoryRoot: MerkleTreeLib.root(exp2),
                    endHeight: height1,
                    claimId: ei.a2
                }),
                "",
                ""
            );

            vm.warp(block.timestamp + 2);
            assertEq(ei.challengeManager.getCurrentTimeUnrivaled(edge1Id), 1, "Edge1 timer");
            assertEq(ei.challengeManager.getCurrentTimeUnrivaled(edge2Id), 0, "Edge2 timer");
        }

        BisectionChildren memory children = bisect(ei.challengeManager, edge1Id, states, 16, states.length - 1);

        vm.warp(challengePeriodSec + 5);

        bytes32[] memory ancestors = new bytes32[](1);
        ancestors[0] = edge1Id;
        ei.challengeManager.confirmEdgeByTime(children.lowerChildId, ancestors);
        ei.challengeManager.confirmEdgeByTime(children.upperChildId, ancestors);
        ei.challengeManager.confirmEdgeByChildren(edge1Id);

        assertTrue(ei.challengeManager.getEdge(edge1Id).status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    function bisect(
        EdgeChallengeManager challengeManager,
        bytes32 edgeId,
        bytes32[] memory states,
        uint256 bisectionSize,
        uint256 endSize
    ) internal returns (BisectionChildren memory) {
        bytes32[] memory middleExp = ProofUtils.expansionFromLeaves(states, 0, bisectionSize + 1);
        bytes32[] memory upperStates = ArrayUtilsLib.slice(states, bisectionSize + 1, endSize + 1);

        (bytes32 lowerChildId, bytes32 upperChildId) = challengeManager.bisectEdge(
            edgeId,
            MerkleTreeLib.root(middleExp),
            abi.encode(middleExp, ProofUtils.generatePrefixProof(bisectionSize + 1, upperStates))
        );

        return BisectionChildren(lowerChildId, upperChildId);
    }

    struct BisectionChildren {
        bytes32 lowerChildId;
        bytes32 upperChildId;
    }

    struct BisectToForkOnlyArgs {
        EdgeChallengeManager challengeManager;
        bytes32 winningId;
        bytes32 losingId;
        bytes32[] winningLeaves;
        bytes32[] losingLeaves;
        bool skipLast;
    }

    function bisectToForkOnly(BisectToForkOnlyArgs memory args)
        internal
        returns (BisectionChildren[6] memory, BisectionChildren[6] memory)
    {
        BisectionChildren[6] memory winningEdges;
        BisectionChildren[6] memory losingEdges;

        winningEdges[5] = BisectionChildren(args.winningId, 0);
        losingEdges[5] = BisectionChildren(args.losingId, 0);

        // height 16
        winningEdges[4] = bisect(
            args.challengeManager, winningEdges[5].lowerChildId, args.winningLeaves, 16, args.winningLeaves.length - 1
        );
        losingEdges[4] = bisect(
            args.challengeManager, losingEdges[5].lowerChildId, args.losingLeaves, 16, args.losingLeaves.length - 1
        );

        // height 8
        winningEdges[3] = bisect(args.challengeManager, winningEdges[4].lowerChildId, args.winningLeaves, 8, 16);
        losingEdges[3] = bisect(args.challengeManager, losingEdges[4].lowerChildId, args.losingLeaves, 8, 16);

        // height 4
        winningEdges[2] = bisect(args.challengeManager, winningEdges[3].lowerChildId, args.winningLeaves, 4, 8);
        losingEdges[2] = bisect(args.challengeManager, losingEdges[3].lowerChildId, args.losingLeaves, 4, 8);

        winningEdges[1] = bisect(args.challengeManager, winningEdges[2].lowerChildId, args.winningLeaves, 2, 4);
        losingEdges[1] = bisect(args.challengeManager, losingEdges[2].lowerChildId, args.losingLeaves, 2, 4);

        // height 2
        winningEdges[0] = bisect(args.challengeManager, winningEdges[1].lowerChildId, args.winningLeaves, 1, 2);
        if (!args.skipLast) {
            losingEdges[0] = bisect(args.challengeManager, losingEdges[1].lowerChildId, args.losingLeaves, 1, 2);
        }

        return (winningEdges, losingEdges);
    }

    function appendRandomStatesBetween(bytes32[] memory currentStates, bytes32 endState, uint256 numStates)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        (bytes32[] memory states, bytes32[] memory exp) = appendRandomStates(currentStates, numStates - 1);
        bytes32[] memory fullStates = ArrayUtilsLib.append(states, endState);
        bytes32[] memory fullExp = MerkleTreeLib.appendLeaf(exp, endState);
        return (fullStates, fullExp);
    }

    function toDynamic(BisectionChildren[6] memory l) internal pure returns (BisectionChildren[] memory) {
        BisectionChildren[] memory d = new BisectionChildren[](6);
        for (uint256 i = 0; i < d.length; i++) {
            d[i] = l[i];
        }
        return d;
    }

    function concat(BisectionChildren[] memory arr1, BisectionChildren[] memory arr2)
        internal
        pure
        returns (BisectionChildren[] memory)
    {
        BisectionChildren[] memory full = new BisectionChildren[](arr1.length + arr2.length);
        for (uint256 i = 0; i < arr1.length; i++) {
            full[i] = arr1[i];
        }
        for (uint256 i = 0; i < arr2.length; i++) {
            full[arr1.length + i] = arr2[i];
        }
        return full;
    }

    function getAncestorsAbove(BisectionChildren[] memory layers, uint256 layer)
        internal
        pure
        returns (bytes32[] memory)
    {
        bytes32[] memory ancestors = new bytes32[](layers.length - 1 - layer);
        for (uint256 i = 0; i < layers.length - layer - 1; i++) {
            ancestors[i] = layers[i + layer + 1].lowerChildId;
        }
        return ancestors;
    }

    function testCanConfirmByClaim() public {
        EdgeInitData memory ei = deployAndInit();

        (bytes32[] memory states1,, BisectionChildren[6] memory edges1,) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(ei.challengeManager, EdgeType.Block, ei.a1, ei.a2, rand.hash(), rand.hash(), false)
        );

        (, bytes32[] memory bigStepExp) = appendRandomStatesBetween(genesisStates(), states1[1], height1);
        bytes32 edge1BigStepId = ei.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: EdgeType.BigStep,
                startHistoryRoot: genesisRoot,
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(bigStepExp),
                endHeight: height1,
                claimId: edges1[0].lowerChildId
            }),
            "",
            ""
        );

        vm.warp(challengePeriodSec + 5);

        ei.challengeManager.confirmEdgeByTime(edge1BigStepId, new bytes32[](0));

        ei.challengeManager.confirmEdgeByClaim(edges1[0].lowerChildId, edge1BigStepId);
        ei.challengeManager.confirmEdgeByTime(edges1[0].upperChildId, getAncestorsAbove(toDynamic(edges1), 0));

        ei.challengeManager.confirmEdgeByChildren(edges1[1].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(edges1[1].upperChildId, getAncestorsAbove(toDynamic(edges1), 1));

        ei.challengeManager.confirmEdgeByChildren(edges1[2].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(edges1[2].upperChildId, getAncestorsAbove(toDynamic(edges1), 2));

        ei.challengeManager.confirmEdgeByChildren(edges1[3].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(edges1[3].upperChildId, getAncestorsAbove(toDynamic(edges1), 3));

        ei.challengeManager.confirmEdgeByChildren(edges1[4].lowerChildId);

        assertTrue(ei.challengeManager.getEdge(edges1[4].lowerChildId).status == EdgeStatus.Confirmed, "Edge confirmed");
    }

    struct CreateEdgesBisectArgs {
        EdgeChallengeManager challengeManager;
        EdgeType eType;
        bytes32 claim1Id;
        bytes32 claim2Id;
        bytes32 endState1;
        bytes32 endState2;
        bool skipLast;
    }

    function createEdgesAndBisectToFork(CreateEdgesBisectArgs memory args)
        internal
        returns (bytes32[] memory, bytes32[] memory, BisectionChildren[6] memory, BisectionChildren[6] memory)
    {
        (bytes32[] memory states1, bytes32[] memory exp1) =
            appendRandomStatesBetween(genesisStates(), args.endState1, height1);
        bytes32 edge1Id = args.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: args.eType,
                startHistoryRoot: genesisRoot,
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp1),
                endHeight: height1,
                claimId: args.claim1Id
            }),
            "",
            ""
        );

        vm.warp(block.timestamp + 1);

        assertEq(args.challengeManager.getCurrentTimeUnrivaled(edge1Id), 1, "Edge1 timer");

        (bytes32[] memory states2, bytes32[] memory exp2) =
            appendRandomStatesBetween(genesisStates(), args.endState2, height1);
        bytes32 edge2Id = args.challengeManager.createLayerZeroEdge(
            CreateEdgeArgs({
                edgeType: args.eType,
                startHistoryRoot: genesisRoot,
                startHeight: 0,
                endHistoryRoot: MerkleTreeLib.root(exp2),
                endHeight: height1,
                claimId: args.claim2Id
            }),
            "",
            ""
        );

        vm.warp(block.timestamp + 2);

        (BisectionChildren[6] memory edges1, BisectionChildren[6] memory edges2) = bisectToForkOnly(
            BisectToForkOnlyArgs(args.challengeManager, edge1Id, edge2Id, states1, states2, args.skipLast)
        );

        return (states1, states2, edges1, edges2);
    }

    function testCanConfirmByClaimSubChallenge() public {
        EdgeInitData memory ei = deployAndInit();

        (
            bytes32[] memory blockStates1,
            bytes32[] memory blockStates2,
            BisectionChildren[6] memory blockEdges1,
            BisectionChildren[6] memory blockEdges2
        ) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(ei.challengeManager, EdgeType.Block, ei.a1, ei.a2, rand.hash(), rand.hash(), false)
        );

        (
            bytes32[] memory bigStepStates1,
            bytes32[] memory bigStepStates2,
            BisectionChildren[6] memory bigStepEdges1,
            BisectionChildren[6] memory bigStepEdges2
        ) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(
                ei.challengeManager,
                EdgeType.BigStep,
                blockEdges1[0].lowerChildId,
                blockEdges2[0].lowerChildId,
                blockStates1[1],
                blockStates2[1],
                false
            )
        );

        (,, BisectionChildren[6] memory smallStepEdges1,) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(
                ei.challengeManager,
                EdgeType.SmallStep,
                bigStepEdges1[0].lowerChildId,
                bigStepEdges2[0].lowerChildId,
                bigStepStates1[1],
                bigStepStates2[1],
                true
            )
        );

        vm.warp(challengePeriodSec + 11);

        BisectionChildren[] memory allWinners =
            concat(concat(toDynamic(smallStepEdges1), toDynamic(bigStepEdges1)), toDynamic(blockEdges1));

        ei.challengeManager.confirmEdgeByTime(allWinners[0].lowerChildId, getAncestorsAbove(allWinners, 0));
        ei.challengeManager.confirmEdgeByTime(allWinners[0].upperChildId, getAncestorsAbove(allWinners, 0));

        ei.challengeManager.confirmEdgeByChildren(allWinners[1].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[1].upperChildId, getAncestorsAbove(allWinners, 1));

        ei.challengeManager.confirmEdgeByChildren(allWinners[2].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[2].upperChildId, getAncestorsAbove(allWinners, 2));

        ei.challengeManager.confirmEdgeByChildren(allWinners[3].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[3].upperChildId, getAncestorsAbove(allWinners, 3));

        ei.challengeManager.confirmEdgeByChildren(allWinners[4].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[4].upperChildId, getAncestorsAbove(allWinners, 4));

        ei.challengeManager.confirmEdgeByChildren(allWinners[5].lowerChildId);

        ei.challengeManager.confirmEdgeByClaim(allWinners[6].lowerChildId, allWinners[5].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[6].upperChildId, getAncestorsAbove(allWinners, 6));

        ei.challengeManager.confirmEdgeByChildren(allWinners[7].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[7].upperChildId, getAncestorsAbove(allWinners, 7));

        ei.challengeManager.confirmEdgeByChildren(allWinners[8].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[8].upperChildId, getAncestorsAbove(allWinners, 8));

        ei.challengeManager.confirmEdgeByChildren(allWinners[9].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[9].upperChildId, getAncestorsAbove(allWinners, 9));

        ei.challengeManager.confirmEdgeByChildren(allWinners[10].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[10].upperChildId, getAncestorsAbove(allWinners, 10));

        ei.challengeManager.confirmEdgeByChildren(allWinners[11].lowerChildId);

        ei.challengeManager.confirmEdgeByClaim(allWinners[12].lowerChildId, allWinners[11].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[12].upperChildId, getAncestorsAbove(allWinners, 12));

        ei.challengeManager.confirmEdgeByChildren(allWinners[13].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[13].upperChildId, getAncestorsAbove(allWinners, 13));

        ei.challengeManager.confirmEdgeByChildren(allWinners[14].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[14].upperChildId, getAncestorsAbove(allWinners, 14));

        ei.challengeManager.confirmEdgeByChildren(allWinners[15].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[15].upperChildId, getAncestorsAbove(allWinners, 15));

        ei.challengeManager.confirmEdgeByChildren(allWinners[16].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[16].upperChildId, getAncestorsAbove(allWinners, 16));

        ei.challengeManager.confirmEdgeByChildren(allWinners[17].lowerChildId);

        assertTrue(
            ei.challengeManager.getEdge(allWinners[14].lowerChildId).status == EdgeStatus.Confirmed, "Edge confirmed"
        );
    }

    function testCanConfirmByOneStep() public {
        EdgeInitData memory ei = deployAndInit();

        (
            bytes32[] memory blockStates1,
            bytes32[] memory blockStates2,
            BisectionChildren[6] memory blockEdges1,
            BisectionChildren[6] memory blockEdges2
        ) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(ei.challengeManager, EdgeType.Block, ei.a1, ei.a2, rand.hash(), rand.hash(), false)
        );

        (
            bytes32[] memory bigStepStates1,
            bytes32[] memory bigStepStates2,
            BisectionChildren[6] memory bigStepEdges1,
            BisectionChildren[6] memory bigStepEdges2
        ) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(
                ei.challengeManager,
                EdgeType.BigStep,
                blockEdges1[0].lowerChildId,
                blockEdges2[0].lowerChildId,
                blockStates1[1],
                blockStates2[1],
                false
            )
        );

        (bytes32[] memory smallStepStates1,, BisectionChildren[6] memory smallStepEdges1,) = createEdgesAndBisectToFork(
            CreateEdgesBisectArgs(
                ei.challengeManager,
                EdgeType.SmallStep,
                bigStepEdges1[0].lowerChildId,
                bigStepEdges2[0].lowerChildId,
                bigStepStates1[1],
                bigStepStates2[1],
                false
            )
        );

        vm.warp(challengePeriodSec + 11);

        BisectionChildren[] memory allWinners =
            concat(concat(toDynamic(smallStepEdges1), toDynamic(bigStepEdges1)), toDynamic(blockEdges1));

        bytes32[] memory firstStates = new bytes32[](2);
        firstStates[0] = smallStepStates1[0];
        firstStates[1] = smallStepStates1[1];

        ei.challengeManager.confirmEdgeByOneStepProof(
            allWinners[0].lowerChildId,
            OneStepData({
                execCtx: ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0))}),
                machineStep: 0,
                beforeHash: firstStates[0],
                proof: abi.encodePacked(firstStates[1])
            }),
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(genesisStates()), 0),
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(firstStates), 1)
        );
        bytes32[] memory above = getAncestorsAbove(allWinners, 0);
        ei.challengeManager.confirmEdgeByTime(allWinners[0].upperChildId, above);
        
        ei.challengeManager.confirmEdgeByChildren(allWinners[1].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[1].upperChildId, getAncestorsAbove(allWinners, 1));

        ei.challengeManager.confirmEdgeByChildren(allWinners[2].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[2].upperChildId, getAncestorsAbove(allWinners, 2));

        ei.challengeManager.confirmEdgeByChildren(allWinners[3].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[3].upperChildId, getAncestorsAbove(allWinners, 3));

        ei.challengeManager.confirmEdgeByChildren(allWinners[4].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[4].upperChildId, getAncestorsAbove(allWinners, 4));

        ei.challengeManager.confirmEdgeByChildren(allWinners[5].lowerChildId);

        ei.challengeManager.confirmEdgeByClaim(allWinners[6].lowerChildId, allWinners[5].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[6].upperChildId, getAncestorsAbove(allWinners, 6));

        ei.challengeManager.confirmEdgeByChildren(allWinners[7].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[7].upperChildId, getAncestorsAbove(allWinners, 7));

        ei.challengeManager.confirmEdgeByChildren(allWinners[8].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[8].upperChildId, getAncestorsAbove(allWinners, 8));

        ei.challengeManager.confirmEdgeByChildren(allWinners[9].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[9].upperChildId, getAncestorsAbove(allWinners, 9));

        ei.challengeManager.confirmEdgeByChildren(allWinners[10].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[10].upperChildId, getAncestorsAbove(allWinners, 10));

        ei.challengeManager.confirmEdgeByChildren(allWinners[11].lowerChildId);

        ei.challengeManager.confirmEdgeByClaim(allWinners[12].lowerChildId, allWinners[11].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[12].upperChildId, getAncestorsAbove(allWinners, 12));

        ei.challengeManager.confirmEdgeByChildren(allWinners[13].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[13].upperChildId, getAncestorsAbove(allWinners, 13));

        ei.challengeManager.confirmEdgeByChildren(allWinners[14].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[14].upperChildId, getAncestorsAbove(allWinners, 14));

        ei.challengeManager.confirmEdgeByChildren(allWinners[15].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[15].upperChildId, getAncestorsAbove(allWinners, 15));

        ei.challengeManager.confirmEdgeByChildren(allWinners[16].lowerChildId);
        ei.challengeManager.confirmEdgeByTime(allWinners[16].upperChildId, getAncestorsAbove(allWinners, 16));

        ei.challengeManager.confirmEdgeByChildren(allWinners[17].lowerChildId);

        assertTrue(
            ei.challengeManager.getEdge(allWinners[17].lowerChildId).status == EdgeStatus.Confirmed, "Edge confirmed"
        );
    }
}
