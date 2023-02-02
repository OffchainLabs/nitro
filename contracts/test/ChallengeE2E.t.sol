// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "../src/DataEntities.sol";
import "./MockAssertionChain.sol";
import "../src/ChallengeManager.sol";

contract AssertionChainTest is Test {
    function setUp() public {
    }

    function testFace() public {
        MockAssertionChain assertionChain = new MockAssertionChain();

        BlockChallengeManager blockChallengeManager = new BlockChallengeManager(address(assertionChain));
        BigStepChallengeManager bigStepChallengeManager = new BigStepChallengeManager(address(blockChallengeManager));
        SmallStepChallengeManager smallStepChallengeManager = new SmallStepChallengeManager(address(bigStepChallengeManager));

        OneStepProofManager oneStepProofManager = new OneStepProofManager();

        // ChallengeManagers challengeManagers = new ChallengeManagers();
        

    }
}
